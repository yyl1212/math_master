package auth

import (
	"context"
	"crypto/subtle"
	"encoding/base64"
	"golang.org/x/crypto/argon2"
	"io"
	"strings"
	"sync"
)

type PasswordHasher interface {
	Hash(context.Context, string) (string, error)
	Verify(context.Context, string, string) (bool, error)
}
type deriveFunc func(password, salt []byte, time, memory uint32, threads uint8, keyLen uint32) []byte

// Argon2Hasher bounds computations even when a request is cancelled.
type Argon2Hasher struct {
	random   io.Reader
	randomMu sync.Mutex
	derive   deriveFunc
	slots    chan struct{}
}

var processArgon2Slots = make(chan struct{}, 4)

func NewArgon2Hasher(random io.Reader) *Argon2Hasher {
	h := newHasher(random, argon2.IDKey)
	h.slots = processArgon2Slots
	return h
}
func newHasher(random io.Reader, derive deriveFunc) *Argon2Hasher {
	return &Argon2Hasher{random: random, derive: derive, slots: make(chan struct{}, 4)}
}
func (h *Argon2Hasher) acquire(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	select {
	case h.slots <- struct{}{}:
		return nil
	default:
		return &RateLimitError{RetryAfterSeconds: 1}
	}
}
func (h *Argon2Hasher) Hash(ctx context.Context, password string) (string, error) {
	if err := ValidateNewPassword(password); err != nil {
		return "", err
	}
	if err := h.acquire(ctx); err != nil {
		return "", err
	}
	defer func() { <-h.slots }()
	var salt [16]byte
	h.randomMu.Lock()
	var err error
	if h.random == nil {
		err = ErrUnavailable
	} else {
		_, err = io.ReadFull(h.random, salt[:])
	}
	h.randomMu.Unlock()
	if err != nil {
		return "", ErrUnavailable
	}
	key := h.derive([]byte(password), salt[:], 2, 19456, 1, 32)
	if err := ctx.Err(); err != nil {
		return "", err
	}
	if len(key) != 32 {
		return "", ErrUnavailable
	}
	return "$argon2id$v=19$m=19456,t=2,p=1$" + base64.RawStdEncoding.EncodeToString(salt[:]) + "$" + base64.RawStdEncoding.EncodeToString(key), nil
}
func parsePHC(phc string) ([]byte, []byte, error) {
	if len(phc) > 150 {
		return nil, nil, ErrUnavailable
	}
	parts := strings.Split(phc, "$")
	if len(parts) != 6 || parts[0] != "" || parts[1] != "argon2id" || parts[2] != "v=19" || parts[3] != "m=19456,t=2,p=1" {
		return nil, nil, ErrUnavailable
	}
	if len(parts[4]) != 22 || len(parts[5]) != 43 {
		return nil, nil, ErrUnavailable
	}
	salt, err := base64.RawStdEncoding.Strict().DecodeString(parts[4])
	if err != nil || len(salt) != 16 {
		return nil, nil, ErrUnavailable
	}
	key, err := base64.RawStdEncoding.Strict().DecodeString(parts[5])
	if err != nil || len(key) != 32 {
		return nil, nil, ErrUnavailable
	}
	return salt, key, nil
}
func (h *Argon2Hasher) Verify(ctx context.Context, phc, password string) (bool, error) {
	salt, want, err := parsePHC(phc)
	if err != nil {
		return false, err
	}
	if err := ValidateNewPassword(password); err != nil {
		return false, err
	}
	if err := h.acquire(ctx); err != nil {
		return false, err
	}
	defer func() { <-h.slots }()
	key := h.derive([]byte(password), salt, 2, 19456, 1, 32)
	if err := ctx.Err(); err != nil {
		return false, err
	}
	if len(key) != 32 {
		return false, ErrUnavailable
	}
	return subtle.ConstantTimeCompare(key, want) == 1, nil
}
