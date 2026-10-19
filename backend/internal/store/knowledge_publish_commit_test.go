package store_test

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
	k "github.com/yyl1212/math_master/backend/internal/knowledgeadmin"
	"github.com/yyl1212/math_master/backend/internal/store"
	"os"
	"strings"
	"testing"
)

type publishLostConnector struct {
	driver.Connector
	Mode string
}

func (c publishLostConnector) Connect(ctx context.Context) (driver.Conn, error) {
	v, e := c.Connector.Connect(ctx)
	if e != nil {
		return nil, e
	}
	return publishLostConn{v, c.Mode}, nil
}

type publishLostConn struct {
	driver.Conn
	Mode string
}

func (c publishLostConn) BeginTx(ctx context.Context, o driver.TxOptions) (driver.Tx, error) {
	v, e := c.Conn.(driver.ConnBeginTx).BeginTx(ctx, o)
	if e != nil {
		return nil, e
	}
	if c.Mode == "statement" {
		return v, nil
	}
	return publishLostTx{v}, nil
}
func (c publishLostConn) ExecContext(ctx context.Context, q string, args []driver.NamedValue) (driver.Result, error) {
	if c.Mode == "statement" && strings.HasPrefix(q, "UPDATE managed_knowledge SET") {
		return c.Conn.(driver.ExecerContext).ExecContext(ctx, "SELECT 1/0", nil)
	}
	return c.Conn.(driver.ExecerContext).ExecContext(ctx, q, args)
}
func (c publishLostConn) QueryContext(ctx context.Context, q string, args []driver.NamedValue) (driver.Rows, error) {
	return c.Conn.(driver.QueryerContext).QueryContext(ctx, q, args)
}
func (c publishLostConn) CheckNamedValue(v *driver.NamedValue) error {
	if a, ok := c.Conn.(driver.NamedValueChecker); ok {
		return a.CheckNamedValue(v)
	}
	return driver.ErrSkip
}

type publishLostTx struct{ driver.Tx }

func (tx publishLostTx) Commit() error {
	if e := tx.Tx.Commit(); e != nil {
		return e
	}
	return errors.New("synthetic connection lost after real commit")
}
func TestPublishStepCommitUnknown(t *testing.T) {
	f := newKnowledgeAdminFixture(t)
	f.Import("admin_a", "seed", f.doc, false)
	a := f.Access("admin_a", "create")
	j, e := f.repo.CreatePublishJob(f.ctx, a, k.PublishCreateInput{Mode: k.PublishAll})
	if e != nil {
		t.Fatal(e)
	}
	config, e := pgx.ParseConfig(os.Getenv("TEST_DATABASE_URL"))
	if e != nil {
		t.Fatal(e)
	}
	if e = f.db.QueryRow("SELECT current_database()").Scan(&config.Database); e != nil {
		t.Fatal(e)
	}
	db := sql.OpenDB(publishLostConnector{Connector: stdlib.GetConnector(*config)})
	defer db.Close()
	repo := store.New(db)
	a = f.Access("admin_a", "lost-step")
	if _, e = repo.StepPublishJob(f.ctx, a, j.ID, k.PublishStepInput{ExpectedSequence: 1}); e == nil || errors.Is(e, k.ErrPublishRolledBack) {
		t.Fatal("unknown outcome reported rolled back", e)
	}
	receipt, e := f.repo.ReadPublishJobReceipt(f.ctx, a, j.ID, a.IdempotencyKey)
	if e != nil || receipt == nil || receipt.Published != 2 {
		t.Fatal(receipt, e)
	}
	r, e := f.repo.StepPublishJob(f.ctx, a, j.ID, k.PublishStepInput{ExpectedSequence: 1})
	if e != nil || r.Sequence != receipt.Sequence || f.count("SELECT count(*) FROM managed_knowledge_events WHERE action='publish'") != 2 {
		t.Fatal(r, e)
	}
}

func publishFaultStore(t *testing.T, f *knowledgeAdminFixture, mode string) *store.Store {
	t.Helper()
	config, e := pgx.ParseConfig(os.Getenv("TEST_DATABASE_URL"))
	if e != nil {
		t.Fatal(e)
	}
	if e = f.db.QueryRow("SELECT current_database()").Scan(&config.Database); e != nil {
		t.Fatal(e)
	}
	db := sql.OpenDB(publishLostConnector{Connector: stdlib.GetConnector(*config), Mode: mode})
	t.Cleanup(func() { db.Close() })
	return store.New(db)
}
