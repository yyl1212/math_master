package testutil

import (
	"context"
	"database/sql"
	"errors"
	"net/url"
	"regexp"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
)

var testDatabasePath = regexp.MustCompile(`^/math_master_test_[a-z0-9_]+$`)
var randomDatabaseName = regexp.MustCompile(`^math_master_test_[0-9a-f]{16}$`)

// IsolatedConfigs rejects connection overrides before any network or SQL operation.
// Explicit Database fields remain authoritative; URLs are never reparsed after selection.
func IsolatedConfigs(raw, name string) (*pgx.ConnConfig, *pgx.ConnConfig, error) {
	fail := func() (*pgx.ConnConfig, *pgx.ConnConfig, error) {
		return nil, nil, errors.New("unsafe isolated test database configuration")
	}
	u, e := url.Parse(raw)
	if e != nil || (u.Scheme != "postgres" && u.Scheme != "postgresql") || u.Host == "" || !testDatabasePath.MatchString(u.Path) || u.Fragment != "" || !randomDatabaseName.MatchString(name) {
		return fail()
	}
	q, e := url.ParseQuery(u.RawQuery)
	if e != nil {
		return fail()
	}
	for key, values := range q {
		if (key != "sslmode" && key != "connect_timeout") || len(values) != 1 {
			return fail()
		}
	}
	base, e := pgx.ParseConfig(u.String())
	if e != nil {
		return fail()
	}
	if base.Database != u.Path[1:] {
		return fail()
	}
	admin, work := base.Copy(), base.Copy()
	admin.Database = "postgres"
	work.Database = name
	return admin, work, nil
}

// OpenVerified checks ownership on every physical connection before caller SQL can run.
func OpenVerified(config *pgx.ConnConfig) *sql.DB {
	expected := config.Database
	return stdlib.OpenDB(*config.Copy(), stdlib.OptionAfterConnect(func(ctx context.Context, conn *pgx.Conn) error {
		var actual string
		if e := conn.QueryRow(ctx, "SELECT current_database()").Scan(&actual); e != nil || actual != expected {
			return errors.New("isolated database ownership check failed")
		}
		return nil
	}))
}
