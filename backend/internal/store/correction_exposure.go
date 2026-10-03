package store

import (
	"context"
	"database/sql"
	"github.com/yyl1212/math_master/backend/internal/correction"
	"github.com/yyl1212/math_master/backend/internal/learning"
	"github.com/yyl1212/math_master/backend/internal/question"
	"strings"
)

type correctionExposureRef struct {
	Kind     string            `json:"kind"`
	Identity question.Identity `json:"identity"`
}

func correctionInstanceExposure(i question.Instance) []correctionExposureRef {
	out := []correctionExposureRef{{Kind: "instance", Identity: i.Identity}}
	if i.Template != nil {
		out = append(out, correctionExposureRef{Kind: "template", Identity: *i.Template})
	}
	return out
}

func correctionDetailExposure(ctx context.Context, tx *sql.Tx, reader, caseID string, refs []correctionExposureRef) error {
	c, e := correctionReadCase(ctx, tx, caseID)
	if e != nil {
		return e
	}
	// Include the exact withdrawn source even when a plan has no mappings yet.
	if c.Withdrawal != nil && c.Withdrawal.Space == "question" {
		var kind, id, sha string
		var version int
		if e = tx.QueryRowContext(ctx, `SELECT kind,target_id,target_version,sha256 FROM question_withdrawals WHERE id=$1`, c.Withdrawal.ID).Scan(&kind, &id, &version, &sha); e != nil {
			return e
		}
		if kind == "instance" || kind == "template" {
			refs = append(refs, correctionExposureRef{Kind: kind, Identity: question.Identity{ID: id, Version: version, SHA256: sha}})
		}
	}
	now, e := dbClock(ctx, tx)
	if e != nil {
		return e
	}
	var overlap bool
	e = tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM assessment_attempts a WHERE a.owner_user_id=$1 AND a.state='active' AND a.sealed AND a.expires_at>$2 AND (
 EXISTS(SELECT 1 FROM assessment_items i CROSS JOIN jsonb_to_recordset($3::jsonb) r(kind text,identity jsonb) WHERE i.attempt_id=a.id AND (r.kind='instance' AND i.binding->'instance'=r.identity OR r.kind='template' AND i.binding->'template'=r.identity))
 OR EXISTS(SELECT 1 FROM correction_cases c WHERE c.id=$4 AND c.kind='grading_rule' AND a.rule_version=c.rule_version AND (c.scope_kind='all' OR a.knowledge_id=c.knowledge_id AND a.knowledge_version=c.knowledge_version AND a.knowledge_sha256=c.knowledge_sha256))
 OR EXISTS(SELECT 1 FROM correction_cases c JOIN learning_evidence_dependencies d ON d.evidence_kind='assessment' AND d.evidence_id=a.id AND d.owner_user_id=a.owner_user_id WHERE c.id=$4 AND c.kind='withdrawal' AND ((c.withdrawal_space='content' AND EXISTS(SELECT 1 FROM content_withdrawals w WHERE w.id=c.withdrawal_id AND w.kind=d.kind AND w.sha256=d.sha256 AND (d.kind='asset' OR w.target_id=d.id AND w.target_version=d.version))) OR (c.withdrawal_space='question' AND EXISTS(SELECT 1 FROM question_withdrawals w WHERE w.id=c.withdrawal_id AND w.kind=d.kind AND w.target_id=d.id AND w.target_version=d.version AND w.sha256=d.sha256))))))`, reader, now, body(refs), caseID).Scan(&overlap)
	if e != nil {
		return e
	}
	if overlap {
		return correction.ErrAnswerOverlap
	}
	recordRefs := []learning.ExposureRef{}
	for _, ref := range refs {
		recordRefs = append(recordRefs, learning.ExposureRef{Kind: ref.Kind, Identity: ref.Identity})
	}
	if _, e = learningRecordExposure(ctx, tx, reader, recordRefs, now); e != nil {
		return e
	}
	id, e := workflowID()
	if e != nil {
		return e
	}
	raw, h, e := correction.Canonical("correction-command-v1", map[string]any{"caseId": caseID, "refs": refs})
	if e != nil {
		return e
	}
	_, e = tx.ExecContext(ctx, `INSERT INTO correction_events(id,subject_kind,subject_id,case_id,owner_user_id,kind,sequence,actor_user_id,body,body_bytes,body_digest) VALUES($1,'exposure',$1,$2,$3,'detail_exposed',1,$3,$4,$5,$6)`, id, caseID, reader, string(raw), raw, h)
	return e
}

// A scope exposure is deliberately independent of a case's attempt cutoff:
// later newly selected questions must also observe the 30-minute cooldown.
const correctionCandidateExposureSQL = ` LEFT JOIN LATERAL(SELECT max(e.recorded_at) exposed_at FROM correction_events e JOIN correction_cases c ON c.id=e.case_id WHERE e.owner_user_id=$1 AND e.kind='detail_exposed' AND (
 (c.kind='grading_rule' AND c.rule_version=1 AND (c.scope_kind='all' OR c.knowledge_id=i.knowledge_id AND c.knowledge_version=i.knowledge_version AND c.knowledge_sha256=(SELECT kv.sha256 FROM knowledge_versions kv WHERE kv.id=i.knowledge_id AND kv.version=i.knowledge_version)))
 OR (c.kind='withdrawal' AND ((c.withdrawal_space='question' AND EXISTS(SELECT 1 FROM question_withdrawals w WHERE w.id=c.withdrawal_id AND ((w.kind='instance' AND w.target_id=i.id AND w.target_version=i.version AND w.sha256=i.sha256) OR (w.kind='template' AND w.target_id=i.template_id AND w.target_version=i.template_version AND w.sha256=i.template_sha256) OR (w.kind='blueprint' AND w.target_id=$6 AND w.target_version=$7))))
 OR (c.withdrawal_space='content' AND EXISTS(SELECT 1 FROM content_withdrawals w WHERE w.id=c.withdrawal_id AND ((w.kind='knowledge' AND w.target_id=i.knowledge_id AND w.target_version=i.knowledge_version AND w.sha256=(SELECT kv.sha256 FROM knowledge_versions kv WHERE kv.id=i.knowledge_id AND kv.version=i.knowledge_version)) OR (w.kind='unit' AND EXISTS(SELECT 1 FROM jsonb_array_elements(i.body#>'{body,body,units}') u JOIN unit_versions uv ON uv.id=u->>'id' AND uv.version=(u->>'version')::integer WHERE uv.id=w.target_id AND uv.version=w.target_version AND uv.sha256=w.sha256)) OR (w.kind='asset' AND EXISTS(SELECT 1 FROM jsonb_array_elements(i.body#>'{body,body,assets}') a WHERE a->>'sha256'=w.sha256))))))))) broad ON true `

func learningCorrectionCandidateSQL(ctx context.Context) string {
	if !correctionEnabled(ctx) {
		return learningCandidateSQL
	}
	query := strings.Replace(learningCandidateSQL, "v.last_seen_at,ex.exposed_at,", "v.last_seen_at,greatest(ex.exposed_at,broad.exposed_at),", 1)
	return strings.Replace(query, " WHERE i.sealed AND", correctionCandidateExposureSQL+" WHERE i.sealed AND", 1)
}
