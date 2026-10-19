package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/yyl1212/math_master/backend/internal/auth"
	k "github.com/yyl1212/math_master/backend/internal/knowledgeadmin"
	"io"
	"mime"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

func publishInput(r io.Reader, max int, dst any) error {
	raw, e := io.ReadAll(io.LimitReader(r, int64(max+1)))
	if len(raw) > max {
		return &k.DecodeError{Code: "INPUT_TOO_LARGE"}
	}
	if e != nil || !utf8.Valid(raw) || !json.Valid(raw) || !validPrivateEscapes(raw) {
		return k.ErrInvalid
	}
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	first, e := dec.Token()
	if e != nil || first != json.Delim('{') {
		return k.ErrInvalid
	}
	if e = walkPrivateJSON(dec, first, 1, nil); e != nil {
		return k.ErrInvalid
	}
	if _, e = dec.Token(); e != io.EOF {
		return k.ErrInvalid
	}
	var fields map[string]json.RawMessage
	if json.Unmarshal(raw, &fields) != nil {
		return k.ErrInvalid
	}
	switch dst.(type) {
	case *k.PublishCreateInput:
		if string(fields["mode"]) == `"all"` {
			if _, exists := fields["items"]; exists {
				return k.ErrInvalid
			}
		}
	case *k.PublishStepInput:
		if v, exists := fields["batchSize"]; exists && string(v) == "null" {
			return k.ErrInvalid
		}
	}
	dec = json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	if dec.Decode(dst) != nil {
		return k.ErrInvalid
	}
	return nil
}
func publishQuery(raw string, allowed ...string) (url.Values, error) {
	q, e := url.ParseQuery(raw)
	if e != nil {
		return nil, k.ErrInvalid
	}
	for key, values := range q {
		found := false
		for _, name := range allowed {
			found = found || key == name
		}
		if !found || len(values) != 1 || values[0] == "" {
			return nil, k.ErrInvalid
		}
	}
	return q, nil
}
func publishQueryWindow(q url.Values) (limit int, offset int64, e error) {
	limit = 20
	for _, key := range []string{"limit", "offset"} {
		v := q.Get(key)
		if v == "" {
			continue
		}
		if !regexpDigits.MatchString(v) {
			return 0, 0, k.ErrInvalid
		}
		n, err := strconv.ParseInt(v, 10, 64)
		if err != nil || n < 0 || n > k.MaxPublishInteger {
			return 0, 0, k.ErrInvalid
		}
		if key == "limit" {
			if n < 1 || n > 100 {
				return 0, 0, k.ErrInvalid
			}
			limit = int(n)
		} else {
			offset = n
		}
	}
	return
}
func servePublishHealth(w http.ResponseWriter, r *http.Request, o KnowledgeOptions) {
	if r.URL.RawPath != "" || r.URL.ForceQuery || r.URL.RawQuery != "" || r.URL.Fragment != "" {
		knowledgeHTTPError(w, r, k.ErrInvalid)
		return
	}
	if r.Method != "GET" {
		w.Header().Set("Allow", "GET")
		knowledgeHTTPError(w, r, errPrivateMethod)
		return
	}
	if r.ContentLength != 0 || len(r.TransferEncoding) > 0 {
		knowledgeHTTPError(w, r, k.ErrInvalid)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 8*time.Second)
	defer cancel()
	health := k.PublishHealth{Capability: true}
	if o.Service != nil {
		if repo, ok := o.Service.Repository.(k.PublishHealthRepository); ok {
			value, e := repo.ReadPublishHealth(ctx)
			if e == nil {
				health.SchemaReady = value.Capability && value.SchemaReady
			}
		}
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	if health.SchemaReady {
		w.WriteHeader(200)
	} else {
		w.WriteHeader(503)
	}
	_ = json.NewEncoder(w).Encode(health)
}
func serveKnowledgePublish(w http.ResponseWriter, r *http.Request, o KnowledgeOptions) {
	fail := func(e error) { knowledgeHTTPError(w, r, e) }
	ctx, cancel := context.WithTimeout(r.Context(), 8*time.Second)
	defer cancel()
	r = r.WithContext(ctx)
	_ = http.NewResponseController(w).SetReadDeadline(time.Now().Add(8 * time.Second))
	if o.Service == nil || o.PublicOrigin == "" {
		fail(k.ErrNotConfigured)
		return
	}
	repo, ok := o.Service.Repository.(k.PublishRepository)
	if !ok {
		fail(k.ErrNotConfigured)
		return
	}
	if r.URL.RawPath != "" || r.URL.ForceQuery {
		fail(k.ErrInvalid)
		return
	}
	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/v3/admin/knowledge/publish-jobs"), "/")
	id, operation := "", "list"
	method := "GET"
	if len(parts) == 1 && parts[0] == "" {
		if r.Method == "POST" {
			operation = "create"
			method = "POST"
		}
	} else if len(parts) == 2 && parts[0] == "" && k.ValidPublishJobID(parts[1]) {
		id = parts[1]
		operation = "read"
	} else if len(parts) == 3 && parts[0] == "" && k.ValidPublishJobID(parts[1]) && parts[2] == "step" {
		id = parts[1]
		operation = "step"
		method = "POST"
	} else {
		fail(auth.ErrNotFound)
		return
	}
	if r.Method != method {
		w.Header().Set("Allow", method)
		fail(errPrivateMethod)
		return
	}
	write := method != "GET"
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
	if len(r.Header.Values("Origin")) > 1 || len(r.Header.Values("X-CSRF-Token")) > 1 || r.Header.Get("Sec-Fetch-Site") == "cross-site" || write && r.Header.Get("Origin") != o.PublicOrigin || !write && r.Header.Get("Origin") != "" && r.Header.Get("Origin") != o.PublicOrigin {
		fail(auth.ErrCSRF)
		return
	}
	a := k.Access{TokenHash: proof.TokenHash, CSRF: proof.CSRF, RequestID: r.Header.Get("X-Request-ID")}
	if write {
		typ, params, e := mime.ParseMediaType(r.Header.Get("Content-Type"))
		if e != nil || typ != "application/json" || len(r.Header.Values("Content-Type")) != 1 {
			fail(k.ErrInvalid)
			return
		}
		for key, v := range params {
			if key != "charset" || !strings.EqualFold(v, "utf-8") {
				fail(k.ErrInvalid)
				return
			}
		}
		if len(r.Header.Values("Idempotency-Key")) != 1 || strings.TrimSpace(r.Header.Get("Idempotency-Key")) == "" || len(r.Header.Get("Idempotency-Key")) > 128 {
			fail(k.ErrInvalid)
			return
		}
		a.IdempotencyKey = r.Header.Get("Idempotency-Key")
		if _, e = o.Service.Repository.KnowledgePreflight(ctx, a); e != nil {
			fail(e)
			return
		}
	} else if r.ContentLength != 0 || len(r.TransferEncoding) > 0 {
		fail(k.ErrInvalid)
		return
	}
	var value any
	switch operation {
	case "create":
		if r.URL.RawQuery != "" {
			e = k.ErrInvalid
			break
		}
		var in k.PublishCreateInput
		if e = publishInput(r.Body, 262144, &in); e == nil {
			e = in.Validate()
		}
		if e == nil {
			value, e = repo.CreatePublishJob(ctx, a, in)
		}
	case "step":
		if r.URL.RawQuery != "" {
			e = k.ErrInvalid
			break
		}
		var in k.PublishStepInput
		if e = publishInput(r.Body, 8192, &in); e == nil {
			_, e = in.Normalize()
		}
		if e == nil {
			value, e = repo.StepPublishJob(ctx, a, id, in)
		}
	case "list":
		var q url.Values
		q, e = publishQuery(r.URL.RawQuery, "limit", "offset", "createKey")
		if e != nil {
			break
		}
		var limit int
		var offset int64
		limit, offset, e = publishQueryWindow(q)
		if e == nil {
			value, e = repo.ListPublishJobs(ctx, a, k.PublishJobQuery{CreateKey: q.Get("createKey"), Limit: limit, Offset: offset})
		}
	case "read":
		var q url.Values
		q, e = publishQuery(r.URL.RawQuery, "state", "limit", "offset", "receiptKey")
		if e != nil {
			break
		}
		if q.Has("receiptKey") {
			if len(q) != 1 {
				e = k.ErrInvalid
				break
			}
			var d k.PublishJobDetail
			d, e = repo.ReadPublishJob(ctx, a, id, k.PublishItemQuery{Limit: 1})
			if e != nil {
				break
			}
			var receipt *k.PublishStepReceipt
			receipt, e = repo.ReadPublishJobReceipt(ctx, a, id, q.Get("receiptKey"))
			value = k.PublishReceiptLookup{ActorID: d.Job.ActorID, JobID: id, Key: q.Get("receiptKey"), Receipt: receipt}
			break
		}
		var limit int
		var offset int64
		limit, offset, e = publishQueryWindow(q)
		if e == nil {
			value, e = repo.ReadPublishJob(ctx, a, id, k.PublishItemQuery{State: k.PublishItemState(q.Get("state")), Limit: limit, Offset: offset})
		}
	}
	if e != nil {
		fail(e)
		return
	}
	b, e := json.Marshal(value)
	if e != nil || len(b) > 1048576 {
		fail(k.ErrNotConfigured)
		return
	}
	knowledgeHTTPResponse(w, r, value, true)
}

var regexpDigits = regexp.MustCompile(`^[0-9]+$`)
