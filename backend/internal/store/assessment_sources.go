package store

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"github.com/yyl1212/math_master/backend/internal/assessment"
	"github.com/yyl1212/math_master/backend/internal/auth"
	"github.com/yyl1212/math_master/backend/internal/learning"
	"github.com/yyl1212/math_master/backend/internal/question"
	"time"
)

// Candidate queries return only accurate identities, coverage and personal facts.
// Private instance bytes are loaded only after selection by learningLoadItems.
const learningCandidateSQL = `WITH recent AS MATERIALIZED (
 SELECT id FROM assessment_attempts WHERE owner_user_id=$1 AND state='submitted' ORDER BY terminal_at DESC,id DESC LIMIT 1
) SELECT i.id,i.version,i.sha256,i.template_id,i.template_version,i.template_sha256,
 coalesce((SELECT jsonb_agg(c.objective_index ORDER BY c.objective_index) FROM question_instance_coverage c WHERE c.instance_id=i.id AND c.instance_version=i.version AND c.knowledge_id=$2 AND c.knowledge_version=$3),'[]'),
 v.last_seen_at,ex.exposed_at,EXISTS(SELECT 1 FROM assessment_items ai JOIN recent r ON r.id=ai.attempt_id WHERE ai.instance_id=i.id AND ai.instance_version=i.version AND ai.instance_sha256=i.sha256)
 FROM question_instances i
 JOIN question_publication_members m ON m.publication_id=$5::uuid AND m.kind='instance' AND m.id=i.id AND m.version=i.version AND m.sha256=i.sha256
 JOIN question_review_decisions rd ON rd.id=m.review_id AND rd.decision='approve'
 JOIN question_submissions qs ON qs.id=m.submission_id AND qs.id=rd.submission_id AND qs.sealed AND qs.status='approved' AND qs.frozen_digest=rd.frozen_digest
 JOIN question_submission_members sm ON sm.submission_id=qs.id AND sm.kind='instance' AND sm.id=i.id AND sm.version=i.version AND sm.sha256=i.sha256
 LEFT JOIN learner_question_views v ON v.owner_user_id=$1 AND v.instance_id=i.id AND v.instance_version=i.version AND v.instance_sha256=i.sha256
 LEFT JOIN LATERAL(SELECT max(e.exposed_at) exposed_at FROM learner_answer_exposures e WHERE e.owner_user_id=$1 AND
 ((e.kind='instance' AND e.id=i.id AND e.version=i.version AND e.sha256=i.sha256) OR (e.kind='template' AND e.id=i.template_id AND e.version=i.template_version AND e.sha256=i.template_sha256))) ex ON true
 WHERE i.sealed AND i.knowledge_id=$2 AND i.knowledge_version=$3
 AND ($6::text IS NULL OR EXISTS(SELECT 1 FROM question_blueprint_sources bs WHERE bs.blueprint_id=$6 AND bs.blueprint_version=$7 AND
 ((bs.kind='instance' AND bs.id=i.id AND bs.version=i.version) OR (bs.kind='template' AND bs.id=i.template_id AND bs.version=i.template_version))))
 AND NOT EXISTS(SELECT 1 FROM question_withdrawals w WHERE
 (w.kind='instance' AND w.target_id=i.id AND w.target_version=i.version AND w.sha256=i.sha256) OR
 (w.kind='template' AND w.target_id=i.template_id AND w.target_version=i.template_version AND w.sha256=i.template_sha256))
 AND (i.template_id IS NULL OR EXISTS(SELECT 1 FROM question_publication_members tm WHERE tm.publication_id=$5::uuid AND tm.kind='template' AND tm.id=i.template_id AND tm.version=i.template_version AND tm.sha256=i.template_sha256))
 AND NOT EXISTS(SELECT 1 FROM jsonb_array_elements(i.body#>'{body,body,units}') u
 LEFT JOIN unit_versions uv ON uv.id=u->>'id' AND uv.version=(u->>'version')::integer
 LEFT JOIN publication_members um ON um.snapshot_id=$4 AND um.kind='unit' AND um.id=uv.id AND um.version=uv.version AND um.availability='active'
 WHERE um.id IS NULL OR EXISTS(SELECT 1 FROM content_withdrawals w WHERE w.kind='unit' AND w.target_id=uv.id AND w.target_version=uv.version AND w.sha256=uv.sha256))
 AND NOT EXISTS(SELECT 1 FROM jsonb_array_elements(i.body#>'{body,body,assets}') a
 LEFT JOIN publication_members am ON am.snapshot_id=$4 AND am.kind='asset' AND am.id=a->>'id' AND am.availability='active'
 LEFT JOIN package_members pm ON pm.package_id=am.package_id AND pm.package_version=am.package_version AND pm.kind='asset' AND pm.id=am.id
 WHERE pm.asset_sha256 IS DISTINCT FROM a->>'sha256' OR EXISTS(SELECT 1 FROM content_withdrawals w WHERE w.kind='asset' AND w.sha256=a->>'sha256'))
 ORDER BY i.id,i.version LIMIT 1001`

func learningSourcePool(ctx context.Context, tx *sql.Tx, actor string, k question.Identity, bp *question.Identity, now time.Time) (assessment.SourcePool, error) {
	out := assessment.SourcePool{Knowledge: k, Candidates: []assessment.Candidate{}}
	if !question.ValidMathID(k.ID) || k.Version < 1 || !question.ValidSHA(k.SHA256) {
		return out, auth.ErrInvalidInput
	}
	var current bool
	err := tx.QueryRowContext(ctx, `SELECT p.id,learning_content_approved(p.id,'knowledge',k.id,k.version,k.sha256) AND NOT EXISTS(SELECT 1 FROM content_withdrawals w WHERE w.kind='knowledge' AND w.target_id=k.id AND w.target_version=k.version AND w.sha256=k.sha256) FROM publication_heads h JOIN publication_snapshots p ON p.id=h.snapshot_id AND p.status='published' JOIN publication_members m ON m.snapshot_id=p.id AND m.kind='knowledge' AND m.availability='active' JOIN knowledge_versions k ON k.id=m.id AND k.version=m.version WHERE h.singleton AND k.id=$1 AND k.version=$2 AND k.sha256=$3`, k.ID, k.Version, k.SHA256).Scan(&out.KnowledgeHead, &current)
	if errors.Is(err, sql.ErrNoRows) || err == nil && !current {
		return out, learning.ErrVersionStale
	}
	if err != nil {
		return out, err
	}
	qhead, e := questionHead(ctx, tx)
	if e != nil {
		return out, e
	}
	if qhead == nil {
		return out, nil
	}
	out.QuestionHead = *qhead
	var published bool
	if e = tx.QueryRowContext(ctx, `SELECT sealed AND status='published' FROM question_publications WHERE id=$1`, *qhead).Scan(&published); e != nil {
		return out, e
	}
	if !published {
		return out, auth.ErrUnavailable
	}
	var bpID *string
	var bpV *int
	if bp != nil {
		if !question.ValidMathID(bp.ID) || bp.Version < 1 || !question.ValidSHA(bp.SHA256) {
			return out, auth.ErrInvalidInput
		}
		var raw []byte
		e = tx.QueryRowContext(ctx, `SELECT b.body_bytes FROM question_blueprints b JOIN question_publication_members m ON m.publication_id=$1 AND m.kind='blueprint' AND m.id=b.id AND m.version=b.version AND m.sha256=b.sha256 JOIN question_review_decisions r ON r.id=m.review_id AND r.decision='approve' JOIN question_submissions s ON s.id=m.submission_id AND s.id=r.submission_id AND s.sealed AND s.status='approved' AND s.frozen_digest=r.frozen_digest JOIN question_submission_members sm ON sm.submission_id=s.id AND sm.kind='blueprint' AND sm.id=b.id AND sm.version=b.version AND sm.sha256=b.sha256 WHERE b.sealed AND b.id=$2 AND b.version=$3 AND b.sha256=$4 AND b.knowledge_id=$5 AND b.knowledge_version=$6 AND NOT EXISTS(SELECT 1 FROM question_withdrawals w WHERE w.kind='blueprint' AND w.target_id=b.id AND w.target_version=b.version AND w.sha256=b.sha256)`, *qhead, bp.ID, bp.Version, bp.SHA256, k.ID, k.Version).Scan(&raw)
		if errors.Is(e, sql.ErrNoRows) {
			return out, learning.ErrVersionStale
		}
		if e != nil {
			return out, e
		}
		var envelope struct {
			Purpose string             `json:"purpose"`
			Body    question.Blueprint `json:"body"`
		}
		h := sha256.Sum256(raw)
		if json.Unmarshal(raw, &envelope) != nil || envelope.Purpose != "question-blueprint-v1" || hex.EncodeToString(h[:]) != bp.SHA256 {
			return out, auth.ErrUnavailable
		}
		out.Blueprint = &envelope.Body
		out.BlueprintIdentity = bp
		bpID = &bp.ID
		bpV = &bp.Version
	}
	rows, e := tx.QueryContext(ctx, learningCandidateSQL, actor, k.ID, k.Version, out.KnowledgeHead, out.QuestionHead, bpID, bpV)
	if e != nil {
		return out, e
	}
	defer rows.Close()
	for rows.Next() {
		var c assessment.Candidate
		var ti, th sql.NullString
		var tv sql.NullInt64
		var seen, exposed sql.NullTime
		var coverage []byte
		if e = rows.Scan(&c.Identity.ID, &c.Identity.Version, &c.Identity.SHA256, &ti, &tv, &th, &coverage, &seen, &exposed, &c.RecentSubmitted); e != nil {
			return out, e
		}
		if json.Unmarshal(coverage, &c.Coverage) != nil {
			return out, auth.ErrUnavailable
		}
		if ti.Valid {
			c.Template = &question.Identity{ID: ti.String, Version: int(tv.Int64), SHA256: th.String}
		}
		if seen.Valid {
			c.Seen = true
			t := seen.Time.UTC()
			c.LastSeenAt = &t
		}
		if exposed.Valid {
			t := exposed.Time.UTC()
			c.ExposedAt = &t
		}
		out.Candidates = append(out.Candidates, c)
		if len(out.Candidates) > 1000 {
			return out, question.ErrLimitExceeded
		}
	}
	return out, rows.Err()
}
