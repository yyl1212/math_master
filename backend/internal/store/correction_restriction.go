package store

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/yyl1212/math_master/backend/internal/assessment"
	"github.com/yyl1212/math_master/backend/internal/auth"
	"github.com/yyl1212/math_master/backend/internal/correction"
	"github.com/yyl1212/math_master/backend/internal/learning"
	"github.com/yyl1212/math_master/backend/internal/question"
	"time"
)

func correctionOriginalEvidenceSQL(ctx context.Context, kind, id string) string {
	if !correctionEnabled(ctx) {
		return "TRUE"
	}
	// All arguments are private aliases/constants, never request query text.
	return fmt.Sprintf(`NOT EXISTS(SELECT 1 FROM correction_cases c WHERE c.sealed AND c.kind='grading_rule' AND EXISTS(
 SELECT 1 FROM assessment_attempts ca WHERE %s='assessment' AND ca.id=%s AND ca.created_at<=c.cutoff AND ca.rule_version=c.rule_version AND (c.scope_kind='all' OR (ca.knowledge_id=c.knowledge_id AND ca.knowledge_version=c.knowledge_version AND ca.knowledge_sha256=c.knowledge_sha256))
 UNION ALL SELECT 1 FROM practice_attempts cp WHERE %s='practice' AND cp.id=%s AND cp.created_at<=c.cutoff AND (cp.seal#>>'{body,ruleVersion}')::integer=c.rule_version AND (c.scope_kind='all' OR (cp.knowledge_id=c.knowledge_id AND cp.knowledge_version=c.knowledge_version AND cp.knowledge_sha256=c.knowledge_sha256))))`, correctionKindSQL(kind), id, correctionKindSQL(kind), id)
}
func correctionRuleRestricted(ctx context.Context, tx *sql.Tx, owner string, ref correction.EvidenceRef) (bool, error) {
	if !correctionEnabled(ctx) || ref.Kind != correction.AssessmentEvidence && ref.Kind != correction.PracticeEvidence {
		return false, nil
	}
	table := "assessment_attempts"
	if ref.Kind == correction.PracticeEvidence {
		table = "practice_attempts"
	}
	suffix := ""
	if ref.Kind == correction.AssessmentEvidence {
		suffix = " AND e.sealed"
	}
	var clean bool
	e := tx.QueryRowContext(ctx, `SELECT `+correctionOriginalEvidenceSQL(ctx, string(ref.Kind), "e.id")+` FROM `+table+` e WHERE e.id=$1 AND e.owner_user_id=$2`+suffix, ref.ID, owner).Scan(&clean)
	if e != nil {
		return false, workflowRowError(e)
	}
	return !clean, nil
}
func correctionEvidenceGuard(ctx context.Context, tx *sql.Tx, owner string, ref correction.EvidenceRef) ([]assessment.RestrictionReason, error) {
	if !correction.ValidEvidence(ref, true) {
		return nil, auth.ErrInvalidInput
	}
	deps, e := learningEvidenceDependencies(ctx, tx, string(ref.Kind), ref.ID)
	if e != nil {
		return nil, e
	}
	rs, e := learningEvidenceRestrictions(ctx, tx, deps)
	if e != nil {
		return nil, e
	}
	blocked, e := correctionRuleRestricted(ctx, tx, owner, ref)
	if e != nil {
		return nil, e
	}
	if blocked {
		rs = append(rs, assessment.GradingIssue)
	}
	return learningUniqueReasons(rs), nil
}
func correctionNewAttemptGuard(ctx context.Context, tx *sql.Tx, owner string, k question.Identity, rule int, now time.Time) error {
	if !correctionEnabled(ctx) {
		return nil
	}
	var blocked bool
	e := tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM correction_cases c WHERE c.sealed AND c.kind='grading_rule' AND c.rule_version=$4 AND (c.scope_kind='all' OR (c.knowledge_id=$1 AND c.knowledge_version=$2 AND c.knowledge_sha256=$3)) AND NOT EXISTS(SELECT 1 FROM correction_plans p WHERE p.case_id=c.id AND p.status='approved' AND p.sealed AND p.algorithm_version=1 AND EXISTS(SELECT 1 FROM correction_events e WHERE e.case_id=c.id AND e.subject_kind='plan' AND e.subject_id=p.id AND e.subject_version=p.version AND e.sequence=p.sequence AND e.kind='plan_approved')))`, k.ID, k.Version, k.SHA256, rule).Scan(&blocked)
	if e != nil {
		return e
	}
	if blocked {
		return &learning.NotReadyError{}
	}
	return nil
}

func correctionKindSQL(kind string) string {
	if kind == "e.evidence" {
		return kind
	}
	return "'" + kind + "'"
}
