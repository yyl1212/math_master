package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/yyl1212/math_master/backend/internal/store"
	"github.com/yyl1212/math_master/backend/internal/study"
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

type currentHealth struct{ schema bool }

func (currentHealth) PingContext(context.Context) error { return nil }
func (currentHealth) ReadTopicSchemaHealth(context.Context) (study.SchemaHealth, error) {
	return study.SchemaHealth{Taxonomy: true, Study: true, Retirement: true, SchemaReady: true}, nil
}
func (c currentHealth) ReadManagedSchemaHealth(context.Context) (store.ManagedSchemaHealth, error) {
	return store.ManagedSchemaHealth{Capability: true, SchemaReady: c.schema, ManagedMode: true}, nil
}
func TestManagedCapabilityReadinessPreservesTopicShape(t *testing.T) {
	for _, ready := range []bool{true, false} {
		w := httptest.NewRecorder()
		NewHealthHandler(currentHealth{ready}, currentHealth{ready}).ServeHTTP(w, httptest.NewRequest("GET", "/readyz", nil))
		var value struct {
			Topic   map[string]bool `json:"topic"`
			Content map[string]bool `json:"content"`
		}
		if json.Unmarshal(w.Body.Bytes(), &value) != nil || len(value.Topic) != 5 || len(value.Content) != 3 || value.Content["schemaReady"] != ready || w.Code != map[bool]int{true: 200, false: 503}[ready] {
			t.Fatal(w.Code, w.Body.String())
		}
	}
}
