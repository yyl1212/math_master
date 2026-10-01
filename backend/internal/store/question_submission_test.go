package store_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"github.com/yyl1212/math_master/backend/internal/auth"
	"github.com/yyl1212/math_master/backend/internal/question"
	"os"
	"strings"
	"testing"
)

func TestQuestionFrozenSubmission(t *testing.T) {
	f := newQuestionFixture(t)
	d, err := f.repo.CreateQuestionDraft(f.ctx, f.Access("author_a", false), f.questionInput)
	if err != nil {
		t.Fatal(err)
	}
	gate, err := f.repo.ValidateQuestionDraft(f.ctx, f.Access("author_a", false), d.ID, question.ValidateInput{ExpectedRevision: 1})
	if err != nil {
		t.Fatal(err)
	}
	for _, input := range []question.SubmitInput{{ExpectedRevision: 2, ExpectedDigest: gate.Digest}, {ExpectedRevision: 1, ExpectedDigest: strings.Repeat("0", 64)}} {
		if _, err = f.repo.SubmitQuestionDraft(f.ctx, f.Access("author_a", false), d.ID, input); !errors.Is(err, question.ErrDraftConflict) {
			t.Fatal("stale proof accepted", err)
		}
	}
	sub, err := f.repo.SubmitQuestionDraft(f.ctx, f.Access("author_a", false), d.ID, question.SubmitInput{ExpectedRevision: 1, ExpectedDigest: gate.Digest})
	if err != nil {
		t.Fatal(err)
	}
	if sub.Status != "pending" || len(sub.Frozen.InstanceIdentities) != 28 || len(sub.Frozen.AuthorIDs) != 1 || len(sub.Frozen.SourceMap) != 1 || len(sub.Frozen.Objectives) != 1 || len(sub.Frozen.GeneratorVersions) != 1 {
		t.Fatal("incomplete frozen metadata")
	}
	foundUnit, foundAsset := false, false
	for _, r := range sub.Frozen.Resolved {
		foundUnit = foundUnit || r.Kind == "unit"
		foundAsset = foundAsset || r.Kind == "asset" && r.SHA256 == f.questionInput.QuestionPackage.Templates[0].Assets[0].SHA256
	}
	if !foundUnit || !foundAsset {
		t.Fatal("unit/asset SHA omitted")
	}
	var before []byte
	if err = f.db.QueryRow(`SELECT frozen_bytes FROM question_submissions WHERE id=$1`, sub.ID).Scan(&before); err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(before, []byte(`"instances":[`)) {
		t.Fatal("instances not in frozen payload")
	}
	if err = os.WriteFile(f.sourceFile, []byte("A changed source file must not rewrite a submission"), 0600); err != nil {
		t.Fatal(err)
	}
	returned, err := f.repo.DecideQuestionReview(f.ctx, f.Access("reviewer_a", false), sub.ID, question.ReviewInput{Decision: "return", Note: "Clarify source mapping before the next independent review."})
	if err != nil || returned.Status != "returned" {
		t.Fatal(err)
	}
	saved := f.questionInput
	saved.SourceMap = append([]question.SourceLink{}, saved.SourceMap...)
	saved.SourceMap[0].Note = "Corrected metadata for a new review; old frozen source is unchanged."
	if _, err = f.repo.SaveQuestionDraft(f.ctx, f.Access("author_a", false), d.ID, question.SaveDraftInput{DraftInput: saved, ExpectedRevision: 2}); err != nil {
		t.Fatal(err)
	}
	var after []byte
	if err = f.db.QueryRow(`SELECT frozen_bytes FROM question_submissions WHERE id=$1`, sub.ID).Scan(&after); err != nil || !bytes.Equal(before, after) {
		t.Fatal("workspace/file rewrote old frozen bytes")
	}
	newGate, err := f.repo.ValidateQuestionDraft(f.ctx, f.Access("author_a", false), d.ID, question.ValidateInput{ExpectedRevision: 3})
	if err != nil || newGate.Digest == gate.Digest {
		t.Fatal("source changes did not change proof", err)
	}
	if _, err = f.repo.SubmitQuestionDraft(f.ctx, f.Access("author_a", false), d.ID, question.SubmitInput{ExpectedRevision: 3, ExpectedDigest: gate.Digest}); !errors.Is(err, question.ErrDraftConflict) {
		t.Fatal("old source digest accepted", err)
	}
	if _, err = f.repo.SubmitQuestionDraft(f.ctx, f.Access("author_a", false), d.ID, question.SubmitInput{ExpectedRevision: 3, ExpectedDigest: newGate.Digest}); err != nil {
		t.Fatal("corrected source snapshot could not be reviewed", err)
	}
}

func TestQuestionCopiedAuthorCannotApprove(t *testing.T) {
	f := newQuestionFixture(t)
	f.exec(`INSERT INTO auth_user_roles VALUES($1,'reviewer'),($2,'reviewer')`, f.ids["author_a"], f.ids["author_b"])
	original := f.QSubmitted("author_a")
	if _, err := f.repo.DecideQuestionReview(f.ctx, f.Access("author_a", false), original.ID, approvedQuestionInput()); !errors.Is(err, auth.ErrForbidden) {
		t.Fatal("author approved own submission", err)
	}
	adopted, err := f.repo.AdoptQuestionDraft(f.ctx, f.Access("author_b", false), question.AdoptInput{PackageID: original.Frozen.QuestionPackage.ID, PackageVersion: 1, Reason: "Preserve inherited authorship in this isolated adopted draft."})
	if err != nil {
		t.Fatal(err)
	}
	gate, err := f.repo.ValidateQuestionDraft(f.ctx, f.Access("author_b", false), adopted.ID, question.ValidateInput{ExpectedRevision: 1})
	if err != nil {
		t.Fatal(err)
	}
	copy, err := f.repo.SubmitQuestionDraft(f.ctx, f.Access("author_b", false), adopted.ID, question.SubmitInput{ExpectedRevision: 1, ExpectedDigest: gate.Digest})
	if err != nil || len(copy.Frozen.AuthorIDs) != 2 {
		t.Fatal("adopt erased original author", err)
	}
	for _, actor := range []string{"author_a", "author_b"} {
		if _, err = f.repo.DecideQuestionReview(f.ctx, f.Access(actor, false), copy.ID, approvedQuestionInput()); !errors.Is(err, auth.ErrForbidden) {
			t.Fatal("copy author approved", actor, err)
		}
	}
	across := f.questionInput
	across.QuestionPackage.ID = "copied-bank"
	d, err := f.repo.CreateQuestionDraft(f.ctx, f.Access("author_b", false), across)
	if err != nil {
		t.Fatal(err)
	}
	gate, err = f.repo.ValidateQuestionDraft(f.ctx, f.Access("author_b", false), d.ID, question.ValidateInput{ExpectedRevision: 1})
	if err != nil {
		t.Fatal(err)
	}
	sub, err := f.repo.SubmitQuestionDraft(f.ctx, f.Access("author_b", false), d.ID, question.SubmitInput{ExpectedRevision: 1, ExpectedDigest: gate.Digest})
	if err != nil || len(sub.Frozen.AuthorIDs) != 2 {
		t.Fatal("cross-package authors lost", err)
	}
	if _, err = f.repo.DecideQuestionReview(f.ctx, f.Access("author_a", false), sub.ID, approvedQuestionInput()); !errors.Is(err, auth.ErrForbidden) {
		t.Fatal("cross-package original author approved", err)
	}
	changed := f.questionInput
	changed.QuestionPackage.ID = "changed-copy-bank"
	changed.QuestionPackage.Templates = append([]question.Template{}, changed.QuestionPackage.Templates...)
	changed.QuestionPackage.Templates[0].ExplanationTemplate += " This is a different fixed version body."
	bad, err := f.repo.CreateQuestionDraft(f.ctx, f.Access("author_b", false), changed)
	if err != nil {
		t.Fatal(err)
	}
	badGate, err := f.repo.ValidateQuestionDraft(f.ctx, f.Access("author_b", false), bad.ID, question.ValidateInput{ExpectedRevision: 1})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = f.repo.SubmitQuestionDraft(f.ctx, f.Access("author_b", false), bad.ID, question.SubmitInput{ExpectedRevision: 1, ExpectedDigest: badGate.Digest}); !errors.Is(err, question.ErrVersionConflict) {
		t.Fatal("changed member reused same version", err)
	}
	if f.count(`SELECT count(*) FROM question_packages WHERE id='changed-copy-bank'`) != 0 {
		t.Fatal("conflicting package partially inserted")
	}
	pending, err := f.repo.ListQuestionSubmissions(f.ctx, f.Access("author_a", false), question.ListQuery{Scope: "review"})
	if err != nil || pending.Total != 0 {
		t.Fatal("review queue includes author", err)
	}
}

func TestQuestionInstancePages(t *testing.T) {
	f := newQuestionFixture(t)
	sub := f.QSubmitted("author_a")
	seen := map[string]bool{}
	offset := 0
	for {
		page, err := f.repo.ListQuestionInstances(f.ctx, f.Access("reviewer_a", false), sub.ID, question.ListQuery{Limit: 7, Offset: offset})
		if err != nil || page.Total != 28 {
			t.Fatal(err)
		}
		raw, _ := json.Marshal(page)
		if len(raw) > question.MaxResponseBytes {
			t.Fatal("oversized instance response")
		}
		for _, i := range page.Items {
			if seen[i.Identity.ID] {
				t.Fatal("duplicate page item")
			}
			seen[i.Identity.ID] = true
			if err = question.VerifyInstance(i); err != nil {
				t.Fatal("frozen instance failed verifier", err)
			}
		}
		offset += page.Limit
		if offset >= page.Total {
			break
		}
	}
	if len(seen) != 28 {
		t.Fatal("partial instance pages")
	}
	if _, err := f.repo.ListQuestionInstances(f.ctx, f.Access("author_b", false), sub.ID, question.ListQuery{}); !errors.Is(err, auth.ErrNotFound) {
		t.Fatal("unbound submission exposed", err)
	}
	if _, err := f.db.Exec(`INSERT INTO question_submission_authors VALUES($1,$2)`, sub.ID, f.ids["author_b"]); err == nil {
		t.Fatal("post-freeze author inserted")
	}
	if _, err := f.db.Exec(`INSERT INTO question_submission_members SELECT * FROM question_submission_members WHERE submission_id=$1 LIMIT 1`, sub.ID); err == nil {
		t.Fatal("post-freeze member inserted")
	}
}

func TestQuestionFrozenResponsibilityChange(t *testing.T) {
	f := newQuestionFixture(t)
	d, err := f.repo.CreateQuestionDraft(f.ctx, f.Access("author_b", false), f.questionInput)
	if err != nil {
		t.Fatal(err)
	}
	gate, err := f.repo.ValidateQuestionDraft(f.ctx, f.Access("author_b", false), d.ID, question.ValidateInput{ExpectedRevision: 1})
	if err != nil {
		t.Fatal(err)
	}
	f.QSubmitted("author_a")
	if _, err = f.repo.SubmitQuestionDraft(f.ctx, f.Access("author_b", false), d.ID, question.SubmitInput{ExpectedRevision: 1, ExpectedDigest: gate.Digest}); !errors.Is(err, question.ErrDraftConflict) {
		t.Fatal("stale authorship proof accepted", err)
	}
	current, err := f.repo.ValidateQuestionDraft(f.ctx, f.Access("author_b", false), d.ID, question.ValidateInput{ExpectedRevision: 1})
	if err != nil || current.Digest == gate.Digest {
		t.Fatal("responsibility omitted from proof", err)
	}
	sub, err := f.repo.SubmitQuestionDraft(f.ctx, f.Access("author_b", false), d.ID, question.SubmitInput{ExpectedRevision: 1, ExpectedDigest: current.Digest})
	if err != nil || len(sub.Frozen.AuthorIDs) != 2 {
		t.Fatal("current authorship not frozen", err)
	}
}
