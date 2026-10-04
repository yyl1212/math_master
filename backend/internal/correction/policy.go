package correction

import "github.com/yyl1212/math_master/backend/internal/auth"

type Action string

const (
	CreateCaseAction    Action = "createCase"
	CreatePlanAction    Action = "createPlan"
	UpdatePlanAction    Action = "updatePlan"
	SubmitPlanAction    Action = "submitPlan"
	DecidePlanAction    Action = "decidePlan"
	RetryJobAction      Action = "retryJob"
	ListCasesAction     Action = "listCases"
	ReadCaseAction      Action = "readCase"
	ListPlansAction     Action = "listPlans"
	ReadPlanAction      Action = "readPlan"
	ListJobsAction      Action = "listJobs"
	ListOwnAction       Action = "listOwn"
	ReadOwnAction       Action = "readOwn"
	ReadOwnDetailAction Action = "readOwnDetail"
)

func IsWrite(a Action) bool {
	switch a {
	case CreateCaseAction, CreatePlanAction, UpdatePlanAction, SubmitPlanAction, DecidePlanAction, RetryJobAction:
		return true
	}
	return false
}
func Authorize(u auth.User, a Action) error {
	if u.ID == "" {
		return auth.ErrAuthenticationRequired
	}
	var roles []auth.Role
	switch a {
	case ListOwnAction, ReadOwnAction, ReadOwnDetailAction:
	case CreateCaseAction, RetryJobAction:
		roles = []auth.Role{auth.RoleAdmin}
	case CreatePlanAction, UpdatePlanAction, SubmitPlanAction:
		roles = []auth.Role{auth.RoleEditor, auth.RoleAdmin}
	case DecidePlanAction:
		roles = []auth.Role{auth.RoleReviewer, auth.RoleAdmin}
	case ListCasesAction, ReadCaseAction, ListPlansAction, ReadPlanAction, ListJobsAction:
		roles = []auth.Role{auth.RoleEditor, auth.RoleReviewer, auth.RoleAdmin}
	default:
		return auth.ErrInvalidInput
	}
	if u.MustChangePassword {
		return auth.ErrPasswordChangeRequired
	}
	if roles == nil {
		return nil
	}
	for _, r := range u.Roles {
		for _, allowed := range roles {
			if r == allowed {
				return nil
			}
		}
	}
	return auth.ErrForbidden
}
func CanReview(u auth.User, creatorID string, authors []string) bool {
	if creatorID == "" || u.ID == creatorID || Authorize(u, DecidePlanAction) != nil {
		return false
	}
	for _, id := range authors {
		if id == u.ID {
			return false
		}
	}
	return true
}
