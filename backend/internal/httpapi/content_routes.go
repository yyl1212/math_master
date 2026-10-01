package httpapi

import (
	"context"
	"database/sql"
	"github.com/yyl1212/math_master/backend/internal/auth"
	"github.com/yyl1212/math_master/backend/internal/publication"
	"mime"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type ContentOptions struct {
	Service      *publication.Service
	PublicOrigin string
	Production   bool
	Configured   bool
}
type contentRoute struct {
	Action          publication.Action
	ID, SHA, Method string
	Limit           int64
	Status          int
	List            bool
}

func routeContent(path, method string) (contentRoute, error) {
	var out contentRoute
	if !strings.HasPrefix(path, "/api/v1/content/") {
		return out, auth.ErrNotFound
	}
	parts := strings.Split(strings.TrimPrefix(path, "/api/v1/content/"), "/")
	for _, part := range parts {
		if part == "" {
			return out, auth.ErrNotFound
		}
	}
	out.Limit = 8192
	out.Status = 200
	choose := func(get, write publication.Action, verb string) {
		out.Method = "GET"
		out.Action = get
		if write != "" {
			out.Method = "GET, " + verb
			if method == verb {
				out.Action = write
				out.Method = verb
			}
		}
	}
	switch {
	case len(parts) == 1 && parts[0] == "drafts":
		choose(publication.ListDraftsAction, publication.CreateDraftAction, "POST")
		out.List = method == "GET"
		if method == "POST" {
			out.Limit = 8 << 20
			out.Status = 201
		}
	case len(parts) == 2 && parts[0] == "drafts" && parts[1] == "adopt":
		out.Action = publication.AdoptDraftAction
		out.Method = "POST"
		out.Status = 201
	case len(parts) == 2 && parts[0] == "drafts":
		choose(publication.ReadDraftAction, publication.SaveDraftAction, "PUT")
		out.ID = parts[1]
		if method == "PUT" {
			out.Limit = 8 << 20
		}
	case len(parts) == 3 && parts[0] == "drafts" && (parts[2] == "validate" || parts[2] == "submit"):
		out.ID = parts[1]
		out.Method = "POST"
		out.Action = publication.ValidateDraftAction
		if parts[2] == "submit" {
			out.Action = publication.SubmitDraftAction
			out.Status = 201
		}
	case len(parts) == 1 && parts[0] == "submissions":
		out.Action = publication.ListSubmissionsAction
		out.Method = "GET"
		out.List = true
	case len(parts) == 2 && parts[0] == "submissions":
		out.Action = publication.ReadSubmissionAction
		out.ID = parts[1]
		out.Method = "GET"
	case len(parts) == 3 && parts[0] == "submissions" && (parts[2] == "revision" || parts[2] == "decision"):
		out.ID = parts[1]
		out.Method = "POST"
		out.Action = publication.DecideReviewAction
		if parts[2] == "revision" {
			out.Action = publication.ReviseSubmissionAction
			out.Status = 201
		}
	case len(parts) == 1 && parts[0] == "publications":
		out.Action = publication.ListPublicationsAction
		out.Method = "GET"
		out.List = true
	case len(parts) == 2 && parts[0] == "publications" && parts[1] == "prepare":
		out.Action = publication.PrepareReleaseAction
		out.Method = "POST"
		out.Status = 201
	case len(parts) == 2 && parts[0] == "publications":
		out.Action = publication.ReadPublicationAction
		out.ID = parts[1]
		out.Method = "GET"
	case len(parts) == 3 && parts[0] == "publications" && parts[2] == "activate":
		out.Action = publication.ActivateReleaseAction
		out.ID = parts[1]
		out.Method = "POST"
	case len(parts) == 1 && parts[0] == "withdrawals":
		out.Action = publication.WithdrawVersionAction
		out.Method = "POST"
		out.Status = 201
	case len(parts) == 2 && parts[0] == "withdrawals" && parts[1] == "preview":
		out.Action = publication.PreviewWithdrawalAction
		out.Method = "POST"
	case len(parts) == 4 && (parts[0] == "drafts" || parts[0] == "submissions") && parts[2] == "assets":
		out.ID = parts[1]
		out.SHA = parts[3]
		out.Method = "GET"
		out.Action = publication.ReadDraftAssetAction
		if parts[0] == "submissions" {
			out.Action = publication.ReadSubmissionAssetAction
		}
	default:
		return out, auth.ErrNotFound
	}
	if out.ID != "" && !publication.ValidID(out.ID) || out.SHA != "" && !publication.ValidSHA(out.SHA) {
		return out, auth.ErrInvalidInput
	}
	if method != out.Method && !(method == "GET" && strings.HasPrefix(out.Method, "GET,")) {
		return out, errPrivateMethod
	}
	return out, nil
}
func contentListQuery(raw string, action publication.Action) (publication.ListQuery, error) {
	var q publication.ListQuery
	if strings.ContainsAny(raw, "%+") {
		return q, auth.ErrInvalidInput
	}
	values, err := url.ParseQuery(raw)
	if err != nil {
		return q, auth.ErrInvalidInput
	}
	for key, v := range values {
		if len(v) != 1 || v[0] == "" {
			return q, auth.ErrInvalidInput
		}
		switch key {
		case "scope":
			q.Scope = v[0]
		case "status":
			q.Status = v[0]
		case "limit", "offset":
			if v[0] == "" {
				return q, auth.ErrInvalidInput
			}
			for _, r := range v[0] {
				if r < '0' || r > '9' {
					return q, auth.ErrInvalidInput
				}
			}
			n, err := strconv.Atoi(v[0])
			if err != nil {
				return q, auth.ErrInvalidInput
			}
			if key == "limit" {
				if n < 1 || n > 100 {
					return q, auth.ErrInvalidInput
				}
				q.Limit = n
			} else {
				if n > 100000 {
					return q, auth.ErrInvalidInput
				}
				q.Offset = n
			}
		default:
			return q, auth.ErrInvalidInput
		}
	}
	var scopes, statuses []string
	switch action {
	case publication.ListDraftsAction:
		scopes = []string{"mine", "all"}
		statuses = []string{"editing", "submitted"}
	case publication.ListSubmissionsAction:
		scopes = []string{"mine", "review", "all"}
		statuses = []string{"pending", "approved", "returned"}
	case publication.ListPublicationsAction:
		scopes = []string{"all"}
		statuses = []string{"draft", "published"}
	}
	contains := func(v string, choices []string) bool {
		if v == "" {
			return true
		}
		for _, c := range choices {
			if c == v {
				return true
			}
		}
		return false
	}
	if !contains(q.Scope, scopes) || !contains(q.Status, statuses) {
		return q, auth.ErrInvalidInput
	}
	return q, nil
}
func contentReady(ctx context.Context, db *sql.DB) (bool, error) {
	if db == nil {
		return false, nil
	}
	var ready bool
	err := db.QueryRowContext(ctx, `SELECT bool_and(to_regclass('public.'||name) IS NOT NULL) FROM unnest(ARRAY['content_workspaces','content_workspace_assets','content_submissions','content_submission_authors','content_submission_members','content_review_decisions','content_publication_manifests','content_withdrawals','content_workflow_events','content_idempotency','auth_sessions','unit_asset_bindings','catalogue_versions']) name`).Scan(&ready)
	return ready, err
}

// ContentReady checks table presence only; migrations remain an operator action.
func ContentReady(ctx context.Context, db *sql.DB) (bool, error) { return contentReady(ctx, db) }
func serveContent(w http.ResponseWriter, r *http.Request, o ContentOptions) {
	id := requestID()
	privateHeaders(w)
	w.Header().Set("X-Request-ID", id)
	ctx, cancel := context.WithTimeout(r.Context(), 8*time.Second)
	defer cancel()
	r = r.Clone(ctx)
	r.Header = r.Header.Clone()
	r.Header.Set("X-Request-ID", id)
	_ = http.NewResponseController(w).SetReadDeadline(time.Now().Add(8 * time.Second))
	if r.URL.RawPath != "" {
		contentError(w, r, auth.ErrInvalidInput)
		return
	}
	route, err := routeContent(r.URL.Path, r.Method)
	if err != nil {
		if err == errPrivateMethod {
			w.Header().Set("Allow", route.Method)
		}
		contentError(w, r, err)
		return
	}
	var query publication.ListQuery
	if r.URL.ForceQuery && r.URL.RawQuery == "" {
		contentError(w, r, auth.ErrInvalidInput)
		return
	}
	if route.List {
		query, err = contentListQuery(r.URL.RawQuery, route.Action)
	} else if r.URL.RawQuery != "" || r.URL.ForceQuery {
		err = auth.ErrInvalidInput
	}
	if err != nil {
		contentError(w, r, err)
		return
	}
	if o.PublicOrigin == "" {
		contentError(w, r, errAuthNotConfigured)
		return
	}
	if !o.Configured || o.Service == nil {
		contentError(w, r, publication.ErrContentNotConfigured)
		return
	}
	write := r.Method != "GET"
	if len(r.Header.Values("Origin")) > 1 || len(r.Header.Values("X-CSRF-Token")) > 1 || r.Header.Get("Sec-Fetch-Site") == "cross-site" || write && r.Header.Get("Origin") != o.PublicOrigin || !write && r.Header.Get("Origin") != "" && r.Header.Get("Origin") != o.PublicOrigin {
		contentError(w, r, auth.ErrCSRF)
		return
	}
	if write {
		typ, params, e := mime.ParseMediaType(r.Header.Get("Content-Type"))
		if e != nil || typ != "application/json" || len(r.Header.Values("Content-Type")) != 1 {
			contentError(w, r, auth.ErrInvalidInput)
			return
		}
		for k, v := range params {
			if k != "charset" || !strings.EqualFold(v, "utf-8") {
				contentError(w, r, auth.ErrInvalidInput)
				return
			}
		}
	} else if r.ContentLength != 0 || len(r.TransferEncoding) > 0 {
		contentError(w, r, auth.ErrInvalidInput)
		return
	}
	cookies, _, err := privateCookies(r, o.Production)
	if err != nil {
		contentError(w, r, err)
		return
	}
	proof, err := auth.DecodeContentProof(cookies, r.Header.Get("X-CSRF-Token"), write)
	if err != nil {
		contentError(w, r, err)
		return
	}
	a := publication.Access{TokenHash: proof.TokenHash, CSRF: proof.CSRF, RequestID: id}
	if publication.IsIdempotent(route.Action) {
		if len(r.Header.Values("Idempotency-Key")) != 1 || !publication.ValidID(r.Header.Get("Idempotency-Key")) {
			contentError(w, r, auth.ErrInvalidInput)
			return
		}
		a.IdempotencyKey = r.Header.Get("Idempotency-Key")
	} else if len(r.Header.Values("Idempotency-Key")) > 1 {
		contentError(w, r, auth.ErrInvalidInput)
		return
	}
	if _, err = o.Service.Preflight(ctx, a, route.Action); err != nil {
		contentError(w, r, err)
		return
	}
	if write && r.ContentLength > route.Limit {
		contentError(w, r, errContentPayloadTooLarge)
		return
	}
	if write || route.SHA != "" {
		release, e := o.Service.AcquireValidation(ctx)
		if e != nil {
			if ctx.Err() != nil {
				contentError(w, r, auth.ErrUnavailable)
			} else {
				contentError(w, r, &auth.RateLimitError{RetryAfterSeconds: 1})
			}
			return
		}
		defer release()
	}
	var value any
	decode := func(v any) bool { err = decodeContentJSON(r.Body, route.Limit, v); return err == nil }
	switch route.Action {
	case publication.ListDraftsAction:
		value, err = o.Service.ListDrafts(ctx, a, query)
	case publication.ReadDraftAction:
		value, err = o.Service.ReadDraft(ctx, a, route.ID)
	case publication.CreateDraftAction:
		var v publication.DraftInput
		if decode(&v) {
			value, err = o.Service.CreateDraft(ctx, a, v)
		}
	case publication.SaveDraftAction:
		var v publication.SaveDraftInput
		if decode(&v) {
			if v.ExpectedRevision < 1 {
				err = auth.ErrInvalidInput
			} else {
				value, err = o.Service.SaveDraft(ctx, a, route.ID, v)
			}
		}
	case publication.AdoptDraftAction:
		var v publication.AdoptInput
		if decode(&v) {
			if !publication.ValidMathID(v.PackageID) || v.PackageVersion < 1 || v.PackageVersion > 2147483647 || !publication.ValidNote(v.Reason) {
				err = auth.ErrInvalidInput
			} else {
				value, err = o.Service.AdoptDraft(ctx, a, v)
			}
		}
	case publication.ValidateDraftAction:
		var v publication.ValidateInput
		if decode(&v) {
			if v.ExpectedRevision < 1 {
				err = auth.ErrInvalidInput
			} else {
				value, err = o.Service.ValidateDraft(ctx, a, route.ID, v)
			}
		}
	case publication.SubmitDraftAction:
		var v publication.SubmitInput
		if decode(&v) {
			if v.ExpectedRevision < 1 || !publication.ValidSHA(v.ExpectedDigest) {
				err = auth.ErrInvalidInput
			} else {
				value, err = o.Service.SubmitDraft(ctx, a, route.ID, v)
			}
		}
	case publication.ListSubmissionsAction:
		value, err = o.Service.ListSubmissions(ctx, a, query)
	case publication.ReadSubmissionAction:
		value, err = o.Service.ReadSubmission(ctx, a, route.ID)
	case publication.ReviseSubmissionAction:
		var v struct{}
		if decode(&v) {
			value, err = o.Service.ReviseSubmission(ctx, a, route.ID)
		}
	case publication.DecideReviewAction:
		var v publication.ReviewInput
		if decode(&v) {
			value, err = o.Service.DecideReview(ctx, a, route.ID, v)
		}
	case publication.ListPublicationsAction:
		value, err = o.Service.ListPublications(ctx, a, query)
	case publication.ReadPublicationAction:
		value, err = o.Service.ReadPublication(ctx, a, route.ID)
	case publication.PrepareReleaseAction:
		var v publication.PrepareInput
		if decode(&v) {
			value, err = o.Service.PrepareRelease(ctx, a, v)
		}
	case publication.ActivateReleaseAction:
		var v publication.ActivateInput
		if decode(&v) {
			value, err = o.Service.ActivateRelease(ctx, a, route.ID, v)
		}
	case publication.PreviewWithdrawalAction:
		var v publication.WithdrawalPreviewInput
		if decode(&v) {
			value, err = o.Service.PreviewWithdrawal(ctx, a, v)
		}
	case publication.WithdrawVersionAction:
		var v publication.WithdrawalInput
		if decode(&v) {
			value, err = o.Service.WithdrawVersion(ctx, a, v)
		}
	case publication.ReadDraftAssetAction:
		var bytes []byte
		bytes, err = o.Service.ReadDraftAsset(ctx, a, route.ID, route.SHA)
		contentAsset(w, r, route.SHA, bytes, err)
		return
	case publication.ReadSubmissionAssetAction:
		var bytes []byte
		bytes, err = o.Service.ReadSubmissionAsset(ctx, a, route.ID, route.SHA)
		contentAsset(w, r, route.SHA, bytes, err)
		return
	}
	if ctx.Err() != nil {
		err = auth.ErrUnavailable
	}
	if err != nil {
		contentError(w, r, err)
		return
	}
	contentResponse(w, r, route.Status, value)
}
