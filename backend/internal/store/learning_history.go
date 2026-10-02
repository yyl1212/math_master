package store

import (
	"context"
	"database/sql"
	"github.com/yyl1212/math_master/backend/internal/assessment"
	"github.com/yyl1212/math_master/backend/internal/auth"
	"github.com/yyl1212/math_master/backend/internal/learning"
	"github.com/yyl1212/math_master/backend/internal/question"
	"time"
)

const learningHistorySQL = `WITH entries AS (
 SELECT id::text id,kind,knowledge_id kid,knowledge_version kv,knowledge_sha256 kh,recorded_at occurred,kind state,'learning-event' evidence,id eid,NULL::text path_id,NULL::integer path_version,NULL::text path_sha,NULL::text attempt,false affected FROM learning_events WHERE owner_user_id=$1
 UNION ALL SELECT id::text,'practice',knowledge_id,knowledge_version,knowledge_sha256,coalesce(terminal_at,created_at),CASE WHEN state='active' AND expires_at<=$2 THEN 'expired' ELSE state END,'practice',id,NULL,NULL,NULL,id::text,false FROM practice_attempts WHERE owner_user_id=$1
 UNION ALL SELECT a.id::text,'assessment',a.knowledge_id,a.knowledge_version,a.knowledge_sha256,coalesce(a.terminal_at,a.created_at),CASE WHEN a.state='active' AND a.expires_at<=$2 THEN 'expired' ELSE a.state END,'assessment',a.id,NULL,NULL,NULL,a.id::text,coalesce(r.outcome='affected',false) FROM assessment_attempts a LEFT JOIN assessment_results r ON r.attempt_id=a.id WHERE a.owner_user_id=$1 AND a.sealed
 UNION ALL SELECT e.id::text,'enrollment',n.knowledge_id,n.knowledge_version,n.knowledge_sha256,e.created_at,'enrolled','',e.id,e.path_id,e.path_version,e.path_sha256,NULL,false FROM learning_path_enrollments e JOIN LATERAL(SELECT knowledge_id,knowledge_version,knowledge_sha256 FROM learning_path_nodes WHERE enrollment_id=e.id ORDER BY position LIMIT 1) n ON true WHERE e.owner_user_id=$1
) `

func learningHistoryPage(ctx context.Context, tx *sql.Tx, actor string, q learning.ListQuery, now time.Time) (question.Page[learning.HistoryEntry], error) {
	out := question.Page[learning.HistoryEntry]{Items: []learning.HistoryEntry{}, Limit: q.Limit, Offset: q.Offset}
	if e := tx.QueryRowContext(ctx, learningHistorySQL+` SELECT count(*) FROM entries`, actor, now).Scan(&out.Total); e != nil {
		return out, e
	}
	rows, e := tx.QueryContext(ctx, learningHistorySQL+` SELECT id,kind,kid,kv,kh,occurred,state,path_id,path_version,path_sha,attempt,
 CASE WHEN affected OR EXISTS(SELECT 1 FROM learning_evidence_dependencies d WHERE d.evidence_kind=e.evidence AND d.evidence_id=e.eid AND (EXISTS(SELECT 1 FROM content_withdrawals w WHERE w.kind=d.kind AND w.sha256=d.sha256 AND (d.kind='asset' OR (w.target_id=d.id AND w.target_version=d.version))) OR EXISTS(SELECT 1 FROM question_withdrawals w WHERE w.kind=d.kind AND w.target_id=d.id AND w.target_version=d.version AND w.sha256=d.sha256))) THEN 'restricted'
 WHEN NOT EXISTS(SELECT 1 FROM publication_heads h JOIN publication_members m ON m.snapshot_id=h.snapshot_id AND m.kind='knowledge' AND m.availability='active' JOIN knowledge_versions k ON k.id=m.id AND k.version=m.version WHERE h.singleton AND k.id=e.kid AND k.version=e.kv AND k.sha256=e.kh) THEN 'stale' ELSE 'effective' END FROM entries e ORDER BY occurred DESC,id DESC LIMIT $3 OFFSET $4`, actor, now, q.Limit, q.Offset)
	if e != nil {
		return out, e
	}
	defer rows.Close()
	for rows.Next() {
		var h learning.HistoryEntry
		var pi, ps sql.NullString
		var pv sql.NullInt64
		if e = rows.Scan(&h.ID, &h.Kind, &h.Knowledge.ID, &h.Knowledge.Version, &h.Knowledge.SHA256, &h.OccurredAt, &h.State, &pi, &pv, &ps, &h.AttemptID, &h.Validity); e != nil {
			return out, e
		}
		h.OccurredAt = h.OccurredAt.UTC()
		if pi.Valid {
			h.Path = &question.Identity{ID: pi.String, Version: int(pv.Int64), SHA256: ps.String}
		}
		out.Items = append(out.Items, h)
	}
	return out, rows.Err()
}
func (s *Store) ListLearningHistory(ctx context.Context, a question.Access, q learning.ListQuery) (question.Page[learning.HistoryEntry], error) {
	var out question.Page[learning.HistoryEntry]
	q, e := learningPageQuery(q)
	if e != nil {
		return out, e
	}
	e = s.learningTx(ctx, a, learning.ListHistoryAction, func(ctx context.Context, tx *sql.Tx, u auth.User, now time.Time) error {
		var e error
		out, e = learningHistoryPage(ctx, tx, u.ID, q, now)
		return e
	})
	if e != nil {
		return question.Page[learning.HistoryEntry]{}, e
	}
	return out, nil
}
func (s *Store) ReadAssessmentResult(ctx context.Context, a question.Access, id string) (assessment.ResultView, error) {
	var out assessment.ResultView
	if !question.ValidID(id) {
		return out, auth.ErrInvalidInput
	}
	e := s.learningTx(ctx, a, learning.ReadAssessmentResultAction, func(ctx context.Context, tx *sql.Tx, u auth.User, now time.Time) error {
		p, e := learningReadAssessment(ctx, tx, u.ID, id, false)
		if e != nil {
			return e
		}
		out, e = learningAssessmentResult(ctx, tx, u.ID, p, now, nil)
		return e
	})
	if e != nil {
		return assessment.ResultView{}, e
	}
	return out, nil
}
