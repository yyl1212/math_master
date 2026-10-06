package store

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"github.com/yyl1212/math_master/backend/internal/auth"
	"github.com/yyl1212/math_master/backend/internal/study"
	"sort"
	"time"
)

type studyCursor struct {
	Version int    `json:"v"`
	At      string `json:"at"`
	ID      string `json:"id"`
}

func encodeStudyCursor(at time.Time, id string) string {
	b, _ := json.Marshal(studyCursor{1, at.UTC().Format(time.RFC3339Nano), id})
	return base64.RawURLEncoding.EncodeToString(b)
}
func decodeStudyCursor(value string) (time.Time, string, error) {
	var out studyCursor
	if len(value) > 512 {
		return time.Time{}, "", study.ErrInvalid
	}
	for _, c := range value {
		if c < 32 || c > 126 {
			return time.Time{}, "", study.ErrInvalid
		}
	}
	raw, e := base64.RawURLEncoding.DecodeString(value)
	if e != nil || json.Unmarshal(raw, &out) != nil || out.Version != 1 || !study.ValidID(out.ID) {
		return time.Time{}, "", study.ErrInvalid
	}
	canonical, _ := json.Marshal(out)
	if !bytes.Equal(raw, canonical) {
		return time.Time{}, "", study.ErrInvalid
	}
	at, e := time.Parse(time.RFC3339Nano, out.At)
	if e != nil {
		return time.Time{}, "", study.ErrInvalid
	}
	return at, out.ID, nil
}
func (s *Store) ListStudyHistory(ctx context.Context, a study.Access, q study.HistoryQuery) (study.HistoryPage, error) {
	out := study.HistoryPage{Items: []study.HistoryEntry{}}
	if q.Limit == 0 {
		q.Limit = 20
	}
	validKind := false
	switch q.Kind {
	case "", "started", "completed", "review-started", "review-finished", "note-saved", "note-deleted":
		validKind = true
	}
	if q.Limit < 1 || q.Limit > 50 || !validKind || (q.KnowledgeID != "" && !study.ValidKnowledgeID(q.KnowledgeID)) || (q.TopicID != "" && !study.ValidKnowledgeID(q.TopicID)) || (q.From != nil && q.To != nil && !q.From.Before(*q.To)) {
		return out, study.ErrInvalid
	}
	var cursorAt any
	cursorID := ""
	if q.Cursor != "" {
		at, id, e := decodeStudyCursor(q.Cursor)
		if e != nil {
			return out, e
		}
		cursorAt = at
		cursorID = id
	}
	e := s.studyReadTx(ctx, a, func(ctx context.Context, tx *sql.Tx, u auth.User) error {
		out.ActorID = u.ID
		out.Limit = q.Limit
		var ids any
		if q.TopicID != "" {
			scope, e := studyScopeTx(ctx, tx)
			if e != nil {
				return e
			}
			nodes, e := studyNodesTx(ctx, tx, scope.pair.TaxonomyVersionID)
			if e != nil {
				return e
			}
			if _, ok := nodes[q.TopicID]; !ok {
				return auth.ErrNotFound
			}
			names := []string{}
			for id := range studyMembership(scope, nodes)[q.TopicID] {
				names = append(names, id)
			}
			sort.Strings(names)
			ids = names
		}
		rows, e := tx.QueryContext(ctx, `SELECT id::text,knowledge_id,knowledge_version,knowledge_sha256,taxonomy_version_id,taxonomy_head::text,kind,recorded_at,note_revision,review_id::text,source_kind,origin_event_id::text FROM study_events WHERE owner_user_id=$1 AND ($2='' OR knowledge_id=$2) AND ($3='' OR kind=$3) AND ($4::timestamptz IS NULL OR recorded_at>=$4) AND ($5::timestamptz IS NULL OR recorded_at<$5) AND ($6::timestamptz IS NULL OR (recorded_at,id)<($6,$7::uuid)) AND ($8::text[] IS NULL OR knowledge_id=ANY($8)) ORDER BY recorded_at DESC,id DESC LIMIT $9`, u.ID, q.KnowledgeID, q.Kind, q.From, q.To, cursorAt, nilIfEmpty(cursorID), ids, q.Limit+1)
		if e != nil {
			return e
		}
		defer rows.Close()
		for rows.Next() {
			var v study.HistoryEntry
			if e = rows.Scan(&v.ID, &v.Knowledge.ID, &v.Knowledge.Version, &v.Knowledge.SHA256, &v.TaxonomyVersionID, &v.TaxonomyHead, &v.Kind, &v.OccurredAt, &v.NoteRevision, &v.ReviewID, &v.SourceKind, &v.OriginEventID); e != nil {
				return e
			}
			v.OccurredAt = v.OccurredAt.UTC()
			out.Items = append(out.Items, v)
		}
		if e = rows.Err(); e != nil {
			return e
		}
		if len(out.Items) > q.Limit {
			out.Items = out.Items[:q.Limit]
			last := out.Items[len(out.Items)-1]
			cursor := encodeStudyCursor(last.OccurredAt, last.ID)
			out.NextCursor = &cursor
		}
		return nil
	})
	return out, e
}
func nilIfEmpty(value string) any {
	if value == "" {
		return nil
	}
	return value
}
