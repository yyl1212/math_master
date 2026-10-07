package store

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"github.com/yyl1212/math_master/backend/internal/auth"
	"github.com/yyl1212/math_master/backend/internal/feedback"
	"github.com/yyl1212/math_master/backend/internal/question"
	"time"
)

type feedbackCursor struct {
	Resource  string
	CreatedAt time.Time
	ID        string
	Sequence  int64
}

func feedbackPageQuery(q feedback.ListQuery, actor, scope, resource string) (int, feedbackCursor, error) {
	limit := q.Limit
	if limit == 0 {
		limit = 20
	}
	c := feedbackCursor{Resource: resource}
	if limit < 1 || limit > 50 || q.Status != nil && !feedback.ValidStatus(*q.Status) || q.Category != nil && !feedback.ValidCategory(*q.Category) || (resource != "" && (q.Status != nil || q.Category != nil)) {
		return 0, c, auth.ErrInvalidInput
	}
	if q.Cursor == "" {
		return limit, c, nil
	}
	if len(q.Cursor) > 512 {
		return 0, c, auth.ErrInvalidInput
	}
	raw, e := base64.RawURLEncoding.DecodeString(q.Cursor)
	if e != nil {
		return 0, c, auth.ErrInvalidInput
	}
	var canonical []byte
	if resource == "" {
		var v struct {
			Version   int       `json:"version"`
			CreatedAt time.Time `json:"createdAt"`
			ID        string    `json:"id"`
		}
		d := json.NewDecoder(bytes.NewReader(raw))
		d.DisallowUnknownFields()
		if d.Decode(&v) != nil || v.Version != 1 || !question.ValidID(v.ID) || v.CreatedAt.IsZero() {
			return 0, c, auth.ErrInvalidInput
		}
		canonical, _ = json.Marshal(v)
		c.CreatedAt = v.CreatedAt
		c.ID = v.ID
	} else {
		var v struct {
			Version  int   `json:"version"`
			Sequence int64 `json:"sequence"`
		}
		d := json.NewDecoder(bytes.NewReader(raw))
		d.DisallowUnknownFields()
		if d.Decode(&v) != nil || v.Version != 1 || v.Sequence < 1 || v.Sequence > feedback.MaxSequence {
			return 0, c, auth.ErrInvalidInput
		}
		canonical, _ = json.Marshal(v)
		c.Sequence = v.Sequence
	}
	if !bytes.Equal(raw, canonical) {
		return 0, c, auth.ErrInvalidInput
	}
	return limit, c, nil
}
func feedbackNextCursor(c feedbackCursor) *string {
	var raw []byte
	if c.Resource == "" {
		raw, _ = json.Marshal(struct {
			Version   int       `json:"version"`
			CreatedAt time.Time `json:"createdAt"`
			ID        string    `json:"id"`
		}{1, c.CreatedAt, c.ID})
	} else {
		raw, _ = json.Marshal(struct {
			Version  int   `json:"version"`
			Sequence int64 `json:"sequence"`
		}{1, c.Sequence})
	}
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
		if !review && (q.Status != nil || q.Category != nil) {
			return auth.ErrInvalidInput
		}
		limit, c, e := feedbackPageQuery(q, u.ID, feedbackReadScope(review), "")
		if e != nil {
			return e
		}
		var cutTime, cutID any
		if c.ID != "" {
			cutTime = c.CreatedAt
			cutID = c.ID
		}
		// A missing legacy capability must not hide independent site/knowledge
		// reports. Filter only legacy sources before pagination; discussions and
		// direct legacy metadata retain their original fail-closed guards.
		legacyAvailable := true
		if e := feedbackSourceConfigured(ctx, tx, feedback.Binding{Source: feedback.Source{Kind: "practice"}}); e != nil {
			if !errors.Is(e, feedback.ErrNotConfigured) {
				return e
			}
			legacyAvailable = false
		}
		rows, e := tx.QueryContext(ctx, `SELECT id::text FROM feedback_tickets WHERE ($1::boolean OR owner_user_id=$2) AND ($3='' OR status=$3) AND ($4='' OR category=$4) AND ($5::timestamptz IS NULL OR (created_at,id)<($5::timestamptz,$6::uuid)) AND ($8::boolean OR source->>'kind' NOT IN ('practice','assessment')) ORDER BY created_at DESC,id DESC LIMIT $7`, review, u.ID, feedbackQueryStatus(q), feedbackQueryCategory(q), cutTime, cutID, limit+1, legacyAvailable)
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

func feedbackQueryStatus(q feedback.ListQuery) string {
	if q.Status == nil {
		return ""
	}
	return string(*q.Status)
}
func feedbackQueryCategory(q feedback.ListQuery) string {
	if q.Category == nil {
		return ""
	}
	return string(*q.Category)
}
