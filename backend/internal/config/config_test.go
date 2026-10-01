package config

import (
	"strings"
	"testing"
	"time"
)

func TestLoadRejectsMissingDatabaseURL(t *testing.T) {
	t.Setenv("DATABASE_URL", "")
	if _, err := Load(); err == nil {
		t.Fatal("missing database configuration must fail")
	}
}

func TestLoadDoesNotExposeSecret(t *testing.T) {
	for _, raw := range []string{"invalid:test-secret", "http://user:test-secret@localhost/db"} {
		t.Setenv("DATABASE_URL", raw)
		_, err := Load()
		if err == nil {
			t.Fatal("invalid database configuration accepted")
		}
		if strings.Contains(err.Error(), "test-secret") {
			t.Fatal("error exposed database credentials")
		}
	}
}

func TestLoadDefaultsAndRejectsInvalidLimits(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://learner:fixture@127.0.0.1/math_master_test_config")
	t.Setenv("HTTP_ADDR", "")
	t.Setenv("SHUTDOWN_TIMEOUT", "")
	c, err := Load()
	if err != nil || c.HTTPAddr != "127.0.0.1:8080" || c.ShutdownTimeout != 5*time.Second {
		t.Fatal("valid configuration defaults unavailable")
	}
	for _, value := range []string{"-1s", "0s", "invalid"} {
		t.Setenv("SHUTDOWN_TIMEOUT", value)
		if _, err := Load(); err == nil {
			t.Fatal("invalid shutdown timeout accepted")
		}
	}
}

func TestAuthConfigCompatibility(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://test:fixture@127.0.0.1/math_master_test_config")
	t.Setenv("HTTP_ADDR", "127.0.0.1:8080")
	t.Setenv("SHUTDOWN_TIMEOUT", "")
	t.Setenv("APP_ENV", "")
	t.Setenv("AUTH_PUBLIC_ORIGIN", "")
	c, err := Load()
	if err != nil || c.AppEnv != "development" || c.PublicOrigin != "" {
		t.Fatal("legacy read-only config failed")
	}
	for _, tt := range []struct{ env, origin, addr string }{{"production", "", "127.0.0.1:8080"}, {"production", "http://math.example", "127.0.0.1:8080"}, {"development", "http://math.example", "127.0.0.1:8080"}, {"development", "http://localhost:3000", "0.0.0.0:8080"}, {"development", "http://user:pass@localhost:3000", "127.0.0.1:8080"}, {"development", "http://localhost:3000/", "127.0.0.1:8080"}, {"development", "http://localhost:3000?x=1", "127.0.0.1:8080"}, {"development", "http://localhost:3000#x", "127.0.0.1:8080"}, {"unknown", "http://localhost:3000", "127.0.0.1:8080"}} {
		t.Setenv("APP_ENV", tt.env)
		t.Setenv("AUTH_PUBLIC_ORIGIN", tt.origin)
		t.Setenv("HTTP_ADDR", tt.addr)
		if _, err := Load(); err == nil {
			t.Fatal("unsafe auth configuration accepted")
		}
	}
	t.Setenv("APP_ENV", "production")
	t.Setenv("AUTH_PUBLIC_ORIGIN", "https://MATH.EXAMPLE:443")
	t.Setenv("HTTP_ADDR", "0.0.0.0:8080")
	c, err = Load()
	if err != nil || c.PublicOrigin != "https://math.example" {
		t.Fatal("origin normalization failed")
	}
}

func TestAuthConfigBrowserCanonicalOrigin(t *testing.T) {
	for _, tt := range []struct{ raw, want string }{{"http://[0:0:0:0:0:0:0:1]:003000", "http://[::1]:3000"}, {"http://[::ffff:127.0.0.1]:3000", "http://[::ffff:7f00:1]:3000"}, {"https://MATH.EXAMPLE:00443", "https://math.example"}} {
		got, err := NormalizeAuthOrigin(tt.raw, strings.HasPrefix(tt.raw, "https:"))
		if err != nil || got != tt.want {
			t.Fatal("Go origin disagrees with browser canonical origin")
		}
	}
}
