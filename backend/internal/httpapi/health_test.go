package httpapi

import (
	"context"
	"errors"
	"net/http/httptest"
	"strings"
	"testing"
)

type unavailableDB struct{}

func (unavailableDB) PingContext(context.Context) error {
	return errors.New("database password=test-secret")
}

func TestHealthSeparatesLivenessAndReadiness(t *testing.T) {
	h := NewHealthHandler(unavailableDB{})
	for _, tc := range []struct {
		path string
		want int
	}{{"/healthz", 200}, {"/readyz", 503}} {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest("GET", tc.path, nil))
		if w.Code != tc.want {
			t.Errorf("%s: got %d, want %d", tc.path, w.Code, tc.want)
		}
		if strings.Contains(w.Body.String(), "test-secret") {
			t.Fatal("health response exposed credentials")
		}
	}
}
