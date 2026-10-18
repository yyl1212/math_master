package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"github.com/yyl1212/math_master/backend/internal/auth"
	"github.com/yyl1212/math_master/backend/internal/feedback"
	"github.com/yyl1212/math_master/backend/internal/pagination"
	"github.com/yyl1212/math_master/backend/internal/question"
	"time"
)

func (s *Store) ListFeedbackTicketPage(ctx context.Context, a question.Access, review bool, q feedback.ListQuery, p pagination.Request) (out feedback.Envelope[pagination.Page[feedback.Metadata]], e error) {
	q.Limit = p.Limit
	q.Cursor = p.Cursor
	e = s.feedbackTx(ctx, a, feedbackReadAction(review, feedback.ListOwnAction, feedback.ListReviewAction), nil, func(ctx context.Context, tx *sql.Tx, u auth.User, _ time.Time) error {
		if !review && (q.Status != nil || q.Category != nil) {
			return auth.ErrInvalidInput
		}
		_, c, e := feedbackPageQuery(q, u.ID, feedbackReadScope(review), "")
		if e != nil {
			return e
		}
		legacy := true
		if err := feedbackSourceConfigured(ctx, tx, feedback.Binding{Source: feedback.Source{Kind: "practice"}}); err != nil {
			if !errors.Is(err, feedback.ErrNotConfigured) {
				return err
			}
			legacy = false
		}
		args := []any{review, u.ID, feedbackQueryStatus(q), feedbackQueryCategory(q), legacy}
		anchor := ""
		if p.Cursor != "" {
			anchor = "(t.created_at,t.id)>=($6::timestamptz,$7::uuid)"
			args = append(args, c.CreatedAt, c.ID)
		}
		out.ActorID = u.ID
		out.Data.Items = []feedback.Metadata{}
		rows, e := queryNumberedRows(ctx, tx, p, &out.Data.Meta, `t.id::text,t.owner_user_id::text,t.target,t.source,t.category,t.status,t.sequence,t.created_at,t.updated_at,t.current_resolution`, `feedback_tickets t`, `($1::boolean OR t.owner_user_id=$2) AND ($3='' OR t.status=$3) AND ($4='' OR t.category=$4) AND ($5::boolean OR t.source->>'kind' NOT IN ('practice','assessment'))`, `t.created_at DESC,t.id DESC`, anchor, args...)
		if e != nil {
			return e
		}
		tickets := []feedbackTicket{}
		for rows.Next() {
			var ticket feedbackTicket
			var target, source, res []byte
			if e = rows.Scan(&ticket.Metadata.ID, &ticket.Owner, &target, &source, &ticket.Metadata.Category, &ticket.Metadata.Status, &ticket.Metadata.Sequence, &ticket.Metadata.CreatedAt, &ticket.Metadata.UpdatedAt, &res); e != nil {
				rows.Close()
				return e
			}
			if json.Unmarshal(target, &ticket.Metadata.Target) != nil || json.Unmarshal(source, &ticket.Source) != nil {
				rows.Close()
				return auth.ErrUnavailable
			}
			if len(res) > 0 {
				var resolution feedback.Resolution
				if json.Unmarshal(res, &resolution) != nil {
					rows.Close()
					return auth.ErrUnavailable
				}
				ticket.Resolution = &resolution
				ticket.Metadata.ResolutionKind = &resolution.Kind
			}
			tickets = append(tickets, ticket)
		}
		e = rows.Err()
		rows.Close()
		if e != nil {
			return e
		}
		for _, ticket := range tickets {
			m, e := feedbackProjectMetadata(ctx, tx, u, ticket)
			if e != nil {
				return e
			}
			out.Data.Items = append(out.Data.Items, m)
		}
		return correctionResponseSize(out)
	})
	return
}
