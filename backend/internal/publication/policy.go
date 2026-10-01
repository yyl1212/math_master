package publication

import (
	"github.com/yyl1212/math_master/backend/internal/auth"
	"regexp"
)

type Action string

const (
	ListDraftsAction          Action = "listDrafts"
	ReadDraftAction           Action = "readDraft"
	CreateDraftAction         Action = "createDraft"
	SaveDraftAction           Action = "saveDraft"
	AdoptDraftAction          Action = "adoptDraft"
	ValidateDraftAction       Action = "validateDraft"
	SubmitDraftAction         Action = "submitDraft"
	ListSubmissionsAction     Action = "listSubmissions"
	ReadSubmissionAction      Action = "readSubmission"
	ReviseSubmissionAction    Action = "reviseSubmission"
	DecideReviewAction        Action = "decideReview"
	ListPublicationsAction    Action = "listPublications"
	ReadPublicationAction     Action = "readPublication"
	PrepareReleaseAction      Action = "prepareRelease"
	ActivateReleaseAction     Action = "activateRelease"
	PreviewWithdrawalAction   Action = "previewWithdrawal"
	WithdrawVersionAction     Action = "withdrawVersion"
	ReadDraftAssetAction      Action = "readDraftAsset"
	ReadSubmissionAssetAction Action = "readSubmissionAsset"
)

var uuidV4 = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)
var sha256Hex = regexp.MustCompile(`^[0-9a-f]{64}$`)

func ValidID(v string) bool  { return uuidV4.MatchString(v) }
func ValidSHA(v string) bool { return sha256Hex.MatchString(v) }
func IsRead(a Action) bool {
	switch a {
	case ListDraftsAction, ReadDraftAction, ListSubmissionsAction, ReadSubmissionAction, ListPublicationsAction, ReadPublicationAction, ReadDraftAssetAction, ReadSubmissionAssetAction:
		return true
	}
	return false
}
func IsHeavy(a Action) bool {
	switch a {
	case SubmitDraftAction, DecideReviewAction, PrepareReleaseAction, ActivateReleaseAction, PreviewWithdrawalAction, WithdrawVersionAction:
		return true
	}
	return false
}
func IsIdempotent(a Action) bool {
	return !IsRead(a) && a != ValidateDraftAction && a != PreviewWithdrawalAction
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
	case PrepareReleaseAction, ActivateReleaseAction, PreviewWithdrawalAction, WithdrawVersionAction, ListPublicationsAction, ReadPublicationAction:
		allowed = []auth.Role{auth.RoleAdmin}
	case ListDraftsAction, ReadDraftAction, ReadDraftAssetAction:
		allowed = []auth.Role{auth.RoleEditor, auth.RoleAdmin}
	case ListSubmissionsAction, ReadSubmissionAction, ReadSubmissionAssetAction:
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
