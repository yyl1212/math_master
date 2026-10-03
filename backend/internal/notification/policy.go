package notification

import "github.com/yyl1212/math_master/backend/internal/auth"

type Action string

const (
	ListAction     Action = "list"
	CountAction    Action = "count"
	ReadAction     Action = "read"
	MarkReadAction Action = "markRead"
)

func Authorize(u auth.User, a Action) error {
	if u.ID == "" {
		return auth.ErrAuthenticationRequired
	}
	switch a {
	case ListAction, CountAction, ReadAction, MarkReadAction:
	default:
		return auth.ErrInvalidInput
	}
	if u.MustChangePassword {
		return auth.ErrPasswordChangeRequired
	}
	return nil
}
