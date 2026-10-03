package feedback

import (
	"context"
	"github.com/yyl1212/math_master/backend/internal/auth"
	"github.com/yyl1212/math_master/backend/internal/question"
)

type Repository interface {
	FeedbackPreflight(context.Context, question.Access, Action) (auth.User, error)
	ReadFeedbackContext(context.Context, question.Access, ContextQuery) (Envelope[Context], error)
	CreateFeedback(context.Context, question.Access, CreateInput) (Envelope[Receipt], error)
	ReplyFeedback(context.Context, question.Access, string, ReplyInput) (Envelope[Receipt], error)
	TransitionFeedback(context.Context, question.Access, string, TransitionInput) (Envelope[Receipt], error)
	ListFeedbackTickets(context.Context, question.Access, bool, ListQuery) (Envelope[Page[Metadata]], error)
	ReadFeedbackTicket(context.Context, question.Access, string, bool) (Envelope[Metadata], error)
	ReadFeedbackEvents(context.Context, question.Access, string, bool, ListQuery) (Envelope[DiscussionPage], error)
}
