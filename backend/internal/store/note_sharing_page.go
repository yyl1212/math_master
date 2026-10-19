package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"github.com/yyl1212/math_master/backend/internal/auth"
	"github.com/yyl1212/math_master/backend/internal/knowledgeadmin"
	"github.com/yyl1212/math_master/backend/internal/notesharing"
	"github.com/yyl1212/math_master/backend/internal/pagination"
)

func (s *Store) ListOwnNotePage(ctx context.Context, a knowledgeadmin.Access, q notesharing.Query, p pagination.Request) (out pagination.ActorPage[notesharing.OwnNoteSummary], e error) {
	q.Limit = p.Limit
	q.Cursor = p.Cursor
	q, e = normalizedShareQuery(q)
	if e != nil {
		return out, e
	}
	p.Limit = q.Limit
	out.Items = []notesharing.OwnNoteSummary{}
	e = s.knowledgeTx(ctx, a, false, false, func(ctx context.Context, tx *sql.Tx, u auth.User) error {
		out.ActorID = u.ID
		if e := sharingConfigured(ctx, tx); e != nil {
			return e
		}
		scope := notesharing.QueryScope("own", u.ID, q)
		c, e := notesharing.DecodeCursor(q.Cursor, scope)
		if e != nil {
			return e
		}
		args := []any{u.ID, q.Q, notesharing.TopicPrefix(q.TopicKey), q.Status, noteWhitespace}
		anchor := ""
		if p.Cursor != "" {
			anchor = "(n.updated_at,n.knowledge_id)>=($6::timestamptz,$7)"
			args = append(args, c.At, c.ID)
		}
		rows, e := queryNumberedRows(ctx, tx, p, &out.Meta, `n.knowledge_id,coalesce(k.point->>'title',''),coalesce(k.point->>'title_zh',''),coalesce((SELECT jsonb_agg(t.topic_key ORDER BY t.topic_key) FROM managed_knowledge_topics t WHERE t.internal_id=k.internal_id AND t.active),'[]'),n.body,n.updated_at,k.published AND k.deleted_at IS NULL,n.revision,coalesce(h.status,'unshared'),coalesce(h.sequence,0),h.current_submission_id::text,h.approved_at,coalesce((SELECT reason FROM note_share_events e WHERE e.owner_user_id=n.owner_user_id AND e.knowledge_id=n.knowledge_id AND e.action IN ('return','hide') ORDER BY sequence DESC LIMIT 1),'')`,
			`managed_study_notes n JOIN managed_knowledge k ON k.internal_id=n.knowledge_id LEFT JOIN note_share_heads h ON h.owner_user_id=n.owner_user_id AND h.knowledge_id=n.knowledge_id`,
			`n.owner_user_id=$1 AND NOT n.deleted AND btrim(n.body,$5)<>'' AND ($2='' OR strpos(lower(n.body),lower($2))>0 OR strpos(lower(coalesce(k.point->>'title','')),lower($2))>0 OR strpos(lower(coalesce(k.point->>'title_zh','')),lower($2))>0) AND ($3='' OR EXISTS(SELECT 1 FROM managed_knowledge_topics t WHERE t.internal_id=k.internal_id AND t.active AND left(t.topic_key,length($3))=$3)) AND ($4='' OR coalesce(h.status,'unshared')=$4)`,
			`n.updated_at DESC,n.knowledge_id DESC`, anchor, args...)
		if e != nil {
			return e
		}
		defer rows.Close()
		for rows.Next() {
			var item notesharing.OwnNoteSummary
			var body string
			var topics []byte
			if e = rows.Scan(&item.KnowledgeID, &item.KnowledgeTitle, &item.KnowledgeTitleZH, &topics, &body, &item.UpdatedAt, &item.Available, &item.NoteRevision, &item.Share.Status, &item.Share.Sequence, &item.Share.SubmissionID, &item.Share.ApprovedAt, &item.Share.LastDecisionReason); e != nil {
				return e
			}
			if json.Unmarshal(topics, &item.TopicKeys) != nil {
				return notesharing.ErrNotReady
			}
			item.Excerpt = notesharing.Excerpt(body)
			item.Share.ActorID = u.ID
			item.Share.KnowledgeID = item.KnowledgeID
			out.Items = append(out.Items, item)
		}
		if e = rows.Err(); e != nil {
			return e
		}
		return nil
	})
	return
}

func (s *Store) ListSharedNotePage(ctx context.Context, a knowledgeadmin.Access, q notesharing.Query, p pagination.Request) (out pagination.ActorPage[notesharing.SharedNoteSummary], e error) {
	q.Limit = p.Limit
	q.Cursor = p.Cursor
	q, e = normalizedShareQuery(q)
	if e != nil {
		return out, e
	}
	if q.Status != "" {
		return out, notesharing.ErrInvalid
	}
	p.Limit = q.Limit
	out.Items = []notesharing.SharedNoteSummary{}
	e = s.knowledgeTx(ctx, a, false, false, func(ctx context.Context, tx *sql.Tx, u auth.User) error {
		out.ActorID = u.ID
		if e := sharingConfigured(ctx, tx); e != nil {
			return e
		}
		scope := notesharing.QueryScope("shared", u.ID, q)
		c, e := notesharing.DecodeCursor(q.Cursor, scope)
		if e != nil {
			return e
		}
		args := []any{q.Q, notesharing.TopicPrefix(q.TopicKey)}
		anchor := ""
		if p.Cursor != "" {
			anchor = "(h.approved_at,p.submission_id)>=($3::timestamptz,$4::uuid)"
			args = append(args, c.At, c.ID)
		}
		rows, e := queryNumberedRows(ctx, tx, p, &out.Meta, sharedNoteColumns,
			`note_share_heads h JOIN note_share_submissions p ON p.submission_id=h.current_submission_id JOIN managed_knowledge k ON k.internal_id=p.knowledge_id JOIN auth_users author ON author.id=p.owner_user_id`,
			`h.status='shared' AND k.published AND k.deleted_at IS NULL AND ($1='' OR strpos(lower(p.title),lower($1))>0 OR strpos(lower(p.body),lower($1))>0) AND ($2='' OR EXISTS(SELECT 1 FROM jsonb_array_elements_text(p.topic_keys) key WHERE left(key,length($2))=$2))`,
			`h.approved_at DESC,p.submission_id DESC`, anchor, args...)
		if e != nil {
			return e
		}
		defer rows.Close()
		for rows.Next() {
			d, e := scanSharedNote(rows)
			if e != nil {
				return e
			}
			out.Items = append(out.Items, d.SharedNoteSummary)
		}
		if e = rows.Err(); e != nil {
			return e
		}
		return nil
	})
	return
}

func (s *Store) ListAdminNoteSharePage(ctx context.Context, a knowledgeadmin.Access, q notesharing.Query, p pagination.Request) (out pagination.ActorPage[notesharing.AdminSummary], e error) {
	q.Limit = p.Limit
	q.Cursor = p.Cursor
	q, e = normalizedShareQuery(q)
	if e != nil {
		return out, e
	}
	if q.Q != "" || q.TopicKey != "" || q.Status == notesharing.Unshared {
		return out, notesharing.ErrInvalid
	}
	if q.Status == "" {
		q.Status = notesharing.Pending
	}
	p.Limit = q.Limit
	out.Items = []notesharing.AdminSummary{}
	e = s.knowledgeTx(ctx, a, false, true, func(ctx context.Context, tx *sql.Tx, u auth.User) error {
		out.ActorID = u.ID
		if e := sharingConfigured(ctx, tx); e != nil {
			return e
		}
		scope := notesharing.QueryScope("admin", u.ID, q)
		c, e := notesharing.DecodeCursor(q.Cursor, scope)
		if e != nil {
			return e
		}
		args := []any{q.Status}
		anchor := ""
		if p.Cursor != "" {
			anchor = "(p.submitted_at,p.submission_id)<=($2::timestamptz,$3::uuid)"
			args = append(args, c.At, c.ID)
		}
		rows, e := queryNumberedRows(ctx, tx, p, &out.Meta, `p.submission_id::text,author.username,p.knowledge_id,p.title,p.submitted_at,h.status,h.sequence`,
			`note_share_heads h JOIN note_share_submissions p ON p.submission_id=h.current_submission_id JOIN auth_users author ON author.id=p.owner_user_id`, `h.status=$1`, `p.submitted_at ASC,p.submission_id ASC`, anchor, args...)
		if e != nil {
			return e
		}
		defer rows.Close()
		for rows.Next() {
			var item notesharing.AdminSummary
			if e = rows.Scan(&item.SubmissionID, &item.AuthorName, &item.KnowledgeID, &item.Title, &item.SubmittedAt, &item.Status, &item.Sequence); e != nil {
				return e
			}
			out.Items = append(out.Items, item)
		}
		if e = rows.Err(); e != nil {
			return e
		}
		return nil
	})
	return
}
