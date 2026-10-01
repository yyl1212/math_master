package httpapi

import (
	"context"
	"github.com/yyl1212/math_master/backend/internal/auth"
	"github.com/yyl1212/math_master/backend/internal/question"
	"net/http"
)

func dispatchQuestion(ctx context.Context, r *http.Request, route questionRoute, a question.Access, q question.ListQuery, knowledge string, s *question.Service) (any, error) {
	decode := func(v any) error { return decodeQuestionJSON(r.Body, route.Limit, v) }
	head := func(v *string) bool { return v == nil || question.ValidID(*v) }
	switch route.Action {
	case question.ListDraftsAction:
		return s.ListDrafts(ctx, a, q)
	case question.ReadDraftAction:
		return s.ReadDraft(ctx, a, route.ID)
	case question.CreateDraftAction:
		var v question.DraftInput
		if e := decode(&v); e != nil {
			return nil, e
		}
		return s.CreateDraft(ctx, a, v)
	case question.SaveDraftAction:
		var v question.SaveDraftInput
		if e := decode(&v); e != nil {
			return nil, e
		}
		if v.ExpectedRevision < 1 || v.ExpectedRevision > 9007199254740991 {
			return nil, auth.ErrInvalidInput
		}
		return s.SaveDraft(ctx, a, route.ID, v)
	case question.AdoptDraftAction:
		var v question.AdoptInput
		if e := decode(&v); e != nil {
			return nil, e
		}
		if !question.ValidMathID(v.PackageID) || v.PackageVersion < 1 || v.PackageVersion > 2147483647 {
			return nil, auth.ErrInvalidInput
		}
		return s.AdoptDraft(ctx, a, v)
	case question.ValidateDraftAction:
		var v question.ValidateInput
		if e := decode(&v); e != nil {
			return nil, e
		}
		if v.ExpectedRevision < 1 || v.ExpectedRevision > 9007199254740991 {
			return nil, auth.ErrInvalidInput
		}
		return s.ValidateDraft(ctx, a, route.ID, v)
	case question.SubmitDraftAction:
		var v question.SubmitInput
		if e := decode(&v); e != nil {
			return nil, e
		}
		if v.ExpectedRevision < 1 || v.ExpectedRevision > 9007199254740991 || !question.ValidSHA(v.ExpectedDigest) {
			return nil, auth.ErrInvalidInput
		}
		return s.SubmitDraft(ctx, a, route.ID, v)
	case question.ListSubmissionsAction:
		return s.ListSubmissions(ctx, a, q)
	case question.ReadSubmissionAction:
		return s.ReadSubmission(ctx, a, route.ID)
	case question.ListInstancesAction:
		return s.ListInstances(ctx, a, route.ID, q)
	case question.ReviseSubmissionAction:
		var v struct{}
		if e := decode(&v); e != nil {
			return nil, e
		}
		return s.ReviseSubmission(ctx, a, route.ID)
	case question.DecideReviewAction:
		var v question.ReviewInput
		if e := decode(&v); e != nil {
			return nil, e
		}
		return s.DecideReview(ctx, a, route.ID, v)
	case question.ListPublicationsAction:
		return s.ListPublications(ctx, a, q)
	case question.ReadPublicationAction:
		return s.ReadPublication(ctx, a, route.ID)
	case question.ListMembersAction:
		return s.ListMembers(ctx, a, route.ID, q)
	case question.ListChangesAction:
		return s.ListChanges(ctx, a, route.ID, q)
	case question.PrepareReleaseAction:
		var v question.PrepareInput
		if e := decode(&v); e != nil {
			return nil, e
		}
		if !head(v.ExpectedKnowledgeHead) || !head(v.ExpectedQuestionHead) || len(v.SubmissionIDs) < 1 || len(v.SubmissionIDs) > 20 {
			return nil, auth.ErrInvalidInput
		}
		for _, id := range v.SubmissionIDs {
			if !question.ValidID(id) {
				return nil, auth.ErrInvalidInput
			}
		}
		return s.PrepareRelease(ctx, a, v)
	case question.ActivateReleaseAction:
		var v question.ActivateInput
		if e := decode(&v); e != nil {
			return nil, e
		}
		if !head(v.ExpectedKnowledgeHead) || !head(v.ExpectedQuestionHead) || !question.ValidSHA(v.ExpectedManifestSHA) {
			return nil, auth.ErrInvalidInput
		}
		return s.ActivateRelease(ctx, a, route.ID, v)
	case question.PreviewWithdrawalAction:
		var v question.WithdrawalPreviewInput
		if e := decode(&v); e != nil {
			return nil, e
		}
		if question.ValidateWithdrawalTarget(v.Target) != nil {
			return nil, auth.ErrInvalidInput
		}
		return s.PreviewWithdrawal(ctx, a, v, q)
	case question.WithdrawVersionAction:
		var v question.WithdrawalInput
		if e := decode(&v); e != nil {
			return nil, e
		}
		if question.ValidateWithdrawalTarget(v.Target) != nil || !head(v.ExpectedKnowledgeHead) || !head(v.ExpectedQuestionHead) {
			return nil, auth.ErrInvalidInput
		}
		return s.WithdrawVersion(ctx, a, v)
	case question.ReadCoverageAction:
		return s.ReadCoverage(ctx, a, question.CoverageQuery{Limit: q.Limit, Offset: q.Offset, KnowledgeID: knowledge})
	}
	return nil, auth.ErrNotFound
}
