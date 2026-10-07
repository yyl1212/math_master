package httpapi

import (
	"context"
	"database/sql"
	"github.com/yyl1212/math_master/backend/internal/auth"
	"github.com/yyl1212/math_master/backend/internal/question"
	"mime"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type QuestionOptions struct {
	Service                *question.Service
	PublicOrigin           string
	Production, Configured bool
}
type questionRoute struct {
	Action     question.Action
	ID, Method string
	Limit      int64
	Status     int
	List       bool
}

func routeQuestion(path, method string) (questionRoute, error) {
	out := questionRoute{Limit: 8192, Status: 200}
	if !strings.HasPrefix(path, "/api/v1/question-bank/") {
		return out, auth.ErrNotFound
	}
	p := strings.Split(strings.TrimPrefix(path, "/api/v1/question-bank/"), "/")
	for _, v := range p {
		if v == "" {
			return out, auth.ErrNotFound
		}
	}
	choose := func(read, write question.Action, verb string) {
		out.Action = read
		out.Method = "GET"
		if write != "" {
			out.Method = "GET, " + verb
			if method == verb {
				out.Action = write
			}
		}
	}
	switch {
	case len(p) == 1 && p[0] == "drafts":
		choose(question.ListDraftsAction, question.CreateDraftAction, "POST")
		out.List = method == "GET"
		if method == "POST" {
			out.Limit = 4 << 20
			out.Status = 201
		}
	case len(p) == 2 && p[0] == "drafts" && p[1] == "adopt":
		out.Action = question.AdoptDraftAction
		out.Method = "POST"
		out.Status = 201
	case len(p) == 2 && p[0] == "drafts":
		choose(question.ReadDraftAction, question.SaveDraftAction, "PUT")
		out.ID = p[1]
		if method == "PUT" {
			out.Limit = 4 << 20
		}
	case len(p) == 3 && p[0] == "drafts" && (p[2] == "validate" || p[2] == "submit"):
		out.Method = "POST"
		out.ID = p[1]
		out.Action = question.ValidateDraftAction
		if p[2] == "submit" {
			out.Action = question.SubmitDraftAction
			out.Status = 201
		}
	case len(p) == 1 && p[0] == "submissions":
		out.Action = question.ListSubmissionsAction
		out.Method = "GET"
		out.List = true
	case len(p) == 2 && p[0] == "submissions":
		out.Action = question.ReadSubmissionAction
		out.Method = "GET"
		out.ID = p[1]
	case len(p) == 3 && p[0] == "submissions" && (p[2] == "instances" || p[2] == "revision" || p[2] == "decision"):
		out.ID = p[1]
		out.Method = "POST"
		out.Action = question.DecideReviewAction
		if p[2] == "instances" {
			out.Action = question.ListInstancesAction
			out.Method = "GET"
			out.List = true
		}
		if p[2] == "revision" {
			out.Action = question.ReviseSubmissionAction
			out.Status = 201
		}
	case len(p) == 1 && p[0] == "publications":
		out.Action = question.ListPublicationsAction
		out.Method = "GET"
		out.List = true
	case len(p) == 2 && p[0] == "publications" && p[1] == "prepare":
		out.Action = question.PrepareReleaseAction
		out.Method = "POST"
		out.Status = 201
	case len(p) == 2 && p[0] == "publications":
		out.Action = question.ReadPublicationAction
		out.Method = "GET"
		out.ID = p[1]
	case len(p) == 3 && p[0] == "publications" && (p[2] == "activate" || p[2] == "members" || p[2] == "changes"):
		out.ID = p[1]
		out.Method = "GET"
		out.List = true
		out.Action = question.ListMembersAction
		if p[2] == "changes" {
			out.Action = question.ListChangesAction
		}
		if p[2] == "activate" {
			out.Action = question.ActivateReleaseAction
			out.Method = "POST"
			out.List = false
		}
	case len(p) == 1 && p[0] == "withdrawals":
		out.Action = question.WithdrawVersionAction
		out.Method = "POST"
		out.Status = 201
	case len(p) == 2 && p[0] == "withdrawals" && p[1] == "preview":
		out.Action = question.PreviewWithdrawalAction
		out.Method = "POST"
		out.List = true
	case len(p) == 1 && p[0] == "coverage":
		out.Action = question.ReadCoverageAction
		out.Method = "GET"
		out.List = true
	default:
		return out, auth.ErrNotFound
	}
	if out.ID != "" && !question.ValidID(out.ID) {
		return out, auth.ErrInvalidInput
	}
	if !(method == out.Method || strings.HasPrefix(out.Method, "GET, ") && (method == "GET" || method == strings.TrimPrefix(out.Method, "GET, "))) {
		return out, errPrivateMethod
	}
	return out, nil
}
func questionQuery(raw string, a question.Action) (question.ListQuery, string, error) {
	var q question.ListQuery
	knowledge := ""
	invalid := func() (question.ListQuery, string, error) { return q, "", auth.ErrInvalidInput }
	if strings.ContainsAny(raw, "%+") {
		return invalid()
	}
	v, e := url.ParseQuery(raw)
	if e != nil {
		return invalid()
	}
	for key, values := range v {
		if len(values) != 1 || values[0] == "" {
			return invalid()
		}
		s := values[0]
		switch key {
		case "scope":
			q.Scope = s
		case "status":
			q.Status = s
		case "knowledgeId":
			if a != question.ReadCoverageAction || !question.ValidMathID(s) {
				return invalid()
			}
			knowledge = s
		case "limit", "offset":
			for _, c := range s {
				if c < '0' || c > '9' {
					return invalid()
				}
			}
			if len(s) > 1 && s[0] == '0' {
				return invalid()
			}
			n, e := strconv.Atoi(s)
			if e != nil {
				return invalid()
			}
			if key == "limit" {
				if n < 1 || n > 100 {
					return invalid()
				}
				q.Limit = n
			} else {
				if n > 100000 {
					return invalid()
				}
				q.Offset = n
			}
		default:
			return invalid()
		}
	}
	allowed := func(value string, choices ...string) bool {
		if value == "" {
			return true
		}
		for _, v := range choices {
			if v == value {
				return true
			}
		}
		return false
	}
	switch a {
	case question.ListDraftsAction:
		if !allowed(q.Scope, "mine", "all") || !allowed(q.Status, "editing", "submitted") {
			return invalid()
		}
	case question.ListSubmissionsAction:
		if !allowed(q.Scope, "mine", "review", "all") || !allowed(q.Status, "pending", "approved", "returned") {
			return invalid()
		}
	case question.ListPublicationsAction:
		if !allowed(q.Scope, "all") || !allowed(q.Status, "prepared", "published") {
			return invalid()
		}
	default:
		if q.Scope != "" || q.Status != "" {
			return invalid()
		}
	}
	return q, knowledge, nil
}

// QuestionReady is a read-only feature check; operators run migrations explicitly.
func QuestionReady(ctx context.Context, db *sql.DB) (bool, error) {
	if db == nil {
		return false, nil
	}
	var ready bool
	err := db.QueryRowContext(ctx, `SELECT bool_and(to_regclass('public.'||name) IS NOT NULL) FROM unnest(ARRAY['question_workspaces','question_workspace_authors','question_packages','question_templates','question_instances','question_blueprints','question_instance_coverage','question_blueprint_sources','question_submissions','question_submission_authors','question_submission_members','question_review_decisions','question_publications','question_publication_members','question_heads','question_withdrawals','question_events','question_idempotency','goose_db_version']) name`).Scan(&ready)
	if err != nil || !ready {
		return false, err
	}
	err = db.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM goose_db_version WHERE version_id=5 AND is_applied)`).Scan(&ready)
	return ready, err
}
func serveQuestion(w http.ResponseWriter, r *http.Request, o QuestionOptions) {
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
		questionError(w, r, e)
	}
	if r.URL.RawPath != "" {
		fail(auth.ErrInvalidInput)
		return
	}
	route, e := routeQuestion(r.URL.Path, r.Method)
	if e != nil {
		if e == errPrivateMethod {
			w.Header().Set("Allow", route.Method)
		}
		fail(e)
		return
	}
	var q question.ListQuery
	knowledge := ""
	if r.URL.ForceQuery && r.URL.RawQuery == "" {
		fail(auth.ErrInvalidInput)
		return
	}
	if route.List {
		q, knowledge, e = questionQuery(r.URL.RawQuery, route.Action)
	} else if r.URL.RawQuery != "" || r.URL.ForceQuery {
		e = auth.ErrInvalidInput
	}
	if e != nil {
		fail(e)
		return
	}
	if o.PublicOrigin == "" {
		fail(errAuthNotConfigured)
		return
	}
	if !o.Configured || o.Service == nil {
		fail(question.ErrNotConfigured)
		return
	}
	write := r.Method != "GET"
	if len(r.Header.Values("Origin")) > 1 || len(r.Header.Values("X-CSRF-Token")) > 1 || r.Header.Get("Sec-Fetch-Site") == "cross-site" || write && r.Header.Get("Origin") != o.PublicOrigin || !write && r.Header.Get("Origin") != "" && r.Header.Get("Origin") != o.PublicOrigin {
		fail(auth.ErrCSRF)
		return
	}
	if write {
		typ, params, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
		if err != nil || typ != "application/json" || len(r.Header.Values("Content-Type")) != 1 {
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
	if question.IsIdempotent(route.Action) {
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
	if write && r.ContentLength > route.Limit {
		fail(errQuestionPayloadTooLarge)
		return
	}
	if write || question.IsHeavy(route.Action) {
		release, err := o.Service.AcquireValidation(ctx)
		if err != nil {
			fail(auth.ErrUnavailable)
			return
		}
		defer release()
	}
	value, e := dispatchQuestion(ctx, r, route, a, q, knowledge, o.Service)
	if e != nil {
		fail(e)
		return
	}
	if ctx.Err() != nil {
		fail(auth.ErrUnavailable)
		return
	}
	questionResponse(w, r, route.Status, value)
}
