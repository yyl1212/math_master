package store

import (
	"context"
	"database/sql"
	"errors"
	"github.com/yyl1212/math_master/backend/internal/auth"
	"github.com/yyl1212/math_master/backend/internal/content"
	"github.com/yyl1212/math_master/backend/internal/publication"
	"os"
	"testing"
	"time"
)

func TestImportTransactionReuse(t *testing.T) {
	s, db, a, _ := workflowGuardFixture(t)
	ctx := context.Background()
	cf, pf, root := "../../../content/catalogue/domains.json", "../../../content/packages/elementary-fractions.v1.json", "../../../content/assets"
	f, err := os.Open(cf)
	if err != nil {
		t.Fatal(err)
	}
	c, err := content.DecodeCatalogue(f)
	f.Close()
	if err != nil {
		t.Fatal(err)
	}
	f, err = os.Open(pf)
	if err != nil {
		t.Fatal(err)
	}
	p, err := content.DecodePackage(f)
	f.Close()
	if err != nil {
		t.Fatal(err)
	}
	v, r := content.ValidateAndSeal(c, p, root)
	if !v.Verify() || len(r.Errors) > 0 {
		t.Fatal("seed unavailable")
	}
	late := s.workflowTx(ctx, a, publication.SubmitDraftAction, nil, func(ctx context.Context, tx *sql.Tx, _ auth.User, _ time.Time) error {
		if _, err := s.importValidatedTx(ctx, tx, v); err != nil {
			return err
		}
		return publication.ErrDraftConflict
	})
	if !errors.Is(late, publication.ErrDraftConflict) {
		t.Fatal(late)
	}
	var n int
	if err = db.QueryRow(`SELECT count(*) FROM imported_packages`).Scan(&n); err != nil || n != 0 {
		t.Fatal("caller rollback left imported content")
	}
	first, err := s.ImportDraft(ctx, v)
	if err != nil {
		t.Fatal(err)
	}
	err = s.workflowTx(ctx, a, publication.SubmitDraftAction, nil, func(ctx context.Context, tx *sql.Tx, _ auth.User, _ time.Time) error {
		r, err := s.importValidatedTx(ctx, tx, v)
		if !r.AlreadyImported || r.SHA256 != first.SHA256 {
			t.Error("caller transaction changed legacy digest")
		}
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	p.ID = "cross-package-conflict"
	p.Knowledge[0].Statement = "Different wording requires a new fixed version."
	v, r = content.ValidateAndSeal(c, p, root)
	if len(r.Errors) > 0 {
		t.Fatal(r.Errors)
	}
	err = s.workflowTx(ctx, a, publication.SubmitDraftAction, nil, func(ctx context.Context, tx *sql.Tx, _ auth.User, _ time.Time) error {
		_, err := s.importValidatedTx(ctx, tx, v)
		return err
	})
	if !errors.Is(err, publication.ErrImmutableConflict) {
		t.Fatalf("cross-package conflict not preserved: %v", err)
	}
	if err = db.QueryRow(`SELECT count(*) FROM imported_packages`).Scan(&n); err != nil || n != 1 {
		t.Fatal("conflict partially committed")
	}
}
