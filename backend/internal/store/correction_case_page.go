package store

import (
	"context"
	"database/sql"
	"github.com/yyl1212/math_master/backend/internal/auth"
	"github.com/yyl1212/math_master/backend/internal/correction"
	"github.com/yyl1212/math_master/backend/internal/pagination"
	"github.com/yyl1212/math_master/backend/internal/question"
	"time"
)

func (s *Store) ListCorrectionCasePage(ctx context.Context, a question.Access, q correction.Query, p pagination.Request) (out correction.Envelope[pagination.Page[correction.CaseMetadata]], e error) {
	q.Limit = p.Limit
	q.Cursor = p.Cursor
	if e = correction.ValidateQuery(q); e != nil {
		return
	}
	e = s.correctionTx(ctx, a, correction.ListCasesAction, nil, func(ctx context.Context, tx *sql.Tx, u auth.User, _ time.Time) error {
		args := []any{}
		anchor := ""
		if p.Cursor != "" {
			c, e := correction.DecodeCursor(p.Cursor)
			if e != nil {
				return e
			}
			anchor = "(t.created_at,t.id)>=($1::timestamptz,$2::uuid)"
			args = append(args, c.CreatedAt, c.ID)
		}
		out.ActorID = u.ID
		out.Data.Items = []correction.CaseMetadata{}
		rows, e := queryNumberedRows(ctx, tx, p, &out.Data.Meta, `t.id::text`, `correction_cases t`, `t.sealed`, `t.created_at DESC,t.id DESC`, anchor, args...)
		if e != nil {
			return e
		}
		ids := []string{}
		for rows.Next() {
			var id string
			if e = rows.Scan(&id); e != nil {
				rows.Close()
				return e
			}
			ids = append(ids, id)
		}
		e = rows.Err()
		rows.Close()
		if e != nil {
			return e
		}
		for _, id := range ids {
			m, e := correctionReadCase(ctx, tx, id)
			if e != nil {
				return e
			}
			out.Data.Items = append(out.Data.Items, m)
		}
		return correctionResponseSize(out)
	})
	return
}
