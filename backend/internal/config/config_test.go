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
