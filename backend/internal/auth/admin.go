package auth

import (
	"context"
	"io"
	"regexp"
	"strings"
	"sync"
	"time"
	"unicode/utf8"
)

type RoleMutation struct {
	ActorID string
	Changed bool
}
type AdminRepository interface {
	ReadSession(context.Context, Digest, bool) (SessionRecord, error)
	ConsumeRates(context.Context, []RateKey) error
	InitializeAdmin(context.Context, string, string, string, string) (User, error)
	ListAccountUsers(context.Context, Digest, UserQuery) (UserPage, error)
	ReplaceAccountRoles(context.Context, SessionProof, string, RolesInput, string) (RoleMutation, error)
	ResetAccountPassword(context.Context, SessionProof, string, string, string, string, string) (RoleMutation, error)
}
type AdminService struct {
	repo     AdminRepository
	hasher   PasswordHasher
	random   io.Reader
	randomMu sync.Mutex
}

func NewAdminService(repo AdminRepository, hasher PasswordHasher, random io.Reader) *AdminService {
	return &AdminService{repo: repo, hasher: hasher, random: random}
}

var userIDPattern = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)

func ValidateUserID(id string) error {
	if !userIDPattern.MatchString(id) {
		return ErrInvalidInput
	}
	return nil
}
func ValidateNote(note string) error {
	if !utf8.ValidString(note) || strings.TrimSpace(note) == "" {
		return ErrInvalidInput
	}
	n := utf8.RuneCountInString(note)
	if n < 10 || n > 1000 {
		return ErrInvalidInput
	}
	return nil
}
func NormalizeUserQuery(query UserQuery) (UserQuery, error) {
	if !utf8.ValidString(query.Q) || len(query.Q) > 128 || query.Offset < 0 {
		return UserQuery{}, ErrInvalidInput
	}
	if query.Limit == 0 {
		query.Limit = 20
	}
	if query.Limit < 1 || query.Limit > 100 {
		return UserQuery{}, ErrInvalidInput
	}
	return query, nil
}
func HasRole(user User, role Role) bool {
	for _, r := range user.Roles {
		if r == role {
			return true
		}
	}
	return false
}
func (s *AdminService) Initialize(ctx context.Context, username, password, requestID string) (User, error) {
	if s.repo == nil || s.hasher == nil || s.random == nil {
		return User{}, ErrUnavailable
	}
	normalized, err := ValidateUsername(username)
	if err != nil {
		return User{}, err
	}
	if err = ValidateNewPassword(password); err != nil {
		return User{}, err
	}
	phc, err := s.hasher.Hash(ctx, password)
	if err != nil {
		return User{}, err
	}
	s.randomMu.Lock()
	id, err := NewID(s.random)
	s.randomMu.Unlock()
	if err != nil {
		return User{}, err
	}
	return s.repo.InitializeAdmin(ctx, id, normalized, phc, requestID)
}
func (s *AdminService) ListUsers(ctx context.Context, cookies Cookies, query UserQuery) (UserPage, error) {
	query, err := NormalizeUserQuery(query)
	if err != nil {
		return UserPage{}, err
	}
	if cookies.Session == "" {
		return UserPage{}, ErrAuthenticationRequired
	}
	hash, err := cookieHash(cookies.Session)
	if err != nil {
		return UserPage{}, err
	}
	return s.repo.ListAccountUsers(ctx, hash, query)
}
func (s *AdminService) writeProof(ctx context.Context, cookies Cookies, csrf string) (SessionProof, error) {
	if cookies.Session == "" {
		return SessionProof{}, ErrAuthenticationRequired
	}
	hash, err := cookieHash(cookies.Session)
	if err != nil {
		return SessionProof{}, err
	}
	record, err := s.repo.ReadSession(ctx, hash, true)
	if err != nil {
		return SessionProof{}, err
	}
	if record.User.MustChangePassword {
		return SessionProof{}, ErrPasswordChangeRequired
	}
	if !HasRole(record.User, RoleAdmin) {
		return SessionProof{}, ErrForbidden
	}
	secret, err := proofSecret(csrf)
	if err != nil {
		return SessionProof{}, err
	}
	if !EqualSecret(secret, record.CSRF) {
		return SessionProof{}, ErrCSRF
	}
	if err = s.repo.ConsumeRates(ctx, []RateKey{globalRate("admin_write", 60), {Scope: "admin_actor", Key: record.User.ID, Limit: 10, Window: time.Minute}}); err != nil {
		return SessionProof{}, err
	}
	return SessionProof{hash, secret}, nil
}
func mutationCookies(mutation RoleMutation, targetID string) CookieDelta {
	if mutation.Changed && mutation.ActorID == targetID {
		return CookieDelta{ClearSession: true, ClearPreauth: true}
	}
	return CookieDelta{}
}
func (s *AdminService) ReplaceRoles(ctx context.Context, cookies Cookies, csrf, target string, input RolesInput, requestID string) (CookieDelta, error) {
	if err := ValidateUserID(target); err != nil {
		return CookieDelta{}, err
	}
	if err := ValidateNote(input.Reason); err != nil {
		return CookieDelta{}, err
	}
	roles, err := NormalizeRoles(input.Roles)
	if err != nil {
		return CookieDelta{}, err
	}
	input.Roles = roles
	proof, err := s.writeProof(ctx, cookies, csrf)
	if err != nil {
		return CookieDelta{}, err
	}
	mutation, err := s.repo.ReplaceAccountRoles(ctx, proof, target, input, requestID)
	if err != nil {
		return CookieDelta{}, err
	}
	return mutationCookies(mutation, target), nil
}
func (s *AdminService) ResetPassword(ctx context.Context, cookies Cookies, csrf, target string, input ResetInput, requestID string) (CookieDelta, error) {
	if err := ValidateUserID(target); err != nil {
		return CookieDelta{}, err
	}
	if err := ValidateNote(input.Reason); err != nil {
		return CookieDelta{}, err
	}
	if err := ValidateNote(input.OwnershipNote); err != nil {
		return CookieDelta{}, err
	}
	if err := ValidateNewPassword(input.TemporaryPassword); err != nil {
		return CookieDelta{}, err
	}
	proof, err := s.writeProof(ctx, cookies, csrf)
	if err != nil {
		return CookieDelta{}, err
	}
	phc, err := s.hasher.Hash(ctx, input.TemporaryPassword)
	if err != nil {
		return CookieDelta{}, err
	}
	mutation, err := s.repo.ResetAccountPassword(ctx, proof, target, phc, input.Reason, input.OwnershipNote, requestID)
	if err != nil {
		return CookieDelta{}, err
	}
	return mutationCookies(mutation, target), nil
}
