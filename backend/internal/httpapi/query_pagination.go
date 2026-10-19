package httpapi

import (
	"github.com/yyl1212/math_master/backend/internal/pagination"
	"net/url"
)

func numberedQuery(raw string, allowed bool, defaultLimit, maxLimit int) (string, pagination.Request, error) {
	values, e := url.ParseQuery(raw)
	if e != nil {
		return raw, pagination.Request{}, pagination.ErrInvalid
	}
	p, e := pagination.Parse(values, defaultLimit, maxLimit)
	if e != nil {
		return raw, p, e
	}
	if !p.Enabled {
		return raw, p, nil
	}
	if !allowed {
		return raw, p, pagination.ErrInvalid
	}
	if values.Has("offset") {
		return raw, p, pagination.ErrInvalid
	}
	values.Del("pagination")
	values.Del("page")
	return values.Encode(), p, nil
}
