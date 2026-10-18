package store

import (
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"github.com/yyl1212/math_master/backend/internal/auth"
	"github.com/yyl1212/math_master/backend/internal/knowledgeadmin"
	"github.com/yyl1212/math_master/backend/internal/pagination"
	"github.com/yyl1212/math_master/backend/internal/study"
)

func (s *Store) ListManagedHistoryPage(ctx context.Context, a knowledgeadmin.Access, q study.HistoryQuery, p pagination.Request) (out pagination.ActorPage[knowledgeadmin.ManagedHistoryEntry], e error) {
	if p.Limit < 1 || p.Limit > 50 {
		return out, knowledgeadmin.ErrInvalid
	}
	var cursor managedHistoryCursor
	if p.Cursor != "" {
		b, err := base64.RawURLEncoding.DecodeString(p.Cursor)
		if err != nil || len(p.Cursor) > 512 || json.Unmarshal(b, &cursor) != nil || cursor.At.IsZero() || !study.ValidID(cursor.ID) {
			return out, knowledgeadmin.ErrInvalid
		}
	}
	out.Items = []knowledgeadmin.ManagedHistoryEntry{}
	e = s.knowledgeTx(ctx, a, false, false, func(ctx context.Context, tx *sql.Tx, u auth.User) error {
		if e := managedStudyMode(ctx, tx); e != nil {
			return e
		}
		out.ActorID = u.ID
		args := []any{u.ID, q.KnowledgeID, q.TopicID, q.Kind, q.From, q.To}
		anchor := ""
		if p.Cursor != "" {
			anchor = "(e.recorded_at,e.event_id)>=($7::timestamptz,$8::uuid)"
			args = append(args, cursor.At, cursor.ID)
		}
		rows, e := queryNumberedRows(ctx, tx, p, &out.Meta, `e.event_id::text,e.knowledge_ref,e.topic_keys,e.kind,e.note_revision,e.review_id::text,e.recorded_at,coalesce(k.point->>'title',''),coalesce(k.point->>'title_zh',''),coalesce(k.published AND k.deleted_at IS NULL,false)`, `managed_study_events e LEFT JOIN managed_knowledge k ON k.internal_id=e.knowledge_id`, `e.owner_user_id=$1 AND ($2='' OR e.knowledge_id=$2) AND ($3='' OR EXISTS(SELECT 1 FROM jsonb_array_elements_text(e.topic_keys) key WHERE key=$3 OR key LIKE $3||'%')) AND ($4='' OR e.kind=$4) AND ($5::timestamptz IS NULL OR e.recorded_at>=$5) AND ($6::timestamptz IS NULL OR e.recorded_at<=$6)`, `e.recorded_at DESC,e.event_id DESC`, anchor, args...)
		if e != nil {
			return e
		}
		defer rows.Close()
		for rows.Next() {
			var v knowledgeadmin.ManagedHistoryEntry
			var ref, topics []byte
			if e = rows.Scan(&v.ID, &ref, &topics, &v.Kind, &v.NoteRevision, &v.ReviewID, &v.RecordedAt, &v.Title, &v.TitleZH, &v.Available); e != nil {
				return e
			}
			if json.Unmarshal(ref, &v.Knowledge) != nil || json.Unmarshal(topics, &v.TopicKeys) != nil {
				return knowledgeadmin.ErrNotConfigured
			}
			v.SourceKind = "managed"
			out.Items = append(out.Items, v)
		}
		return rows.Err()
	})
	return
}
