package store

import (
	"github.com/yyl1212/math_master/backend/internal/knowledgeadmin"
	"github.com/yyl1212/math_master/backend/internal/pagination"
)

func normalizeNumberedKnowledge(q *knowledgeadmin.Query, total int, offset *int) error {
	if q.Numbered == nil {
		return nil
	}
	m, e := pagination.Window(int64(total), q.Numbered.Page, q.Limit)
	if e != nil {
		return knowledgeadmin.ErrInvalid
	}
	q.Offset = int(m.Offset)
	*offset = q.Offset
	return nil
}
