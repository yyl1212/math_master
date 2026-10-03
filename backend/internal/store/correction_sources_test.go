package store_test

import (
	"errors"
	"github.com/yyl1212/math_master/backend/internal/assessment"
	"github.com/yyl1212/math_master/backend/internal/correction"
	"strings"
	"testing"
)

func TestCorrectionWithdrawalFenceRealSource(t *testing.T) {
	f := newCorrectionFixture(t)
	v := f.createDiagnostic("learner_a")
	if _, e := f.repo.SubmitAssessment(f.ctx, f.Access("learner_a", false), v.Summary.ID, f.answers(v, 5)); e != nil {
		t.Fatal(e)
	}
	wid := f.legacyCorrectionWithdrawal(v.Questions[0].Instance)
	input := correction.CaseInput{Kind: correction.WithdrawalCase, Withdrawal: &correction.WithdrawalRef{Space: "question", ID: wid}}
	registered, e := f.repo.CreateCorrectionCase(f.ctx, f.Access("admin_a", false), input)
	if e != nil || registered.Data.Case == nil || registered.Data.Case.Withdrawal.ID != wid || registered.Data.Case.Cutoff != nil {
		t.Fatal(registered, e)
	}
	if _, e = f.repo.CreateCorrectionCase(f.ctx, f.Access("admin_a", false), input); !errors.Is(e, correction.ErrConflict) {
		t.Fatal("duplicate withdrawal case", e)
	}
	input.Withdrawal.Space = "content"
	if _, e = f.repo.CreateCorrectionCase(f.ctx, f.Access("admin_a", false), input); !errors.Is(e, correction.ErrSourceStale) {
		t.Fatal("wrong source namespace accepted", e)
	}
	input.Withdrawal.Space = "question"
	input.Withdrawal.ID = f.ID()
	if _, e = f.repo.CreateCorrectionCase(f.ctx, f.Access("admin_a", false), input); !errors.Is(e, correction.ErrSourceStale) {
		t.Fatal("invented withdrawal accepted", e)
	}
	r, e := f.repo.ReadAssessmentResult(f.ctx, f.Access("learner_a", false), v.Summary.ID)
	if e != nil || r.Validity != assessment.Restricted || r.Items[0].Explanation != nil {
		t.Fatal("withdrawal not immediate", e)
	}
}

func TestCorrectionWithdrawalFenceExactSHAAndSharedAsset(t *testing.T) {
	for _, kind := range []string{"instance", "asset"} {
		t.Run(kind, func(t *testing.T) {
			f := newCorrectionFixture(t)
			v := f.createDiagnostic("learner_a")
			if _, e := f.repo.SubmitAssessment(f.ctx, f.Access("learner_a", false), v.Summary.ID, f.answers(v, 5)); e != nil {
				t.Fatal(e)
			}
			wid := f.ID()
			space := "question"
			if kind == "instance" {
				f.exec(`INSERT INTO question_withdrawals(id,kind,target_id,target_version,sha256,actor_user_id,reason,request_id) VALUES($1,'instance',$2,1,$3,$4,'Isolated malformed SHA proof negative test.','correction-test')`, wid, v.Questions[0].Instance.ID, strings.Repeat("f", 64), f.ids["admin_a"])
			} else {
				space = "content"
				asset := f.questionInput.QuestionPackage.Templates[0].Assets[0]
				f.exec(`INSERT INTO content_withdrawals(id,kind,sha256,actor_user_id,reason,request_id) VALUES($1,'asset',$2,$3,'This digest affects every asset ID containing the same bytes.','correction-test')`, wid, asset.SHA256, f.ids["admin_a"])
			}
			_, e := f.repo.CreateCorrectionCase(f.ctx, f.Access("admin_a", false), correction.CaseInput{Kind: correction.WithdrawalCase, Withdrawal: &correction.WithdrawalRef{Space: space, ID: wid}})
			if kind == "instance" && !errors.Is(e, correction.ErrSourceStale) {
				t.Fatal("false SHA registered", e)
			}
			if kind == "asset" && e != nil {
				t.Fatal("real asset digest rejected", e)
			}
			r, e := f.repo.ReadAssessmentResult(f.ctx, f.Access("learner_a", false), v.Summary.ID)
			if e != nil {
				t.Fatal(e)
			}
			if kind == "instance" && r.Validity != assessment.Effective {
				t.Fatal("false SHA invalidated evidence")
			}
			if kind == "asset" && r.Validity != assessment.Restricted {
				t.Fatal("asset digest did not restrict across IDs")
			}
		})
	}
}
