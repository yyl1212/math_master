package cli

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"flag"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/yyl1212/math_master/backend/internal/content"
	"github.com/yyl1212/math_master/backend/internal/store"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"time"
)

func write(w io.Writer, v any) { _ = json.NewEncoder(w).Encode(v) }
func failure(w io.Writer, code string) int {
	write(w, map[string]any{"error": map[string]string{"code": code, "message": "Command failed; verify configuration and input."}})
	return 1
}
func Run(ctx context.Context, command string, args []string, stdout, stderr io.Writer) int {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	f := flag.NewFlagSet(command, flag.ContinueOnError)
	f.SetOutput(io.Discard)
	var cf, pf, ar, id, out, dir string
	var version, cv int
	switch command {
	case "check", "import":
		f.StringVar(&cf, "catalogue", "", "")
		f.StringVar(&pf, "package", "", "")
		f.StringVar(&ar, "assets", "", "")
	case "export":
		f.StringVar(&id, "id", "", "")
		f.StringVar(&out, "out", "", "")
		f.IntVar(&version, "version", 0, "")
		f.IntVar(&cv, "catalogue-version", 0, "")
	case "migrate":
		f.StringVar(&dir, "dir", "db/migrations", "")
	default:
		return failure(stderr, "INVALID_COMMAND")
	}
	if e := f.Parse(args); e != nil {
		return failure(stderr, "INVALID_ARGUMENT")
	}
	if command == "migrate" {
		if f.NArg() != 1 || f.Arg(0) != "up" {
			return failure(stderr, "INVALID_ARGUMENT")
		}
	} else if f.NArg() != 0 {
		return failure(stderr, "INVALID_ARGUMENT")
	}
	var sealed content.ValidatedPackage
	if command == "check" || command == "import" {
		if cf == "" || pf == "" || ar == "" {
			return failure(stderr, "INVALID_ARGUMENT")
		}
		a, e := os.Open(cf)
		if e != nil {
			return failure(stderr, "INPUT_IO")
		}
		defer a.Close()
		cat, e := content.DecodeCatalogue(a)
		if e != nil {
			write(stderr, map[string]string{"code": "INVALID_CONTENT"})
			return 2
		}
		b, e := os.Open(pf)
		if e != nil {
			return failure(stderr, "INPUT_IO")
		}
		defer b.Close()
		p, e := content.DecodePackage(b)
		if e != nil {
			var de *content.DecodeError
			if errors.As(e, &de) {
				write(stderr, de)
			} else {
				write(stderr, map[string]string{"code": "INVALID_CONTENT"})
			}
			return 2
		}
		v, r := content.ValidateAndSeal(cat, p, ar)
		if len(r.Errors) > 0 {
			write(stdout, r)
			return 2
		}
		if command == "check" {
			write(stdout, r)
			return 0
		}
		sealed = v
	}
	raw := os.Getenv("DATABASE_URL")
	u, e := url.Parse(raw)
	if e != nil || raw == "" || (u.Scheme != "postgres" && u.Scheme != "postgresql") || u.Hostname() == "" || u.Path == "" || u.Path == "/" {
		return failure(stderr, "INVALID_CONFIG")
	}
	db, e := sql.Open("pgx", raw)
	if e != nil {
		return failure(stderr, "INVALID_CONFIG")
	}
	defer db.Close()
	db.SetMaxOpenConns(2)
	s := store.New(db)
	switch command {
	case "import":
		r, e := s.ImportDraft(ctx, sealed)
		if e != nil {
			if errors.Is(e, store.ErrImmutableConflict) {
				write(stderr, map[string]string{"code": "IMMUTABLE_CONFLICT"})
				return 2
			}
			return failure(stderr, "DATABASE_FAILURE")
		}
		write(stdout, r)
	case "migrate":
		if e := store.Up(ctx, db, dir); e != nil {
			return failure(stderr, "MIGRATION_FAILURE")
		}
		write(stdout, map[string]string{"status": "migrated"})
	case "export":
		if id == "" || version < 1 || cv < 1 || out == "" {
			return failure(stderr, "INVALID_ARGUMENT")
		}
		actual, e := s.PackageCatalogueVersion(ctx, id, version)
		if e != nil || actual != cv {
			return failure(stderr, "CATALOGUE_MISMATCH")
		}
		p, e := s.ExportPackage(ctx, id, version)
		if e != nil {
			return failure(stderr, "DATABASE_FAILURE")
		}
		cat, e := s.ExportCatalogue(ctx, cv)
		if e != nil {
			return failure(stderr, "DATABASE_FAILURE")
		}
		if e = os.Mkdir(out, 0700); e != nil {
			return failure(stderr, "OUTPUT_IO")
		}
		complete := false
		defer func() {
			if !complete {
				_ = os.RemoveAll(out)
			}
		}()
		if e = os.Mkdir(filepath.Join(out, "assets"), 0700); e != nil {
			return failure(stderr, "OUTPUT_IO")
		}
		for _, a := range p.Assets {
			if !filepath.IsLocal(a.Path) {
				return failure(stderr, "INVALID_ASSET_PATH")
			}
			b, e := s.ExportAsset(ctx, a.SHA256)
			if e != nil {
				return failure(stderr, "DATABASE_FAILURE")
			}
			path := filepath.Join(out, "assets", a.Path)
			if e = os.MkdirAll(filepath.Dir(path), 0700); e != nil {
				return failure(stderr, "OUTPUT_IO")
			}
			if e = os.WriteFile(path, b, 0600); e != nil {
				return failure(stderr, "OUTPUT_IO")
			}
		}
		for name, obj := range map[string]any{"catalogue.json": cat, "package.json": p} {
			b, e := json.MarshalIndent(obj, "", "  ")
			if e != nil {
				return failure(stderr, "OUTPUT_IO")
			}
			if e = os.WriteFile(filepath.Join(out, name), append(b, '\n'), 0600); e != nil {
				return failure(stderr, "OUTPUT_IO")
			}
		}
		complete = true
		write(stdout, map[string]string{"status": "exported"})
	}
	return 0
}
