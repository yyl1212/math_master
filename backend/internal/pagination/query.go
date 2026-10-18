package pagination

import (
	"net/url"
	"regexp"
	"strconv"
)

var positive = regexp.MustCompile(`^[1-9][0-9]*$`)

func Parse(v url.Values, defaultLimit, maxLimit int) (Request, error) {
	p := Request{Page: 1, Limit: defaultLimit}
	if _, ok := v["pagination"]; !ok {
		if _, ok := v["page"]; ok {
			return p, ErrInvalid
		}
		return p, nil
	}
	if len(v["pagination"]) != 1 || v.Get("pagination") != "page" {
		return p, ErrInvalid
	}
	p.Enabled = true
	for _, key := range []string{"page", "limit", "cursor"} {
		if values, ok := v[key]; ok && (len(values) != 1 || values[0] == "") {
			return p, ErrInvalid
		}
	}
	if s := v.Get("page"); s != "" {
		if !positive.MatchString(s) {
			return p, ErrInvalid
		}
		n, e := strconv.ParseInt(s, 10, 64)
		if e != nil || n > MaxSafeInteger {
			return p, ErrInvalid
		}
		p.Page = n
	}
	if s := v.Get("limit"); s != "" {
		if !positive.MatchString(s) {
			return p, ErrInvalid
		}
		n, e := strconv.Atoi(s)
		if e != nil || n < 1 || n > maxLimit {
			return p, ErrInvalid
		}
		p.Limit = n
	}
	p.Cursor = v.Get("cursor")
	if p.Cursor != "" && v.Has("page") {
		return p, ErrInvalid
	}
	return p, nil
}
