package question

import (
	"context"
	"errors"
	"github.com/yyl1212/math_master/backend/internal/auth"
	"github.com/yyl1212/math_master/backend/internal/publication"
	"testing"
)

type recordingRepository struct {
	user           auth.User
	rates          []auth.RateKey
	preflightError error
}

func (r *recordingRepository) QuestionPreflight(arg0 context.Context, arg1 Access, arg2 Action) (auth.User, error) {
	return r.user, r.preflightError
}
func (r *recordingRepository) ListQuestionDrafts(arg0 context.Context, arg1 Access, arg2 ListQuery) (Page[DraftSummary], error) {
	var v0 Page[DraftSummary]
	return v0, nil
}
func (r *recordingRepository) CreateQuestionDraft(arg0 context.Context, arg1 Access, arg2 DraftInput) (DraftView, error) {
	var v0 DraftView
	return v0, nil
}
func (r *recordingRepository) ReadQuestionDraft(arg0 context.Context, arg1 Access, arg2 string) (DraftView, error) {
	var v0 DraftView
	return v0, nil
}
func (r *recordingRepository) SaveQuestionDraft(arg0 context.Context, arg1 Access, arg2 string, arg3 SaveDraftInput) (DraftView, error) {
	var v0 DraftView
	return v0, nil
}
func (r *recordingRepository) AdoptQuestionDraft(arg0 context.Context, arg1 Access, arg2 AdoptInput) (DraftView, error) {
	var v0 DraftView
	return v0, nil
}
func (r *recordingRepository) ValidateQuestionDraft(arg0 context.Context, arg1 Access, arg2 string, arg3 ValidateInput) (ValidationReport, error) {
	var v0 ValidationReport
	return v0, nil
}
func (r *recordingRepository) SubmitQuestionDraft(arg0 context.Context, arg1 Access, arg2 string, arg3 SubmitInput) (SubmissionView, error) {
	var v0 SubmissionView
	return v0, nil
}
func (r *recordingRepository) ListQuestionSubmissions(arg0 context.Context, arg1 Access, arg2 ListQuery) (Page[SubmissionSummary], error) {
	var v0 Page[SubmissionSummary]
	return v0, nil
}
func (r *recordingRepository) ReadQuestionSubmission(arg0 context.Context, arg1 Access, arg2 string) (SubmissionView, error) {
	var v0 SubmissionView
	return v0, nil
}
func (r *recordingRepository) ListQuestionInstances(arg0 context.Context, arg1 Access, arg2 string, arg3 ListQuery) (Page[Instance], error) {
	var v0 Page[Instance]
	return v0, nil
}
func (r *recordingRepository) ReviseQuestionSubmission(arg0 context.Context, arg1 Access, arg2 string) (DraftView, error) {
	var v0 DraftView
	return v0, nil
}
func (r *recordingRepository) DecideQuestionReview(arg0 context.Context, arg1 Access, arg2 string, arg3 ReviewInput) (SubmissionView, error) {
	var v0 SubmissionView
	return v0, nil
}
func (r *recordingRepository) ListQuestionPublications(arg0 context.Context, arg1 Access, arg2 ListQuery) (PublicationPage, error) {
	var v0 PublicationPage
	return v0, nil
}
func (r *recordingRepository) ReadQuestionPublication(arg0 context.Context, arg1 Access, arg2 string) (PublicationSummary, error) {
	var v0 PublicationSummary
	return v0, nil
}
func (r *recordingRepository) ListQuestionMembers(arg0 context.Context, arg1 Access, arg2 string, arg3 ListQuery) (MemberPage, error) {
	var v0 MemberPage
	return v0, nil
}
func (r *recordingRepository) ListQuestionChanges(arg0 context.Context, arg1 Access, arg2 string, arg3 ListQuery) (ChangePage, error) {
	var v0 ChangePage
	return v0, nil
}
func (r *recordingRepository) PrepareQuestionRelease(arg0 context.Context, arg1 Access, arg2 PrepareInput) (PublicationSummary, error) {
	var v0 PublicationSummary
	return v0, nil
}
func (r *recordingRepository) ActivateQuestionRelease(arg0 context.Context, arg1 Access, arg2 string, arg3 ActivateInput) (PublicationSummary, error) {
	var v0 PublicationSummary
	return v0, nil
}
func (r *recordingRepository) PreviewQuestionWithdrawal(arg0 context.Context, arg1 Access, arg2 WithdrawalPreviewInput, arg3 ListQuery) (WithdrawalPreview, error) {
	var v0 WithdrawalPreview
	return v0, nil
}
func (r *recordingRepository) WithdrawQuestionVersion(arg0 context.Context, arg1 Access, arg2 WithdrawalInput) (WithdrawalResult, error) {
	var v0 WithdrawalResult
	return v0, nil
}
func (r *recordingRepository) ReadQuestionCoverage(arg0 context.Context, arg1 Access, arg2 CoverageQuery) (CoverageReport, error) {
	var v0 CoverageReport
	return v0, nil
}
func (r *recordingRepository) ReadSession(arg0 context.Context, arg1 auth.Digest, arg2 bool) (auth.SessionRecord, error) {
	var v0 auth.SessionRecord
	return v0, nil
}
func (r *recordingRepository) ConsumeRates(arg0 context.Context, arg1 []auth.RateKey) error {
	r.rates = arg1
	return nil
}

func TestQuestionServiceInjection(t *testing.T) {
	repo := &recordingRepository{user: auth.User{ID: actorID, Roles: []auth.Role{auth.RoleEditor}}}
	shared := publication.NewService(nil)
	if _, e := NewService(nil, shared.AcquireValidation); e == nil {
		t.Fatal("nil repository")
	}
	if _, e := NewService(repo, nil); e == nil {
		t.Fatal("nil acquire")
	}
	var typedNil *recordingRepository
	if _, e := NewService(typedNil, shared.AcquireValidation); e == nil {
		t.Fatal("typed nil repo")
	}
	service, e := NewService(repo, shared.AcquireValidation)
	if e != nil {
		t.Fatal(e)
	}
	release1, e := shared.AcquireValidation(context.Background())
	if e != nil {
		t.Fatal(e)
	}
	release2, e := service.AcquireValidation(context.Background())
	if e != nil {
		t.Fatal(e)
	}
	if _, e = service.AcquireValidation(context.Background()); !errors.Is(e, auth.ErrUnavailable) {
		t.Fatal("extra question slots", e)
	}
	release1()
	release1()
	release2()
	release3, e := service.AcquireValidation(context.Background())
	if e != nil {
		t.Fatal("slot leak")
	}
	release3()
	if _, e = service.Preflight(context.Background(), Access{}, ValidateDraftAction); e != nil || len(repo.rates) != 2 || repo.rates[1].Scope != "content_heavy_user" {
		t.Fatal("preflight did not apply shared rates", e)
	}
	repo.user.Roles = []auth.Role{auth.RoleLearner}
	repo.rates = nil
	if _, e = service.Preflight(context.Background(), Access{}, ValidateDraftAction); !errors.Is(e, auth.ErrForbidden) || repo.rates != nil {
		t.Fatal("rate consumed before auth")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, e = service.AcquireValidation(ctx); e == nil {
		t.Fatal("cancelled")
	}
}
