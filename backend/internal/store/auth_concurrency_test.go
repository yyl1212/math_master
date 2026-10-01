package store_test

import (
	"context"
	"crypto/rand"
	"errors"
	"github.com/yyl1212/math_master/backend/internal/auth"
	"testing"
	"time"
)

type verificationGate struct {
	auth.PasswordHasher
	entered chan struct{}
	release chan struct{}
}

func (h *verificationGate) Verify(ctx context.Context, phc, password string) (bool, error) {
	ok, err := h.PasswordHasher.Verify(ctx, phc, password)
	h.entered <- struct{}{}
	<-h.release
	return ok, err
}
func TestLoginRacesPasswordChange(t *testing.T) {
	f := newAuthFixture(t)
	user, cookies, csrf := f.signup("race_user")
	gate := &verificationGate{auth.NewArgon2Hasher(rand.Reader), make(chan struct{}, 1), make(chan struct{})}
	racing, err := auth.NewService(f.repo, gate, rand.Reader)
	if err != nil {
		t.Fatal("gated service failed")
	}
	anon, proof := f.anonymous()
	result := make(chan error, 1)
	go func() {
		_, _, err := racing.Login(f.ctx, anon, proof, auth.LoginInput{Username: user.Username, Password: accountTestPassword}, "old-login")
		result <- err
	}()
	select {
	case <-gate.entered:
	case <-time.After(2 * time.Second):
		t.Fatal("login verification not paused")
	}
	if _, err = f.service.ChangePassword(f.ctx, cookies, csrf, auth.PasswordInput{CurrentPassword: accountTestPassword, NewPassword: changedTestPassword}, "race-change"); err != nil {
		t.Fatal("concurrent password change failed")
	}
	close(gate.release)
	if err = <-result; !errors.Is(err, auth.ErrInvalidCredentials) {
		t.Fatal("old credential created new session")
	}
	if f.count("SELECT count(*) FROM auth_sessions WHERE revoked_at IS NULL") != 0 {
		t.Fatal("old login survived version change")
	}
}
