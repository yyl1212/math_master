package publication

import (
	"context"
	"errors"
	"github.com/yyl1212/math_master/backend/internal/auth"
	"testing"
)

type recordingRepository struct {
	t            *testing.T
	user         auth.User
	preflightErr error
	rates        []auth.RateKey
	called       Action
}

func (r *recordingRepository) Preflight(context.Context, Access, Action) (auth.User, error) {
	return r.user, r.preflightErr
}
func (r *recordingRepository) ReadSession(context.Context, auth.Digest, bool) (auth.SessionRecord, error) {
	r.t.Fatal("unexpected ReadSession")
	return auth.SessionRecord{}, nil
}
func (r *recordingRepository) ConsumeRates(_ context.Context, k []auth.RateKey) error {
	r.rates = k
	return nil
}
func (r *recordingRepository) ListDrafts(_ context.Context, _ Access, q ListQuery) (Page[DraftSummary], error) {
	var zero Page[DraftSummary]
	if r.called != ListDraftsAction {
		r.t.Fatalf("unexpected action %s", "ListDrafts")
	}
	return zero, nil
}
func (r *recordingRepository) CreateDraft(_ context.Context, _ Access, v DraftInput) (DraftView, error) {
	var zero DraftView
	if r.called != CreateDraftAction {
		r.t.Fatalf("unexpected action %s", "CreateDraft")
	}
	return zero, nil
}
func (r *recordingRepository) ReadDraft(_ context.Context, _ Access, id string) (DraftView, error) {
	var zero DraftView
	if r.called != ReadDraftAction {
		r.t.Fatalf("unexpected action %s", "ReadDraft")
	}
	return zero, nil
}
func (r *recordingRepository) SaveDraft(_ context.Context, _ Access, id string, v SaveDraftInput) (DraftView, error) {
	var zero DraftView
	if r.called != SaveDraftAction {
		r.t.Fatalf("unexpected action %s", "SaveDraft")
	}
	return zero, nil
}
func (r *recordingRepository) AdoptDraft(_ context.Context, _ Access, v AdoptInput) (DraftView, error) {
	var zero DraftView
	if r.called != AdoptDraftAction {
		r.t.Fatalf("unexpected action %s", "AdoptDraft")
	}
	return zero, nil
}
func (r *recordingRepository) ValidateDraft(_ context.Context, _ Access, id string, v ValidateInput) (GateReport, error) {
	var zero GateReport
	if r.called != ValidateDraftAction {
		r.t.Fatalf("unexpected action %s", "ValidateDraft")
	}
	return zero, nil
}
func (r *recordingRepository) SubmitDraft(_ context.Context, _ Access, id string, v SubmitInput) (SubmissionView, error) {
	var zero SubmissionView
	if r.called != SubmitDraftAction {
		r.t.Fatalf("unexpected action %s", "SubmitDraft")
	}
	return zero, nil
}
func (r *recordingRepository) ListSubmissions(_ context.Context, _ Access, q ListQuery) (Page[SubmissionSummary], error) {
	var zero Page[SubmissionSummary]
	if r.called != ListSubmissionsAction {
		r.t.Fatalf("unexpected action %s", "ListSubmissions")
	}
	return zero, nil
}
func (r *recordingRepository) ReadSubmission(_ context.Context, _ Access, id string) (SubmissionView, error) {
	var zero SubmissionView
	if r.called != ReadSubmissionAction {
		r.t.Fatalf("unexpected action %s", "ReadSubmission")
	}
	return zero, nil
}
func (r *recordingRepository) ReviseSubmission(_ context.Context, _ Access, id string) (DraftView, error) {
	var zero DraftView
	if r.called != ReviseSubmissionAction {
		r.t.Fatalf("unexpected action %s", "ReviseSubmission")
	}
	return zero, nil
}
func (r *recordingRepository) DecideReview(_ context.Context, _ Access, id string, v ReviewInput) (SubmissionView, error) {
	var zero SubmissionView
	if r.called != DecideReviewAction {
		r.t.Fatalf("unexpected action %s", "DecideReview")
	}
	return zero, nil
}
func (r *recordingRepository) ListPublications(_ context.Context, _ Access, q ListQuery) (PublicationPage, error) {
	var zero PublicationPage
	if r.called != ListPublicationsAction {
		r.t.Fatalf("unexpected action %s", "ListPublications")
	}
	return zero, nil
}
func (r *recordingRepository) ReadPublication(_ context.Context, _ Access, id string) (PublicationView, error) {
	var zero PublicationView
	if r.called != ReadPublicationAction {
		r.t.Fatalf("unexpected action %s", "ReadPublication")
	}
	return zero, nil
}
func (r *recordingRepository) PrepareRelease(_ context.Context, _ Access, v PrepareInput) (PublicationView, error) {
	var zero PublicationView
	if r.called != PrepareReleaseAction {
		r.t.Fatalf("unexpected action %s", "PrepareRelease")
	}
	return zero, nil
}
func (r *recordingRepository) ActivateRelease(_ context.Context, _ Access, id string, v ActivateInput) (PublicationView, error) {
	var zero PublicationView
	if r.called != ActivateReleaseAction {
		r.t.Fatalf("unexpected action %s", "ActivateRelease")
	}
	return zero, nil
}
func (r *recordingRepository) PreviewWithdrawal(_ context.Context, _ Access, v WithdrawalPreviewInput) (WithdrawalPreview, error) {
	var zero WithdrawalPreview
	if r.called != PreviewWithdrawalAction {
		r.t.Fatalf("unexpected action %s", "PreviewWithdrawal")
	}
	return zero, nil
}
func (r *recordingRepository) WithdrawVersion(_ context.Context, _ Access, v WithdrawalInput) (WithdrawalResult, error) {
	var zero WithdrawalResult
	if r.called != WithdrawVersionAction {
		r.t.Fatalf("unexpected action %s", "WithdrawVersion")
	}
	return zero, nil
}
func (r *recordingRepository) ReadDraftAsset(_ context.Context, _ Access, id string, sha string) ([]byte, error) {
	var zero []byte
	if r.called != ReadDraftAssetAction {
		r.t.Fatalf("unexpected action %s", "ReadDraftAsset")
	}
	return zero, nil
}
func (r *recordingRepository) ReadSubmissionAsset(_ context.Context, _ Access, id string, sha string) ([]byte, error) {
	var zero []byte
	if r.called != ReadSubmissionAssetAction {
		r.t.Fatalf("unexpected action %s", "ReadSubmissionAsset")
	}
	return zero, nil
}
func TestContentServicePreflightAndSlots(t *testing.T) {
	r := &recordingRepository{t: t, user: auth.User{ID: "11111111-1111-4111-8111-111111111111", Roles: []auth.Role{auth.RoleLearner, auth.RoleEditor}}}
	s := NewService(r)
	if _, err := s.Preflight(context.Background(), Access{}, CreateDraftAction); err != nil || len(r.rates) != 2 || r.rates[1].Key != r.user.ID {
		t.Fatal("current actor rate policy missing")
	}
	r.rates = nil
	r.user.MustChangePassword = true
	if _, err := s.Preflight(context.Background(), Access{}, CreateDraftAction); !errors.Is(err, auth.ErrPasswordChangeRequired) || len(r.rates) != 0 {
		t.Fatal("invalid user consumed budget")
	}
	r.user.MustChangePassword = false
	r.preflightErr = auth.ErrCSRF
	if _, err := s.Preflight(context.Background(), Access{}, CreateDraftAction); !errors.Is(err, auth.ErrCSRF) {
		t.Fatal("proof failure ignored")
	}
	ctx, cancel := context.WithCancel(context.Background())
	first, err := s.AcquireValidation(ctx)
	if err != nil {
		t.Fatal(err)
	}
	second, err := s.AcquireValidation(ctx)
	if err != nil {
		t.Fatal(err)
	}
	cancel()
	if _, err = s.AcquireValidation(context.Background()); !errors.Is(err, auth.ErrUnavailable) {
		t.Fatal("cancel released still-working slot")
	}
	first()
	first()
	third, err := s.AcquireValidation(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.AcquireValidation(context.Background()); !errors.Is(err, auth.ErrUnavailable) {
		t.Fatal("double release overfilled capacity")
	}
	second()
	third()
	if _, err = s.AcquireValidation(ctx); !errors.Is(err, context.Canceled) {
		t.Fatal("canceled request acquired slot")
	}
}
