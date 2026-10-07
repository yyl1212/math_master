package httpapi

import (
	"context"
	"github.com/yyl1212/math_master/backend/internal/auth"
	"github.com/yyl1212/math_master/backend/internal/correction"
	"github.com/yyl1212/math_master/backend/internal/learning"
	"github.com/yyl1212/math_master/backend/internal/question"
	"github.com/yyl1212/math_master/backend/internal/study"
	"github.com/yyl1212/math_master/backend/internal/taxonomy"
	"net/http"
	"time"
)

type RetirementRoute struct{ Method, Path, Module, Action string }
type ExperienceModeReader interface {
	ReadExperienceMode(context.Context) (taxonomy.ExperienceMode, error)
}

func retirementPolicy(mode taxonomy.ExperienceMode, r RetirementRoute) string {
	if mode != taxonomy.ModeTopics {
		return "allow"
	}
	if v, e := routeLearning(r.Path, r.Method); e == nil {
		if learning.IsRead(v.Action) {
			return "historical-read"
		}
		return "retired"
	}
	if v, e := routeQuestion(r.Path, r.Method); e == nil {
		if question.IsRead(v.Action) {
			return "historical-read"
		}
		return "retired"
	}
	if v, e := routeCorrection(r.Path, r.Method); e == nil {
		if v.Action == correction.RetryJobAction {
			return "allow"
		}
		if correction.IsWrite(v.Action) {
			return "retired"
		}
		return "historical-read"
	}
	return "allow"
}
func serveTopicRetirement(w http.ResponseWriter, r *http.Request, reader ExperienceModeReader) bool {
	route := RetirementRoute{Method: r.Method, Path: r.URL.Path}
	if r.URL.RawPath != "" || r.URL.ForceQuery && r.URL.RawQuery == "" {
		return false
	}
	if reader == nil || retirementPolicy(taxonomy.ModeTopics, route) != "retired" {
		return false
	}
	id := requestID()
	r = r.Clone(r.Context())
	r.Header = r.Header.Clone()
	r.Header.Set("X-Request-ID", id)
	w.Header().Set("X-Request-ID", id)
	ctx, cancel := context.WithTimeout(r.Context(), 8*time.Second)
	defer cancel()
	mode, e := reader.ReadExperienceMode(ctx)
	if e != nil || mode != taxonomy.ModeLegacy && mode != taxonomy.ModeTopics {
		privateError(w, r, auth.ErrUnavailable, auth.CookieDelta{}, false)
		return true
	}
	if retirementPolicy(mode, route) != "retired" {
		return false
	}
	r = r.Clone(ctx)
	learningError(w, r, study.ErrModuleRetired)
	return true
}
