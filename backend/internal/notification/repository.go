package notification

import (
	"context"
	"github.com/yyl1212/math_master/backend/internal/auth"
	"github.com/yyl1212/math_master/backend/internal/correction"
	"github.com/yyl1212/math_master/backend/internal/question"
)

type Repository interface {
	CorrectionPreflight(ctx context.Context, a question.Access, action correction.Action) (auth.User, error)
	ListNotifications(ctx context.Context, a question.Access, q Query) (Envelope[Page[Metadata]], error)
	ReadNotificationCount(ctx context.Context, a question.Access) (Envelope[UnreadCount], error)
	ReadNotification(ctx context.Context, a question.Access, id string) (Envelope[Metadata], error)
	MarkNotificationRead(ctx context.Context, a question.Access, id string) (Envelope[ReadReceipt], error)
}
