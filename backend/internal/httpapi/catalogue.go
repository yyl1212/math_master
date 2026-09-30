package httpapi

import (
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"unicode/utf8"
)

func listDomains(w http.ResponseWriter, r *http.Request, reader Reader) {
	id := r.Header.Get("X-Request-ID")
	bad := func() { errorResponse(w, id, 400, "INVALID_QUERY", "Invalid query parameters.") }
	values, e := urlValues(r)
	if e != nil {
		bad()
		return
	}
	limit, offset := 20, 0
	q := values.Get("q")
	for k, v := range values {
		if len(v) != 1 || (k != "q" && k != "limit" && k != "offset") {
			bad()
			return
		}
	}
	if !utf8.ValidString(q) || len(q) > 512 || strings.ContainsRune(q, 0) {
		bad()
		return
	}
	if values.Has("limit") {
		limit, e = strconv.Atoi(values.Get("limit"))
		if e != nil || limit < 1 || limit > 100 {
			bad()
			return
		}
	}
	if values.Has("offset") {
		offset, e = strconv.Atoi(values.Get("offset"))
		if e != nil || offset < 0 {
			bad()
			return
		}
	}
	items, total, e := reader.ListDomains(r.Context(), q, limit, offset)
	if e != nil {
		apiError(w, id, e)
		return
	}
	response(w, 200, map[string]any{"items": items, "total": total, "limit": limit, "offset": offset})
}

func urlValues(r *http.Request) (url.Values, error) { return url.ParseQuery(r.URL.RawQuery) }
