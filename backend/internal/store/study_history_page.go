package store

import (
	"context"
	"database/sql"
	"github.com/yyl1212/math_master/backend/internal/auth"
	"github.com/yyl1212/math_master/backend/internal/pagination"
	"github.com/yyl1212/math_master/backend/internal/study"
	"sort"
)

func (s *Store) ListStudyHistoryPage(ctx context.Context, a study.Access, q study.HistoryQuery, p pagination.Request) (pagination.ActorPage[study.HistoryEntry], error) {
	out := pagination.ActorPage[study.HistoryEntry]{}
	out.Items = []study.HistoryEntry{}
	q.Limit = p.Limit
	q.Cursor = p.Cursor
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
		args := []any{u.ID, q.KnowledgeID, q.Kind, q.From, q.To, ids}
		anchor := ""
		if p.Cursor != "" {
			anchor = "(p.recorded_at,p.id)>=($7::timestamptz,$8::uuid)"
			args = append(args, cursorAt, cursorID)
		}
		rows, e := queryNumberedRows(ctx, tx, p, &out.Meta, `p.id::text,p.knowledge_id,p.knowledge_version,p.knowledge_sha256,p.taxonomy_version_id,p.taxonomy_head::text,p.kind,p.recorded_at,p.note_revision,p.review_id::text,p.source_kind,p.origin_event_id::text,coalesce(k.body->>'title',''),coalesce(k.body->>'titleZh','')`, `study_events p LEFT JOIN knowledge_versions k ON k.id=p.knowledge_id AND k.version=p.knowledge_version AND k.sha256=p.knowledge_sha256`, `p.owner_user_id=$1 AND ($2='' OR p.knowledge_id=$2) AND ($3='' OR p.kind=$3) AND ($4::timestamptz IS NULL OR p.recorded_at>=$4) AND ($5::timestamptz IS NULL OR p.recorded_at<$5) AND ($6::text[] IS NULL OR p.knowledge_id=ANY($6))`, `p.recorded_at DESC,p.id DESC`, anchor, args...)
		if e != nil {
			return e
		}
		defer rows.Close()
		for rows.Next() {
			var v study.HistoryEntry
			if e = rows.Scan(&v.ID, &v.Knowledge.ID, &v.Knowledge.Version, &v.Knowledge.SHA256, &v.TaxonomyVersionID, &v.TaxonomyHead, &v.Kind, &v.OccurredAt, &v.NoteRevision, &v.ReviewID, &v.SourceKind, &v.OriginEventID, &v.Title, &v.TitleZH); e != nil {
				return e
			}
			v.OccurredAt = v.OccurredAt.UTC()
			out.Items = append(out.Items, v)
		}
		if e = rows.Err(); e != nil {
			return e
		}
		return nil
	})
	return out, e
}
