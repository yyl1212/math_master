package auth

import (
	"context"
	"crypto/sha256"
	"errors"
	"io"
	"sync"
	"time"
)

type Service struct {
	repo     AccountRepository
	hasher   PasswordHasher
	random   io.Reader
	randomMu sync.Mutex
	dummyPHC string
}

func NewService(repo AccountRepository, hasher PasswordHasher, random io.Reader) (*Service, error) {
	if repo == nil || hasher == nil || random == nil {
		return nil, ErrUnavailable
	}
	phc, err := hasher.Hash(context.Background(), "a fixed non-account dummy passphrase")
	if err != nil {
		return nil, ErrUnavailable
	}
	return &Service{repo: repo, hasher: hasher, random: random, dummyPHC: phc}, nil
}
func (s *Service) newSecret() (Secret, error) {
	s.randomMu.Lock()
	defer s.randomMu.Unlock()
	return NewSecret(s.random)
}
func (s *Service) newID() (string, error) {
	s.randomMu.Lock()
	defer s.randomMu.Unlock()
	return NewID(s.random)
}
func cookieHash(value string) (Digest, error) {
	secret, err := DecodeSecret(value)
	if err != nil {
		return Digest{}, err
	}
	return TokenDigest(secret), nil
}
func proofSecret(value string) (Secret, error) {
	secret, err := DecodeSecret(value)
	if err != nil {
		return Secret{}, ErrCSRF
	}
	return secret, nil
}
func (s *Service) Context(ctx context.Context, cookies Cookies) (ContextView, CookieDelta, error) {
	fail := func(err error) (ContextView, CookieDelta, error) { return ContextView{}, CookieDelta{}, err }
	if err := s.repo.ConsumeRates(ctx, contextRate()); err != nil {
		return fail(err)
	}
	delta := CookieDelta{}
	if cookies.Session != "" {
		hash, err := cookieHash(cookies.Session)
		if err != nil {
			return fail(err)
		}
		record, err := s.repo.ReadSession(ctx, hash, true)
		if err == nil {
			return ContextView{User: &record.User, CSRFToken: EncodeSecret(record.CSRF)}, delta, nil
		}
		if !errors.Is(err, ErrAuthenticationRequired) {
			return fail(err)
		}
		delta.ClearSession = true
	}
	if cookies.Preauth != "" {
		hash, err := cookieHash(cookies.Preauth)
		if err != nil {
			return fail(err)
		}
		record, err := s.repo.ReadPreauth(ctx, hash)
		if err == nil {
			return ContextView{CSRFToken: EncodeSecret(record.CSRF)}, delta, nil
		}
		if !errors.Is(err, ErrCSRF) {
			return fail(err)
		}
	}
	if err := s.repo.ConsumeRates(ctx, []RateKey{globalRate("new_preauth", 120)}); err != nil {
		return fail(err)
	}
	token, err := s.newSecret()
	if err != nil {
		return fail(err)
	}
	csrf, err := s.newSecret()
	if err != nil {
		return fail(err)
	}
	if err = s.repo.CreatePreauth(ctx, NewPreauth{TokenHash: TokenDigest(token), CSRF: csrf}); err != nil {
		return fail(err)
	}
	delta.SetPreauth = EncodeSecret(token)
	return ContextView{CSRFToken: EncodeSecret(csrf)}, delta, nil
}
func (s *Service) Session(ctx context.Context, cookies Cookies) (*User, error) {
	if err := s.repo.ConsumeRates(ctx, contextRate()); err != nil {
		return nil, err
	}
	if cookies.Session == "" {
		return nil, nil
	}
	hash, err := cookieHash(cookies.Session)
	if err != nil {
		return nil, err
	}
	record, err := s.repo.ReadSession(ctx, hash, false)
	if errors.Is(err, ErrAuthenticationRequired) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &record.User, nil
}
func (s *Service) anonymousProof(ctx context.Context, cookies Cookies, csrf string) (PreauthProof, error) {
	if cookies.Session != "" {
		hash, err := cookieHash(cookies.Session)
		if err != nil {
			return PreauthProof{}, err
		}
		_, err = s.repo.ReadSession(ctx, hash, false)
		if err == nil {
			return PreauthProof{}, ErrAlreadyAuthenticated
		}
		if !errors.Is(err, ErrAuthenticationRequired) {
			return PreauthProof{}, err
		}
	}
	if cookies.Preauth == "" {
		return PreauthProof{}, ErrCSRF
	}
	hash, err := cookieHash(cookies.Preauth)
	if err != nil {
		return PreauthProof{}, err
	}
	secret, err := proofSecret(csrf)
	if err != nil {
		return PreauthProof{}, err
	}
	record, err := s.repo.ReadPreauth(ctx, hash)
	if err != nil {
		return PreauthProof{}, err
	}
	if !EqualSecret(record.CSRF, secret) {
		return PreauthProof{}, ErrCSRF
	}
	return PreauthProof{hash, secret}, nil
}
func (s *Service) Register(ctx context.Context, cookies Cookies, csrf string, input RegisterInput, requestID string) (User, CookieDelta, error) {
	fail := func(err error) (User, CookieDelta, error) { return User{}, CookieDelta{}, err }
	username, err := ValidateUsername(input.Username)
	if err != nil {
		return fail(err)
	}
	if err = ValidateNewPassword(input.Password); err != nil {
		return fail(err)
	}
	proof, err := s.anonymousProof(ctx, cookies, csrf)
	if err != nil {
		return fail(err)
	}
	if err = s.repo.ConsumeRates(ctx, registerRates(proof.TokenHash)); err != nil {
		return fail(err)
	}
	phc, err := s.hasher.Hash(ctx, input.Password)
	if err != nil {
		return fail(err)
	}
	id, err := s.newID()
	if err != nil {
		return fail(err)
	}
	user, err := s.repo.RegisterLearner(ctx, proof, id, username, phc, requestID)
	if err != nil {
		return fail(err)
	}
	return user, CookieDelta{ClearPreauth: true, ClearSession: cookies.Session != ""}, nil
}
func (s *Service) loginFailure(ctx context.Context, hash Digest, requestID string) error {
	if err := s.repo.RecordLoginFailure(ctx, hash, requestID); err != nil {
		return err
	}
	return ErrInvalidCredentials
}
func (s *Service) Login(ctx context.Context, cookies Cookies, csrf string, input LoginInput, requestID string) (User, CookieDelta, error) {
	fail := func(err error) (User, CookieDelta, error) { return User{}, CookieDelta{}, err }
	username, err := ValidateUsername(input.Username)
	if err != nil {
		return fail(err)
	}
	if err = ValidateNewPassword(input.Password); err != nil {
		return fail(err)
	}
	proof, err := s.anonymousProof(ctx, cookies, csrf)
	if err != nil {
		return fail(err)
	}
	usernameHash := Digest(sha256.Sum256([]byte(username)))
	if err = s.repo.ConsumeRates(ctx, loginRates(usernameHash, proof.TokenHash)); err != nil {
		return fail(err)
	}
	credential, err := s.repo.ReadCredential(ctx, username)
	unknown := errors.Is(err, ErrNotFound)
	if err != nil && !unknown {
		return fail(err)
	}
	phc := credential.PHC
	if unknown {
		phc = s.dummyPHC
	}
	ok, err := s.hasher.Verify(ctx, phc, input.Password)
	if err != nil {
		return fail(err)
	}
	if unknown || !ok {
		return fail(s.loginFailure(ctx, usernameHash, requestID))
	}
	token, err := s.newSecret()
	if err != nil {
		return fail(err)
	}
	secret, err := s.newSecret()
	if err != nil {
		return fail(err)
	}
	user, err := s.repo.LoginSession(ctx, proof, credential.UserID, credential.Version, NewSession{TokenDigest(token), secret}, requestID)
	if errors.Is(err, ErrInvalidCredentials) {
		return fail(s.loginFailure(ctx, usernameHash, requestID))
	}
	if err != nil {
		return fail(err)
	}
	return user, CookieDelta{SetSession: EncodeSecret(token), ClearPreauth: true}, nil
}
func (s *Service) sessionProof(ctx context.Context, cookies Cookies, csrf string) (SessionProof, SessionRecord, error) {
	if cookies.Session == "" {
		return SessionProof{}, SessionRecord{}, ErrAuthenticationRequired
	}
	hash, err := cookieHash(cookies.Session)
	if err != nil {
		return SessionProof{}, SessionRecord{}, err
	}
	record, err := s.repo.ReadSession(ctx, hash, true)
	if err != nil {
		return SessionProof{}, SessionRecord{}, err
	}
	secret, err := proofSecret(csrf)
	if err != nil {
		return SessionProof{}, SessionRecord{}, err
	}
	if !EqualSecret(record.CSRF, secret) {
		return SessionProof{}, SessionRecord{}, ErrCSRF
	}
	return SessionProof{hash, secret}, record, nil
}
func (s *Service) Logout(ctx context.Context, cookies Cookies, csrf string, all bool, requestID string) (CookieDelta, error) {
	proof, _, err := s.sessionProof(ctx, cookies, csrf)
	if err != nil {
		return CookieDelta{}, err
	}
	if err = s.repo.LogoutSession(ctx, proof, all, requestID); err != nil {
		return CookieDelta{}, err
	}
	return CookieDelta{ClearSession: true, ClearPreauth: true}, nil
}
func (s *Service) verifiedCredential(ctx context.Context, record SessionRecord, password string) (Credential, error) {
	credential, err := s.repo.ReadCredential(ctx, record.User.Username)
	if err != nil {
		return Credential{}, err
	}
	if credential.Version != record.CredentialVersion {
		return Credential{}, ErrInvalidCredentials
	}
	ok, err := s.hasher.Verify(ctx, credential.PHC, password)
	if err != nil {
		return Credential{}, err
	}
	if !ok {
		return Credential{}, ErrInvalidCredentials
	}
	return credential, nil
}
func (s *Service) ChangePassword(ctx context.Context, cookies Cookies, csrf string, input PasswordInput, requestID string) (CookieDelta, error) {
	if err := ValidateNewPassword(input.CurrentPassword); err != nil {
		return CookieDelta{}, err
	}
	if err := ValidateNewPassword(input.NewPassword); err != nil {
		return CookieDelta{}, err
	}
	proof, record, err := s.sessionProof(ctx, cookies, csrf)
	if err != nil {
		return CookieDelta{}, err
	}
	if err = s.repo.ConsumeRates(ctx, passwordRates(proof.TokenHash)); err != nil {
		return CookieDelta{}, err
	}
	credential, err := s.verifiedCredential(ctx, record, input.CurrentPassword)
	if err != nil {
		return CookieDelta{}, err
	}
	phc, err := s.hasher.Hash(ctx, input.NewPassword)
	if err != nil {
		return CookieDelta{}, err
	}
	if err = s.repo.ChangePassword(ctx, proof, credential.Version, phc, requestID); err != nil {
		return CookieDelta{}, err
	}
	return CookieDelta{ClearSession: true, ClearPreauth: true}, nil
}
func (s *Service) Reauthenticate(ctx context.Context, cookies Cookies, csrf string, input ReauthInput, requestID string) (time.Time, error) {
	if err := ValidateNewPassword(input.Password); err != nil {
		return time.Time{}, err
	}
	proof, record, err := s.sessionProof(ctx, cookies, csrf)
	if err != nil {
		return time.Time{}, err
	}
	if record.User.MustChangePassword {
		return time.Time{}, ErrPasswordChangeRequired
	}
	if err = s.repo.ConsumeRates(ctx, passwordRates(proof.TokenHash)); err != nil {
		return time.Time{}, err
	}
	credential, err := s.verifiedCredential(ctx, record, input.Password)
	if err != nil {
		return time.Time{}, err
	}
	return s.repo.ReauthenticateSession(ctx, proof, credential.Version, requestID)
}
func (s *Service) Cleanup(ctx context.Context) (int, error) { return s.repo.CleanupAuth(ctx, 2000) }
