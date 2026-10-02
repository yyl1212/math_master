package e2etest

import (
	"bytes"
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"github.com/yyl1212/math_master/backend/internal/auth"
	"github.com/yyl1212/math_master/backend/internal/content"
	"github.com/yyl1212/math_master/backend/internal/publication"
	"github.com/yyl1212/math_master/backend/internal/question"
	"github.com/yyl1212/math_master/backend/internal/store"
	"os"
	"path/filepath"
)

func fixtureAccess(ctx context.Context, accounts *auth.Service, name string, recent bool) (publication.Access, error) {
	v, d, e := accounts.Context(ctx, auth.Cookies{})
	if e != nil {
		return publication.Access{}, e
	}
	_, session, e := accounts.Login(ctx, auth.Cookies{Preauth: d.SetPreauth}, v.CSRFToken, auth.LoginInput{Username: name, Password: fixturePassword}, "question-fixture-login")
	if e != nil {
		return publication.Access{}, e
	}
	c := auth.Cookies{Session: session.SetSession}
	v, _, e = accounts.Context(ctx, c)
	if e != nil {
		return publication.Access{}, e
	}
	if recent {
		if _, e = accounts.Reauthenticate(ctx, c, v.CSRFToken, auth.ReauthInput{Password: fixturePassword}, "question-fixture-reauth"); e != nil {
			return publication.Access{}, e
		}
	}
	proof, e := auth.DecodeContentProof(c, v.CSRFToken, true)
	if e != nil {
		return publication.Access{}, e
	}
	key, e := auth.NewID(rand.Reader)
	if e != nil {
		return publication.Access{}, e
	}
	return publication.Access{TokenHash: proof.TokenHash, CSRF: proof.CSRF, IdempotencyKey: key, RequestID: "question-fixture"}, nil
}
func nextFixtureAccess(a publication.Access) (publication.Access, error) {
	key, e := auth.NewID(rand.Reader)
	a.IdempotencyKey = key
	return a, e
}

// All source bytes are original technical fixtures. Publication uses real distinct
// accounts and the normal transactional workflow in this fresh random database.
func resetQuestion(ctx context.Context, db *sql.DB, s *store.Store, accounts *auth.Service, admin *auth.AdminService, root string, normal content.ValidatedPackage) error {
	if e := resetWorkflow(ctx, db, accounts, admin); e != nil {
		return e
	}
	if _, e := s.ImportDraft(ctx, normal); e != nil {
		return e
	}
	raw, e := os.ReadFile(filepath.Join(root, "backend/internal/content/testdata/workflow-ready.json"))
	if e != nil {
		return e
	}
	var p content.Package
	if json.Unmarshal(raw, &p) != nil {
		return errors.New("question knowledge fixture invalid")
	}
	p.ID = "e2e-question-knowledge"
	p.Knowledge[0].ID = "e2e-question-fractions"
	p.Units[0].ID = "e2e-question-unit"
	p.Units[0].Knowledge.ID = p.Knowledge[0].ID
	p.Assets[0].Knowledge.ID = p.Knowledge[0].ID
	svg, e := os.ReadFile(filepath.Join(root, "backend/internal/content/testdata/workflow-ready.svg"))
	if e != nil {
		return e
	}
	a, e := fixtureAccess(ctx, accounts, "content_editor", false)
	if e != nil {
		return e
	}
	draft, e := s.CreateDraft(ctx, a, publication.DraftInput{CatalogueVersion: 1, Package: p, AssetBytes: []publication.AssetInput{{ID: p.Assets[0].ID, Base64: base64.StdEncoding.EncodeToString(svg)}}, SourceMap: []publication.SourceLink{}})
	if e != nil {
		return e
	}
	a, e = nextFixtureAccess(a)
	if e != nil {
		return e
	}
	sub, e := s.SubmitDraft(ctx, a, draft.ID, publication.SubmitInput{ExpectedRevision: draft.Revision, ExpectedDigest: draft.Gate.Digest})
	if e != nil {
		return e
	}
	reviewer, e := fixtureAccess(ctx, accounts, "content_reviewer", false)
	if e != nil {
		return e
	}
	_, e = s.DecideReview(ctx, reviewer, sub.ID, publication.ReviewInput{Decision: "approve", Checks: publication.ReviewChecks{Mathematics: true, Explanations: true, Relationships: true, Sources: true, Illustrations: true}, IndependenceNote: "Distinct original technical fixture accounts.", Note: "All original fixture learning content checked independently."})
	if e != nil {
		return e
	}
	manager, e := fixtureAccess(ctx, accounts, "content_admin", true)
	if e != nil {
		return e
	}
	pub, e := s.PrepareRelease(ctx, manager, publication.PrepareInput{SubmissionIDs: []string{sub.ID}, ExpectedHead: nil, Reason: "Prepare original question knowledge in isolated browser database."})
	if e != nil {
		return e
	}
	manager, e = nextFixtureAccess(manager)
	if e != nil {
		return e
	}
	_, e = s.ActivateRelease(ctx, manager, pub.ID, publication.ActivateInput{ExpectedHead: nil, ExpectedManifestSHA: pub.ManifestSHA, Reason: "Activate real approved knowledge only in this random browser database."})
	return e
}
func questionFixtureInput(root string) (question.DraftInput, error) {
	raw, e := os.ReadFile(filepath.Join(root, "content/questions/elementary-rationals.v1.json"))
	if e != nil {
		return question.DraftInput{}, e
	}
	p, e := question.DecodePackage(bytes.NewReader(raw))
	if e != nil {
		return question.DraftInput{}, e
	}
	for i := range p.Templates {
		p.Templates[i].Knowledge = question.Ref{ID: "e2e-question-fractions", Version: 1}
		for j := range p.Templates[i].Coverage {
			p.Templates[i].Coverage[j].Knowledge = p.Templates[i].Knowledge
		}
	}
	for i := range p.Blueprints {
		p.Blueprints[i].Knowledge = question.Ref{ID: "e2e-question-fractions", Version: 1}
	}
	return question.DraftInput{CatalogueVersion: 1, QuestionPackage: p, SourceMap: []question.SourceLink{}}, nil
}
