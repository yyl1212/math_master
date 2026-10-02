package store_test

import (
	"github.com/yyl1212/math_master/backend/internal/question"
	"testing"
)

func TestQuestionExportsEditableBodyExposure(t *testing.T) {
	f := newLearningFixture(t)
	n := f.exposureSequence("author_a")
	sub, e := f.repo.ReadQuestionSubmission(f.ctx, f.Access("author_a", false), f.approvedSubmissionID)
	if e != nil || len(sub.Frozen.QuestionPackage.Templates) != 1 || f.exposureSequence("author_a") <= n {
		t.Fatal("editable export read omitted exposure", e)
	}
	n = f.exposureSequence("author_a")
	archive, e := f.repo.ExportQuestionArchive(f.ctx, sub.Frozen.QuestionPackage.ID, 1)
	if e != nil || len(archive.Envelope.QuestionPackage.Templates) != 1 || f.exposureSequence("author_a") != n {
		t.Fatal("trusted offline export fabricated personal actor", e)
	}
	f.QWithdraw(question.WithdrawalTarget{Kind: "template", ID: "lf-addition", Version: 1})
	n = f.exposureSequence("admin_a")
	if _, e = f.repo.ReadQuestionSubmission(f.ctx, f.Access("admin_a", false), sub.ID); e != nil || f.exposureSequence("admin_a") <= n {
		t.Fatal("administrator cannot audit original withdrawn mathematics", e)
	}
}
