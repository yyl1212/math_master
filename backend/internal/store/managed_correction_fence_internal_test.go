package store

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"sync/atomic"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
)

func ManagedCorrectionGuardForTest(ctx context.Context, s *Store, locks bool) error {
	ctx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()
	conn, err := s.db.Conn(ctx)
	if err != nil {
		return err
	}
	defer conn.Close()
	tx, err := conn.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	return managedCorrectionGuard(ctx, tx, locks)
}

func ManagedCorrectionLockIDsForTest() []int64 {
	return []int64{adminLockID, 1296127048, knowledgeLockID}
}

// Count real driver Exec calls without replacing rows or transaction behavior.
func ManagedCorrectionCountingStoreForTest(ctx context.Context, db *sql.DB) (*Store, *atomic.Int32, func(), error) {
	conn, err := db.Conn(ctx)
	if err != nil {
		return nil, nil, nil, err
	}
	var config *pgx.ConnConfig
	err = conn.Raw(func(raw any) error {
		wrapped, ok := raw.(*stdlib.Conn)
		if !ok {
			return errors.New("test requires pgx")
		}
		config = wrapped.Conn().Config().Copy()
		return nil
	})
	conn.Close()
	if err != nil {
		return nil, nil, nil, err
	}
	count := &atomic.Int32{}
	counted := sql.OpenDB(&managedFenceConnector{Connector: stdlib.GetConnector(*config), count: count})
	counted.SetMaxOpenConns(1)
	return New(counted), count, func() { counted.Close() }, nil
}

type managedFenceConnector struct {
	driver.Connector
	count *atomic.Int32
}

func (c *managedFenceConnector) Connect(ctx context.Context) (driver.Conn, error) {
	conn, err := c.Connector.Connect(ctx)
	if err != nil {
		return nil, err
	}
	return &managedFenceConn{Conn: conn, count: c.count}, nil
}

type managedFenceConn struct {
	driver.Conn
	count *atomic.Int32
}

func (c *managedFenceConn) BeginTx(ctx context.Context, options driver.TxOptions) (driver.Tx, error) {
	return c.Conn.(driver.ConnBeginTx).BeginTx(ctx, options)
}

func (c *managedFenceConn) ExecContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Result, error) {
	c.count.Add(1)
	return c.Conn.(driver.ExecerContext).ExecContext(ctx, query, args)
}

func (c *managedFenceConn) QueryContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
	return c.Conn.(driver.QueryerContext).QueryContext(ctx, query, args)
}

func (c *managedFenceConn) CheckNamedValue(value *driver.NamedValue) error {
	return c.Conn.(driver.NamedValueChecker).CheckNamedValue(value)
}
