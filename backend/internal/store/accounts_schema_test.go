package store_test

import (
	"testing"
)

func TestAccountsSchema(t *testing.T) {
	db, _, ctx := setup(t)
	var n int
	if err := db.QueryRowContext(ctx, "SELECT count(*) FROM auth_users").Scan(&n); err != nil || n != 0 {
		t.Fatal("account migration missing")
	}
	// Expected relationships are tested by attempting forbidden writes, not by inspecting DDL text.
	create := func(id, name string) {
		t.Helper()
		tx, err := db.BeginTx(ctx, nil)
		if err != nil {
			t.Fatal("begin failed")
		}
		defer tx.Rollback()
		if _, err = tx.ExecContext(ctx, "INSERT INTO auth_users(id,username,password_phc) VALUES($1,$2,'test-only-hash')", id, name); err != nil {
			t.Fatal("user insert failed")
		}
		if _, err = tx.ExecContext(ctx, "INSERT INTO auth_user_roles(user_id,role) VALUES($1,'learner')", id); err != nil {
			t.Fatal("learner insert failed")
		}
		if err = tx.Commit(); err != nil {
			t.Fatal("learner commit failed")
		}
	}
	id := "10000000-0000-4000-8000-000000000001"
	create(id, "schema_user")
	reject := func(query string, args ...any) {
		t.Helper()
		if _, err := db.ExecContext(ctx, query, args...); err == nil {
			t.Fatal("database accepted invalid account data")
		}
	}
	reject("INSERT INTO auth_users(id,username,password_phc) VALUES('10000000-0000-4000-8000-000000000002','schema_user','test-only-hash')")
	reject("INSERT INTO auth_users(id,username,password_phc) VALUES('10000000-0000-4000-8000-000000000003','MixedCase','test-only-hash')")
	reject("INSERT INTO auth_user_roles(user_id,role) VALUES($1,'owner')", id)
	reject("INSERT INTO auth_user_roles(user_id,role) VALUES('10000000-0000-4000-8000-000000000099','learner')")
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal("begin failed")
	}
	if _, err = tx.ExecContext(ctx, "INSERT INTO auth_users(id,username,password_phc) VALUES('10000000-0000-4000-8000-000000000004','missing_learner','test-only-hash')"); err != nil {
		t.Fatal("learner constraint should be deferred")
	}
	if err = tx.Commit(); err == nil {
		t.Fatal("user committed without learner")
	}
	tx, err = db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal("begin failed")
	}
	if _, err = tx.ExecContext(ctx, "DELETE FROM auth_user_roles WHERE user_id=$1 AND role='learner'", id); err != nil {
		t.Fatal("learner deletion check should be deferred")
	}
	if err = tx.Commit(); err == nil {
		t.Fatal("learner removed at commit")
	}
	reject("UPDATE auth_users SET credential_version=0 WHERE id=$1", id)
	reject("INSERT INTO auth_sessions(token_hash,user_id,credential_version,csrf,absolute_expires_at) VALUES(decode('00','hex'),$1,1,decode(repeat('00',32),'hex'),clock_timestamp()+interval '7 days')", id)
	if _, err = db.ExecContext(ctx, "INSERT INTO auth_audit_events(action,actor_id,target_id,before_roles,after_roles,reason,request_id) VALUES('registered',$1,$1,'{}','{learner}','test event','schema-request')", id); err != nil {
		t.Fatal("audit insert failed")
	}
	reject("UPDATE auth_audit_events SET reason='tampered'")
	reject("DELETE FROM auth_audit_events")
	if err = db.QueryRowContext(ctx, "SELECT count(*) FROM publication_heads").Scan(&n); err != nil || n != 0 {
		t.Fatal("content publication changed")
	}
	if err = db.QueryRowContext(ctx, "SELECT count(*) FROM unit_versions").Scan(&n); err != nil || n != 0 {
		t.Fatal("content table changed")
	}
}
