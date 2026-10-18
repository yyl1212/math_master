package store

import (
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"github.com/yyl1212/math_master/backend/internal/auth"
	"github.com/yyl1212/math_master/backend/internal/pagination"
	"github.com/yyl1212/math_master/backend/internal/study"
	"github.com/yyl1212/math_master/backend/internal/studyreview"
)

func (s *Store) ListReviewHistoryPage(ctx context.Context, a studyreview.Access, q studyreview.HistoryQuery, p pagination.Request) (out studyreview.NumberedHistoryPage, e error) {
	q.Limit = p.Limit
	q.Cursor = p.Cursor
	if e = studyreview.ValidateHistory(q); e != nil {
		return
	}
	filter := knowledgeFingerprint(struct {
		KnowledgeID, TopicKey string
		Mode                  studyreview.Mode
		Rating                studyreview.Rating
	}{q.KnowledgeID, q.TopicKey, q.Mode, q.Rating})
	var cursor reviewHistoryCursor
	if p.Cursor != "" {
		raw, err := base64.RawURLEncoding.DecodeString(p.Cursor)
		if err != nil || json.Unmarshal(raw, &cursor) != nil || cursor.Filter != filter || !study.ValidID(cursor.ID) || cursor.At.IsZero() || cursor.At.Year() > 9999 {
			return out, studyreview.ErrInvalid
		}
	}
	out.Items = []studyreview.HistoryItem{}
	e = s.studyReviewReadTx(ctx, a, func(ctx context.Context, tx *sql.Tx, u auth.User) error {
		out.ActorID = u.ID
		args := []any{u.ID, q.KnowledgeID, topicPrefix(q.TopicKey), q.Mode, q.Rating}
		anchor := ""
		if p.Cursor != "" {
			anchor = "(s.graded_at,s.session_id)>=($6::timestamptz,$7::uuid)"
			args = append(args, cursor.At, cursor.ID)
		}
		rows, e := queryNumberedRows(ctx, tx, p, &out.Meta, `clock_timestamp(),s.session_id::text,s.knowledge_ref,left(p.title,128),left(p.title_zh,128),p.knowledge_type,s.mode,s.target,s.grade,k.published AND k.deleted_at IS NULL,k.content_sha256<>s.knowledge_ref->>'contentSha256'`, `study_review_sessions s JOIN study_review_snapshots p ON p.session_id=s.session_id JOIN managed_knowledge k ON k.internal_id=s.knowledge_id`, `s.owner_user_id=$1 AND s.status='completed' AND ($2='' OR s.knowledge_id=$2) AND ($3='' OR EXISTS(SELECT 1 FROM managed_knowledge_topics t WHERE t.internal_id=s.knowledge_id AND t.active AND (t.topic_key=$3 OR t.topic_key LIKE $3||'%'))) AND ($4='' OR s.mode=$4) AND ($5='' OR s.grade->>'rating'=$5)`, `s.graded_at DESC,s.session_id DESC`, anchor, args...)
		if e != nil {
			return e
		}
		defer rows.Close()
		for rows.Next() {
			var item studyreview.HistoryItem
			var ref, grade []byte
			if e = rows.Scan(&out.AsOf, &item.SessionID, &ref, &item.Title, &item.TitleZH, &item.KnowledgeType, &item.Mode, &item.Target, &grade, &item.Available, &item.MaterialChanged); e != nil {
				return e
			}
			if json.Unmarshal(ref, &item.Knowledge) != nil || json.Unmarshal(grade, &item.Grade) != nil {
				return studyreview.ErrNotConfigured
			}
			out.Items = append(out.Items, item)
		}
		if e = rows.Err(); e != nil {
			return e
		}
		if out.AsOf.IsZero() {
			return tx.QueryRowContext(ctx, `SELECT clock_timestamp()`).Scan(&out.AsOf)
		}
		return nil
	})
	return
}
