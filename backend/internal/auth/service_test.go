package auth

import (
	"context"
	"crypto/rand"
	"errors"
	"testing"
)

type faultRepository struct{ AccountRepository }

func (faultRepository) ConsumeRates(context.Context, []RateKey) error { return nil }
func (faultRepository) ReadSession(context.Context, Digest, bool) (SessionRecord, error) {
	return SessionRecord{}, ErrUnavailable
}

type faultHasher struct{}

func (faultHasher) Hash(context.Context, string) (string, error) { return "", ErrUnavailable }
func (faultHasher) Verify(context.Context, string, string) (bool, error) {
	return false, ErrUnavailable
}
func TestContextRecoveryFault(t *testing.T) {
	service, err := NewService(faultRepository{}, NewArgon2Hasher(rand.Reader), rand.Reader)
	if err != nil {
		t.Fatal("constructor failed")
	}
	var token Secret
	_, delta, err := service.Context(context.Background(), Cookies{Session: EncodeSecret(token)})
	if !errors.Is(err, ErrUnavailable) || delta != (CookieDelta{}) {
		t.Fatal("database fault changed cookie")
	}
	if _, err := NewService(faultRepository{}, faultHasher{}, rand.Reader); !errors.Is(err, ErrUnavailable) {
		t.Fatal("dummy hash failure ignored")
	}
}
