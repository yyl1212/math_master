package httpapi

import (
	"context"
	"github.com/yyl1212/math_master/backend/internal/auth"
	"github.com/yyl1212/math_master/backend/internal/knowledgeadmin"
	"github.com/yyl1212/math_master/backend/internal/study"
	"mime"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

func serveManagedStudy(w http.ResponseWriter, r *http.Request, o KnowledgeOptions) {
	id := requestID()
	w.Header().Set("X-Request-ID", id)
	r = r.Clone(r.Context())
	r.Header = r.Header.Clone()
	r.Header.Set("X-Request-ID", id)
	ctx, cancel := context.WithTimeout(r.Context(), 8*time.Second)
	defer cancel()
	r = r.WithContext(ctx)
	_ = http.NewResponseController(w).SetReadDeadline(time.Now().Add(8 * time.Second))
	repo := o.Study
	if repo == nil && o.Service != nil {
		repo, _ = o.Service.Repository.(knowledgeadmin.StudyRepository)
	}
	if repo == nil || o.PublicOrigin == "" {
		knowledgeHTTPError(w, r, knowledgeadmin.ErrNotConfigured)
		return
	}
	if r.URL.RawPath != "" || r.URL.ForceQuery {
		knowledgeHTTPError(w, r, knowledgeadmin.ErrInvalid)
		return
	}
	write := r.Method != "GET"
	cookies, _, e := privateCookies(r, o.Production)
	if e != nil {
		knowledgeHTTPError(w, r, e)
		return
	}
	proof, e := auth.DecodeContentProof(cookies, r.Header.Get("X-CSRF-Token"), write)
	if e != nil {
		knowledgeHTTPError(w, r, e)
		return
	}
	if len(r.Header.Values("Origin")) > 1 || len(r.Header.Values("X-CSRF-Token")) > 1 || r.Header.Get("Sec-Fetch-Site") == "cross-site" || write && r.Header.Get("Origin") != o.PublicOrigin || !write && r.Header.Get("Origin") != "" && r.Header.Get("Origin") != o.PublicOrigin {
		knowledgeHTTPError(w, r, auth.ErrCSRF)
		return
	}
	a := knowledgeadmin.Access{TokenHash: proof.TokenHash, CSRF: proof.CSRF, RequestID: id}
	if write {
		if len(r.Header.Values("Idempotency-Key")) != 1 || strings.TrimSpace(r.Header.Get("Idempotency-Key")) == "" || len(r.Header.Get("Idempotency-Key")) > 128 {
			knowledgeHTTPError(w, r, knowledgeadmin.ErrInvalid)
			return
		}
		a.IdempotencyKey = r.Header.Get("Idempotency-Key")
		typ, params, e := mime.ParseMediaType(r.Header.Get("Content-Type"))
		if e != nil || typ != "application/json" || len(r.Header.Values("Content-Type")) != 1 {
			knowledgeHTTPError(w, r, knowledgeadmin.ErrInvalid)
			return
		}
		for k, v := range params {
			if k != "charset" || !strings.EqualFold(v, "utf-8") {
				knowledgeHTTPError(w, r, knowledgeadmin.ErrInvalid)
				return
			}
		}
	} else if r.ContentLength != 0 || len(r.TransferEncoding) > 0 {
		knowledgeHTTPError(w, r, knowledgeadmin.ErrInvalid)
		return
	}
	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/v3/study/"), "/")
	var value any
	var query knowledgeadmin.Query
	var history study.HistoryQuery
	if len(parts) == 1 && r.Method == "GET" {
		if parts[0] == "history" {
			history, e = managedHistoryQuery(r.URL.RawQuery)
		} else if parts[0] == "overview" {
			if r.URL.RawQuery != "" {
				e = knowledgeadmin.ErrInvalid
			}
		} else {
			query, e = managedStudyQuery(r.URL.RawQuery)
		}
		if e != nil {
			knowledgeHTTPError(w, r, e)
			return
		}
		switch parts[0] {
		case "overview":
			value, e = repo.ReadManagedOverview(ctx, a)
		case "topics":
			value, e = repo.ListManagedStudyTopics(ctx, a, query)
		case "knowledge":
			value, e = repo.ListManagedStudyKnowledge(ctx, a, query)
		case "history":
			value, e = repo.ListManagedHistory(ctx, a, history)
		default:
			e = knowledgeadmin.ErrNotFound
		}
	} else if len(parts) >= 2 && parts[0] == "knowledge" && managedKnowledgeID.MatchString(parts[1]) {
		if r.URL.RawQuery != "" {
			knowledgeHTTPError(w, r, knowledgeadmin.ErrInvalid)
			return
		}
		kid := parts[1]
		switch {
		case len(parts) == 2 && r.Method == "GET":
			value, e = repo.ReadManagedStudy(ctx, a, kid)
		case len(parts) == 3 && parts[2] == "note":
			switch r.Method {
			case "GET":
				value, e = repo.ReadManagedNote(ctx, a, kid)
			case "PUT", "DELETE":
				var in knowledgeadmin.ManagedNoteInput
				if e = decodeKnowledgeJSON(r.Body, 131072, &in); e == nil {
					if r.Method == "PUT" {
						value, e = repo.SaveManagedNote(ctx, a, kid, in)
					} else {
						value, e = repo.DeleteManagedNote(ctx, a, kid, in)
					}
				}
			default:
				e = errPrivateMethod
			}
		case len(parts) == 3 && (parts[2] == "begin" || parts[2] == "complete" || parts[2] == "start-review" || parts[2] == "finish-review") && r.Method == "POST":
			var in knowledgeadmin.ManagedStudyInput
			if e = decodeKnowledgeJSON(r.Body, 8192, &in); e == nil {
				value, e = repo.ApplyManagedStudy(ctx, a, kid, parts[2], in)
			}
		default:
			e = knowledgeadmin.ErrNotFound
		}
	} else {
		e = knowledgeadmin.ErrNotFound
	}
	if e != nil {
		knowledgeHTTPError(w, r, e)
		return
	}
	knowledgeHTTPResponse(w, r, value, true)
}
func managedHistoryQuery(raw string) (q study.HistoryQuery, e error) {
	v, e := url.ParseQuery(raw)
	if e != nil {
		return q, knowledgeadmin.ErrInvalid
	}
	for k, values := range v {
		if len(values) != 1 || values[0] == "" {
			return q, knowledgeadmin.ErrInvalid
		}
		s := values[0]
		switch k {
		case "knowledgeId":
			q.KnowledgeID = s
			if !managedKnowledgeID.MatchString(s) {
				return q, knowledgeadmin.ErrInvalid
			}
		case "topicKey":
			q.TopicID = topicQueryPrefix(s)
		case "kind":
			q.Kind = s
			switch s {
			case "started", "completed", "review-started", "review-finished", "note-saved", "note-deleted":
			default:
				return q, knowledgeadmin.ErrInvalid
			}
		case "cursor":
			q.Cursor = s
			if len(s) > 512 {
				return q, knowledgeadmin.ErrInvalid
			}
		case "limit":
			n, e := strconv.Atoi(s)
			if e != nil || n < 1 || n > 50 {
				return q, knowledgeadmin.ErrInvalid
			}
			q.Limit = n
		case "from", "to":
			at, e := time.Parse(time.RFC3339Nano, s)
			if e != nil {
				return q, knowledgeadmin.ErrInvalid
			}
			if k == "from" {
				q.From = &at
			} else {
				q.To = &at
			}
		default:
			return q, knowledgeadmin.ErrInvalid
		}
	}
	if q.From != nil && q.To != nil && !q.From.Before(*q.To) {
		return q, knowledgeadmin.ErrInvalid
	}
	return q, nil
}
func topicQueryPrefix(s string) string {
	s = strings.TrimPrefix(s, "msc-")
	s = strings.ToUpper(s)
	if strings.HasSuffix(s, "-XX") {
		return strings.TrimSuffix(s, "-XX")
	}
	if strings.HasSuffix(s, "XX") {
		return strings.TrimSuffix(s, "XX")
	}
	if s == "PROJECT-OTHER" || s == "PROJECT:OTHER" {
		return "project:other"
	}
	return s
}

func managedStudyQuery(raw string) (q knowledgeadmin.Query, e error) {
	v, e := url.ParseQuery(raw)
	if e != nil {
		return q, knowledgeadmin.ErrInvalid
	}
	state := ""
	review := false
	if values, ok := v["state"]; ok {
		if len(values) != 1 || !study.ValidState(study.State(values[0])) {
			return q, knowledgeadmin.ErrInvalid
		}
		state = values[0]
		v.Del("state")
	}
	if values, ok := v["reviewOnly"]; ok {
		if len(values) != 1 || (values[0] != "true" && values[0] != "false") {
			return q, knowledgeadmin.ErrInvalid
		}
		review = values[0] == "true"
		v.Del("reviewOnly")
	}
	q, e = managedQuery(v.Encode(), managedRoute{})
	q.State = state
	q.ReviewOnly = review
	return
}
