package store

import (
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"github.com/yyl1212/math_master/backend/internal/auth"
	"github.com/yyl1212/math_master/backend/internal/study"
	"github.com/yyl1212/math_master/backend/internal/studyreview"
	"time"
)

const reviewSessionSelect = `SELECT s.owner_user_id::text,s.session_id::text,s.knowledge_id,s.review_id::text,s.knowledge_ref,s.mode,s.target,s.status,s.sequence,s.draft_revision,s.answer,s.unanswered,s.created_at,s.compared_at,s.graded_at,s.abandoned_at,s.draft_updated_at,s.grade,p.title,p.title_zh,p.knowledge_type,p.prompt,p.expected,p.sha256 FROM study_review_sessions s JOIN study_review_snapshots p ON p.session_id=s.session_id AND p.owner_user_id=s.owner_user_id AND p.knowledge_id=s.knowledge_id WHERE s.owner_user_id=$1 AND s.session_id=$2`

func readReviewRaw(ctx context.Context, tx *sql.Tx, owner, id string, lock bool) (s studyreview.Session, m studyreview.FrozenMaterial, e error) {
	q := reviewSessionSelect
	if lock {
		q += " FOR UPDATE OF s"
	}
	var ref, grade, prompt, expected []byte
	e = tx.QueryRowContext(ctx, q, owner, id).Scan(&s.ActorID, &s.SessionID, &s.KnowledgeID, &s.ReviewID, &ref, &s.Mode, &s.Target, &s.Status, &s.Sequence, &s.DraftRevision, &s.Answer, &s.Unanswered, &s.CreatedAt, &s.ComparedAt, &s.GradedAt, &s.AbandonedAt, &s.DraftUpdatedAt, &grade, &s.Title, &s.TitleZH, &s.KnowledgeType, &prompt, &expected, &m.SHA256)
	if errors.Is(e, sql.ErrNoRows) {
		return s, m, studyreview.ErrNotFound
	}
	if e != nil {
		return
	}
	if json.Unmarshal(ref, &s.Knowledge) != nil || json.Unmarshal(prompt, &m.Prompt) != nil || json.Unmarshal(expected, &m.Expected) != nil {
		return s, m, studyreview.ErrNotConfigured
	}
	if len(grade) > 0 {
		s.Grade = &studyreview.GradeResult{}
		if json.Unmarshal(grade, s.Grade) != nil {
			return s, m, studyreview.ErrNotConfigured
		}
	}
	s.Prompt = []studyreview.Section{}
	return
}
func readReviewSchedule(ctx context.Context, tx *sql.Tx, owner, kid string, lock bool) (s *studyreview.Schedule, e error) {
	q := `SELECT stage,due_at,last_ref,last_rating,sequence,graded_at FROM study_review_schedules WHERE owner_user_id=$1 AND knowledge_id=$2`
	if lock {
		q += " FOR UPDATE"
	}
	s = &studyreview.Schedule{}
	var ref []byte
	e = tx.QueryRowContext(ctx, q, owner, kid).Scan(&s.Stage, &s.DueAt, &ref, &s.LastRating, &s.Sequence, &s.GradedAt)
	if errors.Is(e, sql.ErrNoRows) {
		return nil, nil
	}
	if e != nil {
		return nil, e
	}
	if json.Unmarshal(ref, &s.LastRef) != nil {
		return nil, studyreview.ErrNotConfigured
	}
	return
}
func (s *Store) ReadReviewSession(ctx context.Context, a studyreview.Access, id string) (out studyreview.Session, e error) {
	if !study.ValidID(id) {
		return out, studyreview.ErrInvalid
	}
	e = s.studyReviewReadTx(ctx, a, func(ctx context.Context, tx *sql.Tx, u auth.User) error {
		current, m, e := readReviewRaw(ctx, tx, u.ID, id, false)
		if e != nil {
			return e
		}
		k, e := managedStudyCurrent(ctx, tx, current.KnowledgeID, true)
		if e != nil {
			return e
		}
		current.Available = k.Published && !k.Deleted
		current.MaterialChanged = k.Ref != current.Knowledge
		current.Schedule, e = readReviewSchedule(ctx, tx, u.ID, current.KnowledgeID, false)
		if e != nil {
			return e
		}
		if current.Available && !current.MaterialChanged {
			current.Prompt = m.Prompt
			if current.Mode == studyreview.Read || current.Status == studyreview.Compared || current.Status == studyreview.Completed {
				current.Reference = &studyreview.ExpectedReference{Sections: m.Expected, SHA256: m.SHA256}
			}
		}
		if e = studyreview.ValidateSessionBudget(current); e != nil {
			return e
		}
		out = current
		return nil
	})
	if e != nil {
		out = studyreview.Session{}
	}
	return
}
func (s *Store) ReadReviewHealth(ctx context.Context) (out studyreview.Health, e error) {
	out.Capability = true
	tx, e := s.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true, Isolation: sql.LevelRepeatableRead})
	if e != nil {
		return out, knowledgeError(e)
	}
	defer tx.Rollback()
	e = knowledgeConfigured(ctx, tx)
	if e == nil {
		e = studyReviewSchemaIntegrity(ctx, tx)
	}
	if e != nil {
		return out, nil
	}
	out.SchemaReady = true
	e = tx.Commit()
	return out, knowledgeError(e)
}

// One statement snapshot classifies each learned knowledge exactly once.
const reviewQueueCTE = `WITH at AS MATERIALIZED(SELECT clock_timestamp() as_of), eligible AS MATERIALIZED (
 SELECT k.internal_id knowledge_id,k.content_sha256,k.point->>'title' title,k.point->>'title_zh' title_zh,k.point->>'type' knowledge_type,NOT study_review_blank(coalesce(k.point->>'proof','')) proof_available,
 active.session_id::text active_session_id,
 CASE WHEN sch.knowledge_id IS NOT NULL THEN jsonb_build_object('stage',sch.stage,'dueAt',sch.due_at,'lastRef',sch.last_ref,'lastRating',sch.last_rating,'sequence',sch.sequence,'gradedAt',sch.graded_at) ELSE NULL END schedule,
 (coalesce(sch.last_ref,CASE WHEN r.last_review_ref IS NOT NULL AND (r.completed_ref IS NULL OR r.last_completed_at IS NULL OR r.last_reviewed_at>r.last_completed_at) THEN r.last_review_ref ELSE coalesce(r.completed_ref,r.last_review_ref) END)->>'contentSha256' IS DISTINCT FROM k.content_sha256) material_changed,
 CASE WHEN active.session_id IS NOT NULL OR r.state='reviewing' THEN 'unfinished'
 WHEN coalesce(sch.last_ref,CASE WHEN r.last_review_ref IS NOT NULL AND (r.completed_ref IS NULL OR r.last_completed_at IS NULL OR r.last_reviewed_at>r.last_completed_at) THEN r.last_review_ref ELSE coalesce(r.completed_ref,r.last_review_ref) END)->>'contentSha256' IS DISTINCT FROM k.content_sha256 THEN 'changed'
 WHEN sch.due_at<=at.as_of THEN 'due' WHEN sch.knowledge_id IS NULL THEN 'unplanned' ELSE 'future' END bucket,
 sch.due_at,at.as_of
 FROM managed_study_records r JOIN managed_knowledge k ON k.internal_id=r.knowledge_id
 LEFT JOIN study_review_schedules sch ON sch.owner_user_id=r.owner_user_id AND sch.knowledge_id=r.knowledge_id
 LEFT JOIN study_review_sessions active ON active.owner_user_id=r.owner_user_id AND active.knowledge_id=r.knowledge_id AND active.status IN ('working','compared')
 CROSS JOIN at WHERE r.owner_user_id=$1 AND r.first_completed_at IS NOT NULL AND k.published AND k.deleted_at IS NULL
) `

func (s *Store) ReadReviewOverview(ctx context.Context, a studyreview.Access) (out studyreview.Overview, e error) {
	e = s.studyReviewReadTx(ctx, a, func(ctx context.Context, tx *sql.Tx, u auth.User) error {
		out.ActorID = u.ID
		return tx.QueryRowContext(ctx, reviewQueueCTE+`SELECT (SELECT as_of FROM at),count(*),count(*) FILTER(WHERE bucket='unfinished'),count(*) FILTER(WHERE bucket='changed'),count(*) FILTER(WHERE bucket='due'),count(*) FILTER(WHERE bucket='unplanned'),count(*) FILTER(WHERE bucket='future') FROM eligible`, u.ID).Scan(&out.AsOf, &out.Total, &out.Unfinished, &out.Changed, &out.Due, &out.Unplanned, &out.Future)
	})
	if e != nil {
		out = studyreview.Overview{}
	}
	return
}
func (s *Store) ListReviewQueue(ctx context.Context, a studyreview.Access, q studyreview.QueueQuery) (out studyreview.QueuePage, e error) {
	if q.Limit == 0 {
		q.Limit = 20
	}
	if e = studyreview.ValidateQueue(q); e != nil {
		return
	}
	e = s.studyReviewReadTx(ctx, a, func(ctx context.Context, tx *sql.Tx, u auth.User) error {
		out = studyreview.QueuePage{ActorID: u.ID, Items: []studyreview.QueueItem{}, Limit: q.Limit, Offset: q.Offset}
		const filter = ` ,filtered AS(SELECT * FROM eligible WHERE ($2='' OR strpos(lower(title),lower($2))>0 OR strpos(lower(title_zh),lower($2))>0) AND ($3='' OR EXISTS(SELECT 1 FROM managed_knowledge_topics t WHERE t.internal_id=knowledge_id AND t.active AND (t.topic_key=$3 OR t.topic_key LIKE $3||'%'))) AND ($4='' OR $4='all' OR bucket=$4 OR ($4='due' AND bucket='changed'))),chosen AS(SELECT * FROM filtered ORDER BY CASE bucket WHEN 'unfinished' THEN 0 WHEN 'changed' THEN 1 WHEN 'due' THEN 2 WHEN 'unplanned' THEN 3 ELSE 4 END,due_at NULLS FIRST,knowledge_id LIMIT $5 OFFSET $6),chosen_topics AS MATERIALIZED(SELECT t.internal_id,jsonb_agg(t.topic_key ORDER BY t.topic_key) topic_keys FROM managed_knowledge_topics t JOIN chosen c ON c.knowledge_id=t.internal_id WHERE t.active GROUP BY t.internal_id) `
		var raw []byte
		e := tx.QueryRowContext(ctx, reviewQueueCTE+filter+`SELECT (SELECT as_of FROM at),(SELECT count(*) FROM filtered),coalesce((SELECT jsonb_agg(jsonb_build_object('knowledge',jsonb_build_object('id',knowledge_id,'contentSha256',content_sha256,'sourceKind','managed'),'title',left(title,128),'titleZh',left(title_zh,128),'knowledgeType',knowledge_type,'topicKeys',coalesce((SELECT topic_keys FROM chosen_topics t WHERE t.internal_id=knowledge_id),'[]'::jsonb),'bucket',bucket,'materialChanged',material_changed,'proofAvailable',proof_available,'activeSessionId',active_session_id,'schedule',schedule) ORDER BY CASE bucket WHEN 'unfinished' THEN 0 WHEN 'changed' THEN 1 WHEN 'due' THEN 2 WHEN 'unplanned' THEN 3 ELSE 4 END,due_at NULLS FIRST,knowledge_id) FROM chosen),'[]'::jsonb)`, u.ID, q.Q, topicPrefix(q.TopicKey), q.Bucket, q.Limit, q.Offset).Scan(&out.AsOf, &out.Total, &raw)
		if e != nil {
			return e
		}
		if json.Unmarshal(raw, &out.Items) != nil {
			return studyreview.ErrNotConfigured
		}
		return nil
	})
	if e != nil {
		out = studyreview.QueuePage{}
	}
	return
}

type reviewHistoryCursor struct {
	At     time.Time `json:"at"`
	ID     string    `json:"id"`
	Filter string    `json:"filter"`
}

func (s *Store) ListReviewHistory(ctx context.Context, a studyreview.Access, q studyreview.HistoryQuery) (out studyreview.HistoryPage, e error) {
	if q.Limit == 0 {
		q.Limit = 20
	}
	if e = studyreview.ValidateHistory(q); e != nil {
		return
	}
	filter := knowledgeFingerprint(struct {
		KnowledgeID, TopicKey string
		Mode                  studyreview.Mode
		Rating                studyreview.Rating
	}{q.KnowledgeID, q.TopicKey, q.Mode, q.Rating})
	var cursor reviewHistoryCursor
	var cursorAt, cursorID any
	if q.Cursor != "" {
		raw, err := base64.RawURLEncoding.DecodeString(q.Cursor)
		if err != nil || json.Unmarshal(raw, &cursor) != nil || cursor.Filter != filter || !study.ValidID(cursor.ID) || cursor.At.IsZero() || cursor.At.Year() > 9999 {
			return out, studyreview.ErrInvalid
		}
		cursorAt = cursor.At
		cursorID = cursor.ID
	}
	e = s.studyReviewReadTx(ctx, a, func(ctx context.Context, tx *sql.Tx, u auth.User) error {
		out = studyreview.HistoryPage{ActorID: u.ID, Items: []studyreview.HistoryItem{}}
		rows, e := tx.QueryContext(ctx, `WITH at AS MATERIALIZED(SELECT clock_timestamp() as_of),chosen AS (
 SELECT s.session_id,s.knowledge_ref,s.mode,s.target,s.grade,s.graded_at,left(p.title,128) title,left(p.title_zh,128) title_zh,p.knowledge_type,k.published AND k.deleted_at IS NULL available,k.content_sha256<>s.knowledge_ref->>'contentSha256' material_changed
 FROM study_review_sessions s JOIN study_review_snapshots p ON p.session_id=s.session_id JOIN managed_knowledge k ON k.internal_id=s.knowledge_id
 WHERE s.owner_user_id=$1 AND s.status='completed' AND ($2='' OR s.knowledge_id=$2) AND ($3='' OR EXISTS(SELECT 1 FROM managed_knowledge_topics t WHERE t.internal_id=s.knowledge_id AND t.active AND (t.topic_key=$3 OR t.topic_key LIKE $3||'%'))) AND ($4='' OR s.mode=$4) AND ($5='' OR s.grade->>'rating'=$5) AND ($6::timestamptz IS NULL OR (s.graded_at,s.session_id)<($6::timestamptz,$7::uuid)) ORDER BY s.graded_at DESC,s.session_id DESC LIMIT $8)
 SELECT at.as_of,coalesce(jsonb_agg(jsonb_build_object('sessionId',session_id,'knowledge',knowledge_ref,'title',title,'titleZh',title_zh,'knowledgeType',knowledge_type,'mode',mode,'target',target,'grade',grade,'available',available,'materialChanged',material_changed) ORDER BY graded_at DESC,session_id DESC) FILTER(WHERE session_id IS NOT NULL),'[]'::jsonb) FROM at LEFT JOIN chosen ON true GROUP BY at.as_of`, u.ID, q.KnowledgeID, topicPrefix(q.TopicKey), q.Mode, q.Rating, cursorAt, cursorID, q.Limit+1)
		if e != nil {
			return e
		}
		defer rows.Close()
		if !rows.Next() {
			return studyreview.ErrNotConfigured
		}
		var raw []byte
		if e = rows.Scan(&out.AsOf, &raw); e != nil {
			return e
		}
		if json.Unmarshal(raw, &out.Items) != nil {
			return studyreview.ErrNotConfigured
		}
		if e = rows.Err(); e != nil {
			return e
		}
		if len(out.Items) > q.Limit {
			out.Items = out.Items[:q.Limit]
			last := out.Items[len(out.Items)-1]
			b := knowledgeJSON(reviewHistoryCursor{At: last.Grade.GradedAt, ID: last.SessionID, Filter: filter})
			encoded := base64.RawURLEncoding.EncodeToString(b)
			out.NextCursor = &encoded
		}
		return nil
	})
	if e != nil {
		out = studyreview.HistoryPage{}
	}
	return
}
