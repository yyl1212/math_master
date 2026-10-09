package store

import (
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"github.com/yyl1212/math_master/backend/internal/auth"
	"github.com/yyl1212/math_master/backend/internal/knowledgeadmin"
	"github.com/yyl1212/math_master/backend/internal/study"
	"time"
)

type managedHistoryCursor struct {
	At time.Time `json:"at"`
	ID string    `json:"id"`
}

func (s *Store) ListManagedHistory(ctx context.Context, a knowledgeadmin.Access, q study.HistoryQuery) (out knowledgeadmin.ManagedHistoryPage, e error) {
	limit := q.Limit
	if limit == 0 {
		limit = 20
	}
	if limit < 1 || limit > 50 || len(q.Cursor) > 512 {
		return out, knowledgeadmin.ErrInvalid
	}
	var cursor managedHistoryCursor
	if q.Cursor != "" {
		b, e := base64.RawURLEncoding.DecodeString(q.Cursor)
		if e != nil || json.Unmarshal(b, &cursor) != nil || cursor.At.IsZero() || !study.ValidID(cursor.ID) {
			return out, knowledgeadmin.ErrInvalid
		}
	}
	e = s.knowledgeTx(ctx, a, false, false, func(ctx context.Context, tx *sql.Tx, u auth.User) error {
		if e := managedStudyMode(ctx, tx); e != nil {
			return e
		}
		out = knowledgeadmin.ManagedHistoryPage{ActorID: u.ID, Items: []knowledgeadmin.ManagedHistoryEntry{}}
		var at, id any
		if !cursor.At.IsZero() {
			at, id = cursor.At, cursor.ID
		}
		rows, e := tx.QueryContext(ctx, `SELECT event_id::text,knowledge_ref,topic_keys,kind,note_revision,review_id::text,recorded_at FROM managed_study_events WHERE owner_user_id=$1 AND ($2='' OR knowledge_id=$2) AND ($3='' OR EXISTS(SELECT 1 FROM jsonb_array_elements_text(topic_keys) key WHERE key=$3 OR key LIKE $3||'%')) AND ($4='' OR kind=$4) AND ($5::timestamptz IS NULL OR recorded_at>=$5) AND ($6::timestamptz IS NULL OR recorded_at<=$6) AND ($7::timestamptz IS NULL OR (recorded_at,event_id)<($7,$8::uuid)) ORDER BY recorded_at DESC,event_id DESC LIMIT $9`, u.ID, q.KnowledgeID, q.TopicID, q.Kind, q.From, q.To, at, id, limit+1)
		if e != nil {
			return e
		}
		defer rows.Close()
		for rows.Next() {
			var v knowledgeadmin.ManagedHistoryEntry
			var ref, topics []byte
			if e = rows.Scan(&v.ID, &ref, &topics, &v.Kind, &v.NoteRevision, &v.ReviewID, &v.RecordedAt); e != nil {
				return e
			}
			if json.Unmarshal(ref, &v.Knowledge) != nil || json.Unmarshal(topics, &v.TopicKeys) != nil {
				return knowledgeadmin.ErrNotConfigured
			}
			v.SourceKind = "managed"
			out.Items = append(out.Items, v)
		}
		if e = rows.Err(); e != nil {
			return e
		}
		if len(out.Items) > limit {
			last := out.Items[limit-1]
			b, _ := json.Marshal(managedHistoryCursor{last.RecordedAt, last.ID})
			c := base64.RawURLEncoding.EncodeToString(b)
			out.NextCursor = &c
			out.Items = out.Items[:limit]
		}
		return nil
	})
	return
}
