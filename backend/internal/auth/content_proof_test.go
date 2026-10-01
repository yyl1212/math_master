package auth

import (
	"errors"
	"testing"
)

func TestContentProof(t *testing.T) {
	var token, csrf Secret
	token[0], csrf[0] = 1, 2
	cookies := Cookies{Session: EncodeSecret(token), Preauth: "ignored-not-an-identity"}
	got, err := DecodeContentProof(cookies, "", false)
	if err != nil || got.TokenHash != TokenDigest(token) || got.CSRF != (Secret{}) {
		t.Fatal("read proof must decode only the session")
	}
	got, err = DecodeContentProof(cookies, EncodeSecret(csrf), true)
	if err != nil || got.CSRF != csrf {
		t.Fatal("write proof must preserve the separate CSRF secret")
	}
	for _, value := range []string{"", "invalid"} {
		_, err = DecodeContentProof(Cookies{Session: value}, EncodeSecret(csrf), true)
		if value == "" && !errors.Is(err, ErrAuthenticationRequired) {
			t.Fatal("missing session accepted")
		}
		if value != "" && !errors.Is(err, ErrInvalidCookie) {
			t.Fatal("invalid session accepted")
		}
	}
	if _, err = DecodeContentProof(cookies, "", true); !errors.Is(err, ErrCSRF) {
		t.Fatal("missing write verification accepted")
	}
	if _, err = DecodeContentProof(cookies, "invalid", true); !errors.Is(err, ErrCSRF) {
		t.Fatal("invalid write verification accepted")
	}
}
