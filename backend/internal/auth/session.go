package auth

import (
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"io"
)

func NewID(random io.Reader) (string, error) {
	var raw [16]byte
	if random == nil {
		return "", ErrUnavailable
	}
	if _, err := io.ReadFull(random, raw[:]); err != nil {
		return "", ErrUnavailable
	}
	raw[6] = (raw[6] & 0x0f) | 0x40
	raw[8] = (raw[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", raw[:4], raw[4:6], raw[6:8], raw[8:10], raw[10:]), nil
}
func NewSecret(random io.Reader) (Secret, error) {
	var secret Secret
	if random == nil {
		return secret, ErrUnavailable
	}
	if _, err := io.ReadFull(random, secret[:]); err != nil {
		return Secret{}, ErrUnavailable
	}
	return secret, nil
}
func EncodeSecret(secret Secret) string { return base64.RawURLEncoding.EncodeToString(secret[:]) }
func DecodeSecret(value string) (Secret, error) {
	var secret Secret
	if len(value) != 43 {
		return secret, ErrInvalidCookie
	}
	for i := 0; i < len(value); i++ {
		c := value[i]
		if !(c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '-' || c == '_') {
			return secret, ErrInvalidCookie
		}
	}
	raw, err := base64.RawURLEncoding.Strict().DecodeString(value)
	if err != nil || len(raw) != 32 {
		return secret, ErrInvalidCookie
	}
	copy(secret[:], raw)
	return secret, nil
}
func TokenDigest(secret Secret) Digest { return sha256.Sum256(secret[:]) }
func EqualSecret(a, b Secret) bool     { return subtle.ConstantTimeCompare(a[:], b[:]) == 1 }
