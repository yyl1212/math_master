package store

import (
	"context"
	"database/sql"
	"errors"
	"github.com/yyl1212/math_master/backend/internal/auth"
	"github.com/yyl1212/math_master/backend/internal/feedback"
	"github.com/yyl1212/math_master/backend/internal/question"
	"time"
)

func feedbackResolutionProof(ctx context.Context, tx *sql.Tx, ticketID string, in feedback.TransitionInput) error {
	if in.Resolution == nil {
		return nil
	}
	var valid bool
	e := tx.QueryRowContext(ctx, `SELECT feedback_resolution_proof(t,$2::jsonb,$3) FROM feedback_tickets t WHERE id=$1`, ticketID, body(in.Resolution), in.Status).Scan(&valid)
	if e != nil {
		return e
	}
	if !valid {
		return auth.ErrInvalidInput
	}
	return nil
}
func (s *Store) TransitionFeedback(ctx context.Context, a question.Access, id string, in feedback.TransitionInput) (feedback.Envelope[feedback.Receipt], error) {
	ctx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()
	var out feedback.Envelope[feedback.Receipt]
	// Authenticate before discovering the immutable related account. The second
	// transaction locks both accounts in UUID order and rechecks the same proof.
	if _, e := s.FeedbackPreflight(ctx, a, feedback.TransitionAction); e != nil {
		return out, e
	}
	if !question.ValidID(id) {
		return out, auth.ErrInvalidInput
	}
	var owner string
	e := s.db.QueryRowContext(ctx, `SELECT owner_user_id::text FROM feedback_tickets WHERE id=$1`, id).Scan(&owner)
	if errors.Is(e, sql.ErrNoRows) {
		return out, auth.ErrNotFound
	}
	if e != nil {
		return out, feedbackError(e)
	}
	e = s.feedbackTx(ctx, a, feedback.TransitionAction, []string{owner}, func(ctx context.Context, tx *sql.Tx, u auth.User, _ time.Time) error {
		t, e := feedbackLoadTicket(ctx, tx, id, true)
		if e != nil {
			return e
		}
		if !feedback.CanHandle(u, t.Owner) {
			return auth.ErrForbidden
		}
		_, digest, e := feedback.CanonicalCommand(feedback.TransitionAction, id, in)
		if e != nil {
			return e
		}
		r, found, e := feedbackReplay(ctx, tx, u.ID, "transition", id, a.IdempotencyKey, digest)
		if e != nil {
			return e
		}
		if found {
			out = feedback.Envelope[feedback.Receipt]{ActorID: u.ID, Data: r}
			return nil
		}
		if t.Metadata.Sequence != in.ExpectedSequence {
			return feedback.ErrConflict
		}
		if e = feedback.ValidateTransition(t.Metadata.Status, in); e != nil {
			return e
		}
		if e = feedbackResolutionProof(ctx, tx, id, in); e != nil {
			return e
		}
		effective := in.Resolution
		if t.Metadata.Status == in.Status {
			effective = t.Resolution
		}
		now, e := dbClock(ctx, tx)
		if e != nil {
			return e
		}
		if e = feedbackConsumeRates(ctx, tx, u.ID, feedback.TransitionAction, now); e != nil {
			return e
		}
		if e = feedbackAppend(ctx, tx, u, t, in.Status, in.Message, in.Resolution, effective, a.RequestID, now); e != nil {
			return e
		}
		t, e = feedbackLoadTicket(ctx, tx, id, false)
		if e != nil {
			return e
		}
		m, e := feedbackProjectMetadata(ctx, tx, u, t)
		if e != nil {
			return e
		}
		r = feedback.Receipt{Status: 200, Ticket: m}
		if e = feedbackRemember(ctx, tx, u.ID, "transition", id, a.IdempotencyKey, digest, r); e != nil {
			return e
		}
		out = feedback.Envelope[feedback.Receipt]{ActorID: u.ID, Data: r}
		return nil
	})
	if e != nil {
		return feedback.Envelope[feedback.Receipt]{}, e
	}
	return out, nil
}
