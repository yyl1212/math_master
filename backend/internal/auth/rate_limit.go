package auth

import (
	"encoding/hex"
	"time"
)

type RateKey struct {
	Scope, Key string
	Limit      int
	Window     time.Duration
}

func globalRate(scope string, limit int) RateKey {
	return RateKey{Scope: scope, Limit: limit, Window: time.Minute}
}
func digestKey(digest Digest) string { return hex.EncodeToString(digest[:]) }
func contextRate() []RateKey         { return []RateKey{globalRate("auth_read", 600)} }
func loginRates(username Digest, preauth Digest) []RateKey {
	return []RateKey{globalRate("login", 120), {Scope: "login_username", Key: digestKey(username), Limit: 10, Window: time.Minute}, {Scope: "login_preauth", Key: digestKey(preauth), Limit: 10, Window: time.Minute}}
}
func registerRates(preauth Digest) []RateKey {
	return []RateKey{globalRate("register", 30), {Scope: "register_preauth", Key: digestKey(preauth), Limit: 3, Window: 10 * time.Minute}}
}
func passwordRates(session Digest) []RateKey {
	return []RateKey{globalRate("password", 120), {Scope: "password_session", Key: digestKey(session), Limit: 10, Window: time.Minute}}
}
