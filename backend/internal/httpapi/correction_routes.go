package httpapi

import (
	"context"
	"github.com/yyl1212/math_master/backend/internal/auth"
	"github.com/yyl1212/math_master/backend/internal/correction"
	"github.com/yyl1212/math_master/backend/internal/question"
	"mime"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type CorrectionOptions struct {
	Service      *correction.Service
	PublicOrigin string
	Production   bool
}
type correctionRoute struct {
	Action  correction.Action
	ID      string
	SHA     string
	Version int
	Detail  bool
	Method  string
}

func routeCorrection(path, method string) (correctionRoute, error) {
	r := correctionRoute{Method: "GET"}
	if !strings.HasPrefix(path, "/api/v1/corrections/") {
		return r, auth.ErrNotFound
	}
	p := strings.Split(strings.TrimPrefix(path, "/api/v1/corrections/"), "/")
	for _, s := range p {
		if s == "" {
			return r, auth.ErrNotFound
		}
	}
	switch {
	case len(p) == 1 && p[0] == "cases":
		r.Action = correction.ListCasesAction
		if method == "POST" {
			r.Action = correction.CreateCaseAction
			r.Method = "POST"
		} else if method != "GET" {
			r.Method = "GET, POST"
		}
	case len(p) == 2 && p[0] == "cases":
		r.Action = correction.ReadCaseAction
		r.ID = p[1]
	case len(p) == 3 && p[0] == "cases" && p[2] == "plans":
		r.Action = correction.ListPlansAction
		r.ID = p[1]
		if method == "POST" {
			r.Action = correction.CreatePlanAction
			r.Method = "POST"
		} else if method != "GET" {
			r.Method = "GET, POST"
		}
	case len(p) == 3 && p[0] == "cases" && p[2] == "jobs":
		r.Action = correction.ListJobsAction
		r.ID = p[1]
	case (len(p) == 4 || len(p) == 5) && p[0] == "plans" && p[2] == "versions":
		r.ID = p[1]
		n, e := strconv.ParseInt(p[3], 10, 32)
		if e != nil || n < 1 || strconv.FormatInt(n, 10) != p[3] {
			return r, auth.ErrInvalidInput
		}
		r.Version = int(n)
		r.Action = correction.ReadPlanAction
		if len(p) == 4 {
			if method == "PUT" {
				r.Action = correction.UpdatePlanAction
				r.Method = "PUT"
			} else if method != "GET" {
				r.Method = "GET, PUT"
			}
		} else {
			switch p[4] {
			case "detail":
				r.Detail = true
			case "submit":
				r.Action = correction.SubmitPlanAction
				r.Method = "POST"
			case "decision":
				r.Action = correction.DecidePlanAction
				r.Method = "POST"
			default:
				return r, auth.ErrNotFound
			}
		}
	case len(p) == 3 && p[0] == "jobs" && p[2] == "retry":
		r.Action = correction.RetryJobAction
		r.ID = p[1]
		r.Method = "POST"
	case len(p) == 1 && p[0] == "evidence":
		r.Action = correction.ListOwnAction
	case len(p) == 4 && p[0] == "results" && p[2] == "assets":
		r.ID = p[1]
		r.SHA = p[3]
		r.Action = correction.ReadOwnDetailAction
		if !question.ValidSHA(r.SHA) {
			return r, auth.ErrInvalidInput
		}
	case (len(p) == 2 || len(p) == 3) && p[0] == "results":
		r.ID = p[1]
		r.Action = correction.ReadOwnAction
		if len(p) == 3 {
			if p[2] != "detail" {
				return r, auth.ErrNotFound
			}
			r.Action = correction.ReadOwnDetailAction
			r.Detail = true
		}
	default:
		return r, auth.ErrNotFound
	}
	if r.ID != "" && !question.ValidID(r.ID) {
		return r, auth.ErrInvalidInput
	}
	if method != r.Method {
		return r, errPrivateMethod
	}
	return r, nil
}
func correctionQuery(raw string, list, evidence bool) (correction.Query, correction.EvidenceRef, error) {
	var q correction.Query
	var ref correction.EvidenceRef
	bad := func() (correction.Query, correction.EvidenceRef, error) { return q, ref, auth.ErrInvalidInput }
	v, e := url.ParseQuery(raw)
	if e != nil || !list && raw != "" {
		return bad()
	}
	if raw != "" {
		for _, pair := range strings.Split(raw, "&") {
			parts := strings.SplitN(pair, "=", 2)
			if len(parts) != 2 || parts[0] == "" || parts[1] == "" || parts[0] != "limit" && parts[0] != "cursor" && (!evidence || parts[0] != "kind" && parts[0] != "id") {
				return bad()
			}
		}
	}
	for k, values := range v {
		if len(values) != 1 || values[0] == "" {
			return bad()
		}
		s := values[0]
		switch k {
		case "limit":
			n, e := strconv.Atoi(s)
			if e != nil || n < 1 || n > 50 || strconv.Itoa(n) != s {
				return bad()
			}
			q.Limit = n
		case "cursor":
			q.Cursor = s
		case "kind":
			if !evidence {
				return bad()
			}
			ref.Kind = correction.EvidenceKind(s)
		case "id":
			if !evidence {
				return bad()
			}
			ref.ID = s
		default:
			return bad()
		}
	}
	if correction.ValidateQuery(q) != nil || evidence && !correction.ValidEvidence(ref, true) {
		return bad()
	}
	return q, ref, nil
}
func correctionPrivateAccess(r *http.Request, origin string, production, write bool) (question.Access, error) {
	var a question.Access
	if len(r.Header.Values("Origin")) > 1 || len(r.Header.Values("X-CSRF-Token")) > 1 || r.Header.Get("Sec-Fetch-Site") == "cross-site" || write && r.Header.Get("Origin") != origin || !write && r.Header.Get("Origin") != "" && r.Header.Get("Origin") != origin {
		return a, auth.ErrCSRF
	}
	if write {
		typ, params, e := mime.ParseMediaType(r.Header.Get("Content-Type"))
		if e != nil || typ != "application/json" || len(r.Header.Values("Content-Type")) != 1 {
			return a, auth.ErrInvalidInput
		}
		for k, v := range params {
			if k != "charset" || !strings.EqualFold(v, "utf-8") {
				return a, auth.ErrInvalidInput
			}
		}
	} else if r.ContentLength != 0 || len(r.TransferEncoding) > 0 {
		return a, auth.ErrInvalidInput
	}
	cookies, _, e := privateCookies(r, production)
	if e != nil {
		return a, e
	}
	proof, e := auth.DecodeContentProof(cookies, r.Header.Get("X-CSRF-Token"), write)
	if e != nil {
		return a, e
	}
	a = question.Access{TokenHash: proof.TokenHash, CSRF: proof.CSRF, RequestID: r.Header.Get("X-Request-ID")}
	if write {
		if len(r.Header.Values("Idempotency-Key")) != 1 || !question.ValidID(r.Header.Get("Idempotency-Key")) {
			return a, auth.ErrInvalidInput
		}
		a.IdempotencyKey = r.Header.Get("Idempotency-Key")
	} else if len(r.Header.Values("Idempotency-Key")) > 1 {
		return a, auth.ErrInvalidInput
	}
	return a, nil
}
func serveCorrection(w http.ResponseWriter, r *http.Request, o CorrectionOptions) {
	id := requestID()
	privateHeaders(w)
	w.Header().Set("X-Request-ID", id)
	ctx, cancel := context.WithTimeout(r.Context(), 8*time.Second)
	defer cancel()
	r = r.Clone(ctx)
	r.Header = r.Header.Clone()
	r.Header.Set("X-Request-ID", id)
	deadline, _ := ctx.Deadline()
	_ = http.NewResponseController(w).SetReadDeadline(deadline)
	fail := func(e error) {
		if ctx.Err() != nil {
			e = auth.ErrUnavailable
		}
		correctionError(w, r, e)
	}
	if r.URL.RawPath != "" || r.URL.ForceQuery && r.URL.RawQuery == "" {
		fail(auth.ErrInvalidInput)
		return
	}
	route, e := routeCorrection(r.URL.Path, r.Method)
	if e != nil {
		if e == errPrivateMethod {
			w.Header().Set("Allow", route.Method)
		}
		fail(e)
		return
	}
	list := route.Action == correction.ListCasesAction || route.Action == correction.ListPlansAction || route.Action == correction.ListJobsAction || route.Action == correction.ListOwnAction
	q, ref, e := correctionQuery(r.URL.RawQuery, list, route.Action == correction.ListOwnAction)
	if e != nil {
		fail(e)
		return
	}
	if o.Service == nil || o.PublicOrigin == "" {
		fail(correction.ErrNotConfigured)
		return
	}
	a, e := correctionPrivateAccess(r, o.PublicOrigin, o.Production, correction.IsWrite(route.Action))
	if e != nil {
		fail(e)
		return
	}
	if _, e = o.Service.Preflight(ctx, a, route.Action); e != nil {
		fail(e)
		return
	}
	if correction.IsWrite(route.Action) && r.ContentLength > correction.MaxRequestBytes {
		fail(auth.ErrInvalidInput)
		return
	}
	v, status, e := dispatchCorrection(ctx, r, route, a, q, ref, o.Service)
	if e != nil {
		fail(e)
		return
	}
	if ctx.Err() != nil {
		fail(auth.ErrUnavailable)
		return
	}
	if route.SHA != "" {
		contentAsset(w, r, route.SHA, v.([]byte), nil)
		return
	}
	correctionResponse(w, r, status, v)
}
