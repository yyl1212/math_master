package store_test

import (
	"errors"
	"fmt"
	"github.com/yyl1212/math_master/backend/internal/auth"
	"github.com/yyl1212/math_master/backend/internal/store"
	"regexp"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestAuthRateLimits(t *testing.T) {
	f, _ := fixedRateFixture(t, time.Date(2026, 10, 1, 12, 0, 59, 999000000, time.UTC))
	for _, limit := range []int{3, 10, 30, 60, 120, 600} {
		scope := fmt.Sprintf("threshold_%d", limit)
		keys := []auth.RateKey{{Scope: scope, Limit: limit, Window: time.Hour}}
		for i := 0; i < limit; i++ {
			if err := f.repo.ConsumeRates(f.ctx, keys); err != nil {
				t.Fatal("quota rejected before limit")
			}
		}
		var limited *auth.RateLimitError
		if err := store.New(f.db).ConsumeRates(f.ctx, keys); !errors.As(err, &limited) || limited.RetryAfterSeconds < 1 || limited.RetryAfterSeconds > 3600 {
			t.Fatal("quota lost on repository restart")
		}
	}
	var successes atomic.Int32
	var wg sync.WaitGroup
	keys := []auth.RateKey{{Scope: "concurrent", Limit: 10, Window: time.Hour}}
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if f.repo.ConsumeRates(f.ctx, keys) == nil {
				successes.Add(1)
			}
		}()
	}
	wg.Wait()
	if successes.Load() != 10 {
		t.Fatal("quota over- or under-admitted concurrency")
	}
	for i := 0; i < 102; i++ {
		err := f.repo.ConsumeRates(f.ctx, []auth.RateKey{{Scope: "bounded", Limit: 2, Window: time.Hour}, {Scope: "bounded_name", Key: fmt.Sprintf("name%d", i), Limit: 10, Window: time.Hour}})
		if i < 2 && err != nil {
			t.Fatal("global quota rejected early")
		}
		if i >= 2 && err == nil {
			t.Fatal("global quota exceeded")
		}
	}
	if f.count("SELECT count(*) FROM auth_rate_limits WHERE scope IN ('bounded','bounded_name')") != 3 || f.count("SELECT attempts FROM auth_rate_limits WHERE scope='bounded'") != 3 {
		t.Fatal("rejected usernames grew rate keys or count overflowed")
	}
	keys = []auth.RateKey{{Scope: "multi", Limit: 100, Window: time.Hour}, {Scope: "multi_short", Key: "same", Limit: 1, Window: time.Minute}, {Scope: "multi_long", Key: "same", Limit: 1, Window: 10 * time.Minute}}
	if err := f.repo.ConsumeRates(f.ctx, keys); err != nil {
		t.Fatal("multiple quotas initial request failed")
	}
	var limited *auth.RateLimitError
	if err := f.repo.ConsumeRates(f.ctx, keys); !errors.As(err, &limited) {
		t.Fatal("multiple quota rejection failed")
	}
	var seconds float64
	if f.db.QueryRowContext(f.ctx, "SELECT extract(epoch from max(window_end)-clock_timestamp()) FROM auth_rate_limits WHERE scope='multi_long'").Scan(&seconds) != nil || float64(limited.RetryAfterSeconds) < seconds {
		t.Fatal("largest retry window missing")
	}
}
func TestAuthCleanup(t *testing.T) {
	f := newAuthFixture(t)
	user, cookies, _ := f.signup("cleanup_user")
	h := tokenHash(t, cookies.Session)
	f.exec("UPDATE auth_sessions SET created_at=clock_timestamp()-interval '3 days',last_seen_at=clock_timestamp()-interval '26 hours',absolute_expires_at=clock_timestamp()-interval '25 hours' WHERE token_hash=$1", h[:])
	f.exec("INSERT INTO auth_preauth(token_hash,csrf,created_at,expires_at) SELECT decode(md5(i::text)||md5('context'||i::text),'hex'),decode(repeat('00',32),'hex'),clock_timestamp()-interval '26 hours',clock_timestamp()-interval '25 hours' FROM generate_series(1,2001)i")
	f.exec("INSERT INTO auth_rate_limits(scope,key,window_start,window_end,attempts) VALUES('expired','',clock_timestamp()-interval '26 hours',clock_timestamp()-interval '25 hours',1)")
	anon, _ := f.anonymous()
	validHash := tokenHash(t, anon.Preauth)
	before := f.count("SELECT count(*) FROM auth_audit_events")
	removed, err := f.service.Cleanup(f.ctx)
	if err != nil || removed != 2000 {
		t.Fatal("cleanup exceeded shared batch budget or failed")
	}
	if f.count("SELECT count(*) FROM auth_preauth WHERE token_hash=$1", validHash[:]) != 1 || f.count("SELECT count(*) FROM auth_audit_events") != before || f.count("SELECT count(*) FROM auth_users WHERE id=$1", user.ID) != 1 {
		t.Fatal("cleanup removed active or permanent data")
	}
	if _, err = f.service.Cleanup(f.ctx); err != nil {
		t.Fatal("second cleanup failed")
	}
	if f.count("SELECT count(*) FROM auth_preauth WHERE expires_at<clock_timestamp()-interval '24 hours'") != 0 {
		t.Fatal("expired preauth retained after batches")
	}
	// Keep a newly expired session within its 24h diagnostic retention window.
	cookies, _ = f.login(user.Username, accountTestPassword)
	h = tokenHash(t, cookies.Session)
	f.exec("UPDATE auth_sessions SET last_seen_at=clock_timestamp()-interval '31 minutes' WHERE token_hash=$1", h[:])
	if _, err = f.service.Cleanup(f.ctx); err != nil || f.count("SELECT count(*) FROM auth_sessions WHERE token_hash=$1", h[:]) != 1 {
		t.Fatal("cleanup removed a recently expired session")
	}
	// A cleanup failure cannot restore an expired session: reads validate expiry independently.
	f.exec("CREATE FUNCTION fail_cleanup() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'forced cleanup failure'; END $$")
	f.exec("CREATE TRIGGER force_cleanup_failure BEFORE DELETE ON auth_preauth FOR EACH ROW EXECUTE FUNCTION fail_cleanup()")
	f.exec("INSERT INTO auth_preauth(token_hash,csrf,created_at,expires_at) VALUES(decode(repeat('ff',32),'hex'),decode(repeat('00',32),'hex'),clock_timestamp()-interval '26 hours',clock_timestamp()-interval '25 hours')")
	if _, err = f.service.Cleanup(f.ctx); !errors.Is(err, auth.ErrUnavailable) {
		t.Fatal("cleanup failure hidden")
	}
	if u, err := f.service.Session(f.ctx, cookies); err != nil || u != nil {
		t.Fatal("cleanup fault restored invalid session")
	}
}

func TestAuthRateLimitsServicePolicies(t *testing.T) {
	t.Run("loginUsernameAndPreauth", func(t *testing.T) {
		f, _ := fixedRateFixture(t, time.Date(2026, 10, 1, 12, 0, 59, 999000000, time.UTC))
		cookies, csrf := f.anonymous()
		for i := 0; i < 11; i++ {
			_, _, err := f.service.Login(f.ctx, cookies, csrf, auth.LoginInput{Username: "UNKNOWN_NAME", Password: accountTestPassword}, "attempt")
			if i < 10 && !errors.Is(err, auth.ErrInvalidCredentials) {
				t.Fatal("credential attempt prematurely blocked")
			}
			if i == 10 {
				var limited *auth.RateLimitError
				if !errors.As(err, &limited) {
					t.Fatal("eleventh login accepted")
				}
			}
		}
	})
	t.Run("registerPreauth", func(t *testing.T) {
		f, _ := fixedRateFixture(t, time.Date(2026, 10, 1, 12, 0, 59, 999000000, time.UTC))
		f.register("taken_name")
		cookies, csrf := f.anonymous()
		for i := 0; i < 4; i++ {
			_, _, err := f.service.Register(f.ctx, cookies, csrf, auth.RegisterInput{Username: "TAKEN_NAME", Password: accountTestPassword}, "attempt")
			if i < 3 && !errors.Is(err, auth.ErrUsernameUnavailable) {
				t.Fatal("registration attempt prematurely blocked")
			}
			if i == 3 {
				var limited *auth.RateLimitError
				if !errors.As(err, &limited) {
					t.Fatal("fourth registration accepted")
				}
			}
		}
	})
	t.Run("sharedRead", func(t *testing.T) {
		f, _ := fixedRateFixture(t, time.Date(2026, 10, 1, 12, 0, 59, 999000000, time.UTC))
		for i := 0; i < 599; i++ {
			if f.repo.ConsumeRates(f.ctx, []auth.RateKey{{Scope: "auth_read", Limit: 600, Window: time.Minute}}) != nil {
				t.Fatal("read budget setup failed")
			}
		}
		if user, err := f.service.Session(f.ctx, auth.Cookies{}); err != nil || user != nil {
			t.Fatal("last session read failed")
		}
		_, delta, err := f.service.Context(f.ctx, auth.Cookies{})
		var limited *auth.RateLimitError
		if !errors.As(err, &limited) || delta != (auth.CookieDelta{}) {
			t.Fatal("session and context did not share read limit")
		}
	})
	t.Run("newPreauth", func(t *testing.T) {
		f, _ := fixedRateFixture(t, time.Date(2026, 10, 1, 12, 0, 59, 999000000, time.UTC))
		for i := 0; i < 119; i++ {
			if f.repo.ConsumeRates(f.ctx, []auth.RateKey{{Scope: "new_preauth", Limit: 120, Window: time.Minute}}) != nil {
				t.Fatal("context budget setup failed")
			}
		}
		f.anonymous()
		_, delta, err := f.service.Context(f.ctx, auth.Cookies{})
		var limited *auth.RateLimitError
		if !errors.As(err, &limited) || delta != (auth.CookieDelta{}) || f.count("SELECT count(*) FROM auth_preauth") != 1 {
			t.Fatal("anonymous creation global budget failed")
		}
	})
	t.Run("sharedPasswordReauth", func(t *testing.T) {
		f, _ := fixedRateFixture(t, time.Date(2026, 10, 1, 12, 0, 59, 999000000, time.UTC))
		_, cookies, csrf := f.signup("limited_password")
		for i := 0; i < 11; i++ {
			var err error
			if i%2 == 0 {
				_, err = f.service.Reauthenticate(f.ctx, cookies, csrf, auth.ReauthInput{Password: changedTestPassword}, "wrong")
			} else {
				_, err = f.service.ChangePassword(f.ctx, cookies, csrf, auth.PasswordInput{CurrentPassword: changedTestPassword, NewPassword: accountTestPassword}, "wrong")
			}
			if i < 10 && !errors.Is(err, auth.ErrInvalidCredentials) {
				t.Fatal("password verification prematurely blocked")
			}
			if i == 10 {
				var limited *auth.RateLimitError
				if !errors.As(err, &limited) {
					t.Fatal("password and reauth did not share limit")
				}
			}
		}
	})
}

// 只改变随机测试库的时钟；所有新连接使用相同 search_path，保留真实多连接竞争。
func fixedRateFixture(t *testing.T, at time.Time) (*authFixture, func(time.Time)) {
	t.Helper()
	f := newAuthFixture(t)
	var database string
	if err := f.db.QueryRowContext(f.ctx, "SELECT current_database()").Scan(&database); err != nil ||
		!regexp.MustCompile(`^math_master_test_[0-9a-f]{16}$`).MatchString(database) {
		t.Fatal("fixed clock requires an isolated random test database")
	}
	f.exec("CREATE TABLE auth_rate_test_clock (singleton boolean PRIMARY KEY CHECK(singleton), current_at timestamptz NOT NULL)")
	f.exec("INSERT INTO auth_rate_test_clock VALUES(true,$1)", at)
	f.exec("CREATE FUNCTION public.clock_timestamp() RETURNS timestamptz LANGUAGE SQL STABLE AS 'SELECT current_at FROM public.auth_rate_test_clock WHERE singleton'")
	// 名称已验证为严格的随机测试库格式，数据库设置不会影响角色或其他数据库。
	f.exec(fmt.Sprintf(`ALTER DATABASE "%s" SET search_path TO public, pg_catalog`, database))
	// 回收默认 search_path 的原连接；以后每条物理连接均读取数据库默认值。
	f.db.SetMaxIdleConns(0)
	f.db.SetMaxIdleConns(2)
	return f, func(next time.Time) {
		t.Helper()
		f.exec("UPDATE public.auth_rate_test_clock SET current_at=$1 WHERE singleton", next)
	}
}

func TestAuthRateClockFixture(t *testing.T) {
	at := time.Date(2026, 10, 1, 12, 0, 59, 999000000, time.UTC)
	f, advance := fixedRateFixture(t, at)
	checkClock := func(want time.Time) {
		t.Helper()
		var got time.Time
		if err := f.db.QueryRowContext(f.ctx, "SELECT clock_timestamp()").Scan(&got); err != nil {
			t.Fatal("test database clock query failed")
		}
		if !got.Equal(want) {
			t.Fatal("test database clock is not fixed at the requested instant")
		}
	}
	checkClock(at)
	// 重新建立物理连接后也必须使用同一测试时钟。
	f.db.SetMaxIdleConns(0)
	f.db.SetMaxIdleConns(2)
	checkClock(at)
	at = at.Add(time.Millisecond)
	advance(at)
	checkClock(at)
}

func TestAuthRateLimitFixedWindowBoundary(t *testing.T) {
	at := time.Date(2026, 10, 1, 12, 0, 59, 999000000, time.UTC)
	f, advance := fixedRateFixture(t, at)
	cookies, csrf := f.anonymous()
	attempt := func() error {
		_, _, err := f.service.Login(f.ctx, cookies, csrf, auth.LoginInput{Username: "UNKNOWN_NAME", Password: accountTestPassword}, "fixed-window")
		return err
	}
	for i := 0; i < 10; i++ {
		if !errors.Is(attempt(), auth.ErrInvalidCredentials) {
			t.Fatal("same-window login blocked before its quota")
		}
	}
	var limited *auth.RateLimitError
	if !errors.As(attempt(), &limited) || limited.RetryAfterSeconds != 1 {
		t.Fatal("same-window eleventh login must be limited until the next minute")
	}
	advance(at.Add(time.Millisecond))
	if !errors.Is(attempt(), auth.ErrInvalidCredentials) {
		t.Fatal("login quota did not reset at the next UTC minute")
	}
	if f.count("SELECT count(*) FROM auth_rate_limits WHERE scope='login_username'") != 2 ||
		f.count("SELECT attempts FROM auth_rate_limits WHERE scope='login_username' AND window_start=$1", at.Truncate(time.Minute)) != 11 ||
		f.count("SELECT attempts FROM auth_rate_limits WHERE scope='login_username' AND window_start=$1", at.Truncate(time.Minute).Add(time.Minute)) != 1 {
		t.Fatal("fixed windows did not retain their independent saturated counts")
	}
}
