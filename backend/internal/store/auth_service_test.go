package store_test

import (
	"errors"
	"github.com/yyl1212/math_master/backend/internal/auth"
	"testing"
	"time"
)

func TestAccountRegistration(t *testing.T) {
	f := newAuthFixture(t)
	cookies, csrf := f.anonymous()
	user, delta, err := f.service.Register(f.ctx, cookies, csrf, auth.RegisterInput{Username: "Math_User", Password: accountTestPassword}, "register")
	if err != nil || user.Username != "math_user" || len(user.Roles) != 1 || user.Roles[0] != auth.RoleLearner || delta.SetSession != "" || !delta.ClearPreauth {
		t.Fatal("registration contract failed")
	}
	if f.count("SELECT count(*) FROM auth_sessions") != 0 || f.count("SELECT count(*) FROM publication_heads") != 0 {
		t.Fatal("registration created unrelated state")
	}
	if _, _, err = f.service.Register(f.ctx, cookies, csrf, auth.RegisterInput{Username: "different_user", Password: accountTestPassword}, "replay"); !errors.Is(err, auth.ErrCSRF) {
		t.Fatal("consumed preauth reused")
	}
	fresh, proof := f.anonymous()
	if _, _, err = f.service.Register(f.ctx, fresh, proof, auth.RegisterInput{Username: "MATH_USER", Password: accountTestPassword}, "duplicate"); !errors.Is(err, auth.ErrUsernameUnavailable) {
		t.Fatal("duplicate normalized username accepted")
	}
	signedIn, signedProof := f.login("math_user", accountTestPassword)
	if _, _, err = f.service.Login(f.ctx, signedIn, signedProof, auth.LoginInput{Username: "math_user", Password: accountTestPassword}, "switch"); !errors.Is(err, auth.ErrAlreadyAuthenticated) {
		t.Fatal("signed in identity switched")
	}
	f.exec("UPDATE auth_users SET password_phc='broken' WHERE id=$1", user.ID)
	fresh, proof = f.anonymous()
	if _, _, err = f.service.Login(f.ctx, fresh, proof, auth.LoginInput{Username: "math_user", Password: accountTestPassword}, "corrupt"); !errors.Is(err, auth.ErrUnavailable) {
		t.Fatal("corrupt hash disguised as credentials")
	}
}
func TestSessionLifecycle(t *testing.T) {
	for _, tt := range []struct {
		name, change string
		valid        bool
	}{{"idle15m", "last_seen_at=clock_timestamp()-interval '15 minutes'", true}, {"idle30m", "last_seen_at=clock_timestamp()-interval '30 minutes'", false}, {"absolute7d", "created_at=clock_timestamp()-interval '8 days',absolute_expires_at=clock_timestamp()-interval '1 second'", false}, {"version", "credential_version=credential_version+1", false}, {"revoked", "revoked_at=clock_timestamp()", false}} {
		t.Run(tt.name, func(t *testing.T) {
			f := newAuthFixture(t)
			_, cookies, _ := f.signup("expiry_user")
			h := tokenHash(t, cookies.Session)
			f.exec("UPDATE auth_sessions SET "+tt.change+" WHERE token_hash=$1", h[:])
			u, err := f.service.Session(f.ctx, cookies)
			if err != nil || (u != nil) != tt.valid {
				t.Fatal("session validity failed")
			}
		})
	}
	t.Run("boundedSessionsAndLogout", func(t *testing.T) {
		f := newAuthFixture(t)
		u := f.register("many_sessions")
		all := make([]auth.Cookies, 6)
		var csrf string
		for i := range all {
			all[i], csrf = f.login(u.Username, accountTestPassword)
		}
		if f.count("SELECT count(*) FROM auth_sessions WHERE user_id=$1 AND revoked_at IS NULL", u.ID) != 5 {
			t.Fatal("sixth session did not revoke oldest")
		}
		if user, err := f.service.Session(f.ctx, all[0]); err != nil || user != nil {
			t.Fatal("oldest session survived")
		}
		if delta, err := f.service.Logout(f.ctx, all[5], csrf, true, "logout-all"); err != nil || !delta.ClearSession {
			t.Fatal("logout all failed")
		}
		for _, cookies := range all {
			if user, err := f.service.Session(f.ctx, cookies); err != nil || user != nil {
				t.Fatal("logout left valid session")
			}
		}
	})
	for _, tt := range []struct {
		age   string
		touch bool
	}{{"30 seconds", false}, {"90 seconds", true}} {
		t.Run("activity"+tt.age, func(t *testing.T) {
			f := newAuthFixture(t)
			_, cookies, _ := f.signup("activity_user")
			h := tokenHash(t, cookies.Session)
			f.exec("UPDATE auth_sessions SET last_seen_at=clock_timestamp()-$2::interval WHERE token_hash=$1", h[:], tt.age)
			var before, after time.Time
			if f.db.QueryRowContext(f.ctx, "SELECT last_seen_at FROM auth_sessions WHERE token_hash=$1", h[:]).Scan(&before) != nil {
				t.Fatal("timestamp query failed")
			}
			if _, _, err := f.service.Context(f.ctx, cookies); err != nil {
				t.Fatal("context failed")
			}
			if f.db.QueryRowContext(f.ctx, "SELECT last_seen_at FROM auth_sessions WHERE token_hash=$1", h[:]).Scan(&after) != nil {
				t.Fatal("timestamp query failed")
			}
			if after.After(before) != tt.touch {
				t.Fatal("session touch interval failed")
			}
		})
	}
}
func TestContextRecovery(t *testing.T) {
	f := newAuthFixture(t)
	cookies, csrf := f.anonymous()
	view, delta, err := f.service.Context(f.ctx, cookies)
	if err != nil || view.CSRFToken != csrf || delta.SetPreauth != "" {
		t.Fatal("valid preauth not reused")
	}
	var seconds float64
	h := tokenHash(t, cookies.Preauth)
	if f.db.QueryRowContext(f.ctx, "SELECT extract(epoch from expires_at-created_at) FROM auth_preauth WHERE token_hash=$1", h[:]).Scan(&seconds) != nil || seconds != 600 {
		t.Fatal("preauth lifetime incorrect")
	}
	_, signed, _ := f.signup("recovery_user")
	h = tokenHash(t, signed.Session)
	f.exec("UPDATE auth_sessions SET revoked_at=clock_timestamp() WHERE token_hash=$1", h[:])
	view, delta, err = f.service.Context(f.ctx, signed)
	if err != nil || view.User != nil || !delta.ClearSession || delta.SetPreauth == "" {
		t.Fatal("expired context not recovered")
	}
	f.exec("ALTER TABLE auth_sessions RENAME TO auth_sessions_unavailable")
	view, delta, err = f.service.Context(f.ctx, signed)
	if !errors.Is(err, auth.ErrUnavailable) || delta != (auth.CookieDelta{}) || view.User != nil {
		t.Fatal("DB fault changed browser identity")
	}
}
func TestPasswordAndReauth(t *testing.T) {
	f := newAuthFixture(t)
	user, cookies, csrf := f.signup("password_user")
	h := tokenHash(t, cookies.Session)
	if f.count("SELECT count(*) FROM auth_sessions WHERE token_hash=$1 AND reauthenticated_at IS NULL", h[:]) != 1 {
		t.Fatal("login granted recent verification")
	}
	if _, err := f.service.ChangePassword(f.ctx, cookies, csrf, auth.PasswordInput{CurrentPassword: "an incorrect test passphrase", NewPassword: changedTestPassword}, "wrong"); !errors.Is(err, auth.ErrInvalidCredentials) {
		t.Fatal("wrong current password accepted")
	}
	if f.count("SELECT credential_version FROM auth_users WHERE id=$1", user.ID) != 1 || f.count("SELECT count(*) FROM auth_sessions WHERE revoked_at IS NULL") != 1 {
		t.Fatal("failed password change mutated state")
	}
	until, err := f.service.Reauthenticate(f.ctx, cookies, csrf, auth.ReauthInput{Password: accountTestPassword}, "reauth")
	if err != nil {
		t.Fatal("reauth failed")
	}
	var at time.Time
	if f.db.QueryRowContext(f.ctx, "SELECT reauthenticated_at FROM auth_sessions WHERE token_hash=$1", h[:]).Scan(&at) != nil || !until.Equal(at.Add(5*time.Minute)) {
		t.Fatal("reauth window failed")
	}
	delta, err := f.service.ChangePassword(f.ctx, cookies, csrf, auth.PasswordInput{CurrentPassword: accountTestPassword, NewPassword: changedTestPassword}, "change")
	if err != nil || !delta.ClearSession {
		t.Fatal("password change failed")
	}
	if f.count("SELECT credential_version FROM auth_users WHERE id=$1", user.ID) != 2 || f.count("SELECT count(*) FROM auth_sessions WHERE revoked_at IS NULL") != 0 {
		t.Fatal("change did not revoke all credentials")
	}
	f.login(user.Username, changedTestPassword)
}
func TestAtomicAudit(t *testing.T) {
	f := newAuthFixture(t)
	cookies, csrf := f.anonymous()
	f.exec("CREATE FUNCTION fail_auth_audit() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'forced failure'; END $$")
	f.exec("CREATE TRIGGER force_auth_audit_failure BEFORE INSERT ON auth_audit_events FOR EACH ROW EXECUTE FUNCTION fail_auth_audit()")
	if _, _, err := f.service.Register(f.ctx, cookies, csrf, auth.RegisterInput{Username: "atomic_user", Password: accountTestPassword}, "atomic"); !errors.Is(err, auth.ErrUnavailable) {
		t.Fatal("audit failure not surfaced")
	}
	if f.count("SELECT count(*) FROM auth_users") != 0 || f.count("SELECT count(*) FROM auth_preauth WHERE consumed_at IS NOT NULL") != 0 {
		t.Fatal("audit failure left partial registration")
	}
	f.exec("DROP TRIGGER force_auth_audit_failure ON auth_audit_events")
	f.register("atomic_user")
	cookies, csrf = f.anonymous()
	if _, _, err := f.service.Login(f.ctx, cookies, csrf, auth.LoginInput{Username: "UNKNOWN_USER", Password: accountTestPassword}, "failed-login"); !errors.Is(err, auth.ErrInvalidCredentials) {
		t.Fatal("unknown user leaked distinction")
	}
	if f.count("SELECT count(*) FROM auth_audit_events WHERE action='login_failed' AND octet_length(username_hash)=32 AND actor_id IS NULL AND target_id IS NULL AND reason='' AND ownership_note=''") != 1 {
		t.Fatal("login failure audit leaked identity")
	}
	f.exec("CREATE TRIGGER force_auth_audit_failure BEFORE INSERT ON auth_audit_events FOR EACH ROW EXECUTE FUNCTION fail_auth_audit()")
	if _, _, err := f.service.Login(f.ctx, cookies, csrf, auth.LoginInput{Username: "atomic_user", Password: accountTestPassword}, "atomic-login"); !errors.Is(err, auth.ErrUnavailable) {
		t.Fatal("login audit failure not surfaced")
	}
	if f.count("SELECT count(*) FROM auth_sessions") != 0 {
		t.Fatal("audit failure left session")
	}
}
