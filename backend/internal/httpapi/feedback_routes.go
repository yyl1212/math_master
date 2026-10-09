package httpapi

import (
	"context"
	"github.com/yyl1212/math_master/backend/internal/auth"
	"github.com/yyl1212/math_master/backend/internal/feedback"
	"github.com/yyl1212/math_master/backend/internal/question"
	"mime"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type FeedbackOptions struct {
	Service      *feedback.Service
	PublicOrigin string
	Production   bool
}
type feedbackRoute struct {
	Action           feedback.Action
	ID, Kind, Method string
	Review           bool
}

func routeFeedback(path, method string) (feedbackRoute, error) {
	r := feedbackRoute{Method: "GET"}
	if !strings.HasPrefix(path, "/api/v1/feedback/") {
		return r, auth.ErrNotFound
	}
	p := strings.Split(strings.TrimPrefix(path, "/api/v1/feedback/"), "/")
	for _, s := range p {
		if s == "" {
			return r, auth.ErrNotFound
		}
	}
	switch {
	case len(p) == 2 && p[0] == "contexts" && p[1] == "site":
		r.Action = feedback.ReadContextAction
		r.Kind = "site"
	case len(p) == 3 && p[0] == "contexts" && (p[1] == "knowledge" || p[1] == "path" || p[1] == "managed-knowledge" || p[1] == "practice" || p[1] == "assessment"):
		r.Action = feedback.ReadContextAction
		r.Kind = p[1]
		r.ID = p[2]
	case len(p) == 1 && p[0] == "tickets":
		r.Action = feedback.ListOwnAction
		if method == "POST" {
			r.Action = feedback.CreateAction
			r.Method = "POST"
		} else if method != "GET" {
			r.Method = "GET, POST"
		}
	case len(p) == 2 && p[0] == "tickets":
		r.Action = feedback.ReadOwnAction
		r.ID = p[1]
	case len(p) == 3 && p[0] == "tickets" && p[2] == "events":
		r.Action = feedback.DiscussOwnAction
		r.ID = p[1]
	case len(p) == 3 && p[0] == "tickets" && p[2] == "reply":
		r.Action = feedback.ReplyAction
		r.ID = p[1]
		r.Method = "POST"
	case len(p) == 2 && p[0] == "review" && p[1] == "tickets":
		r.Action = feedback.ListReviewAction
		r.Review = true
	case len(p) == 3 && p[0] == "review" && p[1] == "tickets":
		r.Action = feedback.ReadReviewAction
		r.Review = true
		r.ID = p[2]
	case len(p) == 4 && p[0] == "review" && p[1] == "tickets" && p[3] == "events":
		r.Action = feedback.DiscussReviewAction
		r.Review = true
		r.ID = p[2]
	case len(p) == 4 && p[0] == "review" && p[1] == "tickets" && p[3] == "transition":
		r.Action = feedback.TransitionAction
		r.Review = true
		r.ID = p[2]
		r.Method = "POST"
	default:
		return r, auth.ErrNotFound
	}
	if r.ID != "" {
		valid := question.ValidID(r.ID)
		if r.Kind == "knowledge" || r.Kind == "path" || r.Kind == "managed-knowledge" {
			valid = question.ValidMathID(r.ID)
		}
		if !valid {
			return r, auth.ErrInvalidInput
		}
	}
	if method != r.Method {
		return r, errPrivateMethod
	}
	return r, nil
}
func feedbackQuery(raw string, r feedbackRoute) (feedback.ListQuery, feedback.ContextQuery, error) {
	q := feedback.ListQuery{}
	c := feedback.ContextQuery{Kind: r.Kind, ID: r.ID}
	bad := func() (feedback.ListQuery, feedback.ContextQuery, error) { return q, c, auth.ErrInvalidInput }
	values, e := url.ParseQuery(raw)
	if e != nil {
		return bad()
	}
	list := r.Action == feedback.ListOwnAction || r.Action == feedback.ListReviewAction || r.Action == feedback.DiscussOwnAction || r.Action == feedback.DiscussReviewAction
	original := map[string]string{}
	if raw != "" {
		for _, pair := range strings.Split(raw, "&") {
			parts := strings.Split(pair, "=")
			if len(parts) != 2 || parts[0] == "" {
				return bad()
			}
			original[parts[0]] = parts[1]
		}
	}
	for k, v := range values {
		if len(v) != 1 || v[0] == "" {
			return bad()
		}
		s := v[0]
		switch k {
		case "area":
			if r.Kind != "site" || !feedback.ValidArea(feedback.Area(s)) {
				return bad()
			}
			c.Area = feedback.Area(s)
		case "partKind":
			if (r.Kind != "knowledge" && r.Kind != "practice" && r.Kind != "assessment") || (s != "asset" && (s != "unit" || r.Kind != "knowledge")) {
				return bad()
			}
			c.PartKind = s
		case "partId":
			if (r.Kind != "knowledge" && r.Kind != "practice" && r.Kind != "assessment") || !question.ValidMathID(s) {
				return bad()
			}
			c.PartID = s
		case "position", "limit":
			if original[k] != s || s[0] < '1' || s[0] > '9' {
				return bad()
			}
			for _, ch := range s {
				if ch < '0' || ch > '9' {
					return bad()
				}
			}
			n, e := strconv.Atoi(s)
			if e != nil {
				return bad()
			}
			if k == "position" {
				if r.Kind != "assessment" || n < 1 || n > 5 {
					return bad()
				}
				c.Position = n
			} else {
				if !list || n < 1 || n > 50 {
					return bad()
				}
				q.Limit = n
			}
		case "cursor":
			if !list || len(s) > 512 {
				return bad()
			}
			q.Cursor = s
		case "status":
			if r.Action != feedback.ListReviewAction || !feedback.ValidStatus(feedback.Status(s)) {
				return bad()
			}
			status := feedback.Status(s)
			q.Status = &status
		case "category":
			if r.Action != feedback.ListReviewAction || !feedback.ValidCategory(feedback.Category(s)) {
				return bad()
			}
			cat := feedback.Category(s)
			q.Category = &cat
		default:
			return bad()
		}
	}
	if (r.Kind == "site" && c.Area == "") || (r.Kind == "assessment" && c.Position == 0) || (c.PartKind == "") != (c.PartID == "") {
		return bad()
	}
	return q, c, nil
}
func serveFeedback(w http.ResponseWriter, r *http.Request, o FeedbackOptions) {
	id := requestID()
	privateHeaders(w)
	w.Header().Set("X-Request-ID", id)
	ctx, cancel := context.WithTimeout(r.Context(), 8*time.Second)
	defer cancel()
	r = r.Clone(ctx)
	r.Header = r.Header.Clone()
	r.Header.Set("X-Request-ID", id)
	_ = http.NewResponseController(w).SetReadDeadline(time.Now().Add(8 * time.Second))
	fail := func(e error) {
		if ctx.Err() != nil {
			e = auth.ErrUnavailable
		}
		feedbackError(w, r, e)
	}
	if r.URL.RawPath != "" || r.URL.ForceQuery && r.URL.RawQuery == "" {
		fail(auth.ErrInvalidInput)
		return
	}
	route, e := routeFeedback(r.URL.Path, r.Method)
	if e != nil {
		if e == errPrivateMethod {
			w.Header().Set("Allow", route.Method)
		}
		fail(e)
		return
	}
	q, c, e := feedbackQuery(r.URL.RawQuery, route)
	if e != nil {
		fail(e)
		return
	}
	if o.Service == nil || o.PublicOrigin == "" {
		fail(feedback.ErrNotConfigured)
		return
	}
	write := feedback.IsWrite(route.Action)
	if len(r.Header.Values("Origin")) > 1 || len(r.Header.Values("X-CSRF-Token")) > 1 || r.Header.Get("Sec-Fetch-Site") == "cross-site" || write && r.Header.Get("Origin") != o.PublicOrigin || !write && r.Header.Get("Origin") != "" && r.Header.Get("Origin") != o.PublicOrigin {
		fail(auth.ErrCSRF)
		return
	}
	if write {
		typ, params, e := mime.ParseMediaType(r.Header.Get("Content-Type"))
		if e != nil || typ != "application/json" || len(r.Header.Values("Content-Type")) != 1 {
			fail(auth.ErrInvalidInput)
			return
		}
		for k, v := range params {
			if k != "charset" || !strings.EqualFold(v, "utf-8") {
				fail(auth.ErrInvalidInput)
				return
			}
		}
	} else if r.ContentLength != 0 || len(r.TransferEncoding) > 0 {
		fail(auth.ErrInvalidInput)
		return
	}
	cookies, _, e := privateCookies(r, o.Production)
	if e != nil {
		fail(e)
		return
	}
	proof, e := auth.DecodeContentProof(cookies, r.Header.Get("X-CSRF-Token"), write)
	if e != nil {
		fail(e)
		return
	}
	a := question.Access{TokenHash: proof.TokenHash, CSRF: proof.CSRF, RequestID: id}
	if write {
		if len(r.Header.Values("Idempotency-Key")) != 1 || !question.ValidID(r.Header.Get("Idempotency-Key")) {
			fail(auth.ErrInvalidInput)
			return
		}
		a.IdempotencyKey = r.Header.Get("Idempotency-Key")
	} else if len(r.Header.Values("Idempotency-Key")) > 1 {
		fail(auth.ErrInvalidInput)
		return
	}
	if _, e = o.Service.Preflight(ctx, a, route.Action); e != nil {
		fail(e)
		return
	}
	if write && r.ContentLength > feedback.MaxRequestBytes {
		fail(auth.ErrInvalidInput)
		return
	}
	v, status, e := dispatchFeedback(ctx, r, route, a, q, c, o.Service)
	if e != nil {
		fail(e)
		return
	}
	if ctx.Err() != nil {
		fail(auth.ErrUnavailable)
		return
	}
	feedbackResponse(w, r, status, v)
}
