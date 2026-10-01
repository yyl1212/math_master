package publication

import (
	"context"
	"github.com/yyl1212/math_master/backend/internal/auth"
)

type Repository interface {
	Preflight(context.Context, Access, Action) (auth.User, error)
	ReadSession(context.Context, auth.Digest, bool) (auth.SessionRecord, error)
	ConsumeRates(context.Context, []auth.RateKey) error
	ListDrafts(ctx context.Context, a Access, q ListQuery) (Page[DraftSummary], error)
	CreateDraft(ctx context.Context, a Access, v DraftInput) (DraftView, error)
	ReadDraft(ctx context.Context, a Access, id string) (DraftView, error)
	SaveDraft(ctx context.Context, a Access, id string, v SaveDraftInput) (DraftView, error)
	AdoptDraft(ctx context.Context, a Access, v AdoptInput) (DraftView, error)
	ValidateDraft(ctx context.Context, a Access, id string, v ValidateInput) (GateReport, error)
	SubmitDraft(ctx context.Context, a Access, id string, v SubmitInput) (SubmissionView, error)
	ListSubmissions(ctx context.Context, a Access, q ListQuery) (Page[SubmissionSummary], error)
	ReadSubmission(ctx context.Context, a Access, id string) (SubmissionView, error)
	ReviseSubmission(ctx context.Context, a Access, id string) (DraftView, error)
	DecideReview(ctx context.Context, a Access, id string, v ReviewInput) (SubmissionView, error)
	ListPublications(ctx context.Context, a Access, q ListQuery) (PublicationPage, error)
	ReadPublication(ctx context.Context, a Access, id string) (PublicationView, error)
	PrepareRelease(ctx context.Context, a Access, v PrepareInput) (PublicationView, error)
	ActivateRelease(ctx context.Context, a Access, id string, v ActivateInput) (PublicationView, error)
	PreviewWithdrawal(ctx context.Context, a Access, v WithdrawalPreviewInput) (WithdrawalPreview, error)
	WithdrawVersion(ctx context.Context, a Access, v WithdrawalInput) (WithdrawalResult, error)
	ReadDraftAsset(ctx context.Context, a Access, id string, sha string) ([]byte, error)
	ReadSubmissionAsset(ctx context.Context, a Access, id string, sha string) ([]byte, error)
}
