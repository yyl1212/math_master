package question

import (
	"context"
	"github.com/yyl1212/math_master/backend/internal/auth"
	"reflect"
)

type Service struct {
	repo    Repository
	acquire func(context.Context) (func(), error)
}

func NewService(repo Repository, acquire func(context.Context) (func(), error)) (*Service, error) {
	if repo == nil || acquire == nil {
		return nil, ErrNotConfigured
	}
	v := reflect.ValueOf(repo)
	if v.Kind() == reflect.Pointer && v.IsNil() {
		return nil, ErrNotConfigured
	}
	return &Service{repo: repo, acquire: acquire}, nil
}
func (s *Service) AcquireValidation(ctx context.Context) (func(), error) {
	if s == nil || s.acquire == nil {
		return nil, ErrNotConfigured
	}
	return s.acquire(ctx)
}
func (s *Service) Preflight(ctx context.Context, a Access, action Action) (auth.User, error) {
	if s == nil || s.repo == nil {
		return auth.User{}, ErrNotConfigured
	}
	u, e := s.repo.QuestionPreflight(ctx, a, action)
	if e != nil {
		return auth.User{}, e
	}
	if e = Authorize(u, action); e != nil {
		return auth.User{}, e
	}
	rates, e := Rates(u.ID, action)
	if e != nil {
		return auth.User{}, e
	}
	if e = s.repo.ConsumeRates(ctx, rates); e != nil {
		return auth.User{}, e
	}
	return u, nil
}
func (s *Service) ListDrafts(arg0 context.Context, arg1 Access, arg2 ListQuery) (Page[DraftSummary], error) {
	return s.repo.ListQuestionDrafts(arg0, arg1, arg2)
}
func (s *Service) CreateDraft(arg0 context.Context, arg1 Access, arg2 DraftInput) (DraftView, error) {
	return s.repo.CreateQuestionDraft(arg0, arg1, arg2)
}
func (s *Service) ReadDraft(arg0 context.Context, arg1 Access, arg2 string) (DraftView, error) {
	return s.repo.ReadQuestionDraft(arg0, arg1, arg2)
}
func (s *Service) SaveDraft(arg0 context.Context, arg1 Access, arg2 string, arg3 SaveDraftInput) (DraftView, error) {
	return s.repo.SaveQuestionDraft(arg0, arg1, arg2, arg3)
}
func (s *Service) AdoptDraft(arg0 context.Context, arg1 Access, arg2 AdoptInput) (DraftView, error) {
	return s.repo.AdoptQuestionDraft(arg0, arg1, arg2)
}
func (s *Service) ValidateDraft(arg0 context.Context, arg1 Access, arg2 string, arg3 ValidateInput) (ValidationReport, error) {
	return s.repo.ValidateQuestionDraft(arg0, arg1, arg2, arg3)
}
func (s *Service) SubmitDraft(arg0 context.Context, arg1 Access, arg2 string, arg3 SubmitInput) (SubmissionView, error) {
	return s.repo.SubmitQuestionDraft(arg0, arg1, arg2, arg3)
}
func (s *Service) ListSubmissions(arg0 context.Context, arg1 Access, arg2 ListQuery) (Page[SubmissionSummary], error) {
	return s.repo.ListQuestionSubmissions(arg0, arg1, arg2)
}
func (s *Service) ReadSubmission(arg0 context.Context, arg1 Access, arg2 string) (SubmissionView, error) {
	return s.repo.ReadQuestionSubmission(arg0, arg1, arg2)
}
func (s *Service) ListInstances(arg0 context.Context, arg1 Access, arg2 string, arg3 ListQuery) (Page[Instance], error) {
	return s.repo.ListQuestionInstances(arg0, arg1, arg2, arg3)
}
func (s *Service) ReviseSubmission(arg0 context.Context, arg1 Access, arg2 string) (DraftView, error) {
	return s.repo.ReviseQuestionSubmission(arg0, arg1, arg2)
}
func (s *Service) DecideReview(arg0 context.Context, arg1 Access, arg2 string, arg3 ReviewInput) (SubmissionView, error) {
	return s.repo.DecideQuestionReview(arg0, arg1, arg2, arg3)
}
func (s *Service) ListPublications(arg0 context.Context, arg1 Access, arg2 ListQuery) (PublicationPage, error) {
	return s.repo.ListQuestionPublications(arg0, arg1, arg2)
}
func (s *Service) ReadPublication(arg0 context.Context, arg1 Access, arg2 string) (PublicationSummary, error) {
	return s.repo.ReadQuestionPublication(arg0, arg1, arg2)
}
func (s *Service) ListMembers(arg0 context.Context, arg1 Access, arg2 string, arg3 ListQuery) (MemberPage, error) {
	return s.repo.ListQuestionMembers(arg0, arg1, arg2, arg3)
}
func (s *Service) ListChanges(arg0 context.Context, arg1 Access, arg2 string, arg3 ListQuery) (ChangePage, error) {
	return s.repo.ListQuestionChanges(arg0, arg1, arg2, arg3)
}
func (s *Service) PrepareRelease(arg0 context.Context, arg1 Access, arg2 PrepareInput) (PublicationSummary, error) {
	return s.repo.PrepareQuestionRelease(arg0, arg1, arg2)
}
func (s *Service) ActivateRelease(arg0 context.Context, arg1 Access, arg2 string, arg3 ActivateInput) (PublicationSummary, error) {
	return s.repo.ActivateQuestionRelease(arg0, arg1, arg2, arg3)
}
func (s *Service) PreviewWithdrawal(arg0 context.Context, arg1 Access, arg2 WithdrawalPreviewInput, arg3 ListQuery) (WithdrawalPreview, error) {
	return s.repo.PreviewQuestionWithdrawal(arg0, arg1, arg2, arg3)
}
func (s *Service) WithdrawVersion(arg0 context.Context, arg1 Access, arg2 WithdrawalInput) (WithdrawalResult, error) {
	return s.repo.WithdrawQuestionVersion(arg0, arg1, arg2)
}
func (s *Service) ReadCoverage(arg0 context.Context, arg1 Access, arg2 CoverageQuery) (CoverageReport, error) {
	return s.repo.ReadQuestionCoverage(arg0, arg1, arg2)
}
