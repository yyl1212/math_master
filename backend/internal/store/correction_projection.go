package store

import (
	"context"
	"database/sql"
	"errors"
	"github.com/yyl1212/math_master/backend/internal/assessment"
	"github.com/yyl1212/math_master/backend/internal/learning"
	"github.com/yyl1212/math_master/backend/internal/question"
)

func learningInvalidateProjection(ctx context.Context, tx *sql.Tx, owner string, k question.Identity) {
	if p := learningProjectionFor(ctx, tx, owner); p != nil {
		delete(p.evidence, k)
		delete(p.passes, k)
	}
}
func correctionApprovedResultSQL(r string) string {
	return r + `.sealed AND ` + r + `.plan_id IS NOT NULL AND EXISTS(SELECT 1 FROM correction_plans p WHERE p.id=` + r + `.plan_id AND p.version=` + r + `.plan_version AND p.status='approved' AND p.sealed AND p.algorithm_version=1 AND EXISTS(SELECT 1 FROM correction_events v WHERE v.subject_kind='plan' AND v.subject_id=p.id AND v.subject_version=p.version AND v.kind='plan_approved'))`
}
func correctionLeafSQL(r string) string {
	return `NOT EXISTS(SELECT 1 FROM correction_results child WHERE child.owner_user_id=` + r + `.owner_user_id AND child.evidence_kind=` + r + `.evidence_kind AND child.evidence_id=` + r + `.evidence_id AND child.id<>` + r + `.id AND ` + correctionApprovedResultSQL("child") + ` AND (child.parent_result_id=` + r + `.id OR child.basis#>'{body,parentResultIds}' @> jsonb_build_array(` + r + `.id::text)))`
}

// All branches are considered. A superseding failed/retake result prevents a
// passed ancestor from becoming a fallback; timestamps never select a branch.
func correctionEffectiveResultSQL(r string) string {
	return correctionApprovedResultSQL(r) + ` AND ` + correctionLeafSQL(r) + `
 AND NOT EXISTS(SELECT 1 FROM correction_results sibling WHERE sibling.owner_user_id=` + r + `.owner_user_id AND sibling.evidence_kind=` + r + `.evidence_kind AND sibling.evidence_id=` + r + `.evidence_id AND sibling.id<>` + r + `.id AND ` + correctionApprovedResultSQL("sibling") + ` AND ` + correctionLeafSQL("sibling") + ` AND (sibling.basis#>'{body,effectiveItems}' IS DISTINCT FROM ` + r + `.basis#>'{body,effectiveItems}' OR sibling.status IS DISTINCT FROM ` + r + `.status))
 AND NOT EXISTS(SELECT 1 FROM correction_dependencies d WHERE d.result_id=` + r + `.id AND d.role='effective' AND (EXISTS(SELECT 1 FROM content_withdrawals w WHERE w.kind=d.kind AND w.sha256=d.sha256 AND (d.kind='asset' OR w.target_id=d.id AND w.target_version=d.version)) OR EXISTS(SELECT 1 FROM question_withdrawals w WHERE w.kind=d.kind AND w.sha256=d.sha256 AND w.target_id=d.id AND w.target_version=d.version)))
 AND NOT EXISTS(SELECT 1 FROM correction_cases c WHERE c.sealed AND c.kind='grading_rule' AND correction_case_applies(c.id,` + r + `.evidence_kind,` + r + `.evidence_id,` + r + `.owner_user_id) AND NOT (` + r + `.handled_case_ids @> jsonb_build_array(c.id::text)))
 AND NOT EXISTS(SELECT 1 FROM jsonb_array_elements_text(` + r + `.handled_case_ids) handled WHERE NOT EXISTS(SELECT 1 FROM jsonb_array_elements(` + r + `.basis#>'{body,planRefs}') ref JOIN correction_plans approved ON approved.id=(ref->>'id')::uuid AND approved.version=(ref->>'version')::integer WHERE approved.case_id=handled::uuid AND approved.status='approved' AND approved.sealed AND approved.algorithm_version=1))`
}
func correctionAssessmentProjectionSQL(ctx context.Context) string {
	original := `SELECT a.id attempt_id,a.owner_user_id,a.knowledge_id,a.knowledge_version,a.knowledge_sha256,a.mode,a.terminal_at,r.outcome,NULL::uuid correction_id FROM assessment_attempts a JOIN assessment_results r ON r.attempt_id=a.id WHERE a.state='submitted' AND r.outcome IN ('passed','failed') AND ` + learningCleanEvidenceSQL("assessment", "a.id") + ` AND ` + correctionOriginalEvidenceSQL(ctx, "assessment", "a.id")
	if !correctionEnabled(ctx) {
		return original
	}
	return original + ` UNION ALL SELECT a.id,a.owner_user_id,a.knowledge_id,a.knowledge_version,a.knowledge_sha256,a.mode,a.terminal_at,CASE WHEN cr.status='corrected_passed' THEN 'passed' ELSE 'failed' END,cr.id FROM assessment_attempts a JOIN correction_results cr ON cr.owner_user_id=a.owner_user_id AND cr.evidence_kind='assessment' AND cr.evidence_id=a.id AND cr.knowledge_id=a.knowledge_id AND cr.knowledge_version=a.knowledge_version AND cr.knowledge_sha256=a.knowledge_sha256 WHERE a.state='submitted' AND cr.status IN ('corrected_passed','corrected_failed') AND ` + correctionEffectiveResultSQL("cr")
}
func correctionPassingEvidence(ctx context.Context, tx *sql.Tx, owner string, k question.Identity) (*learning.QualificationView, error) {
	current, e := learningIsCurrent(ctx, tx, k)
	if e != nil || !current {
		return nil, e
	}
	completion, e := learningCurrentCompletion(ctx, tx, owner, k)
	if e != nil {
		return nil, e
	}
	var id, mode string
	var cid *string
	e = tx.QueryRowContext(ctx, `WITH candidates AS (`+correctionAssessmentProjectionSQL(ctx)+`) SELECT attempt_id::text,mode,correction_id::text FROM candidates WHERE owner_user_id=$1 AND knowledge_id=$2 AND knowledge_version=$3 AND knowledge_sha256=$4 AND outcome='passed' AND (mode='diagnostic' OR $5::uuid IS NOT NULL) ORDER BY (mode='diagnostic') DESC,terminal_at DESC,attempt_id,correction_id LIMIT 1`, owner, k.ID, k.Version, k.SHA256, completion).Scan(&id, &mode, &cid)
	if errors.Is(e, sql.ErrNoRows) {
		return nil, nil
	}
	if e != nil {
		return nil, e
	}
	v := &learning.QualificationView{Knowledge: k, Kind: "normal", EvidenceAttemptID: id, CompletedEventID: completion, Validity: assessment.Effective, CorrectionID: cid}
	if mode == "diagnostic" {
		v.Kind = "diagnostic"
		v.CompletedEventID = nil
	}
	return v, nil
}
