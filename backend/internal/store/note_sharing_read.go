package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"github.com/yyl1212/math_master/backend/internal/auth"
	"github.com/yyl1212/math_master/backend/internal/knowledgeadmin"
	"github.com/yyl1212/math_master/backend/internal/notesharing"
	"github.com/yyl1212/math_master/backend/internal/study"
	"time"
)

const sharedNoteFrom = ` FROM note_share_heads h JOIN note_share_submissions p ON p.submission_id=h.current_submission_id JOIN managed_knowledge k ON k.internal_id=p.knowledge_id JOIN auth_users author ON author.id=p.owner_user_id WHERE h.status='shared' AND k.published AND k.deleted_at IS NULL `
const sharedNoteColumns = `p.submission_id::text,p.title,p.body,p.body_sha256,author.username,h.approved_at,p.knowledge_id,p.knowledge_title,p.knowledge_title_zh,p.topic_keys,p.note_knowledge_ref<>jsonb_build_object('id',k.internal_id,'contentSha256',k.content_sha256,'sourceKind','managed')`

type shareScanner interface{ Scan(...any) error }

func scanSharedNote(row shareScanner) (out notesharing.SharedNoteDetail, e error) {
	var topics []byte
	var bodySHA string
	e = row.Scan(&out.SubmissionID, &out.Title, &out.Body, &bodySHA, &out.AuthorName, &out.ApprovedAt, &out.KnowledgeID, &out.KnowledgeTitle, &out.KnowledgeTitleZH, &topics, &out.MaterialChanged)
	if e != nil {
		return
	}
	if json.Unmarshal(topics, &out.TopicKeys) != nil || knowledgeadmin.DigestBytes([]byte(out.Body)) != bodySHA || notesharing.ValidateBody(out.Body) != nil {
		return out, notesharing.ErrNotReady
	}
	out.Excerpt = notesharing.Excerpt(out.Body)
	return
}
func (s *Store) publicSharingTx(ctx context.Context, mode bool, fn func(context.Context, *sql.Tx) error) error {
	ctx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()
	tx, e := s.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true, Isolation: sql.LevelRepeatableRead})
	if e != nil {
		return e
	}
	defer tx.Rollback()
	if _, e = tx.ExecContext(ctx, "SET LOCAL lock_timeout='1s'"); e != nil {
		return e
	}
	if mode {
		if e = knowledgeConfigured(ctx, tx); e != nil {
			return e
		}
		if e = managedStudyMode(ctx, tx); e != nil {
			return e
		}
	}
	if e = noteSharingSchemaIntegrity(ctx, tx); e != nil {
		return e
	}
	if e = fn(ctx, tx); e != nil {
		return e
	}
	return tx.Commit()
}
func (s *Store) ReadSchemaHealth(ctx context.Context) (out notesharing.NoteSharingHealth, e error) {
	out.Capability = true
	e = s.publicSharingTx(ctx, false, func(context.Context, *sql.Tx) error { out.SchemaReady = true; return nil })
	if errors.Is(e, notesharing.ErrNotReady) {
		return out, nil
	}
	return
}
func (s *Store) Latest(ctx context.Context) (out notesharing.LatestSharedNotePage, e error) {
	out.Items = []notesharing.SharedNoteSummary{}
	e = s.publicSharingTx(ctx, true, func(ctx context.Context, tx *sql.Tx) error {
		rows, e := tx.QueryContext(ctx, "SELECT "+sharedNoteColumns+sharedNoteFrom+" ORDER BY h.approved_at DESC,p.submission_id DESC LIMIT 10")
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
		return rows.Err()
	})
	return
}
func (s *Store) ReadPublic(ctx context.Context, id string) (out notesharing.SharedNoteDetail, e error) {
	if !study.ValidID(id) {
		return out, notesharing.ErrInvalid
	}
	e = s.publicSharingTx(ctx, true, func(ctx context.Context, tx *sql.Tx) error {
		query := `WITH permitted AS (SELECT p.submission_id` + sharedNoteFrom + ` ORDER BY h.approved_at DESC,p.submission_id DESC LIMIT 10) SELECT ` + sharedNoteColumns + sharedNoteFrom + ` AND p.submission_id=$1 AND p.submission_id IN (SELECT submission_id FROM permitted)`
		out, e = scanSharedNote(tx.QueryRowContext(ctx, query, id))
		if errors.Is(e, sql.ErrNoRows) {
			return notesharing.ErrNotPublic
		}
		return e
	})
	return
}
func (s *Store) ReadShared(ctx context.Context, a knowledgeadmin.Access, id string) (out notesharing.AuthenticatedSharedNoteDetail, e error) {
	if !study.ValidID(id) {
		return out, notesharing.ErrInvalid
	}
	e = s.knowledgeTx(ctx, a, false, false, func(ctx context.Context, tx *sql.Tx, u auth.User) error {
		out.ActorID = u.ID
		if e := sharingConfigured(ctx, tx); e != nil {
			return e
		}
		out.Note, e = scanSharedNote(tx.QueryRowContext(ctx, "SELECT "+sharedNoteColumns+sharedNoteFrom+" AND p.submission_id=$1", id))
		if errors.Is(e, sql.ErrNoRows) {
			return notesharing.ErrNotPublic
		}
		return e
	})
	return
}
func (s *Store) ListShared(ctx context.Context, a knowledgeadmin.Access, q notesharing.Query) (out notesharing.SharedNotePage, e error) {
	q, e = normalizedShareQuery(q)
	if e != nil {
		return out, e
	}
	if q.Status != "" {
		return out, notesharing.ErrInvalid
	}
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
		rows, e := tx.QueryContext(ctx, "SELECT "+sharedNoteColumns+sharedNoteFrom+` AND ($1='' OR strpos(lower(p.title),lower($1))>0 OR strpos(lower(p.body),lower($1))>0) AND ($2='' OR EXISTS(SELECT 1 FROM jsonb_array_elements_text(p.topic_keys) key WHERE left(key,length($2))=$2)) AND ($3::timestamptz IS NULL OR (h.approved_at,p.submission_id)<($3,$4::uuid)) ORDER BY h.approved_at DESC,p.submission_id DESC LIMIT $5`, q.Q, notesharing.TopicPrefix(q.TopicKey), cursorAt(c), nullableCursorID(c.ID), q.Limit+1)
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
		if len(out.Items) > q.Limit {
			out.Items = out.Items[:q.Limit]
			last := out.Items[len(out.Items)-1]
			next := notesharing.EncodeCursor(last.ApprovedAt, last.SubmissionID, scope)
			out.NextCursor = &next
		}
		return nil
	})
	return
}

type shareSnapshot struct {
	ID, OwnerID, KnowledgeID, Title, Body, BodySHA, KnowledgeTitle, KnowledgeTitleZH string
	NoteRevision                                                                     int64
	NoteKnowledgeRef                                                                 knowledgeadmin.Ref
	TopicKeys                                                                        []string
	SubmittedAt                                                                      time.Time
}

func readShareSnapshot(ctx context.Context, tx *sql.Tx, id string) (out shareSnapshot, e error) {
	var ref, topics []byte
	e = tx.QueryRowContext(ctx, `SELECT submission_id::text,owner_user_id::text,knowledge_id,note_revision,note_knowledge_ref,title,body,body_sha256,knowledge_title,knowledge_title_zh,topic_keys,submitted_at FROM note_share_submissions WHERE submission_id=$1`, id).Scan(&out.ID, &out.OwnerID, &out.KnowledgeID, &out.NoteRevision, &ref, &out.Title, &out.Body, &out.BodySHA, &out.KnowledgeTitle, &out.KnowledgeTitleZH, &topics, &out.SubmittedAt)
	if errors.Is(e, sql.ErrNoRows) {
		return out, notesharing.ErrNotFound
	}
	if e != nil {
		return out, e
	}
	if json.Unmarshal(ref, &out.NoteKnowledgeRef) != nil || json.Unmarshal(topics, &out.TopicKeys) != nil || out.NoteKnowledgeRef.ID != out.KnowledgeID || knowledgeadmin.DigestBytes([]byte(out.Body)) != out.BodySHA || notesharing.ValidateBody(out.Body) != nil {
		return out, notesharing.ErrNotReady
	}
	return
}
func (s *Store) ReadOwnHead(ctx context.Context, a knowledgeadmin.Access, kid string) (out notesharing.Head, e error) {
	e = s.knowledgeTx(ctx, a, false, false, func(ctx context.Context, tx *sql.Tx, u auth.User) error {
		if e := sharingConfigured(ctx, tx); e != nil {
			return e
		}
		k, e := managedStudyCurrent(ctx, tx, kid, false)
		if e != nil {
			return e
		}
		n, e := readManagedNote(ctx, tx, u.ID, kid, false)
		if e != nil {
			return e
		}
		if (!k.Published || k.Deleted) && n.Revision == 0 {
			return notesharing.ErrNotFound
		}
		out, e = readShareHead(ctx, tx, u.ID, kid, false)
		return e
	})
	return
}

const noteWhitespace = " \t\n\r\f\v\u0085\u00a0\u1680\u2000\u2001\u2002\u2003\u2004\u2005\u2006\u2007\u2008\u2009\u200a\u2028\u2029\u202f\u205f\u3000"

func normalizedShareQuery(q notesharing.Query) (notesharing.Query, error) {
	if e := notesharing.ValidateQuery(q); e != nil {
		return q, e
	}
	if q.Limit == 0 {
		q.Limit = notesharing.DefaultLimit
	}
	return q, nil
}
func cursorAt(c notesharing.Cursor) any {
	if c.At.IsZero() {
		return nil
	}
	return c.At
}
func (s *Store) ListOwnNotes(ctx context.Context, a knowledgeadmin.Access, q notesharing.Query) (out notesharing.OwnNotePage, e error) {
	q, e = normalizedShareQuery(q)
	if e != nil {
		return out, e
	}
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
		rows, e := tx.QueryContext(ctx, `SELECT n.knowledge_id,coalesce(k.point->>'title',''),coalesce(k.point->>'title_zh',''),
coalesce((SELECT jsonb_agg(t.topic_key ORDER BY t.topic_key) FROM managed_knowledge_topics t WHERE t.internal_id=k.internal_id AND t.active),'[]'),
n.body,n.updated_at,k.published AND k.deleted_at IS NULL,n.revision,coalesce(h.status,'unshared'),coalesce(h.sequence,0),h.current_submission_id::text,h.approved_at,
coalesce((SELECT reason FROM note_share_events e WHERE e.owner_user_id=n.owner_user_id AND e.knowledge_id=n.knowledge_id AND e.action IN ('return','hide') ORDER BY sequence DESC LIMIT 1),'')
FROM managed_study_notes n JOIN managed_knowledge k ON k.internal_id=n.knowledge_id LEFT JOIN note_share_heads h ON h.owner_user_id=n.owner_user_id AND h.knowledge_id=n.knowledge_id
WHERE n.owner_user_id=$1 AND NOT n.deleted AND btrim(n.body,$8)<>''
AND ($2='' OR strpos(lower(n.body),lower($2))>0 OR strpos(lower(coalesce(k.point->>'title','')),lower($2))>0 OR strpos(lower(coalesce(k.point->>'title_zh','')),lower($2))>0)
AND ($3='' OR EXISTS(SELECT 1 FROM managed_knowledge_topics t WHERE t.internal_id=k.internal_id AND t.active AND left(t.topic_key,length($3))=$3))
AND ($4='' OR coalesce(h.status,'unshared')=$4)
AND ($5::timestamptz IS NULL OR (n.updated_at,n.knowledge_id)<($5,$6)) ORDER BY n.updated_at DESC,n.knowledge_id DESC LIMIT $7`, u.ID, q.Q, notesharing.TopicPrefix(q.TopicKey), q.Status, cursorAt(c), c.ID, q.Limit+1, noteWhitespace)
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
		if len(out.Items) > q.Limit {
			out.Items = out.Items[:q.Limit]
			last := out.Items[len(out.Items)-1]
			next := notesharing.EncodeCursor(last.UpdatedAt, last.KnowledgeID, scope)
			out.NextCursor = &next
		}
		return nil
	})
	return
}
func (s *Store) ListAdmin(ctx context.Context, a knowledgeadmin.Access, q notesharing.Query) (out notesharing.AdminNoteSharePage, e error) {
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
		rows, e := tx.QueryContext(ctx, `SELECT p.submission_id::text,author.username,p.knowledge_id,p.title,p.submitted_at,h.status,h.sequence FROM note_share_heads h JOIN note_share_submissions p ON p.submission_id=h.current_submission_id JOIN auth_users author ON author.id=p.owner_user_id WHERE h.status=$1 AND ($2::timestamptz IS NULL OR (p.submitted_at,p.submission_id)>($2,$3::uuid)) ORDER BY p.submitted_at ASC,p.submission_id ASC LIMIT $4`, q.Status, cursorAt(c), nullableCursorID(c.ID), q.Limit+1)
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
		if len(out.Items) > q.Limit {
			out.Items = out.Items[:q.Limit]
			last := out.Items[len(out.Items)-1]
			next := notesharing.EncodeCursor(last.SubmittedAt, last.SubmissionID, scope)
			out.NextCursor = &next
		}
		return nil
	})
	return
}
func nullableCursorID(id string) any {
	if id == "" {
		return nil
	}
	return id
}
func shareActionStatus(action string) string {
	switch action {
	case "submit":
		return notesharing.Pending
	case "approve":
		return notesharing.Shared
	case "return":
		return notesharing.Returned
	case "withdraw":
		return notesharing.Withdrawn
	case "hide":
		return notesharing.Hidden
	}
	return ""
}
func (s *Store) ReadAdmin(ctx context.Context, a knowledgeadmin.Access, id, eventsCursor string) (out notesharing.AdminNoteShareDetail, e error) {
	if !study.ValidID(id) {
		return out, notesharing.ErrInvalid
	}
	out.Events = []notesharing.ShareEventSummary{}
	e = s.knowledgeTx(ctx, a, false, true, func(ctx context.Context, tx *sql.Tx, u auth.User) error {
		out.ActorID = u.ID
		if e := sharingConfigured(ctx, tx); e != nil {
			return e
		}
		snap, e := readShareSnapshot(ctx, tx, id)
		if e != nil {
			return e
		}
		out.SubmissionID = snap.ID
		out.KnowledgeID = snap.KnowledgeID
		out.Title = snap.Title
		out.SubmittedAt = snap.SubmittedAt
		out.Body = snap.Body
		out.NoteRevision = snap.NoteRevision
		out.NoteKnowledgeRef = snap.NoteKnowledgeRef
		out.KnowledgeTitle = snap.KnowledgeTitle
		out.KnowledgeTitleZH = snap.KnowledgeTitleZH
		out.TopicKeys = snap.TopicKeys
		if e = tx.QueryRowContext(ctx, "SELECT username FROM auth_users WHERE id=$1", snap.OwnerID).Scan(&out.AuthorName); e != nil {
			return e
		}
		var action string
		if e = tx.QueryRowContext(ctx, `SELECT action,sequence FROM note_share_events WHERE submission_id=$1 ORDER BY sequence DESC LIMIT 1`, id).Scan(&action, &out.Sequence); e != nil {
			return notesharing.ErrNotReady
		}
		out.Status = shareActionStatus(action)
		if e = tx.QueryRowContext(ctx, `SELECT max(recorded_at) FROM note_share_events WHERE submission_id=$1 AND action='approve'`, id).Scan(&out.ApprovedAt); e != nil {
			return e
		}
		scope := notesharing.QueryScope("events", id, notesharing.Query{})
		c, e := notesharing.DecodeCursor(eventsCursor, scope)
		if e != nil {
			return e
		}
		rows, e := tx.QueryContext(ctx, `SELECT e.event_id::text,e.action,actor.username,e.reason,e.sequence,e.recorded_at FROM note_share_events e JOIN auth_users actor ON actor.id=e.actor_user_id WHERE e.submission_id=$1 AND ($2::timestamptz IS NULL OR (e.recorded_at,e.event_id)<($2,$3::uuid)) ORDER BY e.recorded_at DESC,e.event_id DESC LIMIT 51`, id, cursorAt(c), nullableCursorID(c.ID))
		if e != nil {
			return e
		}
		defer rows.Close()
		ids := []string{}
		for rows.Next() {
			var v notesharing.ShareEventSummary
			var eid string
			if e = rows.Scan(&eid, &v.Action, &v.ActorName, &v.Reason, &v.Sequence, &v.RecordedAt); e != nil {
				return e
			}
			out.Events = append(out.Events, v)
			ids = append(ids, eid)
		}
		if e = rows.Err(); e != nil {
			return e
		}
		if len(out.Events) > 50 {
			out.Events = out.Events[:50]
			next := notesharing.EncodeCursor(out.Events[49].RecordedAt, ids[49], scope)
			out.NextEventsCursor = &next
		}
		return nil
	})
	return
}
