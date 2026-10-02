package e2etest

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"github.com/yyl1212/math_master/backend/internal/auth"
	"github.com/yyl1212/math_master/backend/internal/cli"
	"github.com/yyl1212/math_master/backend/internal/content"
	"github.com/yyl1212/math_master/backend/internal/publication"
	"github.com/yyl1212/math_master/backend/internal/question"
	"github.com/yyl1212/math_master/backend/internal/store"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
)

type questionControls struct {
	db             *sql.DB
	repo           *store.Store
	accounts       *auth.Service
	admin          *auth.AdminService
	contentService *publication.Service
	root, workURL  string
	slots          []func()
	holdSave       atomic.Bool
}

func (c *questionControls) release() {
	for _, stop := range c.slots {
		stop()
	}
	c.slots = nil
}
func (c *questionControls) change(ctx context.Context, scene string) (bool, error) {
	switch scene {
	case "question-hold-next-save":
		c.holdSave.Store(true)
		return true, nil
	case "question-hold-slots":
		c.release()
		for n := 0; n < 2; n++ {
			stop, e := c.contentService.AcquireValidation(ctx)
			if e != nil {
				c.release()
				return true, e
			}
			c.slots = append(c.slots, stop)
		}
		return true, nil
	case "question-release-slots":
		c.release()
		return true, nil
	case "question-migration-missing":
		_, e := c.db.ExecContext(ctx, `ALTER TABLE question_idempotency RENAME TO question_missing_fixture`)
		return true, e
	case "question-migration-recover":
		_, e := c.db.ExecContext(ctx, `ALTER TABLE question_missing_fixture RENAME TO question_idempotency`)
		return true, e
	case "question-revoke-editor":
		return true, c.revokeEditor(ctx)
	case "question-advance-knowledge":
		return true, c.advanceKnowledge(ctx)
	case "question-cli-import":
		return true, c.importCLI(ctx)
	}
	return false, nil
}
func (c *questionControls) advanceKnowledge(ctx context.Context) error {
	var id string
	if e := c.db.QueryRowContext(ctx, `SELECT id::text FROM content_submissions WHERE sealed AND status='approved' ORDER BY created_at LIMIT 1`).Scan(&id); e != nil {
		return e
	}
	a, e := fixtureAccess(ctx, c.accounts, "content_admin", true)
	if e != nil {
		return e
	}
	var head string
	if e = c.db.QueryRowContext(ctx, `SELECT snapshot_id FROM publication_heads`).Scan(&head); e != nil {
		return e
	}
	p, e := c.repo.PrepareRelease(ctx, a, publication.PrepareInput{SubmissionIDs: []string{id}, ExpectedHead: &head, Reason: "Advance the real knowledge snapshot for isolated stale-head verification."})
	if e != nil {
		return e
	}
	a, e = nextFixtureAccess(a)
	if e != nil {
		return e
	}
	_, e = c.repo.ActivateRelease(ctx, a, p.ID, publication.ActivateInput{ExpectedHead: &head, ExpectedManifestSHA: p.ManifestSHA, Reason: "Activate unchanged fixed knowledge under a new real approved snapshot."})
	return e
}
func (c *questionControls) revokeEditor(ctx context.Context) error {
	v, d, e := c.accounts.Context(ctx, auth.Cookies{})
	if e != nil {
		return e
	}
	_, session, e := c.accounts.Login(ctx, auth.Cookies{Preauth: d.SetPreauth}, v.CSRFToken, auth.LoginInput{Username: "auth_admin", Password: fixturePassword}, "question-control-login")
	if e != nil {
		return e
	}
	cookies := auth.Cookies{Session: session.SetSession}
	v, _, e = c.accounts.Context(ctx, cookies)
	if e != nil {
		return e
	}
	if _, e = c.accounts.Reauthenticate(ctx, cookies, v.CSRFToken, auth.ReauthInput{Password: fixturePassword}, "question-control-reauth"); e != nil {
		return e
	}
	var id string
	if e = c.db.QueryRowContext(ctx, `SELECT id::text FROM auth_users WHERE username='content_editor'`).Scan(&id); e != nil {
		return e
	}
	_, e = c.admin.ReplaceRoles(ctx, cookies, v.CSRFToken, id, auth.RolesInput{Roles: []auth.Role{auth.RoleLearner}, Reason: "Remove real editor and reviewer responsibilities for this isolated browser test."}, "question-control-roles")
	return e
}
func (c *questionControls) importCLI(ctx context.Context) error {
	u, e := url.Parse(c.workURL)
	if e != nil || !strings.HasPrefix(strings.TrimPrefix(u.Path, "/"), "math_master_test_") {
		return errors.New("unsafe question CLI database")
	}
	var current string
	if e = c.db.QueryRowContext(ctx, `SELECT current_database()`).Scan(&current); e != nil || current != strings.TrimPrefix(u.Path, "/") {
		return errors.New("question CLI database mismatch")
	}
	input, e := questionFixtureInput(c.root)
	if e != nil {
		return e
	}
	refs := question.ReferenceSnapshot{CatalogueVersion: 1, Knowledge: []question.FixedKnowledge{}, Units: []question.FixedUnit{}, Assets: []content.AssetView{}}
	var head string
	if e = c.db.QueryRowContext(ctx, `SELECT snapshot_id FROM publication_heads`).Scan(&head); e != nil {
		return e
	}
	refs.KnowledgeHead = &head
	if e = c.db.QueryRowContext(ctx, `SELECT sha256 FROM catalogue_versions WHERE version=1`).Scan(&refs.CatalogueSHA256); e != nil {
		return e
	}
	var k question.FixedKnowledge
	var goals []byte
	k.Identity.ID = "e2e-question-fractions"
	k.Identity.Version = 1
	if e = c.db.QueryRowContext(ctx, `SELECT sha256,body->>'title',body->>'titleZh',body->'objectives' FROM knowledge_versions WHERE id=$1 AND version=1`, k.Identity.ID).Scan(&k.Identity.SHA256, &k.Title, &k.TitleZh, &goals); e != nil {
		return e
	}
	if json.Unmarshal(goals, &k.Objectives) != nil {
		return errors.New("question CLI objectives unavailable")
	}
	refs.Knowledge = append(refs.Knowledge, k)
	sealed, _, e := question.ValidateAndSeal(ctx, input, refs)
	if e != nil {
		return e
	}
	g, v := question.UsedEngineVersions(sealed.Package, sealed.Instances)
	input.QuestionPackage = sealed.Package
	archive := question.Archive{Envelope: input, PackageSHA: sealed.PackageSHA, Instances: sealed.Instances, GeneratorVersions: g, VerifierVersions: v, SourceResponsibility: question.SourceResponsibility{AuthorIDs: []string{}, LegacyUnattributed: true}}
	dir, e := os.MkdirTemp("", "math-master-question-cli-")
	if e != nil {
		return e
	}
	defer os.RemoveAll(dir)
	raw, e := json.Marshal(archive)
	if e != nil {
		return e
	}
	file := filepath.Join(dir, "archive.json")
	if e = os.WriteFile(file, raw, 0600); e != nil {
		return e
	}
	previous, present := os.LookupEnv("DATABASE_URL")
	if e = os.Setenv("DATABASE_URL", c.workURL); e != nil {
		return e
	}
	defer func() {
		if present {
			_ = os.Setenv("DATABASE_URL", previous)
		} else {
			_ = os.Unsetenv("DATABASE_URL")
		}
	}()
	if cli.RunQuestion(ctx, "import", []string{"--archive", file}, io.Discard, io.Discard) != 0 {
		return errors.New("real question CLI import failed")
	}
	return nil
}

type questionDatabaseState struct {
	Workspaces  int            `json:"workspaces"`
	MaxRevision int64          `json:"maxRevision"`
	Submissions int            `json:"submissions"`
	Reviews     int            `json:"reviews"`
	Published   int            `json:"published"`
	Head        *string        `json:"head"`
	ManifestSHA *string        `json:"manifestSha"`
	Members     int            `json:"members"`
	Withdrawals int            `json:"withdrawals"`
	Events      map[string]int `json:"events"`
}

func (c *questionControls) state(ctx context.Context) (questionDatabaseState, error) {
	out := questionDatabaseState{Events: map[string]int{}}
	var head, sha sql.NullString
	e := c.db.QueryRowContext(ctx, `SELECT (SELECT count(*) FROM question_workspaces),(SELECT coalesce(max(revision),0) FROM question_workspaces),(SELECT count(*) FROM question_submissions),(SELECT count(*) FROM question_review_decisions),(SELECT count(*) FROM question_publications WHERE status='published'),(SELECT publication_id::text FROM question_heads),(SELECT p.manifest_sha256 FROM question_publications p JOIN question_heads h ON h.publication_id=p.id),(SELECT count(*) FROM question_publication_members m JOIN question_heads h ON h.publication_id=m.publication_id),(SELECT count(*) FROM question_withdrawals)`).Scan(&out.Workspaces, &out.MaxRevision, &out.Submissions, &out.Reviews, &out.Published, &head, &sha, &out.Members, &out.Withdrawals)
	if e != nil {
		return out, e
	}
	if head.Valid {
		out.Head = &head.String
	}
	if sha.Valid {
		out.ManifestSHA = &sha.String
	}
	rows, e := c.db.QueryContext(ctx, `SELECT action,count(*) FROM question_events GROUP BY action`)
	if e != nil {
		return out, e
	}
	defer rows.Close()
	for rows.Next() {
		var action string
		var n int
		if e = rows.Scan(&action, &n); e != nil {
			return out, e
		}
		out.Events[action] = n
	}
	return out, rows.Err()
}
