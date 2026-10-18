// Package pagination defines opt-in numbered pages without changing cursor contracts.
package pagination

import "errors"

const MaxSafeInteger int64 = 9007199254740991

var ErrInvalid = errors.New("invalid page")

type Request struct {
	Enabled bool
	Page    int64
	Limit   int
	Cursor  string
}
type Meta struct {
	Total  int64 `json:"total"`
	Limit  int   `json:"limit"`
	Offset int64 `json:"offset"`
}
type Page[T any] struct {
	Items []T `json:"items"`
	Meta
}
type ActorPage[T any] struct {
	ActorID string `json:"actorId"`
	Page[T]
}

func Window(total, page int64, limit int) (Meta, error) {
	m := Meta{Total: total, Limit: limit}
	if total < 0 || total > MaxSafeInteger || page < 1 || page > MaxSafeInteger || limit < 1 || limit > 100 {
		return m, ErrInvalid
	}
	if total == 0 {
		return m, nil
	}
	last := (total-1)/int64(limit) + 1
	if page > last {
		page = last
	}
	m.Offset = (page - 1) * int64(limit)
	return m, nil
}
