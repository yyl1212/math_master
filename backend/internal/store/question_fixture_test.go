package store_test

import (
	"crypto/sha256"
	"database/sql"
	"fmt"
	"github.com/yyl1212/math_master/backend/internal/question"
	"os"
	"path/filepath"
	"testing"
)

type questionFixture struct {
	*workflowFixture
	questionInput question.DraftInput
	sourceFile    string
}

func newQuestionFixture(t *testing.T) *questionFixture {
	f := &questionFixture{workflowFixture: newWorkflowFixture(t)}
	f.Activate(f.Prepare(f.Approved("author_a", "reviewer_a"), nil), nil)
	f.questionInput = questionDraftInput(t)
	p := f.Input().Package
	f.questionInput.QuestionPackage.Templates[0].Units = []question.Ref{{ID: p.Units[0].ID, Version: 1}}
	f.questionInput.QuestionPackage.Templates[0].Assets = []question.AssetRef{{ID: p.Assets[0].ID, SHA256: p.Assets[0].SHA256}}
	raw := []byte(`{"note":"Original technical question source snapshot"}`)
	f.sourceFile = filepath.Join(t.TempDir(), "source.json")
	if err := os.WriteFile(f.sourceFile, raw, 0600); err != nil {
		t.Fatal(err)
	}
	sha := fmt.Sprintf("%x", sha256.Sum256(raw))
	f.questionInput.SourceMap = []question.SourceLink{{Knowledge: question.Ref{ID: "workflow-fractions", Version: 1}, BatchSHA256: sha, RelativePath: "questions/source.json", SHA256: sha, LegacyID: "technical-question-source", Note: "Original technical snapshot used only in random test databases."}}
	return f
}
func (f *questionFixture) QSubmitted(owner string) question.SubmissionView {
	f.t.Helper()
	d, err := f.repo.CreateQuestionDraft(f.ctx, f.Access(owner, false), f.questionInput)
	if err != nil {
		f.t.Fatal(err)
	}
	gate, err := f.repo.ValidateQuestionDraft(f.ctx, f.Access(owner, false), d.ID, question.ValidateInput{ExpectedRevision: d.Revision})
	if err != nil || !gate.ReadyToSubmit {
		f.t.Fatal("question not ready", err)
	}
	sub, err := f.repo.SubmitQuestionDraft(f.ctx, f.Access(owner, false), d.ID, question.SubmitInput{ExpectedRevision: d.Revision, ExpectedDigest: gate.Digest})
	if err != nil {
		f.t.Fatal(err)
	}
	return sub
}
func approvedQuestionInput() question.ReviewInput {
	return question.ReviewInput{Decision: "approve", Checks: question.ReviewChecks{Mathematics: true, Explanations: true, Objectives: true, Sources: true, Illustrations: true, Generation: true}, IndependenceNote: "Different isolated author and reviewer accounts.", GenerationNote: "Every finite instance and independent verifier were checked.", Note: "All six requirements reviewed on the complete technical payload."}
}

func (f *questionFixture) QApproved(owner, reviewer string) question.SubmissionView {
	f.t.Helper()
	sub := f.QSubmitted(owner)
	out, err := f.repo.DecideQuestionReview(f.ctx, f.Access(reviewer, false), sub.ID, approvedQuestionInput())
	if err != nil {
		f.t.Fatal(err)
	}
	return out
}
func (f *questionFixture) QHead() *string {
	f.t.Helper()
	var id string
	err := f.db.QueryRow(`SELECT publication_id::text FROM question_heads WHERE singleton`).Scan(&id)
	if err == sql.ErrNoRows {
		return nil
	}
	if err != nil {
		f.t.Fatal(err)
	}
	return &id
}
func (f *questionFixture) KHead() *string {
	f.t.Helper()
	var id string
	if err := f.db.QueryRow(`SELECT snapshot_id FROM publication_heads WHERE singleton`).Scan(&id); err != nil {
		f.t.Fatal(err)
	}
	return &id
}
func (f *questionFixture) QPrepare(subs ...string) question.PublicationSummary {
	f.t.Helper()
	out, err := f.repo.PrepareQuestionRelease(f.ctx, f.Access("admin_a", false), question.PrepareInput{SubmissionIDs: subs, ExpectedKnowledgeHead: f.KHead(), ExpectedQuestionHead: f.QHead(), Reason: "Prepare the fixed question evidence in an isolated test."})
	if err != nil {
		f.t.Fatal(err)
	}
	return out
}
func (f *questionFixture) QActivate(p question.PublicationSummary) question.PublicationSummary {
	f.t.Helper()
	out, err := f.repo.ActivateQuestionRelease(f.ctx, f.Access("admin_a", true), p.ID, question.ActivateInput{ExpectedKnowledgeHead: p.BaseKnowledgeHead, ExpectedQuestionHead: p.BaseQuestionHead, ExpectedManifestSHA: p.ManifestSHA, Reason: "Explicitly activate this fixed evidence in an isolated test."})
	if err != nil {
		f.t.Fatal(err)
	}
	return out
}
