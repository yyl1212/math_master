package cli

import (
	"context"
	"database/sql"
	"errors"
	"flag"
	"github.com/yyl1212/math_master/backend/internal/store"
	"github.com/yyl1212/math_master/backend/internal/taxonomy"
	"io"
	"os"
	"time"
)

func RunTopicCatalogue(ctx context.Context, args []string, out, stderr io.Writer) int {
	f := flag.NewFlagSet("topic-catalogue-import", flag.ContinueOnError)
	f.SetOutput(io.Discard)
	var archive string
	f.StringVar(&archive, "archive", "", "")
	fail := func(code string, status int) int {
		write(stderr, map[string]string{"code": code, "message": "Verify the protected configuration and source archive."})
		return status
	}
	if e := f.Parse(args); e != nil || f.NArg() != 0 || archive == "" {
		return fail("INVALID_ARGUMENT", 2)
	}
	before, e := os.Lstat(archive)
	if e != nil || !before.Mode().IsRegular() {
		return fail("INPUT_IO", 1)
	}
	if before.Size() > 32<<20 {
		return fail("INPUT_LIMIT", 2)
	}
	file, e := os.Open(archive)
	if e != nil {
		return fail("INPUT_IO", 1)
	}
	defer file.Close()
	opened, e := file.Stat()
	if e != nil || !os.SameFile(before, opened) || opened.Size() != before.Size() || !opened.ModTime().Equal(before.ModTime()) {
		return fail("INPUT_CHANGED", 1)
	}
	ctx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()
	stop := context.AfterFunc(ctx, func() { file.Close() })
	defer stop()
	input, e := taxonomy.DecodeCapturedBatch(file)
	if e != nil {
		return fail("INVALID_TAXONOMY", 2)
	}
	after, e := file.Stat()
	current, statErr := os.Lstat(archive)
	if e != nil || statErr != nil || !os.SameFile(before, after) || !os.SameFile(before, current) || after.Size() != before.Size() || !after.ModTime().Equal(before.ModTime()) {
		return fail("INPUT_CHANGED", 1)
	}
	raw := os.Getenv("DATABASE_URL")
	if raw == "" {
		return fail("DATABASE_NOT_CONFIGURED", 1)
	}
	db, e := sql.Open("pgx", raw)
	if e != nil {
		return fail("DATABASE_NOT_CONFIGURED", 1)
	}
	defer db.Close()
	db.SetMaxOpenConns(2)
	db.SetMaxIdleConns(1)
	v, e := store.New(db).InstallTaxonomyBatch(ctx, input)
	if e != nil {
		if errors.Is(e, taxonomy.ErrInvalid) {
			return fail("INVALID_TAXONOMY", 2)
		}
		return fail("IMPORT_FAILED", 1)
	}
	write(out, map[string]any{"taxonomyVersionId": v.ID, "batch": v.Batch, "status": "draft", "published": false})
	return 0
}
