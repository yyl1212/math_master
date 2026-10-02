package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"github.com/yyl1212/math_master/backend/internal/assessment"
	"github.com/yyl1212/math_master/backend/internal/auth"
	"github.com/yyl1212/math_master/backend/internal/learning"
	"github.com/yyl1212/math_master/backend/internal/question"
)

func learningLoadItems(ctx context.Context, tx *sql.Tx, seal assessment.Seal) ([]question.Instance, error) {
	out := make([]question.Instance, 0, len(seal.Items))
	if len(seal.Items) != 1 && len(seal.Items) != 5 {
		return out, auth.ErrInvalidInput
	}
	rows, e := tx.QueryContext(ctx, `SELECT r.position,i.sha256,i.body_bytes,m.evidence FROM jsonb_to_recordset($1::jsonb) r(position integer,instance jsonb)
 JOIN question_instances i ON i.id=r.instance->>'id' AND i.version=(r.instance->>'version')::integer AND i.sha256=r.instance->>'sha256' AND i.sealed
 JOIN question_publication_members m ON m.publication_id=$2 AND m.kind='instance' AND m.id=i.id AND m.version=i.version AND m.sha256=i.sha256
 JOIN question_publications p ON p.id=m.publication_id AND p.sealed AND p.status='published'
 JOIN question_review_decisions rd ON rd.id=m.review_id AND rd.decision='approve'
 JOIN question_submissions s ON s.id=m.submission_id AND s.id=rd.submission_id AND s.sealed AND s.status='approved' AND s.frozen_digest=rd.frozen_digest
 JOIN question_submission_members sm ON sm.submission_id=s.id AND sm.kind='instance' AND sm.id=i.id AND sm.version=i.version AND sm.sha256=i.sha256 ORDER BY r.position`, body(seal.Items), seal.QuestionPublicationID)
	if e != nil {
		return out, e
	}
	defer rows.Close()
	for rows.Next() {
		var pos int
		var sha string
		var raw, proof []byte
		if e = rows.Scan(&pos, &sha, &raw, &proof); e != nil {
			return out, e
		}
		var env struct {
			Purpose string            `json:"purpose"`
			Body    question.Instance `json:"body"`
		}
		var approval question.MemberEvidence
		if pos < 1 || pos > len(seal.Items) || json.Unmarshal(raw, &env) != nil || env.Purpose != "question-instance-body-v1" || json.Unmarshal(proof, &approval) != nil {
			return out, auth.ErrUnavailable
		}
		env.Body.Identity.SHA256 = sha
		_, actual, e := question.CanonicalInstance(env.Body)
		if e != nil || actual != sha || env.Body.Identity != seal.Items[pos-1].Instance || body(approval) != body(seal.Items[pos-1].Approval) {
			return out, auth.ErrUnavailable
		}
		if env.Body.Body.Knowledge.ID != seal.Knowledge.ID || env.Body.Body.Knowledge.Version != seal.Knowledge.Version {
			return out, auth.ErrUnavailable
		}
		out = append(out, env.Body)
	}
	if e = rows.Err(); e != nil {
		return out, e
	}
	if len(out) != len(seal.Items) {
		return out, auth.ErrNotFound
	}
	return out, nil
}

// Fixed evidence is restricted only by permanent accurate withdrawals. Ordinary
// publication replacement retains immutable historical mathematics and approval.
func learningEvidenceRestrictions(ctx context.Context, tx *sql.Tx, deps []learning.EvidenceDependency) ([]assessment.RestrictionReason, error) {
	out := []assessment.RestrictionReason{}
	if len(deps) == 0 {
		return out, nil
	}
	input := make([]map[string]any, 0, len(deps))
	for _, d := range deps {
		input = append(input, map[string]any{"kind": d.Kind, "id": d.ID, "version": d.Version, "sha256": d.SHA256})
	}
	rows, e := tx.QueryContext(ctx, `WITH input AS (SELECT * FROM jsonb_to_recordset($1::jsonb) r(kind text,id text,version integer,sha256 text)),
 d AS (SELECT * FROM input UNION SELECT 'asset',b.asset_id,NULL::integer,b.asset_sha256 FROM input i
 JOIN unit_versions uv ON i.kind='unit' AND uv.id=i.id AND uv.version=i.version AND uv.sha256=i.sha256
 JOIN unit_asset_bindings b ON b.unit_id=uv.id AND b.unit_version=uv.version)
 SELECT DISTINCT d.kind||'-withdrawn' FROM d WHERE
 EXISTS(SELECT 1 FROM content_withdrawals w WHERE w.kind=d.kind AND w.sha256=d.sha256 AND ((d.kind='asset') OR (w.target_id=d.id AND w.target_version=d.version)))
 OR EXISTS(SELECT 1 FROM question_withdrawals w WHERE w.kind=d.kind AND w.target_id=d.id AND w.target_version=d.version AND w.sha256=d.sha256) ORDER BY 1`, body(input))
	if e != nil {
		return out, e
	}
	defer rows.Close()
	for rows.Next() {
		var r assessment.RestrictionReason
		if e = rows.Scan(&r); e != nil {
			return out, e
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func learningEvidenceDependencies(ctx context.Context, tx *sql.Tx, kind, id string) ([]learning.EvidenceDependency, error) {
	out := []learning.EvidenceDependency{}
	rows, e := tx.QueryContext(ctx, `SELECT kind,id,version,sha256 FROM learning_evidence_dependencies WHERE evidence_kind=$1 AND evidence_id=$2 ORDER BY kind,id,version,sha256`, kind, id)
	if e != nil {
		return out, e
	}
	defer rows.Close()
	for rows.Next() {
		var d learning.EvidenceDependency
		var v sql.NullInt64
		if e = rows.Scan(&d.Kind, &d.ID, &v, &d.SHA256); e != nil {
			return out, e
		}
		if v.Valid {
			n := int(v.Int64)
			d.Version = &n
		}
		out = append(out, d)
	}
	return out, rows.Err()
}
