package store_test

import (
	"github.com/yyl1212/math_master/backend/internal/learning"
	"github.com/yyl1212/math_master/backend/internal/publication"
	"testing"
)

func TestCorrectionProcessMaterialAndPathStaticNotices(t *testing.T) {
	for _, kind := range []string{"unit", "path"} {
		t.Run(kind, func(t *testing.T) {
			f := newCorrectionFixture(t)
			target := f.questionInput.QuestionPackage.Templates[0].Units[0].ID
			if kind == "unit" {
				if _, e := f.repo.StartLearning(f.ctx, f.Access("learner_a", false), f.knowledge.ID, f.startInput()); e != nil {
					t.Fatal(e)
				}
			} else {
				path := f.publishLearningGraph()
				target = path.ID
				if _, e := f.repo.EnrollLearningPath(f.ctx, f.Access("learner_a", false), path.ID, learning.EnrollInput{Path: path, ExpectedKnowledgeHead: *f.KHead()}); e != nil {
					t.Fatal(e)
				}
			}
			if _, e := f.repo.WithdrawVersion(f.ctx, f.Access("admin_a", true), publication.WithdrawalInput{Target: publication.WithdrawalTarget{Kind: kind, ID: target, Version: 1}, ExpectedHead: f.KHead(), Reason: "Original material or route withdrawn for a finite notification fixture."}); e != nil {
				t.Fatal(e)
			}
			f.cRunAll()
			typ := "review_material"
			if kind == "path" {
				typ = "path_unavailable"
			}
			if f.count(`SELECT count(*) FROM notifications WHERE type=$1`, typ) != 1 || f.count(`SELECT count(*) FROM correction_results WHERE status='review_material' AND score IS NULL AND passed IS NULL`) != 1 {
				t.Fatal("material/route fabricated grade")
			}
		})
	}
}
