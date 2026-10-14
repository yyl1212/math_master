package studyreview

import "context"

type Repository interface {
	ReadReviewHealth(context.Context) (Health, error)
	ReadReviewOverview(context.Context, Access) (Overview, error)
	ListReviewQueue(context.Context, Access, QueueQuery) (QueuePage, error)
	StartReviewSession(context.Context, Access, StartInput) (CommandReceipt, error)
	ReadReviewSession(context.Context, Access, string) (Session, error)
	SaveReviewDraft(context.Context, Access, string, DraftInput) (CommandReceipt, error)
	CompareReviewSession(context.Context, Access, string, CompareInput) (CommandReceipt, error)
	GradeReviewSession(context.Context, Access, string, GradeInput) (CommandReceipt, error)
	AbandonReviewSession(context.Context, Access, string, AbandonInput) (CommandReceipt, error)
	ListReviewHistory(context.Context, Access, HistoryQuery) (HistoryPage, error)
}
