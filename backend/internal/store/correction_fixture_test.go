package store_test

import (
	"database/sql"
	"encoding/json"
	"github.com/yyl1212/math_master/backend/internal/assessment"
	"github.com/yyl1212/math_master/backend/internal/correction"
	"github.com/yyl1212/math_master/backend/internal/question"
	"testing"
	"time"
)

var correctionTableNames = []string{"correction_cases", "correction_plans", "correction_events", "correction_jobs", "correction_results", "correction_dependencies", "correction_idempotency", "correction_rate_limits", "notifications", "notification_reads"}

func correctionTableCount(t *testing.T, db *sql.DB) int {
	t.Helper()
	var n int
	if e := db.QueryRow(`SELECT count(*) FROM information_schema.tables WHERE table_schema='public' AND table_name=ANY($1::text[])`, correctionTableNames).Scan(&n); e != nil {
		t.Fatal(e)
	}
	return n
}

type correctionFixture struct{ *learningFixture }

func newCorrectionFixture(t *testing.T) *correctionFixture {
	t.Helper()
	f := &correctionFixture{newLearningFixture(t)}
	if correctionTableCount(t, f.db) != 10 {
		t.Fatal("ten correction tables required")
	}
	return f
}
func corrJSON(v any) string {
	b, e := json.Marshal(v)
	if e != nil {
		panic(e)
	}
	return string(b)
}
func (f *correctionFixture) cEvent(tx *sql.Tx, subject, id, kind, actor, caseID string, version *int, sequence int64, value any) error {
	raw, h, e := correction.Canonical("correction-command-v1", value)
	if e != nil {
		return e
	}
	var actorID any
	if actor != "" {
		actorID = actor
	}
	_, e = tx.Exec(`INSERT INTO correction_events(id,subject_kind,subject_id,subject_version,case_id,kind,sequence,actor_user_id,body,body_bytes,body_digest) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)`, f.ID(), subject, id, version, caseID, kind, sequence, actorID, string(raw), raw, h)
	return e
}
func (f *correctionFixture) cCase() string {
	f.t.Helper()
	id := f.ID()
	tx, e := f.db.Begin()
	if e != nil {
		f.t.Fatal(e)
	}
	defer tx.Rollback()
	var now time.Time
	if e = tx.QueryRow(`SELECT clock_timestamp()`).Scan(&now); e != nil {
		f.t.Fatal(e)
	}
	_, e = tx.Exec(`INSERT INTO correction_cases(id,kind,rule_version,scope_kind,cutoff,creator_user_id,created_at) VALUES($1,'grading_rule',1,'all',$2,$3,$2)`, id, now, f.ids["admin_a"])
	if e != nil {
		f.t.Fatal(e)
	}
	if e = f.cEvent(tx, "case", id, "case_registered", f.ids["admin_a"], id, nil, 1, map[string]any{"caseId": id}); e != nil {
		f.t.Fatal(e)
	}
	_, e = tx.Exec(`UPDATE correction_cases SET sealed=true WHERE id=$1`, id)
	if e != nil {
		f.t.Fatal(e)
	}
	if e = tx.Commit(); e != nil {
		f.t.Fatal(e)
	}
	return id
}
func (f *correctionFixture) cDraft(caseID string) correction.PlanRef {
	f.t.Helper()
	p := correction.PlanRef{ID: f.ID(), Version: 1}
	tx, e := f.db.Begin()
	if e != nil {
		f.t.Fatal(e)
	}
	defer tx.Rollback()
	in := correction.PlanInput{AlgorithmVersion: 1, Mappings: []correction.Mapping{}, Reason: "Original independent correction fixture."}
	_, e = tx.Exec(`INSERT INTO correction_plans(id,version,case_id,creator_user_id,body) VALUES($1,1,$2,$3,$4)`, p.ID, caseID, f.ids["author_a"], corrJSON(in))
	if e != nil {
		f.t.Fatal(e)
	}
	if e = f.cEvent(tx, "plan", p.ID, "plan_created", f.ids["author_a"], caseID, &p.Version, 1, in); e != nil {
		f.t.Fatal(e)
	}
	if e = tx.Commit(); e != nil {
		f.t.Fatal(e)
	}
	return p
}
func (f *correctionFixture) cPending(caseID string, p correction.PlanRef) error {
	tx, e := f.db.Begin()
	if e != nil {
		return e
	}
	defer tx.Rollback()
	var input correction.PlanInput
	var raw []byte
	if e = tx.QueryRow(`SELECT body FROM correction_plans WHERE id=$1 AND version=$2`, p.ID, p.Version).Scan(&raw); e != nil {
		return e
	}
	if e = json.Unmarshal(raw, &input); e != nil {
		return e
	}
	proof := correction.PlanProof{CaseID: caseID, AlgorithmVersion: 1, Mappings: []correction.ResolvedMapping{}, Authors: []string{f.ids["author_a"]}, ContentApprovalIDs: []string{}, QuestionApprovalIDs: []string{}}
	frozen, h, e := correction.Canonical("correction-plan-v1", struct {
		Input correction.PlanInput `json:"input"`
		Proof correction.PlanProof `json:"proof"`
	}{input, proof})
	if e != nil {
		return e
	}
	if e = f.cEvent(tx, "plan", p.ID, "plan_submitted", f.ids["author_a"], caseID, &p.Version, 2, map[string]any{"digest": h}); e != nil {
		return e
	}
	_, e = tx.Exec(`UPDATE correction_plans SET status='pending',sequence=2,sealed=true,frozen_body=$3,frozen_bytes=$4,frozen_digest=$5 WHERE id=$1 AND version=$2`, p.ID, p.Version, string(frozen), frozen, h)
	if e != nil {
		return e
	}
	return tx.Commit()
}

func (f *correctionFixture) cSubmittedBasis(actor string) (string, correction.Basis) {
	f.t.Helper()
	v := f.createDiagnostic(actor)
	if _, e := f.repo.SubmitAssessment(f.ctx, f.Access(actor, false), v.Summary.ID, f.answers(v, 5)); e != nil {
		f.t.Fatal(e)
	}
	return v.Summary.ID, f.cLoadBasis(v.Summary.ID)
}
func (f *correctionFixture) cLoadBasis(evidenceID string) correction.Basis {
	f.t.Helper()
	b := correction.Basis{OriginalAnswers: []assessment.Answer{}, OriginalItems: []question.Instance{}, EffectiveItems: []question.Instance{}, HandledCaseIDs: []string{}, ParentResultIDs: []string{}, PlanRefs: []correction.PlanRef{}, AuditDeps: []correction.Dependency{}, EffectiveDeps: []correction.Dependency{}}
	var raw []byte
	if e := f.db.QueryRow(`SELECT seal->'body' FROM assessment_attempts WHERE id=$1`, evidenceID).Scan(&raw); e != nil {
		f.t.Fatal(e)
	}
	if e := json.Unmarshal(raw, &b.OriginalSeal); e != nil {
		f.t.Fatal(e)
	}
	for _, binding := range b.OriginalSeal.Items {
		var i question.Instance
		if e := f.db.QueryRow(`SELECT body->'body' FROM question_instances WHERE id=$1 AND version=$2 AND sha256=$3`, binding.Instance.ID, binding.Instance.Version, binding.Instance.SHA256).Scan(&raw); e != nil {
			f.t.Fatal(e)
		}
		if e := json.Unmarshal(raw, &i); e != nil {
			f.t.Fatal(e)
		}
		i.Identity = binding.Instance
		b.OriginalItems = append(b.OriginalItems, i)
	}
	b.EffectiveItems = append(b.EffectiveItems, b.OriginalItems...)
	rows, e := f.db.Query(`SELECT answer FROM assessment_answers WHERE attempt_id=$1 ORDER BY position`, evidenceID)
	if e != nil {
		f.t.Fatal(e)
	}
	for rows.Next() {
		var a assessment.Answer
		if e = rows.Scan(&raw); e != nil {
			f.t.Fatal(e)
		}
		if e = json.Unmarshal(raw, &a); e != nil {
			f.t.Fatal(e)
		}
		b.OriginalAnswers = append(b.OriginalAnswers, a)
	}
	if e = rows.Err(); e != nil {
		f.t.Fatal(e)
	}
	rows.Close()
	rows, e = f.db.Query(`SELECT kind,id,version,sha256 FROM learning_evidence_dependencies WHERE evidence_kind='assessment' AND evidence_id=$1 ORDER BY kind,id`, evidenceID)
	if e != nil {
		f.t.Fatal(e)
	}
	for rows.Next() {
		var d correction.Dependency
		var ver sql.NullInt64
		if e = rows.Scan(&d.Kind, &d.ID, &ver, &d.SHA256); e != nil {
			f.t.Fatal(e)
		}
		if ver.Valid {
			v := int(ver.Int64)
			d.Version = &v
		}
		b.EffectiveDeps = append(b.EffectiveDeps, d)
	}
	if e = rows.Err(); e != nil {
		f.t.Fatal(e)
	}
	rows.Close()
	b.AuditDeps = append(b.AuditDeps, b.EffectiveDeps...)
	return b
}
func (f *correctionFixture) cResult(evidenceID, owner, caseID string, p correction.PlanRef, b correction.Basis, withEvent bool, omitKnowledge ...bool) (string, error) {
	id := f.ID()
	source := "schema:" + id
	tx, e := f.db.Begin()
	if e != nil {
		return id, e
	}
	defer tx.Rollback()
	jobID := f.ID()
	_, e = tx.Exec(`INSERT INTO correction_jobs(id,source_key,case_id,plan_id,plan_version,type) VALUES($1,$2,$3,$4,$5,'approved_plan')`, jobID, source, caseID, p.ID, p.Version)
	if e != nil {
		return id, e
	}
	if e = f.cEvent(tx, "job", jobID, "job_created", "", caseID, nil, 1, map[string]any{"state": "queued"}); e != nil {
		return id, e
	}
	raw, h, e := correction.Canonical("correction-result-v1", b)
	if e != nil {
		return id, e
	}
	verdict, e := correction.Evaluate(b)
	if e != nil {
		return id, e
	}
	var kid any = b.OriginalSeal.Knowledge.ID
	var kv any = b.OriginalSeal.Knowledge.Version
	var kh any = b.OriginalSeal.Knowledge.SHA256
	if len(omitKnowledge) > 0 && omitKnowledge[0] {
		kid = nil
		kv = nil
		kh = nil
	}
	_, e = tx.Exec(`INSERT INTO correction_results(id,owner_user_id,case_id,plan_id,plan_version,evidence_kind,evidence_id,knowledge_id,knowledge_version,knowledge_sha256,status,reason,score,passed,correctness,handled_case_ids,basis,basis_bytes,basis_digest,source_key) VALUES($1,$2,$3,$4,$5,'assessment',$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19)`, id, owner, caseID, p.ID, p.Version, evidenceID, kid, kv, kh, verdict.Status, verdict.Reason, verdict.Score, verdict.Passed, corrJSON(verdict.Correct), corrJSON(b.HandledCaseIDs), string(raw), raw, h, source)
	if e != nil {
		return id, e
	}
	for _, role := range []string{"audit", "effective"} {
		deps := b.AuditDeps
		if role == "effective" {
			deps = b.EffectiveDeps
		}
		for _, d := range deps {
			positions := []int{}
			for n, i := range b.EffectiveItems {
				if d.Kind == "instance" && d.ID == i.Identity.ID || d.Kind == "template" && i.Template != nil && d.ID == i.Template.ID {
					positions = append(positions, n+1)
				}
			}
			_, e = tx.Exec(`INSERT INTO correction_dependencies(result_id,role,kind,id,version,sha256,positions) VALUES($1,$2,$3,$4,$5,$6,$7)`, id, role, d.Kind, d.ID, d.Version, d.SHA256, positions)
			if e != nil {
				return id, e
			}
		}
	}
	if withEvent {
		if e = f.cEvent(tx, "result", id, "result_sealed", "", caseID, nil, 1, map[string]any{"digest": h}); e != nil {
			return id, e
		}
	}
	if _, e = tx.Exec(`UPDATE correction_results SET sealed=true WHERE id=$1`, id); e != nil {
		return id, e
	}
	return id, tx.Commit()
}
