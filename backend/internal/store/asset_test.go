package store_test

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"github.com/yyl1212/math_master/backend/internal/content"
	"github.com/yyl1212/math_master/backend/internal/store"
	"testing"
)

type publicAssetReader interface {
	GetPublishedAsset(context.Context, string) ([]byte, error)
}

func assetReader(t *testing.T, s *store.Store) publicAssetReader {
	t.Helper()
	r, ok := any(s).(publicAssetReader)
	if !ok {
		t.Fatal("public asset reader is not implemented")
	}
	return r
}
func TestPublicAssetRequiresEffectiveUnitBinding(t *testing.T) {
	for _, tc := range []struct{ name, withdraw string }{
		{"unit", "UPDATE publication_members SET availability='withdrawn' WHERE kind='unit'"},
		{"asset", "UPDATE publication_members SET availability='withdrawn' WHERE kind='asset'"},
		{"knowledge", "UPDATE publication_members SET availability='withdrawn' WHERE kind='knowledge' AND id='equivalent-fractions'"},
		{"prerequisite", "UPDATE publication_members SET availability='withdrawn' WHERE kind='knowledge' AND id='numbers'"},
		{"snapshot", "UPDATE publication_snapshots SET status='withdrawn'"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			db, s, ctx := setup(t)
			v := input(t, nil)
			if _, e := s.ImportDraft(ctx, v); e != nil {
				t.Fatal(e)
			}
			r := assetReader(t, s)
			digest := v.Package().Assets[0].SHA256
			if _, e := r.GetPublishedAsset(ctx, digest); !errors.Is(e, store.ErrNotFound) {
				t.Fatal("draft exposed", e)
			}
			if _, e := db.ExecContext(ctx, "UPDATE publication_snapshots SET status='published'; INSERT INTO publication_heads SELECT true,id FROM publication_snapshots LIMIT 1"); e != nil {
				t.Fatal(e)
			}
			b, e := r.GetPublishedAsset(ctx, digest)
			if e != nil || fmt.Sprintf("%x", sha256.Sum256(b)) != digest {
				t.Fatal("published asset unavailable", e)
			}
			if _, e = db.ExecContext(ctx, tc.withdraw); e != nil {
				t.Fatal(e)
			}
			if _, e = r.GetPublishedAsset(ctx, digest); !errors.Is(e, store.ErrNotFound) {
				t.Fatal("withdrawn asset exposed", e)
			}
			for _, bad := range []string{"", "../private", "ABC", "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"} {
				if _, e = r.GetPublishedAsset(ctx, bad); !errors.Is(e, store.ErrNotFound) {
					t.Fatal("invalid/missing asset exposed", e)
				}
			}
		})
	}
}
func TestPublicAssetRejectsMixedDigest(t *testing.T) {
	db, s, ctx := setup(t)
	v := input(t, nil)
	if _, e := s.ImportDraft(ctx, v); e != nil {
		t.Fatal(e)
	}
	next := revisedAsset(t, v, 2)
	if _, e := s.ImportDraft(ctx, next); e != nil {
		t.Fatal(e)
	}
	if _, e := db.ExecContext(ctx, "UPDATE publication_snapshots SET status='published'; INSERT INTO publication_heads SELECT true,snapshot_id FROM publication_members WHERE package_id='fractions-revised' LIMIT 1"); e != nil {
		t.Fatal(e)
	}
	r := assetReader(t, s)
	if _, e := r.GetPublishedAsset(ctx, next.Package().Assets[0].SHA256); e != nil {
		t.Fatal(e)
	}
	if _, e := db.ExecContext(ctx, "UPDATE publication_members SET package_id=$1,package_version=$2,version=1 WHERE snapshot_id=(SELECT snapshot_id FROM publication_heads) AND kind='unit'", v.Package().ID, v.Package().Version); e != nil {
		t.Fatal(e)
	}
	for _, p := range []content.Package{v.Package(), next.Package()} {
		if _, e := r.GetPublishedAsset(ctx, p.Assets[0].SHA256); !errors.Is(e, store.ErrNotFound) {
			t.Fatal("mismatched binding exposed", e)
		}
	}
}
func TestPublicAssetRetainsOtherEffectiveReference(t *testing.T) {
	db, s, ctx := setup(t)
	v := input(t, func(p *content.Package) {
		p.ID = "shared-asset-test"
		u := p.Units[0]
		u.ID = "second-explanation"
		p.Units = append(p.Units, u)
	})
	if _, e := s.ImportDraft(ctx, v); e != nil {
		t.Fatal(e)
	}
	if _, e := db.ExecContext(ctx, "UPDATE publication_snapshots SET status='published'; INSERT INTO publication_heads SELECT true,id FROM publication_snapshots LIMIT 1"); e != nil {
		t.Fatal(e)
	}
	if _, e := db.ExecContext(ctx, "UPDATE publication_members SET availability='withdrawn' WHERE kind='unit' AND id='second-explanation'"); e != nil {
		t.Fatal(e)
	}
	if _, e := assetReader(t, s).GetPublishedAsset(ctx, v.Package().Assets[0].SHA256); e != nil {
		t.Fatal("other active reference lost", e)
	}
}
