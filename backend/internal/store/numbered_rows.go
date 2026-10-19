package store

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/yyl1212/math_master/backend/internal/pagination"
)

// queryNumberedRows receives only static, repository-owned SQL fragments. Counting
// and selecting share one statement snapshot, including on READ COMMITTED reads.
func queryNumberedRows(ctx context.Context, tx *sql.Tx, p pagination.Request, meta *pagination.Meta, columns, source, filter, order, anchor string, args ...any) (*numberedRows, error) {
	if _, e := pagination.Window(0, p.Page, p.Limit); e != nil {
		return nil, e
	}
	*meta = pagination.Meta{Limit: p.Limit}
	pageArg, limitArg := len(args)+1, len(args)+2
	rank := fmt.Sprintf("($%d::bigint-1)", pageArg)
	if anchor != "" {
		rank = fmt.Sprintf("CASE WHEN $%d::bigint>0 THEN (SELECT count(*) FROM %s WHERE (%s) AND (%s))/$%d::bigint ELSE 0 END", pageArg, source, filter, anchor, limitArg)
	}
	query := fmt.Sprintf(`WITH __page_count AS MATERIALIZED (SELECT count(*)::bigint total FROM %s WHERE %s),__page_window AS MATERIALIZED (SELECT total,least(%s,CASE WHEN total>0 THEN (total-1)/$%d::bigint ELSE 0 END)*$%d::bigint AS page_offset FROM __page_count) SELECT %s,__page_window.total,__page_window.page_offset FROM %s CROSS JOIN __page_window WHERE %s ORDER BY %s LIMIT $%d OFFSET (SELECT page_offset FROM __page_window)`, source, filter, rank, limitArg, limitArg, columns, source, filter, order, limitArg)
	rows, e := tx.QueryContext(ctx, query, append(args, p.Page, p.Limit)...)
	if e != nil {
		return nil, e
	}
	return &numberedRows{Rows: rows, meta: meta}, nil
}

type numberedRows struct {
	*sql.Rows
	meta *pagination.Meta
}

func (r *numberedRows) Scan(dest ...any) error {
	return r.Rows.Scan(append(dest, &r.meta.Total, &r.meta.Offset)...)
}
