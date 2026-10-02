package store

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"github.com/yyl1212/math_master/backend/internal/auth"
	"github.com/yyl1212/math_master/backend/internal/feedback"
	"github.com/yyl1212/math_master/backend/internal/question"
	"time"
)

type feedbackCursor struct {
	Actor     string    `json:"actor"`
	Scope     string    `json:"scope"`
	Resource  string    `json:"resource"`
	Status    string    `json:"status"`
	Category  string    `json:"category"`
	CreatedAt time.Time `json:"createdAt"`
	ID        string    `json:"id"`
	Sequence  int64     `json:"sequence"`
}

func feedbackPageQuery(q feedback.ListQuery, actor, scope, resource string) (int, feedbackCursor, error) {
	limit := q.Limit
	if limit == 0 {
		limit = 20
	}
	c := feedbackCursor{Actor: actor, Scope: scope, Resource: resource}
	if limit < 1 || limit > 50 || q.Status != nil && !feedback.ValidStatus(*q.Status) || q.Category != nil && !feedback.ValidCategory(*q.Category) || (resource != "" && (q.Status != nil || q.Category != nil)) {
		return 0, c, auth.ErrInvalidInput
	}
	if q.Status != nil {
		c.Status = string(*q.Status)
	}
	if q.Category != nil {
		c.Category = string(*q.Category)
	}
	if q.Cursor == "" {
		return limit, c, nil
	}
	if len(q.Cursor) > 1024 {
		return 0, c, auth.ErrInvalidInput
	}
	raw, e := base64.RawURLEncoding.DecodeString(q.Cursor)
	if e != nil {
		return 0, c, auth.ErrInvalidInput
	}
	var previous feedbackCursor
	d := json.NewDecoder(bytes.NewReader(raw))
	d.DisallowUnknownFields()
	if d.Decode(&previous) != nil || previous.Actor != c.Actor || previous.Scope != c.Scope || previous.Resource != c.Resource || previous.Status != c.Status || previous.Category != c.Category {
		return 0, c, auth.ErrInvalidInput
	}
	canonical, _ := json.Marshal(previous)
	if !bytes.Equal(raw, canonical) {
		return 0, c, auth.ErrInvalidInput
	}
	if resource == "" {
		if !question.ValidID(previous.ID) || previous.CreatedAt.IsZero() || previous.Sequence != 0 {
			return 0, c, auth.ErrInvalidInput
		}
	} else if previous.Sequence < 1 || previous.Sequence > feedback.MaxSequence || previous.ID != "" || !previous.CreatedAt.IsZero() {
		return 0, c, auth.ErrInvalidInput
	}
	return limit, previous, nil
}
func feedbackNextCursor(c feedbackCursor) *string {
	raw, _ := json.Marshal(c)
	s := base64.RawURLEncoding.EncodeToString(raw)
	return &s
}
func feedbackReadAction(review bool, own, queue feedback.Action) feedback.Action {
	if review {
		return queue
	}
	return own
}
func feedbackReadScope(review bool) string {
	if review {
		return "review"
	}
	return "own"
}
func (s *Store) ReadFeedbackTicket(ctx context.Context, a question.Access, id string, review bool) (feedback.Envelope[feedback.Metadata], error) {
	var out feedback.Envelope[feedback.Metadata]
	action := feedbackReadAction(review, feedback.ReadOwnAction, feedback.ReadReviewAction)
	e := s.feedbackTx(ctx, a, action, nil, func(ctx context.Context, tx *sql.Tx, u auth.User, _ time.Time) error {
		t, e := feedbackLoadTicket(ctx, tx, id, false)
		if e != nil {
			return e
		}
		if !review && t.Owner != u.ID {
			return auth.ErrNotFound
		}
		m, e := feedbackProjectMetadata(ctx, tx, u, t)
		if e != nil {
			return e
		}
		out = feedback.Envelope[feedback.Metadata]{ActorID: u.ID, Data: m}
		return nil
	})
	if e != nil {
		return feedback.Envelope[feedback.Metadata]{}, e
	}
	return out, nil
}
func (s *Store) ListFeedbackTickets(ctx context.Context, a question.Access, review bool, q feedback.ListQuery) (feedback.Envelope[feedback.Page[feedback.Metadata]], error) {
	var out feedback.Envelope[feedback.Page[feedback.Metadata]]
	action := feedbackReadAction(review, feedback.ListOwnAction, feedback.ListReviewAction)
	e := s.feedbackTx(ctx, a, action, nil, func(ctx context.Context, tx *sql.Tx, u auth.User, _ time.Time) error {
		limit, c, e := feedbackPageQuery(q, u.ID, feedbackReadScope(review), "")
		if e != nil {
			return e
		}
		var cutTime, cutID any
		if c.ID != "" {
			cutTime = c.CreatedAt
			cutID = c.ID
		}
		rows, e := tx.QueryContext(ctx, `SELECT id::text FROM feedback_tickets WHERE ($1::boolean OR owner_user_id=$2) AND ($3='' OR status=$3) AND ($4='' OR category=$4) AND ($5::timestamptz IS NULL OR (created_at,id)<($5::timestamptz,$6::uuid)) ORDER BY created_at DESC,id DESC LIMIT $7`, review, u.ID, c.Status, c.Category, cutTime, cutID, limit+1)
		if e != nil {
			return e
		}
		ids := []string{}
		for rows.Next() {
			var id string
			if e = rows.Scan(&id); e != nil {
				rows.Close()
				return e
			}
			ids = append(ids, id)
		}
		e = rows.Err()
		rows.Close()
		if e != nil {
			return e
		}
		page := feedback.Page[feedback.Metadata]{Items: []feedback.Metadata{}}
		more := len(ids) > limit
		if more {
			ids = ids[:limit]
		}
		for _, id := range ids {
			t, e := feedbackLoadTicket(ctx, tx, id, false)
			if e != nil {
				return e
			}
			m, e := feedbackProjectMetadata(ctx, tx, u, t)
			if e != nil {
				return e
			}
			page.Items = append(page.Items, m)
		}
		if more {
			last := page.Items[len(page.Items)-1]
			c.ID = last.ID
			c.CreatedAt = last.CreatedAt
			c.Sequence = 0
			page.NextCursor = feedbackNextCursor(c)
		}
		out = feedback.Envelope[feedback.Page[feedback.Metadata]]{ActorID: u.ID, Data: page}
		return nil
	})
	if e != nil {
		return feedback.Envelope[feedback.Page[feedback.Metadata]]{}, e
	}
	return out, nil
}
func (s *Store) ReadFeedbackEvents(ctx context.Context, a question.Access, id string, review bool, q feedback.ListQuery) (feedback.Envelope[feedback.DiscussionPage], error) {
	var out feedback.Envelope[feedback.DiscussionPage]
	action := feedbackReadAction(review, feedback.DiscussOwnAction, feedback.DiscussReviewAction)
	e := s.feedbackTx(ctx, a, action, nil, func(ctx context.Context, tx *sql.Tx, u auth.User, _ time.Time) error {
		t, e := feedbackLoadTicket(ctx, tx, id, true)
		if e != nil {
			return e
		}
		if !review && t.Owner != u.ID {
			return auth.ErrNotFound
		}
		limit, c, e := feedbackPageQuery(q, u.ID, feedbackReadScope(review), id)
		if e != nil {
			return e
		}
		b, e := feedbackResolveTarget(ctx, tx, t.Owner, t.Metadata.Target, t.Source, false)
		if e != nil {
			return e
		}
		if e = feedbackDiscussionExposure(ctx, tx, u.ID, b); e != nil {
			return e
		}
		page := feedback.DiscussionPage{Items: []feedback.EventView{}}
		if e = tx.QueryRowContext(ctx, `SELECT original_title,original_location FROM feedback_tickets WHERE id=$1`, id).Scan(&page.Title, &page.Location); e != nil {
			return e
		}
		rows, e := tx.QueryContext(ctx, `SELECT sequence,kind,CASE WHEN actor_user_id=$2 THEN 'submitter' ELSE 'review_team' END,from_status,to_status,message,resolution,recorded_at FROM feedback_events WHERE ticket_id=$1 AND sequence>$3 ORDER BY sequence LIMIT $4`, id, t.Owner, c.Sequence, limit+1)
		if e != nil {
			return e
		}
		for rows.Next() {
			var item feedback.EventView
			var from sql.NullString
			var res []byte
			if e = rows.Scan(&item.Sequence, &item.Kind, &item.Actor, &from, &item.To, &item.Message, &res, &item.RecordedAt); e != nil {
				rows.Close()
				return e
			}
			if from.Valid {
				s := feedback.Status(from.String)
				item.From = &s
			}
			item.RecordedAt = item.RecordedAt.UTC()
			if len(res) > 0 {
				var r feedback.Resolution
				if json.Unmarshal(res, &r) != nil {
					rows.Close()
					return auth.ErrUnavailable
				}
				if u.ID == t.Owner {
					r.DuplicateOf = nil
				}
				item.Resolution = &r
			}
			page.Items = append(page.Items, item)
		}
		e = rows.Err()
		rows.Close()
		if e != nil {
			return e
		}
		if len(page.Items) > limit {
			page.Items = page.Items[:limit]
			c.Sequence = page.Items[limit-1].Sequence
			page.NextCursor = feedbackNextCursor(c)
		}
		out = feedback.Envelope[feedback.DiscussionPage]{ActorID: u.ID, Data: page}
		return nil
	})
	if e != nil {
		return feedback.Envelope[feedback.DiscussionPage]{}, e
	}
	return out, nil
}
