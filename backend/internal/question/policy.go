package question

import (
	"github.com/yyl1212/math_master/backend/internal/auth"
	"github.com/yyl1212/math_master/backend/internal/publication"
	"regexp"
)

type Action string

const (
	ListDraftsAction        Action = "listDrafts"
	ReadDraftAction         Action = "readDraft"
	CreateDraftAction       Action = "createDraft"
	SaveDraftAction         Action = "saveDraft"
	AdoptDraftAction        Action = "adoptDraft"
	ValidateDraftAction     Action = "validateDraft"
	SubmitDraftAction       Action = "submitDraft"
	ListSubmissionsAction   Action = "listSubmissions"
	ReadSubmissionAction    Action = "readSubmission"
	ReviseSubmissionAction  Action = "reviseSubmission"
	DecideReviewAction      Action = "decideReview"
	ListPublicationsAction  Action = "listPublications"
	ReadPublicationAction   Action = "readPublication"
	PrepareReleaseAction    Action = "prepareRelease"
	ActivateReleaseAction   Action = "activateRelease"
	PreviewWithdrawalAction Action = "previewWithdrawal"
	WithdrawVersionAction   Action = "withdrawVersion"
	ListInstancesAction     Action = "listInstances"
	ListMembersAction       Action = "listMembers"
	ListChangesAction       Action = "listChanges"
	ReadCoverageAction      Action = "readCoverage"
)

func ValidMathID(v string) bool { return publication.ValidMathID(v) }
func ValidID(v string) bool     { return publication.ValidID(v) }
func ValidSHA(v string) bool    { return publication.ValidSHA(v) }

var generatedID = regexp.MustCompile(`^qi-[0-9a-f]{64}$`)

func ValidInstanceID(v string) bool { return ValidMathID(v) || generatedID.MatchString(v) }
func IsRead(a Action) bool {
	switch a {
	case ListDraftsAction, ReadDraftAction, ListSubmissionsAction, ReadSubmissionAction, ListPublicationsAction, ReadPublicationAction, ListInstancesAction, ListMembersAction, ListChangesAction, ReadCoverageAction:
		return true
	}
	return false
}
func IsHeavy(a Action) bool {
	switch a {
	case ValidateDraftAction, ReadCoverageAction, SubmitDraftAction, DecideReviewAction, PrepareReleaseAction, ActivateReleaseAction, PreviewWithdrawalAction, WithdrawVersionAction:
		return true
	}
	return false
}
func IsIdempotent(a Action) bool {
	return validAction(a) && !IsRead(a) && a != ValidateDraftAction && a != PreviewWithdrawalAction
}
func Authorize(u auth.User, a Action) error {
	if u.ID == "" {
		return auth.ErrAuthenticationRequired
	}
	var allowed []auth.Role
	switch a {
	case CreateDraftAction, SaveDraftAction, AdoptDraftAction, ValidateDraftAction, SubmitDraftAction, ReviseSubmissionAction:
		allowed = []auth.Role{auth.RoleEditor}
	case DecideReviewAction:
		allowed = []auth.Role{auth.RoleReviewer}
	case PrepareReleaseAction, ActivateReleaseAction, PreviewWithdrawalAction, WithdrawVersionAction, ListPublicationsAction, ReadPublicationAction, ListMembersAction, ListChangesAction:
		allowed = []auth.Role{auth.RoleAdmin}
	case ReadCoverageAction:
		allowed = []auth.Role{auth.RoleReviewer, auth.RoleAdmin}
	case ListDraftsAction, ReadDraftAction:
		allowed = []auth.Role{auth.RoleEditor, auth.RoleAdmin}
	case ListSubmissionsAction, ReadSubmissionAction, ListInstancesAction:
		allowed = []auth.Role{auth.RoleEditor, auth.RoleReviewer, auth.RoleAdmin}
	default:
		return auth.ErrInvalidInput
	}
	if u.MustChangePassword {
		return auth.ErrPasswordChangeRequired
	}
	for _, role := range u.Roles {
		for _, want := range allowed {
			if role == want {
				return nil
			}
		}
	}
	return auth.ErrForbidden
}

func validAction(a Action) bool {
	return Authorize(auth.User{ID: "11111111-1111-4111-8111-111111111111", Roles: []auth.Role{auth.RoleEditor, auth.RoleReviewer, auth.RoleAdmin}}, a) == nil
}
