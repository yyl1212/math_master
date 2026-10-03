package store

import (
	"context"
	"database/sql"
	"github.com/yyl1212/math_master/backend/internal/auth"
	"github.com/yyl1212/math_master/backend/internal/correction"
	"github.com/yyl1212/math_master/backend/internal/notification"
	"github.com/yyl1212/math_master/backend/internal/question"
	"time"
)

func (s *Store) notificationTx(ctx context.Context, a question.Access, action notification.Action, fn func(context.Context, *sql.Tx, auth.User, time.Time) error) error {
	if action == notification.MarkReadAction && !question.ValidID(a.IdempotencyKey) {
		return auth.ErrInvalidInput
	}
	return s.correctionTx(ctx, a, correction.ReadOwnAction, nil, func(ctx context.Context, tx *sql.Tx, u auth.User, _ time.Time) error {
		write := action == notification.MarkReadAction
		fresh, _, now, e := managedIdentity(ctx, tx, a, write, false, nil)
		if e != nil {
			return e
		}
		if e = notification.Authorize(fresh, action); e != nil {
			return e
		}
		if fresh.ID != u.ID {
			return auth.ErrAuthenticationRequired
		}
		if e = fn(ctx, tx, fresh, now); e != nil {
			return e
		}
		fresh, _, _, e = managedIdentity(ctx, tx, a, write, false, nil)
		if e != nil {
			return e
		}
		return notification.Authorize(fresh, action)
	})
}
func notificationReadOne(ctx context.Context, tx *sql.Tx, owner, id string) (notification.Metadata, error) {
	var m notification.Metadata
	e := tx.QueryRowContext(ctx, `SELECT n.id::text,n.type,n.evidence_kind,n.evidence_id::text,n.case_id::text,n.result_id::text,n.created_at,r.read_at FROM notifications n LEFT JOIN notification_reads r ON r.notification_id=n.id AND r.owner_user_id=n.owner_user_id WHERE n.id=$1 AND n.owner_user_id=$2`, id, owner).Scan(&m.ID, &m.Type, &m.Evidence.Kind, &m.Evidence.ID, &m.CaseID, &m.ResultID, &m.CreatedAt, &m.ReadAt)
	if e != nil {
		return m, workflowRowError(e)
	}
	m.CreatedAt = m.CreatedAt.UTC()
	if m.ReadAt != nil {
		v := m.ReadAt.UTC()
		m.ReadAt = &v
	}
	return m, nil
}
func (s *Store) ListNotifications(ctx context.Context, a question.Access, q notification.Query) (notification.Envelope[notification.Page[notification.Metadata]], error) {
	var out notification.Envelope[notification.Page[notification.Metadata]]
	if e := notification.ValidateQuery(q); e != nil {
		return out, e
	}
	e := s.notificationTx(ctx, a, notification.ListAction, func(ctx context.Context, tx *sql.Tx, u auth.User, _ time.Time) error {
		keys, next, e := correctionPageKeys(ctx, tx, `SELECT t.id::text,0,t.created_at FROM notifications t WHERE t.owner_user_id=$1`, []any{u.ID}, correction.Query{Limit: q.Limit, Cursor: q.Cursor})
		if e != nil {
			return e
		}
		out.ActorID = u.ID
		out.Data.Items = []notification.Metadata{}
		out.Data.NextCursor = next
		for _, k := range keys {
			m, e := notificationReadOne(ctx, tx, u.ID, k.ID)
			if e != nil {
				return e
			}
			out.Data.Items = append(out.Data.Items, m)
		}
		return correctionResponseSize(out)
	})
	if e != nil {
		return notification.Envelope[notification.Page[notification.Metadata]]{}, e
	}
	return out, nil
}
func (s *Store) ReadNotificationCount(ctx context.Context, a question.Access) (notification.Envelope[notification.UnreadCount], error) {
	var out notification.Envelope[notification.UnreadCount]
	e := s.notificationTx(ctx, a, notification.CountAction, func(ctx context.Context, tx *sql.Tx, u auth.User, _ time.Time) error {
		out.ActorID = u.ID
		return tx.QueryRowContext(ctx, `SELECT count(*) FROM notifications n WHERE n.owner_user_id=$1 AND NOT EXISTS(SELECT 1 FROM notification_reads r WHERE r.owner_user_id=n.owner_user_id AND r.notification_id=n.id)`, u.ID).Scan(&out.Data.Count)
	})
	if e != nil {
		return notification.Envelope[notification.UnreadCount]{}, e
	}
	return out, nil
}
func (s *Store) ReadNotification(ctx context.Context, a question.Access, id string) (notification.Envelope[notification.Metadata], error) {
	var out notification.Envelope[notification.Metadata]
	if !question.ValidID(id) {
		return out, auth.ErrInvalidInput
	}
	e := s.notificationTx(ctx, a, notification.ReadAction, func(ctx context.Context, tx *sql.Tx, u auth.User, _ time.Time) error {
		v, e := notificationReadOne(ctx, tx, u.ID, id)
		if e != nil {
			return e
		}
		out = notification.Envelope[notification.Metadata]{ActorID: u.ID, Data: v}
		return correctionResponseSize(out)
	})
	if e != nil {
		return notification.Envelope[notification.Metadata]{}, e
	}
	return out, nil
}
func (s *Store) MarkNotificationRead(ctx context.Context, a question.Access, id string) (notification.Envelope[notification.ReadReceipt], error) {
	var out notification.Envelope[notification.ReadReceipt]
	if !question.ValidID(id) {
		return out, auth.ErrInvalidInput
	}
	_, digest, e := correction.Canonical("notification-command-v1", struct {
		Action   string   `json:"action"`
		Resource string   `json:"resource"`
		Input    struct{} `json:"input"`
	}{Action: "notification-read", Resource: id})
	if e != nil {
		return out, e
	}
	ctx = correctionWithCommand(ctx, "notification-read", id, a.IdempotencyKey)
	e = s.notificationTx(ctx, a, notification.MarkReadAction, func(ctx context.Context, tx *sql.Tx, u auth.User, _ time.Time) error {
		if _, e := notificationReadOne(ctx, tx, u.ID, id); e != nil {
			return e
		}
		out.ActorID = u.ID
		r, found, e := notificationReplay(ctx, tx, u.ID, id, a.IdempotencyKey, digest)
		if e != nil {
			return e
		}
		if found {
			out.Data = r
			return nil
		}
		if _, e = tx.ExecContext(ctx, `INSERT INTO notification_reads(owner_user_id,notification_id) VALUES($1,$2) ON CONFLICT DO NOTHING`, u.ID, id); e != nil {
			return e
		}
		r.Status = 200
		r.NotificationID = id
		if e = tx.QueryRowContext(ctx, `SELECT read_at FROM notification_reads WHERE owner_user_id=$1 AND notification_id=$2`, u.ID, id).Scan(&r.ReadAt); e != nil {
			return e
		}
		r.ReadAt = r.ReadAt.UTC()
		now, e := dbClock(ctx, tx)
		if e != nil {
			return e
		}
		if e = correctionConsumeRate(ctx, tx, u.ID, "notification-read", now); e != nil {
			return e
		}
		if e = notificationRemember(ctx, tx, u.ID, id, a.IdempotencyKey, digest, r); e != nil {
			return e
		}
		out.Data = r
		return correctionResponseSize(out)
	})
	if e != nil {
		return notification.Envelope[notification.ReadReceipt]{}, e
	}
	return out, nil
}
