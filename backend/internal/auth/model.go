package auth

import "errors"

type Role string

const (
	RoleLearner  Role = "learner"
	RoleEditor   Role = "editor"
	RoleReviewer Role = "reviewer"
	RoleAdmin    Role = "admin"
)

type User struct {
	ID                 string `json:"id"`
	Username           string `json:"username"`
	Roles              []Role `json:"roles"`
	MustChangePassword bool   `json:"mustChangePassword"`
}
type Secret [32]byte
type Digest [32]byte

var (
	ErrInvalidInput           = errors.New("invalid account input")
	ErrInvalidCookie          = errors.New("invalid account cookie")
	ErrInvalidCredentials     = errors.New("invalid credentials")
	ErrAuthenticationRequired = errors.New("authentication required")
	ErrCSRF                   = errors.New("request verification failed")
	ErrForbidden              = errors.New("forbidden")
	ErrUsernameUnavailable    = errors.New("username unavailable")
	ErrAlreadyAuthenticated   = errors.New("already authenticated")
	ErrLastAdminRequired      = errors.New("last administrator required")
	ErrPasswordChangeRequired = errors.New("password change required")
	ErrReauthRequired         = errors.New("reauthentication required")
	ErrNotFound               = errors.New("account resource not found")
	ErrUnavailable            = errors.New("account service unavailable")
	ErrAlreadyInitialized     = errors.New("administrator already initialized")
)

type RateLimitError struct{ RetryAfterSeconds int }

func (*RateLimitError) Error() string { return "account rate limit exceeded" }
