package publication

import (
	"context"
	"github.com/yyl1212/math_master/backend/internal/auth"
	"sync"
)

type Service struct {
	repo  Repository
	slots chan struct{}
}

func NewService(repo Repository) *Service { return &Service{repo: repo, slots: make(chan struct{}, 2)} }
func (s *Service) Preflight(ctx context.Context, a Access, action Action) (auth.User, error) {
	if s == nil || s.repo == nil {
		return auth.User{}, ErrContentNotConfigured
	}
	u, err := s.repo.Preflight(ctx, a, action)
	if err != nil {
		return auth.User{}, err
	}
	if err = Authorize(u, action); err != nil {
		return auth.User{}, err
	}
	rates, err := Rates(u.ID, action)
	if err != nil {
		return auth.User{}, err
	}
	if err = s.repo.ConsumeRates(ctx, rates); err != nil {
		return auth.User{}, err
	}
	return u, nil
}
func (s *Service) AcquireValidation(ctx context.Context) (func(), error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	select {
	case s.slots <- struct{}{}:
		var once sync.Once
		return func() { once.Do(func() { <-s.slots }) }, nil
	default:
		return nil, auth.ErrUnavailable
	}
}
func (s *Service) ListDrafts(ctx context.Context, a Access, q ListQuery) (Page[DraftSummary], error) {
	return s.repo.ListDrafts(ctx, a, q)
}
func (s *Service) CreateDraft(ctx context.Context, a Access, v DraftInput) (DraftView, error) {
	return s.repo.CreateDraft(ctx, a, v)
}
func (s *Service) ReadDraft(ctx context.Context, a Access, id string) (DraftView, error) {
	return s.repo.ReadDraft(ctx, a, id)
}
func (s *Service) SaveDraft(ctx context.Context, a Access, id string, v SaveDraftInput) (DraftView, error) {
	return s.repo.SaveDraft(ctx, a, id, v)
}
func (s *Service) AdoptDraft(ctx context.Context, a Access, v AdoptInput) (DraftView, error) {
	return s.repo.AdoptDraft(ctx, a, v)
}
func (s *Service) ValidateDraft(ctx context.Context, a Access, id string, v ValidateInput) (GateReport, error) {
	return s.repo.ValidateDraft(ctx, a, id, v)
}
func (s *Service) SubmitDraft(ctx context.Context, a Access, id string, v SubmitInput) (SubmissionView, error) {
	return s.repo.SubmitDraft(ctx, a, id, v)
}
func (s *Service) ListSubmissions(ctx context.Context, a Access, q ListQuery) (Page[SubmissionSummary], error) {
	return s.repo.ListSubmissions(ctx, a, q)
}
func (s *Service) ReadSubmission(ctx context.Context, a Access, id string) (SubmissionView, error) {
	return s.repo.ReadSubmission(ctx, a, id)
}
func (s *Service) ReviseSubmission(ctx context.Context, a Access, id string) (DraftView, error) {
	return s.repo.ReviseSubmission(ctx, a, id)
}
func (s *Service) DecideReview(ctx context.Context, a Access, id string, v ReviewInput) (SubmissionView, error) {
	return s.repo.DecideReview(ctx, a, id, v)
}
func (s *Service) ListPublications(ctx context.Context, a Access, q ListQuery) (PublicationPage, error) {
	return s.repo.ListPublications(ctx, a, q)
}
func (s *Service) ReadPublication(ctx context.Context, a Access, id string) (PublicationView, error) {
	return s.repo.ReadPublication(ctx, a, id)
}
func (s *Service) PrepareRelease(ctx context.Context, a Access, v PrepareInput) (PublicationView, error) {
	return s.repo.PrepareRelease(ctx, a, v)
}
func (s *Service) ActivateRelease(ctx context.Context, a Access, id string, v ActivateInput) (PublicationView, error) {
	return s.repo.ActivateRelease(ctx, a, id, v)
}
func (s *Service) PreviewWithdrawal(ctx context.Context, a Access, v WithdrawalPreviewInput) (WithdrawalPreview, error) {
	return s.repo.PreviewWithdrawal(ctx, a, v)
}
func (s *Service) WithdrawVersion(ctx context.Context, a Access, v WithdrawalInput) (WithdrawalResult, error) {
	return s.repo.WithdrawVersion(ctx, a, v)
}
func (s *Service) ReadDraftAsset(ctx context.Context, a Access, id string, sha string) ([]byte, error) {
	return s.repo.ReadDraftAsset(ctx, a, id, sha)
}
func (s *Service) ReadSubmissionAsset(ctx context.Context, a Access, id string, sha string) ([]byte, error) {
	return s.repo.ReadSubmissionAsset(ctx, a, id, sha)
}
