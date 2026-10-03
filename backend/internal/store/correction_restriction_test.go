package store_test

import (
	"context"
	"crypto/rand"
	"errors"
	"github.com/pressly/goose/v3"
	"github.com/yyl1212/math_master/backend/internal/assessment"
	"github.com/yyl1212/math_master/backend/internal/auth"
	"github.com/yyl1212/math_master/backend/internal/correction"
	"github.com/yyl1212/math_master/backend/internal/learning"
	"github.com/yyl1212/math_master/backend/internal/question"
	"github.com/yyl1212/math_master/backend/internal/store"
	"github.com/yyl1212/math_master/backend/internal/testutil"
	"os"
	"testing"
	"time"
)

func TestCorrectionCasesPracticeStopsGradingAllowsAbandon(t *testing.T) {
	f := newCorrectionFixture(t)
	p := f.createPractice("learner_a")
	f.registerRule(nil, 1)
	if _, e := f.repo.AnswerPractice(f.ctx, f.Access("learner_a", false), p.Summary.ID, assessment.Answer{Kind: "numeric", Raw: stringPointer("0")}); !errors.Is(e, learning.ErrVersionStale) {
		t.Fatal("practice graded under issue", e)
	}
	if _, e := f.repo.RevealPractice(f.ctx, f.Access("learner_a", false), p.Summary.ID); !errors.Is(e, learning.ErrVersionStale) {
		t.Fatal("practice revealed under issue", e)
	}
	if p, e := f.repo.AbandonPractice(f.ctx, f.Access("learner_a", false), p.Summary.ID); e != nil || p.Summary.State != "abandoned" {
		t.Fatal("abandon blocked", e)
	}
	if _, e := f.repo.CreatePractice(f.ctx, f.Access("learner_b", false), f.practiceInput()); !errors.Is(e, learning.ErrAssessmentNotReady) {
		t.Fatal("new practice opened", e)
	}
	if f.count(`SELECT count(*) FROM practice_attempts WHERE answer IS NOT NULL`) != 0 {
		t.Fatal("restricted answer stored")
	}
}
func TestCorrectionPartialConfigNeverEnabledLegacy(t *testing.T) {
	db := testutil.Database(t)
	ctx := context.Background()
	provider, e := goose.NewProvider(goose.DialectPostgres, db, os.DirFS("../../../db/migrations"))
	if e != nil {
		t.Fatal(e)
	}
	if _, e = provider.UpTo(ctx, 7); e != nil {
		t.Fatal(e)
	}
	repo := store.New(db)
	service, e := auth.NewService(repo, auth.NewArgon2Hasher(rand.Reader), rand.Reader)
	if e != nil {
		t.Fatal(e)
	}
	fixture := &authFixture{t, db, repo, service, ctx}
	_, cookies, csrf := fixture.signup("legacy_learner")
	proof, e := auth.DecodeContentProof(cookies, csrf, true)
	if e != nil {
		t.Fatal(e)
	}
	a := question.Access{TokenHash: proof.TokenHash, CSRF: proof.CSRF}
	if correctionTableCount(t, db) != 0 {
		t.Fatal("legacy database unexpectedly migrated eight")
	}
	if _, e = repo.ReadLearningOverview(ctx, a); e != nil {
		t.Fatal("never-enabled learning failed", e)
	}
	if _, e = repo.CorrectionPreflight(ctx, a, correction.ListOwnAction); !errors.Is(e, correction.ErrNotConfigured) {
		t.Fatal("legacy correction enabled", e)
	}
}
func waitCorrectionSQL(t *testing.T, f *correctionFixture, pattern string) {
	t.Helper()
	deadline := time.Now().Add(650 * time.Millisecond)
	for time.Now().Before(deadline) {
		var waiting bool
		if e := f.db.QueryRow(`SELECT EXISTS(SELECT 1 FROM pg_stat_activity WHERE datname=current_database() AND pid<>pg_backend_pid() AND wait_event_type='Lock' AND query LIKE $1)`, pattern).Scan(&waiting); e != nil {
			t.Fatal(e)
		}
		if waiting {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("request did not reach expected lock")
}
func TestCorrectionCutoffRegistrationCommitFence(t *testing.T) {
	for _, registrationFirst := range []bool{true, false} {
		name := "learner-first"
		if registrationFirst {
			name = "registration-first"
		}
		t.Run(name, func(t *testing.T) {
			f := newCorrectionFixture(t)
			v := f.createDiagnostic("learner_a")
			in := f.answers(v, 5)
			admin, learner := f.Access("admin_a", false), f.Access("learner_a", false)
			block, e := f.db.Begin()
			if e != nil {
				t.Fatal(e)
			}
			defer block.Rollback()
			var id string
			query := `SELECT id::text FROM assessment_attempts WHERE id=$1 FOR UPDATE`
			target := v.Summary.ID
			if registrationFirst {
				query = `SELECT id::text FROM auth_users WHERE id=$1 FOR UPDATE`
				target = f.ids["admin_a"]
			}
			if e = block.QueryRow(query, target).Scan(&id); e != nil {
				t.Fatal(e)
			}
			caseDone := make(chan error, 1)
			resultDone := make(chan assessment.ResultView, 1)
			resultError := make(chan error, 1)
			register := func() { _, e := f.repo.CreateCorrectionCase(f.ctx, admin, ruleCaseInput(nil, 1)); caseDone <- e }
			submit := func() {
				r, e := f.repo.SubmitAssessment(f.ctx, learner, v.Summary.ID, in)
				resultDone <- r
				resultError <- e
			}
			if registrationFirst {
				go register()
				waitCorrectionSQL(t, f, "%FROM auth_users%FOR UPDATE%")
				go submit()
				waitCorrectionSQL(t, f, "%pg_advisory_xact_lock_shared%")
			} else {
				go submit()
				waitCorrectionSQL(t, f, "%FROM assessment_attempts WHERE id=%")
				go register()
				waitCorrectionSQL(t, f, "%pg_advisory_xact_lock($1)%")
			}
			if e = block.Commit(); e != nil {
				t.Fatal(e)
			}
			if e = <-caseDone; e != nil {
				t.Fatal(e)
			}
			result := <-resultDone
			if e = <-resultError; e != nil {
				t.Fatal(e)
			}
			if registrationFirst && result.Outcome != assessment.Affected {
				t.Fatal("pre-registration snapshot granted after case commit", result)
			}
			if !registrationFirst && result.Outcome != assessment.Passed {
				t.Fatal("earlier qualification did not precede case", result)
			}
			d, e := f.repo.ReadLearningKnowledge(f.ctx, f.Access("learner_a", false), f.knowledge.ID, 1)
			if e != nil || d.State.Qualification != nil {
				t.Fatal("current projection missed committed case", e)
			}
		})
	}
}
func TestCorrectionCommitIdentityRoleRevokedDuringLockWait(t *testing.T) {
	f := newCorrectionFixture(t)
	a := f.Access("admin_a", false)
	block, e := f.db.Begin()
	if e != nil {
		t.Fatal(e)
	}
	defer block.Rollback()
	var id string
	if e = block.QueryRow(`SELECT id::text FROM auth_users WHERE id=$1 FOR UPDATE`, f.ids["admin_a"]).Scan(&id); e != nil {
		t.Fatal(e)
	}
	done := make(chan error, 1)
	go func() { _, e := f.repo.CreateCorrectionCase(f.ctx, a, ruleCaseInput(nil, 1)); done <- e }()
	waitCorrectionSQL(t, f, "%FROM auth_users%FOR UPDATE%")
	if _, e = block.Exec(`DELETE FROM auth_user_roles WHERE user_id=$1 AND role='admin'`, id); e != nil {
		t.Fatal(e)
	}
	if e = block.Commit(); e != nil {
		t.Fatal(e)
	}
	if e = <-done; !errors.Is(e, auth.ErrForbidden) {
		t.Fatal("stale role survived lock wait", e)
	}
	if f.count(`SELECT count(*) FROM correction_cases`) != 0 || f.count(`SELECT count(*) FROM correction_rate_limits`) != 0 {
		t.Fatal("identity failure committed facts")
	}
}
