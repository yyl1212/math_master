package e2etest

import (
	"context"
	"encoding/base64"
	"errors"
	"github.com/yyl1212/math_master/backend/internal/content"
	"github.com/yyl1212/math_master/backend/internal/contentaudit"
	"github.com/yyl1212/math_master/backend/internal/publication"
	"github.com/yyl1212/math_master/backend/internal/question"
	"os"
	"path/filepath"
)

// This scene is reachable only through the existing loopback/token/random-database
// harness. Workflow decisions are technical fixtures, not real independent review.
func (c *questionControls) contentAcceptanceScene(ctx context.Context) error {
	c.release()
	c.holdSave.Store(false)
	if e := resetWorkflow(ctx, c.db, c.accounts, c.admin); e != nil {
		return e
	}
	input, e := contentaudit.LoadDraft(ctx, c.root)
	if e != nil {
		return e
	}
	v, report := content.ValidateAndSeal(input.Catalogue, input.Content, input.AssetsRoot)
	if len(report.Errors) > 0 {
		return errors.New("acceptance fixture content invalid")
	}
	if _, e = c.repo.ImportDraft(ctx, v); e != nil {
		return e
	}
	in := publication.DraftInput{CatalogueVersion: 1, Package: input.Content, AssetBytes: []publication.AssetInput{}, SourceMap: []publication.SourceLink{}}
	for _, a := range input.Content.Assets {
		b, e := os.ReadFile(filepath.Join(input.AssetsRoot, a.Path))
		if e != nil {
			return e
		}
		in.AssetBytes = append(in.AssetBytes, publication.AssetInput{ID: a.ID, Base64: base64.StdEncoding.EncodeToString(b)})
	}
	if e = learningPublishContent(ctx, c.db, c.repo, c.accounts, in); e != nil {
		return e
	}
	author, e := fixtureAccess(ctx, c.accounts, "content_editor", false)
	if e != nil {
		return e
	}
	reviewer, e := fixtureAccess(ctx, c.accounts, "content_reviewer", false)
	if e != nil {
		return e
	}
	manager, e := fixtureAccess(ctx, c.accounts, "content_admin", true)
	if e != nil {
		return e
	}
	next := func(a *publication.Access) error { var e error; *a, e = nextFixtureAccess(*a); return e }
	ids := []string{}
	for _, bank := range input.Questions {
		if e = next(&author); e != nil {
			return e
		}
		d, e := c.repo.CreateQuestionDraft(ctx, author, question.DraftInput{CatalogueVersion: 1, QuestionPackage: bank, SourceMap: []question.SourceLink{}})
		if e != nil {
			return e
		}
		if e = next(&author); e != nil {
			return e
		}
		gate, e := c.repo.ValidateQuestionDraft(ctx, author, d.ID, question.ValidateInput{ExpectedRevision: d.Revision})
		if e != nil || !gate.ReadyToSubmit {
			return errors.New("acceptance fixture question invalid")
		}
		if e = next(&author); e != nil {
			return e
		}
		s, e := c.repo.SubmitQuestionDraft(ctx, author, d.ID, question.SubmitInput{ExpectedRevision: d.Revision, ExpectedDigest: gate.Digest})
		if e != nil {
			return e
		}
		if e = next(&reviewer); e != nil {
			return e
		}
		_, e = c.repo.DecideQuestionReview(ctx, reviewer, s.ID, question.ReviewInput{Decision: "approve", Checks: question.ReviewChecks{Mathematics: true, Explanations: true, Objectives: true, Sources: true, Illustrations: true, Generation: true}, IndependenceNote: "Distinct technical fixture accounts; no actual human independence attested.", GenerationNote: "Every finite instance verified by the original technical workflow.", Note: "Isolated simulation only; P6b real mathematical approval remains pending."})
		if e != nil {
			return e
		}
		ids = append(ids, s.ID)
	}
	var kh string
	if e = c.db.QueryRowContext(ctx, `SELECT snapshot_id FROM publication_heads WHERE singleton`).Scan(&kh); e != nil {
		return e
	}
	p, e := c.repo.PrepareQuestionRelease(ctx, manager, question.PrepareInput{SubmissionIDs: ids, ExpectedKnowledgeHead: &kh, Reason: "Prepare complete original acceptance draft in isolated database."})
	if e != nil {
		return e
	}
	if e = next(&manager); e != nil {
		return e
	}
	_, e = c.repo.ActivateQuestionRelease(ctx, manager, p.ID, question.ActivateInput{ExpectedKnowledgeHead: &kh, ExpectedManifestSHA: p.ManifestSHA, Reason: "Activate only the isolated acceptance workflow simulation."})
	return e
}
