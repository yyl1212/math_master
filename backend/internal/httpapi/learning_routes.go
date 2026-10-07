package httpapi

import (
	"context"
	"github.com/yyl1212/math_master/backend/internal/auth"
	"github.com/yyl1212/math_master/backend/internal/learning"
	"github.com/yyl1212/math_master/backend/internal/question"
	"mime"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type LearningOptions struct {
	Learning     *learning.Service
	PublicOrigin string
	Production   bool
}
type learningRoute struct {
	Action          learning.Action
	ID, SHA, Method string
	Status          int
}

func routeLearning(path, method string) (learningRoute, error) {
	r := learningRoute{Status: 200}
	if !strings.HasPrefix(path, "/api/v1/learning/") {
		return r, auth.ErrNotFound
	}
	p := strings.Split(strings.TrimPrefix(path, "/api/v1/learning/"), "/")
	for _, s := range p {
		if s == "" {
			return r, auth.ErrNotFound
		}
	}
	r.Method = "GET"
	mathID := false
	switch {
	case len(p) == 1 && p[0] == "overview":
		r.Action = learning.ReadOverviewAction
	case len(p) == 1 && p[0] == "knowledge":
		r.Action = learning.ListKnowledgeAction
	case len(p) == 2 && p[0] == "knowledge":
		r.Action = learning.ReadKnowledgeAction
		r.ID = p[1]
		mathID = true
	case len(p) == 3 && p[0] == "knowledge" && (p[2] == "start" || p[2] == "complete"):
		r.Method = "POST"
		r.ID = p[1]
		mathID = true
		r.Action = learning.StartKnowledgeAction
		if p[2] == "complete" {
			r.Action = learning.CompleteKnowledgeAction
		}
		if p[2] == "start" {
			r.Status = 201
		}
	case len(p) == 3 && p[0] == "paths" && p[2] == "enroll":
		r.Method = "POST"
		r.ID = p[1]
		mathID = true
		r.Status = 201
		r.Action = learning.EnrollPathAction
	case len(p) == 1 && p[0] == "paths":
		r.Action = learning.ListPathsAction
	case len(p) == 2 && p[0] == "paths":
		r.Action = learning.ReadPathAction
		r.ID = p[1]
	case len(p) == 3 && p[0] == "paths" && p[2] == "nodes":
		r.Action = learning.ListPathNodesAction
		r.ID = p[1]
	case len(p) == 1 && p[0] == "practice":
		r.Action = learning.CreatePracticeAction
		r.Method = "POST"
		r.Status = 201
	case len(p) == 2 && p[0] == "practice":
		r.Action = learning.ReadPracticeAction
		r.ID = p[1]
	case len(p) == 3 && p[0] == "practice":
		r.Method = "POST"
		r.ID = p[1]
		switch p[2] {
		case "answer":
			r.Action = learning.AnswerPracticeAction
		case "reveal":
			r.Action = learning.RevealPracticeAction
		case "abandon":
			r.Action = learning.AbandonPracticeAction
		default:
			return r, auth.ErrNotFound
		}
	case len(p) == 1 && p[0] == "assessments":
		r.Action = learning.CreateAssessmentAction
		r.Method = "POST"
		r.Status = 201
	case len(p) == 2 && p[0] == "assessments":
		r.Action = learning.ReadAssessmentAction
		r.ID = p[1]
	case len(p) == 3 && p[0] == "assessments":
		r.ID = p[1]
		switch p[2] {
		case "submit":
			r.Action = learning.SubmitAssessmentAction
			r.Method = "POST"
		case "abandon":
			r.Action = learning.AbandonAssessmentAction
			r.Method = "POST"
		case "result":
			r.Action = learning.ReadAssessmentResultAction
		default:
			return r, auth.ErrNotFound
		}
	case len(p) == 1 && p[0] == "history":
		r.Action = learning.ListHistoryAction
	case len(p) == 3 && p[0] == "assets":
		r.Action = learning.ReadAssetAction
		r.ID = p[1]
		r.SHA = p[2]
		if !question.ValidSHA(r.SHA) {
			return r, auth.ErrInvalidInput
		}
	default:
		return r, auth.ErrNotFound
	}
	if r.ID != "" && ((mathID && !question.ValidMathID(r.ID)) || (!mathID && !question.ValidID(r.ID))) {
		return r, auth.ErrInvalidInput
	}
	if method != r.Method {
		return r, errPrivateMethod
	}
	return r, nil
}
func learningQuery(raw string, a learning.Action) (learning.ListQuery, int, error) {
	q := learning.ListQuery{}
	version := 0
	bad := func() (learning.ListQuery, int, error) { return q, 0, auth.ErrInvalidInput }
	if strings.ContainsAny(raw, "%+") {
		return bad()
	}
	v, e := url.ParseQuery(raw)
	if e != nil {
		return bad()
	}
	list := a == learning.ListKnowledgeAction || a == learning.ListPathsAction || a == learning.ListPathNodesAction || a == learning.ListHistoryAction
	for k, vs := range v {
		if len(vs) != 1 || vs[0] == "" {
			return bad()
		}
		s := vs[0]
		for _, c := range s {
			if c < '0' || c > '9' {
				return bad()
			}
		}
		if len(s) > 1 && s[0] == '0' {
			return bad()
		}
		n, e := strconv.Atoi(s)
		if e != nil {
			return bad()
		}
		switch k {
		case "version":
			if a != learning.ReadKnowledgeAction || n < 1 || n > 2147483647 {
				return bad()
			}
			version = n
		case "limit":
			if !list || n < 1 || n > 100 {
				return bad()
			}
			q.Limit = n
		case "offset":
			if !list || n > 100000 {
				return bad()
			}
			q.Offset = n
		default:
			return bad()
		}
	}
	if a == learning.ReadKnowledgeAction && version == 0 {
		return bad()
	}
	return q, version, nil
}
func serveLearning(w http.ResponseWriter, r *http.Request, o LearningOptions) {
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
		learningError(w, r, e)
	}
	if r.URL.RawPath != "" || r.URL.ForceQuery && r.URL.RawQuery == "" {
		fail(auth.ErrInvalidInput)
		return
	}
	route, e := routeLearning(r.URL.Path, r.Method)
	if e != nil {
		if e == errPrivateMethod {
			w.Header().Set("Allow", route.Method)
		}
		fail(e)
		return
	}
	q, version, e := learningQuery(r.URL.RawQuery, route.Action)
	if e != nil {
		fail(e)
		return
	}
	if o.PublicOrigin == "" {
		fail(errAuthNotConfigured)
		return
	}
	if o.Learning == nil {
		fail(learning.ErrNotConfigured)
		return
	}
	write := r.Method != "GET"
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
	if learning.IsIdempotent(route.Action) {
		if len(r.Header.Values("Idempotency-Key")) != 1 || !question.ValidID(r.Header.Get("Idempotency-Key")) {
			fail(auth.ErrInvalidInput)
			return
		}
		a.IdempotencyKey = r.Header.Get("Idempotency-Key")
	} else if len(r.Header.Values("Idempotency-Key")) > 1 {
		fail(auth.ErrInvalidInput)
		return
	}
	if _, e = o.Learning.Preflight(ctx, a, route.Action); e != nil {
		fail(e)
		return
	}
	if write && r.ContentLength > learningBodyLimit {
		fail(errQuestionPayloadTooLarge)
		return
	}
	if learning.IsHeavy(route.Action) {
		release, e := o.Learning.AcquireValidation(ctx)
		if e != nil {
			fail(auth.ErrUnavailable)
			return
		}
		defer release()
	}
	v, e := dispatchLearning(ctx, r, route, a, q, version, o.Learning)
	if e != nil {
		fail(e)
		return
	}
	if ctx.Err() != nil {
		fail(auth.ErrUnavailable)
		return
	}
	if route.Action == learning.ReadAssetAction {
		contentAsset(w, r, route.SHA, v.([]byte), nil)
		return
	}
	learningResponse(w, r, route.Status, v)
}
