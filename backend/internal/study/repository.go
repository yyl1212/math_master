package study

import (
	"context"
	"github.com/yyl1212/math_master/backend/internal/auth"
)

type Repository interface {
	StudyPreflight(context.Context, Access, Action) (auth.User, error)
	ConsumeRates(context.Context, []auth.RateKey) error
	BeginStudy(context.Context, Access, string, CommandInput) (StudyDetail, error)
	CompleteStudy(context.Context, Access, string, CommandInput) (StudyDetail, error)
	StartStudyReview(context.Context, Access, string, CommandInput) (StudyDetail, error)
	FinishStudyReview(context.Context, Access, string, ReviewInput) (StudyDetail, error)
	ReadStudyOverview(context.Context, Access) (Overview, error)
	ListStudyTopics(context.Context, Access, ListQuery) (Page[TopicProgress], error)
	ListStudyKnowledge(context.Context, Access, ListQuery) (Page[StudyDetail], error)
	ReadStudyKnowledge(context.Context, Access, string) (StudyDetail, error)
	ListStudyHistory(context.Context, Access, HistoryQuery) (HistoryPage, error)
	ReadStudyNote(context.Context, Access, string) (NoteView, error)
	SaveStudyNote(context.Context, Access, string, NoteInput) (NoteReceipt, error)
	DeleteStudyNote(context.Context, Access, string, NoteDeleteInput) (NoteReceipt, error)
}
