package auth

import (
	"context"
	"time"
)

type Cookies struct{ Session, Preauth string }
type PreauthProof struct {
	TokenHash Digest
	CSRF      Secret
}
type SessionProof struct {
	TokenHash Digest
	CSRF      Secret
}
type NewPreauth struct {
	TokenHash Digest
	CSRF      Secret
}
type NewSession struct {
	TokenHash Digest
	CSRF      Secret
}
type Credential struct {
	UserID, PHC string
	Version     int64
}
type SessionRecord struct {
	User              User
	CSRF              Secret
	CredentialVersion int64
	ReauthenticatedAt *time.Time
}
type PreauthRecord struct{ CSRF Secret }
type CookieDelta struct {
	SetSession   string `json:"-"`
	SetPreauth   string `json:"-"`
	ClearSession bool   `json:"-"`
	ClearPreauth bool   `json:"-"`
}
type ContextView struct {
	User      *User  `json:"user"`
	CSRFToken string `json:"csrfToken"`
}
type RegisterInput struct {
	Username string `json:"username"`
	Password string `json:"password"`
}
type LoginInput struct {
	Username string `json:"username"`
	Password string `json:"password"`
}
type PasswordInput struct {
	CurrentPassword string `json:"currentPassword"`
	NewPassword     string `json:"newPassword"`
}
type ReauthInput struct {
	Password string `json:"password"`
}
type RolesInput struct {
	Roles  []Role `json:"roles"`
	Reason string `json:"reason"`
}
type ResetInput struct {
	TemporaryPassword string `json:"temporaryPassword"`
	Reason            string `json:"reason"`
	OwnershipNote     string `json:"ownershipNote"`
}
type UserQuery struct {
	Q             string
	Limit, Offset int
}
type UserPage struct {
	Items  []User `json:"items"`
	Total  int    `json:"total"`
	Limit  int    `json:"limit"`
	Offset int    `json:"offset"`
}
type AccountRepository interface {
	ReadCredential(context.Context, string) (Credential, error)
	RecordLoginFailure(context.Context, Digest, string) error
	ReadSession(context.Context, Digest, bool) (SessionRecord, error)
	ReadPreauth(context.Context, Digest) (PreauthRecord, error)
	CreatePreauth(context.Context, NewPreauth) error
	RegisterLearner(context.Context, PreauthProof, string, string, string, string) (User, error)
	LoginSession(context.Context, PreauthProof, string, int64, NewSession, string) (User, error)
	LogoutSession(context.Context, SessionProof, bool, string) error
	ChangePassword(context.Context, SessionProof, int64, string, string) error
	ReauthenticateSession(context.Context, SessionProof, int64, string) (time.Time, error)
	ConsumeRates(context.Context, []RateKey) error
	CleanupAuth(context.Context, int) (int, error)
}
