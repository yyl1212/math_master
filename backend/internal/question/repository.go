package question

import (
	"context"
	"github.com/yyl1212/math_master/backend/internal/auth"
)

type Repository interface {
	QuestionPreflight(context.Context, Access, Action) (auth.User, error)
	ListQuestionDrafts(context.Context, Access, ListQuery) (Page[DraftSummary], error)
	CreateQuestionDraft(context.Context, Access, DraftInput) (DraftView, error)
	ReadQuestionDraft(context.Context, Access, string) (DraftView, error)
	SaveQuestionDraft(context.Context, Access, string, SaveDraftInput) (DraftView, error)
	AdoptQuestionDraft(context.Context, Access, AdoptInput) (DraftView, error)
	ValidateQuestionDraft(context.Context, Access, string, ValidateInput) (ValidationReport, error)
	SubmitQuestionDraft(context.Context, Access, string, SubmitInput) (SubmissionView, error)
	ListQuestionSubmissions(context.Context, Access, ListQuery) (Page[SubmissionSummary], error)
	ReadQuestionSubmission(context.Context, Access, string) (SubmissionView, error)
	ListQuestionInstances(context.Context, Access, string, ListQuery) (Page[Instance], error)
	ReviseQuestionSubmission(context.Context, Access, string) (DraftView, error)
	DecideQuestionReview(context.Context, Access, string, ReviewInput) (SubmissionView, error)
	ListQuestionPublications(context.Context, Access, ListQuery) (PublicationPage, error)
	ReadQuestionPublication(context.Context, Access, string) (PublicationSummary, error)
	ListQuestionMembers(context.Context, Access, string, ListQuery) (MemberPage, error)
	ListQuestionChanges(context.Context, Access, string, ListQuery) (ChangePage, error)
	PrepareQuestionRelease(context.Context, Access, PrepareInput) (PublicationSummary, error)
	ActivateQuestionRelease(context.Context, Access, string, ActivateInput) (PublicationSummary, error)
	PreviewQuestionWithdrawal(context.Context, Access, WithdrawalPreviewInput, ListQuery) (WithdrawalPreview, error)
	WithdrawQuestionVersion(context.Context, Access, WithdrawalInput) (WithdrawalResult, error)
	ReadQuestionCoverage(context.Context, Access, CoverageQuery) (CoverageReport, error)
	ReadSession(context.Context, auth.Digest, bool) (auth.SessionRecord, error)
	ConsumeRates(context.Context, []auth.RateKey) error
}
