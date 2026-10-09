package httpapi

import (
	"context"
	"encoding/json"
	"github.com/yyl1212/math_master/backend/internal/store"
	"github.com/yyl1212/math_master/backend/internal/study"
	"net/http"
	"time"
)

type Pinger interface{ PingContext(context.Context) error }

type TopicHealthReader interface {
	ReadTopicSchemaHealth(context.Context) (study.SchemaHealth, error)
}

type ManagedHealthReader interface {
	ReadManagedSchemaHealth(context.Context) (store.ManagedSchemaHealth, error)
}

func NewHealthHandler(pinger Pinger, topics ...TopicHealthReader) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	})
	mux.HandleFunc("GET /readyz", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), time.Second)
		defer cancel()
		w.Header().Set("Content-Type", "application/json")
		if pinger == nil || pinger.PingContext(ctx) != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			_ = json.NewEncoder(w).Encode(map[string]string{"status": "unavailable"})
			return
		}
		payload := map[string]any{"status": "ready"}
		ready := true
		if len(topics) > 0 && topics[0] != nil {
			health, e := topics[0].ReadTopicSchemaHealth(ctx)
			payload["topic"] = health
			ready = e == nil && health.SchemaReady
			if current, ok := topics[0].(ManagedHealthReader); ok {
				h, e := current.ReadManagedSchemaHealth(ctx)
				payload["content"] = h
				ready = ready && e == nil && h.SchemaReady
			}
		}
		if !ready {
			payload["status"] = "unavailable"
			w.WriteHeader(http.StatusServiceUnavailable)
		}
		_ = json.NewEncoder(w).Encode(payload)
	})
	return mux
}
