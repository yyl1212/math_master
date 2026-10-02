package store_test

import (
	"errors"
	"github.com/yyl1212/math_master/backend/internal/learning"
	"github.com/yyl1212/math_master/backend/internal/question"
	"testing"
)

func (f *learningFixture) exposureSequence(actor string) int64 {
	f.t.Helper()
	var n int64
	e := f.db.QueryRow(`SELECT coalesce((SELECT sequence FROM learner_exposure_state WHERE owner_user_id=$1),0)`, f.ids[actor]).Scan(&n)
	if e != nil {
		f.t.Fatal(e)
	}
	return n
}
func TestLearningExposureActiveAttempt(t *testing.T) {
	f := newLearningFixture(t)
	before := f.exposureSequence("author_a")
	_, e := f.repo.ReadQuestionSubmission(f.ctx, f.Access("author_a", false), f.approvedSubmissionID)
	if e != nil || f.exposureSequence("author_a") <= before {
		t.Fatal("answer delivery must advance exposure before returning", e)
	}
}
func TestLearningExposureQuestionEntrypoints(t *testing.T) {
	f := newLearningFixture(t)
	var d question.DraftView
	var gate question.ValidationReport
	var sub question.SubmissionView
	advance := func(name, actor string, fn func() error) {
		t.Helper()
		t.Run(name, func(t *testing.T) {
			n := f.exposureSequence(actor)
			if e := fn(); e != nil {
				t.Fatal(e)
			}
			if f.exposureSequence(actor) <= n {
				t.Fatal("answer entrypoint omitted exposure")
			}
		})
	}
	key := f.Access("author_a", false)
	advance("create", "author_a", func() error { var e error; d, e = f.repo.CreateQuestionDraft(f.ctx, key, f.questionInput); return e })
	advance("create replay", "author_a", func() error { _, e := f.repo.CreateQuestionDraft(f.ctx, key, f.questionInput); return e })
	advance("read draft", "author_a", func() error { _, e := f.repo.ReadQuestionDraft(f.ctx, f.Access("author_a", false), d.ID); return e })
	saveKey := f.Access("author_a", false)
	savedInput := question.SaveDraftInput{DraftInput: f.questionInput, ExpectedRevision: d.Revision}
	advance("save", "author_a", func() error {
		var e error
		d, e = f.repo.SaveQuestionDraft(f.ctx, saveKey, d.ID, savedInput)
		return e
	})
	advance("save replay", "author_a", func() error { _, e := f.repo.SaveQuestionDraft(f.ctx, saveKey, d.ID, savedInput); return e })
	advance("validate", "author_a", func() error {
		var e error
		gate, e = f.repo.ValidateQuestionDraft(f.ctx, f.Access("author_a", false), d.ID, question.ValidateInput{ExpectedRevision: d.Revision})
		return e
	})
	key = f.Access("author_a", false)
	advance("submit", "author_a", func() error {
		var e error
		sub, e = f.repo.SubmitQuestionDraft(f.ctx, key, d.ID, question.SubmitInput{ExpectedRevision: d.Revision, ExpectedDigest: gate.Digest})
		return e
	})
	advance("submit replay", "author_a", func() error {
		_, e := f.repo.SubmitQuestionDraft(f.ctx, key, d.ID, question.SubmitInput{ExpectedRevision: d.Revision, ExpectedDigest: gate.Digest})
		return e
	})
	advance("read submission", "author_a", func() error {
		_, e := f.repo.ReadQuestionSubmission(f.ctx, f.Access("author_a", false), sub.ID)
		return e
	})
	advance("instances page", "reviewer_a", func() error {
		_, e := f.repo.ListQuestionInstances(f.ctx, f.Access("reviewer_a", false), sub.ID, question.ListQuery{Limit: 1})
		return e
	})
	key = f.Access("reviewer_a", false)
	advance("review", "reviewer_a", func() error {
		_, e := f.repo.DecideQuestionReview(f.ctx, key, sub.ID, approvedQuestionInput())
		return e
	})
	advance("review replay", "reviewer_a", func() error {
		_, e := f.repo.DecideQuestionReview(f.ctx, key, sub.ID, approvedQuestionInput())
		return e
	})
	reviseKey := f.Access("author_a", false)
	advance("revise", "author_a", func() error { _, e := f.repo.ReviseQuestionSubmission(f.ctx, reviseKey, sub.ID); return e })
	advance("revise replay", "author_a", func() error { _, e := f.repo.ReviseQuestionSubmission(f.ctx, reviseKey, sub.ID); return e })
	adoptKey := f.Access("author_b", false)
	adoptInput := question.AdoptInput{PackageID: f.questionInput.QuestionPackage.ID, PackageVersion: 1, Reason: "Explicitly adopt original test mathematics."}
	advance("adopt", "author_b", func() error { _, e := f.repo.AdoptQuestionDraft(f.ctx, adoptKey, adoptInput); return e })
	advance("adopt replay", "author_b", func() error { _, e := f.repo.AdoptQuestionDraft(f.ctx, adoptKey, adoptInput); return e })
	n := f.exposureSequence("author_a")
	if _, e := f.repo.ListQuestionDrafts(f.ctx, f.Access("author_a", false), question.ListQuery{}); e != nil {
		t.Fatal(e)
	}
	if _, e := f.repo.ListQuestionSubmissions(f.ctx, f.Access("author_a", false), question.ListQuery{Scope: "mine"}); e != nil {
		t.Fatal(e)
	}
	if f.exposureSequence("author_a") != n {
		t.Fatal("summary-only list counted as answer delivery")
	}
}
func TestLearningExposureMissingTableFailsClosed(t *testing.T) {
	f := newLearningFixture(t)
	f.exec(`ALTER TABLE learner_answer_exposures RENAME TO missing_answer_exposures`)
	if _, e := f.repo.ReadQuestionSubmission(f.ctx, f.Access("author_a", false), f.approvedSubmissionID); !errors.Is(e, learning.ErrNotConfigured) {
		t.Fatal("enabled partial schema delivered answers", e)
	}
}
func TestLearningExposureBeforePublication(t *testing.T) {
	f := newLearningFixture(t)
	id := *f.items[0].Template
	// Draft delivery was recorded before the fixture's actual QActivate.
	var n int
	if e := f.db.QueryRow(`SELECT count(*) FROM learner_answer_exposures WHERE owner_user_id=$1 AND kind='template' AND id=$2 AND version=$3 AND sha256=$4`, f.ids["author_a"], id.ID, id.Version, id.SHA256).Scan(&n); e != nil || n != 1 {
		t.Fatal("prepublication accurate template identity lost", n, e)
	}
}
