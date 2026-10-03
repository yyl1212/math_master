package feedback

import "github.com/yyl1212/math_master/backend/internal/auth"

func IsWrite(a Action) bool { return a == CreateAction || a == ReplyAction || a == TransitionAction }
func IsReview(a Action) bool {
	return a == TransitionAction || a == ListReviewAction || a == ReadReviewAction || a == DiscussReviewAction
}
func validAction(a Action) bool {
	switch a {
	case ReadContextAction, CreateAction, ReplyAction, TransitionAction, ListOwnAction, ReadOwnAction, DiscussOwnAction, ListReviewAction, ReadReviewAction, DiscussReviewAction:
		return true
	}
	return false
}
func Authorize(u auth.User, a Action) error {
	if u.ID == "" {
		return auth.ErrAuthenticationRequired
	}
	if !validAction(a) {
		return auth.ErrInvalidInput
	}
	if u.MustChangePassword {
		return auth.ErrPasswordChangeRequired
	}
	if !IsReview(a) {
		return nil
	}
	for _, r := range u.Roles {
		if r == auth.RoleReviewer || r == auth.RoleAdmin {
			return nil
		}
	}
	return auth.ErrForbidden
}
func CanHandle(u auth.User, ownerID string) bool {
	return ownerID != "" && u.ID != ownerID && Authorize(u, TransitionAction) == nil
}
