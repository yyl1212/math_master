package store

import (
	"context"
	"database/sql"
	"github.com/yyl1212/math_master/backend/internal/auth"
	"github.com/yyl1212/math_master/backend/internal/correction"
	"github.com/yyl1212/math_master/backend/internal/notification"
	"github.com/yyl1212/math_master/backend/internal/pagination"
	"github.com/yyl1212/math_master/backend/internal/question"
	"time"
)

func (s *Store) ListNotificationPage(ctx context.Context, a question.Access, q notification.Query, p pagination.Request) (out notification.Envelope[pagination.Page[notification.Metadata]], e error) {
	q.Limit = p.Limit
	q.Cursor = p.Cursor
	if e = notification.ValidateQuery(q); e != nil {
		return
	}
	e = s.notificationTx(ctx, a, notification.ListAction, func(ctx context.Context, tx *sql.Tx, u auth.User, _ time.Time) error {
		args := []any{u.ID}
		anchor := ""
		if p.Cursor != "" {
			c, e := correction.DecodeCursor(p.Cursor)
			if e != nil {
				return e
			}
			anchor = "(n.created_at,n.id)>=($2::timestamptz,$3::uuid)"
			args = append(args, c.CreatedAt, c.ID)
		}
		out.ActorID = u.ID
		out.Data.Items = []notification.Metadata{}
		rows, e := queryNumberedRows(ctx, tx, p, &out.Data.Meta, `n.id::text,n.type,n.evidence_kind,n.evidence_id::text,n.case_id::text,n.result_id::text,n.created_at,r.read_at`, `notifications n LEFT JOIN notification_reads r ON r.notification_id=n.id AND r.owner_user_id=n.owner_user_id`, `n.owner_user_id=$1`, `n.created_at DESC,n.id DESC`, anchor, args...)
		if e != nil {
			return e
		}
		defer rows.Close()
		for rows.Next() {
			var m notification.Metadata
			if e = rows.Scan(&m.ID, &m.Type, &m.Evidence.Kind, &m.Evidence.ID, &m.CaseID, &m.ResultID, &m.CreatedAt, &m.ReadAt); e != nil {
				return e
			}
			m.CreatedAt = m.CreatedAt.UTC()
			out.Data.Items = append(out.Data.Items, m)
		}
		if e = rows.Err(); e != nil {
			return e
		}
		return correctionResponseSize(out)
	})
	return
}
