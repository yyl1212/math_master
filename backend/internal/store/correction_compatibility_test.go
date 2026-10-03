package store_test

import (
	"errors"
	"github.com/yyl1212/math_master/backend/internal/assessment"
	"github.com/yyl1212/math_master/backend/internal/correction"
	"github.com/yyl1212/math_master/backend/internal/learning"
	"github.com/yyl1212/math_master/backend/internal/notification"
	"testing"
)

func TestCorrectionCompatibilityRequiredGuardsFailClosed(t *testing.T) {
	for _, mode := range []string{"check", "foreign-key", "unique", "function", "trigger-table", "disabled-trigger", "weakened-check", "unvalidated-fk", "unique-index", "trigger-function"} {
		t.Run(mode, func(t *testing.T) {
			f := newCorrectionFixture(t)
			switch mode {
			case "check":
				f.exec(`ALTER TABLE correction_jobs DROP CONSTRAINT correction_jobs_attempt_check`)
			case "foreign-key":
				f.exec(`ALTER TABLE notification_reads DROP CONSTRAINT notification_reads_notification_id_owner_user_id_fkey`)
			case "unique":
				f.exec(`ALTER TABLE notifications DROP CONSTRAINT notifications_dedup_key_key`)
			case "function":
				f.exec(`DROP FUNCTION correction_equivalent_json(jsonb,jsonb)`)
			case "trigger-table":
				f.exec(`DROP TRIGGER correction_rates_immutable ON correction_rate_limits`)
				f.exec(`CREATE TRIGGER correction_rates_immutable BEFORE UPDATE OR DELETE ON auth_rate_limits FOR EACH ROW EXECUTE FUNCTION correction_immutable()`)
			case "weakened-check":
				f.exec(`ALTER TABLE correction_jobs DROP CONSTRAINT correction_jobs_attempt_check`)
				f.exec(`ALTER TABLE correction_jobs ADD CONSTRAINT correction_jobs_attempt_check CHECK (attempt BETWEEN 0 AND 100)`)
			case "unvalidated-fk":
				f.exec(`ALTER TABLE notification_reads DROP CONSTRAINT notification_reads_notification_id_owner_user_id_fkey`)
				f.exec(`ALTER TABLE notification_reads ADD CONSTRAINT notification_reads_notification_id_owner_user_id_fkey FOREIGN KEY(notification_id,owner_user_id) REFERENCES notifications(id,owner_user_id) NOT VALID`)
			case "unique-index":
				f.exec(`DROP INDEX correction_qualification_once`)
			case "trigger-function":
				f.exec(`DROP TRIGGER correction_rates_immutable ON correction_rate_limits`)
				f.exec(`CREATE TRIGGER correction_rates_immutable BEFORE UPDATE OR DELETE ON correction_rate_limits FOR EACH ROW EXECUTE FUNCTION reject_auth_audit_update()`)
			case "disabled-trigger":
				f.exec(`ALTER TABLE correction_results DISABLE TRIGGER correction_result_basis`)
			}
			if _, e := f.repo.CorrectionPreflight(f.ctx, f.Access("admin_a", false), correction.ListCasesAction); !errors.Is(e, correction.ErrNotConfigured) {
				t.Fatal("correction ignored missing schema guard", mode, e)
			}
			if _, e := f.repo.LearningPreflight(f.ctx, f.Access("learner_a", false), learning.ReadOverviewAction); !errors.Is(e, correction.ErrNotConfigured) {
				t.Fatal("learning opened damaged correction schema", mode, e)
			}
			if _, e := f.repo.ListNotifications(f.ctx, f.Access("learner_a", false), notification.Query{Limit: 50}); !errors.Is(e, correction.ErrNotConfigured) {
				t.Fatal("notifications opened damaged correction schema", mode, e)
			}
		})
	}
}
func TestCorrectionCompatibilityOriginalFailedReceiptAndRollback(t *testing.T) {
	f := newCorrectionFixture(t)
	aid, b := f.cLegacyFailedFour()
	originalAttempt := f.createDiagnostic("learner_b")
	if _, e := f.repo.SubmitAssessment(f.ctx, f.Access("learner_b", false), originalAttempt.Summary.ID, f.answers(originalAttempt, 3)); e != nil {
		t.Fatal(e)
	}
	var receiptsBefore string
	if e := f.db.QueryRow(`SELECT coalesce(jsonb_agg(encode(receipt_bytes,'hex') ORDER BY action,target),'[]')::text FROM learning_idempotency WHERE owner_user_id=$1`, f.ids["learner_b"]).Scan(&receiptsBefore); e != nil {
		t.Fatal(e)
	}
	c := f.registerRule(nil, 1)
	f.cFinishRoots()
	f.cApproveAPI(c.ID)
	f.cRunAll()
	var receiptsAfter string
	if e := f.db.QueryRow(`SELECT coalesce(jsonb_agg(encode(receipt_bytes,'hex') ORDER BY action,target),'[]')::text FROM learning_idempotency WHERE owner_user_id=$1`, f.ids["learner_b"]).Scan(&receiptsAfter); e != nil || receiptsBefore != receiptsAfter {
		t.Fatal("original receipt bytes changed", e)
	}
	original, e := f.repo.ReadAssessmentResult(f.ctx, f.Access("learner_b", false), originalAttempt.Summary.ID)
	if e != nil || original.Outcome != assessment.Failed || original.Score == nil || *original.Score != 3 || original.Passed == nil || *original.Passed {
		t.Fatal("original failed result changed", e)
	}
	if f.count(`SELECT count(*) FROM assessment_results WHERE attempt_id=$1 AND outcome='failed' AND score=3 AND NOT passed`, aid) != 1 || f.count(`SELECT count(*) FROM correction_results WHERE evidence_id=$1 AND status='corrected_passed' AND score=4 AND passed`, aid) != 1 {
		t.Fatal("old score overwritten")
	}
	if _, e := f.db.Exec(`INSERT INTO learning_qualification_events(id,owner_user_id,knowledge_id,knowledge_version,knowledge_sha256,kind,attempt_id) VALUES($1,$2,$3,$4,$5,'diagnostic',$6)`, f.ID(), f.ids["learner_a"], b.OriginalSeal.Knowledge.ID, b.OriginalSeal.Knowledge.Version, b.OriginalSeal.Knowledge.SHA256, aid); e == nil {
		t.Fatal("original failed qualification guard lost")
	}
	if e := downCorrection(t, f.db); e == nil {
		t.Fatal("nonempty correction data rolled down")
	}
	if correctionTableCount(t, f.db) != 10 {
		t.Fatal("partial destructive rollback")
	}
}
