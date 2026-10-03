package store_test

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"github.com/yyl1212/math_master/backend/internal/auth"
	"github.com/yyl1212/math_master/backend/internal/content"
	"github.com/yyl1212/math_master/backend/internal/publication"
	"os"
	"strings"
	"testing"
)

type workflowFixture struct {
	*authFixture
	ids    map[string]string
	access map[string]publication.Access
}

func newWorkflowFixture(t *testing.T, initialMigration ...int) *workflowFixture {
	t.Helper()
	f := &workflowFixture{authFixture: newAuthFixture(t, initialMigration...), ids: map[string]string{}, access: map[string]publication.Access{}}
	if _, err := f.repo.ImportDraft(f.ctx, input(t, nil)); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"author_a", "author_b", "reviewer_a", "reviewer_b", "admin_a"} {
		u, cookies, csrf := f.signup(name)
		f.ids[name] = u.ID
		role := "editor"
		if strings.HasPrefix(name, "reviewer") {
			role = "reviewer"
		}
		if strings.HasPrefix(name, "admin") {
			role = "admin"
		}
		f.exec(`INSERT INTO auth_user_roles(user_id,role) VALUES($1,$2)`, u.ID, role)
		proof, err := auth.DecodeContentProof(cookies, csrf, true)
		if err != nil {
			t.Fatal(err)
		}
		f.access[name] = publication.Access{TokenHash: proof.TokenHash, CSRF: proof.CSRF, RequestID: "workflow-fixture"}
	}
	return f
}
func (f *workflowFixture) Access(name string, recent bool) publication.Access {
	f.t.Helper()
	a, ok := f.access[name]
	if !ok {
		f.t.Fatal("unknown fixture actor")
	}
	a.IdempotencyKey = f.ID()
	if recent {
		f.exec(`UPDATE auth_sessions SET reauthenticated_at=clock_timestamp() WHERE token_hash=$1`, a.TokenHash[:])
	}
	return a
}
func (f *workflowFixture) ID() string {
	f.t.Helper()
	id, err := auth.NewID(rand.Reader)
	if err != nil {
		f.t.Fatal(err)
	}
	return id
}
func (f *workflowFixture) Input() publication.DraftInput {
	f.t.Helper()
	raw, err := os.ReadFile("../content/testdata/workflow-ready.json")
	if err != nil {
		f.t.Fatal(err)
	}
	var p content.Package
	if json.Unmarshal(raw, &p) != nil {
		f.t.Fatal("fixture decode failed")
	}
	p.Knowledge[0].ID = "workflow-fractions"
	p.Units[0].ID = "workflow-fractions-unit"
	p.Units[0].Knowledge.ID = p.Knowledge[0].ID
	p.Assets[0].Knowledge.ID = p.Knowledge[0].ID
	svg, err := os.ReadFile("../content/testdata/workflow-ready.svg")
	if err != nil {
		f.t.Fatal(err)
	}
	return publication.DraftInput{CatalogueVersion: 1, Package: p, AssetBytes: []publication.AssetInput{{ID: p.Assets[0].ID, Base64: base64.StdEncoding.EncodeToString(svg)}}, SourceMap: []publication.SourceLink{{Knowledge: content.VersionRef{ID: p.Knowledge[0].ID, Version: 1}, BatchSHA256: strings.Repeat("a", 64), RelativePath: "foundations/fractions.json", SHA256: strings.Repeat("b", 64), LegacyID: "technical-fixture", Note: "A source mapping used only by technical tests."}}}
}
func (f *workflowFixture) Submitted(owner string) publication.SubmissionView {
	f.t.Helper()
	a := f.Access(owner, false)
	d, err := f.repo.CreateDraft(f.ctx, a, f.Input())
	if err != nil {
		f.t.Fatal(err)
	}
	a = f.Access(owner, false)
	sub, err := f.repo.SubmitDraft(f.ctx, a, d.ID, publication.SubmitInput{ExpectedRevision: d.Revision, ExpectedDigest: d.Gate.Digest})
	if err != nil {
		f.t.Fatal(err)
	}
	return sub
}

// Task 3 uses direct, isolated SQL review fixtures until Task 4 implements decisions.
func (f *workflowFixture) DecideFixture(sub publication.SubmissionView, reviewer, decision string) {
	f.t.Helper()
	tx, err := f.db.BeginTx(f.ctx, nil)
	if err != nil {
		f.t.Fatal(err)
	}
	defer tx.Rollback()
	if _, err = tx.ExecContext(f.ctx, `INSERT INTO content_review_decisions(id,submission_id,reviewer_user_id,frozen_digest,decision,checks,independence_note,note) VALUES($1,$2,$3,$4,$5,'{"mathematics":true,"explanations":true,"relationships":true,"sources":true,"illustrations":true}','Independent technical fixture accounts.','This is an isolated technical state fixture.')`, f.ID(), sub.ID, f.ids[reviewer], sub.Frozen.FrozenDigest, decision); err != nil {
		f.t.Fatal(err)
	}
	status := "approved"
	if decision == "return" {
		status = "returned"
	}
	if _, err = tx.ExecContext(f.ctx, `UPDATE content_submissions SET status=$2 WHERE id=$1`, sub.ID, status); err != nil {
		f.t.Fatal(err)
	}
	if decision == "return" {
		if _, err = tx.ExecContext(f.ctx, `UPDATE content_workspaces SET status='editing',revision=revision+1 WHERE id=$1`, sub.WorkspaceID); err != nil {
			f.t.Fatal(err)
		}
	}
	if err = tx.Commit(); err != nil {
		f.t.Fatal(err)
	}
}
func newMaterial(v publication.DraftInput) publication.DraftInput {
	v.Package.Version++
	for i := range v.Package.Knowledge {
		v.Package.Knowledge[i].Version++
	}
	for i := range v.Package.Units {
		v.Package.Units[i].Version++
		v.Package.Units[i].Knowledge.Version++
	}
	for i := range v.Package.Assets {
		v.Package.Assets[i].Knowledge.Version++
	}
	for i := range v.SourceMap {
		v.SourceMap[i].Knowledge.Version++
	}
	return v
}

func (f *workflowFixture) Approved(owner, reviewer string) publication.SubmissionView {
	f.t.Helper()
	sub := f.Submitted(owner)
	out, err := f.repo.DecideReview(f.ctx, f.Access(reviewer, false), sub.ID, approvedReviewInput())
	if err != nil {
		f.t.Fatal(err)
	}
	return out
}
