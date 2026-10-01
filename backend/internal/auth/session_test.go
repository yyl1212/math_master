package auth

import (
	"bytes"
	"encoding/hex"
	"errors"
	"io"
	"strings"
	"testing"
)

type brokenRandom struct{}

func (brokenRandom) Read([]byte) (int, error) { return 0, io.ErrUnexpectedEOF }

func TestTokenAndIDBoundaries(t *testing.T) {
	secret, err := NewSecret(bytes.NewReader(make([]byte, 32)))
	if err != nil {
		t.Fatal("secret generation failed")
	}
	encoded := EncodeSecret(secret)
	if encoded != strings.Repeat("A", 43) {
		t.Fatal("unexpected canonical token encoding")
	}
	decoded, err := DecodeSecret(encoded)
	if err != nil || !EqualSecret(secret, decoded) {
		t.Fatal("token roundtrip failed")
	}
	for _, bad := range []string{encoded + "=", encoded[:42], strings.Repeat("!", 43), encoded[:42] + "B", encoded + "\n"} {
		if _, err := DecodeSecret(bad); !errors.Is(err, ErrInvalidCookie) {
			t.Fatal("noncanonical token accepted")
		}
	}
	digest := TokenDigest(secret)
	if hex.EncodeToString(digest[:]) != "66687aadf862bd776c8fc18b8e9f8e20089714856ee233b3902a591d0d5f2925" {
		t.Fatal("token digest is not SHA256")
	}
	changed := secret
	changed[0] = 1
	if EqualSecret(secret, changed) {
		t.Fatal("different secrets equal")
	}
	id, err := NewID(bytes.NewReader(make([]byte, 16)))
	if err != nil || id != "00000000-0000-4000-8000-000000000000" {
		t.Fatal("UUID v4 generation failed")
	}
	if _, err := NewSecret(brokenRandom{}); err == nil {
		t.Fatal("failed random accepted for token")
	}
	if _, err := NewID(brokenRandom{}); err == nil {
		t.Fatal("failed random accepted for ID")
	}
}
