package httpapi

import (
	"context"
	"errors"
	"github.com/yyl1212/math_master/backend/internal/auth"
	"github.com/yyl1212/math_master/backend/internal/study"
	"github.com/yyl1212/math_master/backend/internal/studyreview"
	"mime"
	"net/http"
	"strings"
	"time"
)

type studyReviewRoute struct{ kind, id, method string }

func isStudyReviewPath(path string) bool {
	return path == "/api/v3/study/review" || strings.HasPrefix(path, "/api/v3/study/review/")
}
func routeStudyReview(path string) (out studyReviewRoute, e error) {
	out.method = "GET"
	if !strings.HasPrefix(path, "/api/v3/study/review/") {
		return out, studyreview.ErrNotFound
	}
	parts := strings.Split(strings.TrimPrefix(path, "/api/v3/study/review/"), "/")
	if len(parts) == 1 {
		switch parts[0] {
		case "health", "overview", "queue", "history":
			out.kind = parts[0]
			return
		case "sessions":
			out.kind = "start"
			out.method = "POST"
			return
		}
	}
	if len(parts) >= 2 && len(parts) <= 3 && parts[0] == "sessions" {
		if !study.ValidID(parts[1]) {
			return out, errReviewRequest
		}
		out.id = parts[1]
		out.kind = "session"
		if len(parts) == 2 {
			return
		}
		out.kind = parts[2]
		out.method = "POST"
		switch out.kind {
		case "draft":
			out.method = "PUT"
		case "compare", "grade", "abandon":
		default:
			return out, studyreview.ErrNotFound
		}
		return
	}
	return out, studyreview.ErrNotFound
}
func serveStudyReview(w http.ResponseWriter, r *http.Request, o KnowledgeOptions) {
	id := requestID()
	w.Header().Set("X-Request-ID", id)
	ctx, cancel := context.WithTimeout(r.Context(), 8*time.Second)
	defer cancel()
	r = r.Clone(ctx)
	r.Header = r.Header.Clone()
	r.Header.Set("X-Request-ID", id)
	_ = http.NewResponseController(w).SetReadDeadline(time.Now().Add(8 * time.Second))
	if r.URL.RawPath != "" || r.URL.ForceQuery || r.URL.Fragment != "" {
		studyReviewHTTPError(w, r, errReviewRequest)
		return
	}
	route, e := routeStudyReview(r.URL.Path)
	if e != nil {
		studyReviewHTTPError(w, r, e)
		return
	}
	if r.Method != route.method {
		w.Header().Set("Allow", route.method)
		studyReviewHTTPError(w, r, errPrivateMethod)
		return
	}
	if r.Method == "GET" && (r.ContentLength != 0 || len(r.TransferEncoding) > 0) {
		studyReviewHTTPError(w, r, errReviewRequest)
		return
	}
	rawQuery, numbered, pageErr := numberedQuery(r.URL.RawQuery, route.kind == "history" || route.kind == "queue", 20, 50)
	if pageErr != nil {
		studyReviewHTTPError(w, r, errReviewRequest)
		return
	}
	q, h, e := reviewQuery(rawQuery, route.kind)
	if numbered.Enabled && route.kind == "queue" {
		if numbered.Cursor != "" {
			studyReviewHTTPError(w, r, errReviewRequest)
			return
		}
		q.Numbered = &numbered
	}
	if e != nil {
		studyReviewHTTPError(w, r, e)
		return
	}
	repo := o.Reviews
	if repo == nil && o.Service != nil {
		repo, _ = o.Service.Repository.(studyreview.Repository)
	}
	if repo == nil {
		if route.kind == "health" {
			studyReviewResponse(w, r, studyreview.Health{Capability: true}, 503, false)
		} else {
			studyReviewHTTPError(w, r, studyreview.ErrNotConfigured)
		}
		return
	}
	private := route.kind != "health"
	write := r.Method != "GET"
	var a studyreview.Access
	if private {
		if o.PublicOrigin == "" {
			studyReviewHTTPError(w, r, studyreview.ErrNotConfigured)
			return
		}
		cookies, _, err := privateCookies(r, o.Production)
		if err != nil {
			studyReviewHTTPError(w, r, err)
			return
		}
		proof, err := auth.DecodeContentProof(cookies, r.Header.Get("X-CSRF-Token"), write)
		if err != nil {
			studyReviewHTTPError(w, r, err)
			return
		}
		if len(r.Header.Values("Origin")) > 1 || len(r.Header.Values("X-CSRF-Token")) > 1 || r.Header.Get("Sec-Fetch-Site") == "cross-site" || write && r.Header.Get("Origin") != o.PublicOrigin || !write && r.Header.Get("Origin") != "" && r.Header.Get("Origin") != o.PublicOrigin {
			studyReviewHTTPError(w, r, auth.ErrCSRF)
			return
		}
		a = studyreview.Access{TokenHash: proof.TokenHash, CSRF: proof.CSRF, RequestID: id}
		if write {
			if len(r.Header.Values("Idempotency-Key")) != 1 || strings.TrimSpace(r.Header.Get("Idempotency-Key")) == "" || len(r.Header.Get("Idempotency-Key")) > 128 {
				studyReviewHTTPError(w, r, errReviewRequest)
				return
			}
			a.IdempotencyKey = r.Header.Get("Idempotency-Key")
			typ, params, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
			if err != nil || typ != "application/json" || len(r.Header.Values("Content-Type")) != 1 {
				studyReviewHTTPError(w, r, errReviewRequest)
				return
			}
			for k, v := range params {
				if k != "charset" || !strings.EqualFold(v, "utf-8") {
					studyReviewHTTPError(w, r, errReviewRequest)
					return
				}
			}
		}
	}
	var value any
	switch route.kind {
	case "health":
		health, err := repo.ReadReviewHealth(ctx)
		status := 200
		if err != nil || !health.SchemaReady {
			status = 503
			health = studyreview.Health{Capability: true}
		}
		studyReviewResponse(w, r, health, status, false)
		return
	case "overview":
		value, e = repo.ReadReviewOverview(ctx, a)
	case "queue":
		value, e = repo.ListReviewQueue(ctx, a, q)
	case "history":
		if numbered.Enabled {
			if paged, ok := repo.(studyreview.NumberedRepository); ok {
				value, e = paged.ListReviewHistoryPage(ctx, a, h, numbered)
			} else {
				e = studyreview.ErrNotConfigured
			}
		} else {
			value, e = repo.ListReviewHistory(ctx, a, h)
		}
	case "session":
		value, e = repo.ReadReviewSession(ctx, a, route.id)
	case "start":
		var in studyreview.StartInput
		if e = decodeStudyReview(r, &in); e == nil {
			value, e = repo.StartReviewSession(ctx, a, in)
		}
	case "draft":
		var in studyreview.DraftInput
		if e = decodeStudyReview(r, &in); e == nil {
			value, e = repo.SaveReviewDraft(ctx, a, route.id, in)
		}
	case "compare":
		var in studyreview.CompareInput
		if e = decodeStudyReview(r, &in); e == nil {
			value, e = repo.CompareReviewSession(ctx, a, route.id, in)
		}
	case "grade":
		var in studyreview.GradeInput
		if e = decodeStudyReview(r, &in); e == nil {
			value, e = repo.GradeReviewSession(ctx, a, route.id, in)
		}
	case "abandon":
		var in studyreview.AbandonInput
		if e = decodeStudyReview(r, &in); e == nil {
			value, e = repo.AbandonReviewSession(ctx, a, route.id, in)
		}
	}
	if e != nil {
		if r.Method == "GET" && errors.Is(e, studyreview.ErrInvalid) {
			e = errReviewRequest
		}
		studyReviewHTTPError(w, r, e)
		return
	}
	studyReviewResponse(w, r, value, 200, private)
}
