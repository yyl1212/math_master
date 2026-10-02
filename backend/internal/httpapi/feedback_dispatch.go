package httpapi

import (
	"context"
	"github.com/yyl1212/math_master/backend/internal/auth"
	"github.com/yyl1212/math_master/backend/internal/feedback"
	"github.com/yyl1212/math_master/backend/internal/question"
	"net/http"
)

func dispatchFeedback(ctx context.Context, r *http.Request, route feedbackRoute, a question.Access, q feedback.ListQuery, c feedback.ContextQuery, s *feedback.Service) (any, int, error) {
	switch route.Action {
	case feedback.ReadContextAction:
		v, e := s.ReadFeedbackContext(ctx, a, c)
		return v, 200, e
	case feedback.ListOwnAction, feedback.ListReviewAction:
		v, e := s.ListFeedbackTickets(ctx, a, route.Review, q)
		return v, 200, e
	case feedback.ReadOwnAction, feedback.ReadReviewAction:
		v, e := s.ReadFeedbackTicket(ctx, a, route.ID, route.Review)
		return v, 200, e
	case feedback.DiscussOwnAction, feedback.DiscussReviewAction:
		v, e := s.ReadFeedbackEvents(ctx, a, route.ID, route.Review, q)
		return v, 200, e
	case feedback.CreateAction, feedback.ReplyAction, feedback.TransitionAction:
		in, e := readFeedbackInput(r.Body, route.Action)
		if e != nil {
			return nil, 0, e
		}
		var v feedback.Envelope[feedback.Receipt]
		switch route.Action {
		case feedback.CreateAction:
			v, e = s.CreateFeedback(ctx, a, in.(feedback.CreateInput))
		case feedback.ReplyAction:
			v, e = s.ReplyFeedback(ctx, a, route.ID, in.(feedback.ReplyInput))
		case feedback.TransitionAction:
			v, e = s.TransitionFeedback(ctx, a, route.ID, in.(feedback.TransitionInput))
		}
		return v, v.Data.Status, e
	}
	return nil, 0, auth.ErrNotFound
}
