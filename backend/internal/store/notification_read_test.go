package store_test

import (
	"errors"
	"github.com/yyl1212/math_master/backend/internal/auth"
	"github.com/yyl1212/math_master/backend/internal/correction"
	"github.com/yyl1212/math_master/backend/internal/notification"
	"reflect"
	"testing"
	"time"
)

func TestNotificationCursorIsolation(t *testing.T) {
	f := newCorrectionFixture(t)
	_, aid, _ := f.cReadableResult()
	var cid string
	if e := f.db.QueryRow(`SELECT id::text FROM correction_cases LIMIT 1`).Scan(&cid); e != nil {
		t.Fatal(e)
	}
	f.exec(`ALTER TABLE notifications ALTER COLUMN created_at SET DEFAULT '2026-01-01 01:02:03.123456+00'`)
	wanted := map[string]bool{}
	for n := 0; n < 51; n++ {
		id := f.ID()
		f.exec(`INSERT INTO notifications(id,owner_user_id,dedup_key,type,evidence_kind,evidence_id,case_id) VALUES($1,$2,($1::uuid)::text,'checking','assessment',$3,$4)`, id, f.ids["learner_a"], aid, cid)
		wanted[id] = true
	}
	q := notification.Query{Limit: 50}
	seen := map[string]bool{}
	var cursor string
	for {
		page, e := f.repo.ListNotifications(f.ctx, f.Access("learner_a", false), q)
		if e != nil {
			t.Fatal(e)
		}
		for _, m := range page.Data.Items {
			if seen[m.ID] {
				t.Fatal("duplicate")
			}
			seen[m.ID] = true
		}
		if page.Data.NextCursor == nil {
			break
		}
		q.Cursor = *page.Data.NextCursor
		cursor = q.Cursor
	}
	for id := range wanted {
		if !seen[id] {
			t.Fatal("missing", id)
		}
	}
	foreign, e := f.repo.ListNotifications(f.ctx, f.Access("learner_b", false), notification.Query{Cursor: cursor})
	if e != nil || len(foreign.Data.Items) != 0 {
		t.Fatal("cursor expanded own WHERE", e)
	}
	for id := range wanted {
		v, e := f.repo.ReadNotification(f.ctx, f.Access("learner_b", false), id)
		if !errors.Is(e, auth.ErrNotFound) || v.Data.ID != "" {
			t.Fatal("foreign read", e)
		}
		if _, e = f.repo.MarkNotificationRead(f.ctx, f.Access("learner_b", false), id); !errors.Is(e, auth.ErrNotFound) {
			t.Fatal("foreign mark", e)
		}
		break
	}
	count, e := f.repo.ReadNotificationCount(f.ctx, f.Access("learner_a", false))
	if e != nil || count.Data.Count != int64(len(seen)) {
		t.Fatal("count", count, e)
	}
	if _, e = f.repo.ListNotifications(f.ctx, f.Access("learner_a", false), notification.Query{Limit: 51}); !errors.Is(e, auth.ErrInvalidInput) {
		t.Fatal("limit", e)
	}
}
func TestNotificationReadReplay(t *testing.T) {
	f := newCorrectionFixture(t)
	f.cReadableResult()
	page, e := f.repo.ListNotifications(f.ctx, f.Access("learner_a", false), notification.Query{})
	if e != nil || len(page.Data.Items) == 0 {
		t.Fatal(e)
	}
	id := page.Data.Items[0].ID
	a := f.Access("learner_a", false)
	first, e := f.repo.MarkNotificationRead(f.ctx, a, id)
	if e != nil || first.Data.Status != 200 || first.Data.ReadAt.IsZero() {
		t.Fatal(e)
	}
	replay, e := f.repo.MarkNotificationRead(f.ctx, a, id)
	if e != nil || !reflect.DeepEqual(first, replay) {
		t.Fatal("receipt changed", e)
	}
	second, e := f.repo.MarkNotificationRead(f.ctx, f.Access("learner_a", false), id)
	if e != nil || !second.Data.ReadAt.Equal(first.Data.ReadAt) {
		t.Fatal("first read time changed", e)
	}
	if f.count(`SELECT count(*) FROM correction_rate_limits WHERE owner_user_id=$1 AND scope='notification-read'`, f.ids["learner_a"]) != 2 {
		t.Fatal("replay quota")
	}
	one, e := f.repo.ReadNotification(f.ctx, f.Access("learner_a", false), id)
	if e != nil || one.Data.ReadAt == nil || !one.Data.ReadAt.Equal(first.Data.ReadAt) {
		t.Fatal("read projection", e)
	}
	count, e := f.repo.ReadNotificationCount(f.ctx, f.Access("learner_a", false))
	if e != nil || count.Data.Count != 0 {
		t.Fatal(count, e)
	}
	f.exec(`UPDATE auth_users SET credential_version=credential_version+1 WHERE id=$1`, f.ids["learner_a"])
	if _, e = f.repo.MarkNotificationRead(f.ctx, a, id); !errors.Is(e, auth.ErrAuthenticationRequired) {
		t.Fatal("replay omitted fresh auth", e)
	}
}
func TestNotificationRateAndRollback(t *testing.T) {
	f := newCorrectionFixture(t)
	f.cReadableResult()
	page, e := f.repo.ListNotifications(f.ctx, f.Access("learner_a", false), notification.Query{})
	if e != nil {
		t.Fatal(e)
	}
	id := page.Data.Items[0].ID
	f.exec(`CREATE FUNCTION isolated_notification_receipt() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'isolated receipt failure'; END $$`)
	f.exec(`CREATE TRIGGER isolated_notification_receipt BEFORE INSERT ON correction_idempotency FOR EACH ROW EXECUTE FUNCTION isolated_notification_receipt()`)
	if _, e = f.repo.MarkNotificationRead(f.ctx, f.Access("learner_a", false), id); e == nil {
		t.Fatal("fault accepted")
	}
	if f.count(`SELECT count(*) FROM notification_reads`) != 0 || f.count(`SELECT count(*) FROM correction_rate_limits WHERE scope='notification-read'`) != 0 {
		t.Fatal("partial read")
	}
	f.exec(`DROP TRIGGER isolated_notification_receipt ON correction_idempotency`)
	var now time.Time
	if e = f.db.QueryRow(`SELECT clock_timestamp()`).Scan(&now); e != nil {
		t.Fatal(e)
	}
	for n := 0; n < 300; n++ {
		f.exec(`INSERT INTO correction_rate_limits(owner_user_id,scope,command_key,consumed_at) VALUES($1,'notification-read',$2,$3)`, f.ids["learner_a"], f.ID(), now.Add(-time.Minute))
	}
	out, e := f.repo.MarkNotificationRead(f.ctx, f.Access("learner_a", false), id)
	var rate *correction.RateError
	if !errors.As(e, &rate) || out.ActorID != "" || rate.RetryAt.IsZero() || f.count(`SELECT count(*) FROM notification_reads`) != 0 {
		t.Fatal("rate guard", e)
	}
}

func TestNotificationReadCSRFAndCurrentSession(t *testing.T) {
	f := newCorrectionFixture(t)
	f.cReadableResult()
	page, e := f.repo.ListNotifications(f.ctx, f.Access("learner_a", false), notification.Query{})
	if e != nil {
		t.Fatal(e)
	}
	id := page.Data.Items[0].ID
	a := f.Access("learner_a", false)
	a.CSRF = auth.Secret{}
	if _, e = f.repo.MarkNotificationRead(f.ctx, a, id); !errors.Is(e, auth.ErrCSRF) {
		t.Fatal("missing command CSRF", e)
	}
	if f.count(`SELECT count(*) FROM notification_reads`) != 0 {
		t.Fatal("CSRF read committed")
	}
	a = f.Access("learner_a", false)
	if _, e = f.repo.MarkNotificationRead(f.ctx, a, id); e != nil {
		t.Fatal(e)
	}
	before := f.count(`SELECT count(*) FROM correction_rate_limits WHERE scope='notification-read'`)
	f.exec(`UPDATE auth_sessions SET revoked_at=clock_timestamp() WHERE user_id=$1`, f.ids["learner_a"])
	if _, e = f.repo.MarkNotificationRead(f.ctx, a, id); !errors.Is(e, auth.ErrAuthenticationRequired) {
		t.Fatal("revoked replay", e)
	}
	if f.count(`SELECT count(*) FROM correction_rate_limits WHERE scope='notification-read'`) != before {
		t.Fatal("revoked replay consumed quota")
	}
}
