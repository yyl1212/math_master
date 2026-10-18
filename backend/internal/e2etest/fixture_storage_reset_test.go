package e2etest

import (
	"context"
	"crypto/rand"
	"database/sql"
	"database/sql/driver"
	"errors"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/yyl1212/math_master/backend/internal/auth"
	"github.com/yyl1212/math_master/backend/internal/store"
	"github.com/yyl1212/math_master/backend/internal/testutil"
)

func fixtureStorageSetup(t *testing.T) (*sql.DB, *store.Store, *auth.Service, *auth.AdminService, string) {
	t.Helper()
	db := testutil.Database(t)
	root, err := rootDir()
	if err != nil {
		t.Fatal("fixture root unavailable")
	}
	if err = store.Up(context.Background(), db, root+"/db/migrations"); err != nil {
		t.Fatal("fixture migrations failed")
	}
	repo := store.New(db)
	accounts, admin, err := fixtureAccounts(repo)
	if err != nil {
		t.Fatal("fixture account service failed")
	}
	return db, repo, accounts, admin, root
}

// A split reset adds storage round trips even when account creation stays real.
func TestFixtureStorageResetTransport(t *testing.T) {
	for _, workflow := range []bool{false, true} {
		t.Run(map[bool]string{false: "accounts", true: "workflow"}[workflow], func(t *testing.T) {
			db, _, _, _, _ := fixtureStorageSetup(t)
			counted, count := fixtureStorageCountingDB(t, db)
			accounts, admin, err := fixtureAccounts(store.New(counted))
			if err != nil {
				t.Fatal("real account service failed")
			}
			if workflow {
				err = resetWorkflow(context.Background(), counted, accounts, admin)
			} else {
				err = resetAccounts(context.Background(), counted, accounts, admin)
			}
			if err != nil {
				t.Fatal("real fixture reset failed")
			}
			if got := count.Load(); got != 1 {
				t.Fatalf("storage reset Exec round trips=%d, want 1", got)
			}
		})
	}
}

// A later storage error must not commit a mode change, deleted actors or reset sequence.
func TestFixtureStorageResetRollback(t *testing.T) {
	for _, late := range []bool{false, true} {
		t.Run(map[bool]string{false: "account-foreign-key", true: "catalogue-truncate"}[late], func(t *testing.T) {
			db, repo, accounts, admin, root := fixtureStorageSetup(t)
			ctx := context.Background()
			if err := resetManagedKnowledge(ctx, db, repo, accounts, admin, root); err != nil {
				t.Fatal("real managed fixture seed failed")
			}
			if late {
				_, err := db.Exec(`CREATE FUNCTION fixture_abort_catalogue_reset() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'test-only reset rejection'; END $$;
CREATE TRIGGER fixture_abort_catalogue_reset BEFORE TRUNCATE ON catalogue_versions FOR EACH STATEMENT EXECUTE FUNCTION fixture_abort_catalogue_reset();`)
				if err != nil {
					t.Fatal("real truncate failure fixture failed")
				}
			} else {
				_, err := db.Exec(`CREATE TABLE fixture_reset_blocker(user_id uuid REFERENCES auth_users(id)); INSERT INTO fixture_reset_blocker SELECT id FROM auth_users WHERE username='auth_admin';`)
				if err != nil {
					t.Fatal("real foreign key fixture failed")
				}
			}
			before := fixtureStorageFingerprint(t, db)
			var err error
			if late {
				err = resetWorkflow(ctx, db, accounts, admin)
			} else {
				err = resetAccounts(ctx, db, accounts, admin)
			}
			if err == nil {
				t.Fatal("reset accepted a real storage failure")
			}
			after := fixtureStorageFingerprint(t, db)
			for table, digest := range before {
				if after[table] != digest {
					t.Errorf("failed reset changed %s", table)
				}
			}
		})
	}
	t.Run("cancel-waiting-truncate", func(t *testing.T) {
		db, repo, accounts, admin, root := fixtureStorageSetup(t)
		if err := resetManagedKnowledge(context.Background(), db, repo, accounts, admin, root); err != nil {
			t.Fatal("real managed fixture seed failed")
		}
		before := fixtureStorageFingerprint(t, db)
		blocker, err := db.BeginTx(context.Background(), nil)
		if err != nil {
			t.Fatal("real lock fixture unavailable")
		}
		defer blocker.Rollback()
		if _, err = blocker.Exec(`LOCK TABLE auth_users IN ACCESS SHARE MODE`); err != nil {
			t.Fatal("real lock fixture failed")
		}
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		done := make(chan error, 1)
		go func() { done <- resetAccounts(ctx, db, accounts, admin) }()
		observed := false
		until := time.Now().Add(2 * time.Second)
		for time.Now().Before(until) {
			if err = db.QueryRow(`SELECT EXISTS(SELECT 1 FROM pg_locks WHERE relation='auth_users'::regclass AND mode='AccessExclusiveLock' AND NOT granted)`).Scan(&observed); err != nil {
				t.Fatal("real waiting lock inspection failed")
			}
			if observed {
				break
			}
			time.Sleep(5 * time.Millisecond)
		}
		if !observed {
			t.Fatal("reset never waited for the real truncate lock")
		}
		cancel()
		select {
		case err = <-done:
			if err == nil {
				t.Fatal("cancelled reset succeeded")
			}
		case <-time.After(2 * time.Second):
			t.Fatal("cancelled reset did not return")
		}
		if err = blocker.Rollback(); err != nil {
			t.Fatal("real blocker release failed")
		}
		after := fixtureStorageFingerprint(t, db)
		for table, digest := range before {
			if after[table] != digest {
				t.Errorf("cancelled reset changed %s", table)
			}
		}
	})
}

func fixtureStorageFingerprint(t *testing.T, db *sql.DB) map[string]string {
	t.Helper()
	values := map[string]string{}
	for _, table := range []string{"goose_db_version", "knowledge_admin_state", "topic_learning_state", "auth_users", "auth_user_roles", "auth_sessions", "auth_preauth", "auth_audit_events", "managed_knowledge", "catalogue_versions"} {
		var digest string
		err := db.QueryRow(`SELECT md5(coalesce(string_agg(row_to_json(t)::text,E'\n' ORDER BY row_to_json(t)::text),'') ) FROM public.` + table + ` t`).Scan(&digest)
		if err != nil {
			t.Fatal("reset fingerprint query failed")
		}
		values[table] = digest
	}
	var seq string
	if err := db.QueryRow(`SELECT last_value::text||'/'||is_called::text FROM auth_audit_events_id_seq`).Scan(&seq); err != nil {
		t.Fatal("real sequence inspection failed")
	}
	values["auth-audit-sequence"] = seq
	var trigger string
	if err := db.QueryRow(`SELECT tgenabled::text FROM pg_trigger WHERE tgname='managed_knowledge_marker_guard'`).Scan(&trigger); err != nil {
		t.Fatal("marker trigger inspection failed")
	}
	values["managed-marker-trigger"] = trigger
	return values
}

// Moving catalogue cleanup must preserve standalone content and fresh normal actors.
func TestFixtureStorageResetContracts(t *testing.T) {
	db, repo, accounts, admin, root := fixtureStorageSetup(t)
	ctx := context.Background()
	if err := resetWorkflow(ctx, db, accounts, admin); err != nil {
		t.Fatal("real workflow initialization failed")
	}
	fixtureStorageActors(t, db, 6)
	normal, err := loadFixture(root, false)
	if err != nil {
		t.Fatal("original content fixture unavailable")
	}
	if _, err = repo.ImportDraft(ctx, normal); err != nil {
		t.Fatal("original catalogue seed failed")
	}
	var count int
	if err = db.QueryRow(`SELECT count(*) FROM catalogue_versions`).Scan(&count); err != nil || count == 0 {
		t.Fatal("original catalogue seed missing")
	}
	view, pre, err := accounts.Context(ctx, auth.Cookies{})
	if err != nil {
		t.Fatal("real session context failed")
	}
	_, session, err := accounts.Login(ctx, auth.Cookies{Preauth: pre.SetPreauth}, view.CSRFToken, auth.LoginInput{Username: "auth_learner", Password: fixturePassword}, "reset-contract-login")
	if err != nil {
		t.Fatal("real learner login failed")
	}
	if err = resetAccounts(ctx, db, accounts, admin); err != nil {
		t.Fatal("standalone account reset failed")
	}
	fixtureStorageActors(t, db, 2)
	var retained int
	if err = db.QueryRow(`SELECT count(*) FROM catalogue_versions`).Scan(&retained); err != nil || retained != count {
		t.Fatal("standalone reset deleted catalogue")
	}
	view, _, err = accounts.Context(ctx, auth.Cookies{Session: session.SetSession})
	if err != nil || view.User != nil {
		t.Fatal("old session survived reset")
	}
	if err = resetWorkflow(ctx, db, accounts, admin); err != nil {
		t.Fatal("workflow account reset failed")
	}
	fixtureStorageActors(t, db, 6)
	if err = db.QueryRow(`SELECT count(*) FROM catalogue_versions`).Scan(&count); err != nil || count != 0 {
		t.Fatal("workflow reset retained catalogue")
	}
}

func fixtureStorageActors(t *testing.T, db *sql.DB, expected int) {
	t.Helper()
	rows, err := db.Query(`SELECT username,password_phc FROM auth_users ORDER BY username`)
	if err != nil {
		t.Fatal("real actor lookup failed")
	}
	actors := map[string]string{}
	for rows.Next() {
		var name, phc string
		if err = rows.Scan(&name, &phc); err != nil {
			t.Fatal("real actor decode failed")
		}
		actors[name] = phc
	}
	err = rows.Err()
	rows.Close()
	if err != nil || len(actors) != expected {
		t.Fatal("fixture actor set changed")
	}
	want := map[string]string{"auth_admin": "admin,learner", "auth_learner": "learner", "content_admin": "admin,learner", "content_editor": "editor,learner,reviewer", "content_editor_two": "editor,learner,reviewer", "content_reviewer": "learner,reviewer"}
	hashes := map[string]bool{}
	hasher := auth.NewArgon2Hasher(rand.Reader)
	for name, phc := range actors {
		rolesWanted, known := want[name]
		if !known {
			t.Fatal("unexpected fixture actor")
		}
		var roles string
		if err = db.QueryRow(`SELECT string_agg(r.role,',' ORDER BY r.role) FROM auth_users u JOIN auth_user_roles r ON r.user_id=u.id WHERE u.username=$1`, name).Scan(&roles); err != nil || roles != rolesWanted {
			t.Fatal("fixture roles changed")
		}
		if hashes[phc] || !strings.HasPrefix(phc, "$argon2id$v=19$m=19456,t=2,p=1$") {
			t.Fatal("fixture password salt or parameters changed")
		}
		hashes[phc] = true
		if ok, e := hasher.Verify(context.Background(), phc, fixturePassword); e != nil || !ok {
			t.Fatal("fixture no longer uses real password verification")
		}
	}
	if expected == 2 {
		if _, ok := actors["auth_admin"]; !ok {
			t.Fatal("standalone administrator missing")
		}
		if _, ok := actors["auth_learner"]; !ok {
			t.Fatal("standalone learner missing")
		}
	}
}

func fixtureStorageCountingDB(t *testing.T, db *sql.DB) (*sql.DB, *atomic.Int32) {
	t.Helper()
	conn, err := db.Conn(context.Background())
	if err != nil {
		t.Fatal("real driver unavailable")
	}
	var cfg *pgx.ConnConfig
	err = conn.Raw(func(raw any) error {
		c, ok := raw.(*stdlib.Conn)
		if !ok {
			return errors.New("real pgx driver required")
		}
		cfg = c.Conn().Config().Copy()
		return nil
	})
	conn.Close()
	if err != nil {
		t.Fatal("real driver config unavailable")
	}
	count := &atomic.Int32{}
	connector := stdlib.GetConnector(*cfg, stdlib.OptionAfterConnect(func(ctx context.Context, c *pgx.Conn) error {
		var actual string
		if err := c.QueryRow(ctx, "SELECT current_database()").Scan(&actual); err != nil || actual != cfg.Database {
			return errors.New("isolated database ownership check failed")
		}
		return nil
	}))
	counted := sql.OpenDB(&fixtureStorageConnector{Connector: connector, count: count})
	counted.SetMaxOpenConns(1)
	t.Cleanup(func() { counted.Close() })
	return counted, count
}

type fixtureStorageConnector struct {
	driver.Connector
	count *atomic.Int32
}

func (c *fixtureStorageConnector) Connect(ctx context.Context) (driver.Conn, error) {
	conn, err := c.Connector.Connect(ctx)
	if err != nil {
		return nil, err
	}
	return &fixtureStorageConn{Conn: conn, count: c.count}, nil
}

type fixtureStorageConn struct {
	driver.Conn
	count *atomic.Int32
}

func (c *fixtureStorageConn) BeginTx(ctx context.Context, options driver.TxOptions) (driver.Tx, error) {
	return c.Conn.(driver.ConnBeginTx).BeginTx(ctx, options)
}
func (c *fixtureStorageConn) ExecContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Result, error) {
	queryStart := strings.TrimSpace(query)
	for _, prefix := range []string{"ALTER TABLE goose_db_version DISABLE TRIGGER", "TRUNCATE note_share_receipts", "TRUNCATE study_review_receipts", "UPDATE topic_learning_state SET experience_mode", "TRUNCATE catalogue_versions,"} {
		if strings.HasPrefix(queryStart, prefix) {
			c.count.Add(1)
			break
		}
	}
	return c.Conn.(driver.ExecerContext).ExecContext(ctx, query, args)
}
func (c *fixtureStorageConn) QueryContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
	return c.Conn.(driver.QueryerContext).QueryContext(ctx, query, args)
}
func (c *fixtureStorageConn) CheckNamedValue(value *driver.NamedValue) error {
	return c.Conn.(driver.NamedValueChecker).CheckNamedValue(value)
}
func (c *fixtureStorageConn) Ping(ctx context.Context) error {
	return c.Conn.(driver.Pinger).Ping(ctx)
}
func (c *fixtureStorageConn) ResetSession(ctx context.Context) error {
	return c.Conn.(driver.SessionResetter).ResetSession(ctx)
}

func TestFixtureStorageResetPublishJobs(t *testing.T) {
	db, _, accounts, admin, _ := fixtureStorageSetup(t)
	ctx := context.Background()
	if e := resetAccounts(ctx, db, accounts, admin); e != nil {
		t.Fatal(e)
	}
	if _, e := db.Exec(`INSERT INTO managed_knowledge_publish_jobs(owner_user_id,mode,key,input_sha256,total,pending) SELECT id,'all','reset-publish',repeat('a',64),0,0 FROM auth_users WHERE username='auth_admin';INSERT INTO managed_knowledge_publish_receipts(owner_user_id,action,key,job_id,input_sha256,receipt) SELECT owner_user_id,'create',key,job_id,input_sha256,'{}' FROM managed_knowledge_publish_jobs`); e != nil {
		t.Fatal(e)
	}
	counted, count := fixtureStorageCountingDB(t, db)
	if e := resetFixtureStorage(ctx, counted, false); e != nil {
		t.Fatal(e)
	}
	if count.Load() != 1 {
		t.Fatal("reset split", count.Load())
	}
	var n int
	if e := db.QueryRow("SELECT count(*) FROM managed_knowledge_publish_jobs").Scan(&n); e != nil || n != 0 {
		t.Fatal(n, e)
	}
}
