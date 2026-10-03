package store_test

import (
	"errors"
	"github.com/yyl1212/math_master/backend/internal/auth"
	"github.com/yyl1212/math_master/backend/internal/correction"
	"github.com/yyl1212/math_master/backend/internal/learning"
	"testing"
)

func TestCorrectionCommitIdentityFreshReplay(t *testing.T) {
	for _, mode := range []string{"role", "credential", "password", "csrf"} {
		t.Run(mode, func(t *testing.T) {
			f := newCorrectionFixture(t)
			a := f.Access("admin_a", false)
			in := ruleCaseInput(nil, 1)
			if _, e := f.repo.CreateCorrectionCase(f.ctx, a, in); e != nil {
				t.Fatal(e)
			}
			var want error
			switch mode {
			case "role":
				f.exec(`DELETE FROM auth_user_roles WHERE user_id=$1 AND role='admin'`, f.ids["admin_a"])
				want = auth.ErrForbidden
			case "credential":
				f.exec(`UPDATE auth_users SET credential_version=credential_version+1 WHERE id=$1`, f.ids["admin_a"])
				want = auth.ErrAuthenticationRequired
			case "password":
				f.exec(`UPDATE auth_users SET must_change_password=true WHERE id=$1`, f.ids["admin_a"])
				want = auth.ErrPasswordChangeRequired
			case "csrf":
				a.CSRF[0] ^= 1
				want = auth.ErrCSRF
			}
			if _, e := f.repo.CreateCorrectionCase(f.ctx, a, in); !errors.Is(e, want) {
				t.Fatal("stale identity replay allowed", e, want)
			}
			if f.count(`SELECT count(*) FROM correction_cases`) != 1 {
				t.Fatal("failed identity wrote")
			}
		})
	}
}
func TestCorrectionPartialConfigFailsClosed(t *testing.T) {
	for _, mode := range []string{"table", "version", "column", "trigger", "empty-down"} {
		t.Run(mode, func(t *testing.T) {
			f := newCorrectionFixture(t)
			switch mode {
			case "table":
				f.exec(`DROP TABLE notification_reads`)
			case "version":
				f.exec(`DELETE FROM goose_db_version WHERE version_id=8`)
			case "column":
				f.exec(`ALTER TABLE correction_jobs DROP COLUMN error_class`)
			case "trigger":
				f.exec(`DROP TRIGGER correction_case_registered ON correction_cases`)
			case "empty-down":
				if e := downCorrection(t, f.db); e != nil {
					t.Fatal(e)
				}
			}
			if _, e := f.repo.CorrectionPreflight(f.ctx, f.Access("admin_a", false), correction.ListCasesAction); !errors.Is(e, correction.ErrNotConfigured) {
				t.Fatal("correction opened partial schema", e)
			}
			if _, e := f.repo.ReadLearningOverview(f.ctx, f.Access("learner_a", false)); !errors.Is(e, correction.ErrNotConfigured) {
				t.Fatal("learning ignored enabled correction schema", e)
			}
			if _, e := f.repo.LearningPreflight(f.ctx, f.Access("learner_a", false), learning.ReadOverviewAction); !errors.Is(e, correction.ErrNotConfigured) {
				t.Fatal("preflight ignored enabled schema", e)
			}
		})
	}
}
