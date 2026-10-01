package store

import (
	"context"
	"database/sql"
	"github.com/pressly/goose/v3"
	"os"
	"time"
)

func migration(ctx context.Context, db *sql.DB, dir string, down bool) error {
	ctx, c := context.WithTimeout(ctx, 30*time.Second)
	defer c()
	p, e := goose.NewProvider(goose.DialectPostgres, db, os.DirFS(dir))
	if e != nil {
		return e
	}
	if down {
		_, e = p.DownTo(ctx, 0)
	} else {
		_, e = p.Up(ctx)
	}
	return e
}
func Up(ctx context.Context, db *sql.DB, dir string) error   { return migration(ctx, db, dir, false) }
func Down(ctx context.Context, db *sql.DB, dir string) error { return migration(ctx, db, dir, true) }
