package store_test

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"github.com/yyl1212/math_master/backend/internal/content"
	"github.com/yyl1212/math_master/backend/internal/store"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCatalogueSearchAndPagination(t *testing.T) {
	_, s, ctx := setup(t)
	if _, e := s.ImportDraft(ctx, input(t, nil)); e != nil {
		t.Fatal(e)
	}
	items, total, e := s.ListDomains(ctx, "", 5, 0)
	if e != nil || total != 16 || len(items) != 5 || items[0].ContentStatus != "planned" {
		t.Fatal(total, len(items), e)
	}
	all, total, e := s.ListDomains(ctx, "", 100, 0)
	n := 0
	for _, d := range all {
		n += len(d.Topics)
	}
	if e != nil || n != 56 || total != 16 {
		t.Fatal(n, total, e)
	}
	for _, q := range []string{"Markov", "概率"} {
		items, total, e := s.ListDomains(ctx, q, 20, 0)
		if e != nil || total == 0 || len(items) == 0 {
			t.Fatal(q, total, e)
		}
	}
	for _, q := range []string{"' OR 1=1 --", "%", "_"} {
		_, total, e := s.ListDomains(ctx, q, 20, 0)
		if e != nil || total != 0 {
			t.Fatal("unsafe or wildcard search", q, total, e)
		}
	}
	items, _, e = s.ListDomains(ctx, "", 5, 5)
	if e != nil || len(items) != 5 || items[0].Order != 6 {
		t.Fatal(items, e)
	}
}
func TestPublicReadsHideDraftAndWithdrawnVersions(t *testing.T) {
	db, s, ctx := setup(t)
	p := input(t, nil)
	if _, e := s.ImportDraft(ctx, p); e != nil {
		t.Fatal(e)
	}
	id := p.Package().Knowledge[0].ID
	if _, e := s.GetPublishedKnowledge(ctx, id); !errors.Is(e, store.ErrNotFound) {
		t.Fatal("draft exposed")
	}
	if _, e := s.GetPublishedPath(ctx, p.Package().Paths[0].ID); !errors.Is(e, store.ErrNotFound) {
		t.Fatal("draft path exposed")
	}
	fixture, e := os.ReadFile("testdata/fixture.sql")
	if e != nil {
		t.Fatal(e)
	}
	if _, e = db.ExecContext(ctx, string(fixture)); e != nil {
		t.Fatal(e)
	}
	if _, e = s.GetPublishedKnowledge(ctx, id); e != nil {
		t.Fatal(e)
	}
	if _, e = db.ExecContext(ctx, "UPDATE publication_members SET availability='withdrawn' WHERE kind='knowledge' AND id=$1", id); e != nil {
		t.Fatal(e)
	}
	for _, k := range p.Package().Knowledge {
		if _, e := s.GetPublishedKnowledge(ctx, k.ID); !errors.Is(e, store.ErrNotFound) {
			t.Fatal("withdrawn prerequisite closure leaked", k.ID, e)
		}
	}
	if _, e := s.GetPublishedPath(ctx, p.Package().Paths[0].ID); !errors.Is(e, store.ErrNotFound) {
		t.Fatal("path with withdrawn prerequisite leaked")
	}
	if _, e = db.ExecContext(ctx, "UPDATE publication_snapshots SET status='withdrawn'"); e != nil {
		t.Fatal(e)
	}
	if _, e = s.GetPublishedKnowledge(ctx, id); !errors.Is(e, store.ErrNotFound) {
		t.Fatal("withdrawn snapshot exposed")
	}
}
func TestPublicKnowledgeKeepsPrerequisiteVersionRefs(t *testing.T) {
	db, s, ctx := setup(t)
	p := input(t, nil)
	if _, e := s.ImportDraft(ctx, p); e != nil {
		t.Fatal(e)
	}
	b, _ := os.ReadFile("testdata/fixture.sql")
	if _, e := db.ExecContext(ctx, string(b)); e != nil {
		t.Fatal(e)
	}
	v, e := s.GetPublishedKnowledge(ctx, "equivalent-fractions")
	if e != nil || v.Knowledge.Version != 1 || len(v.Units) != 1 || len(v.Assets) != 1 {
		t.Fatal(v, e)
	}
	for _, r := range v.Knowledge.Relations {
		k, e := s.GetPublishedKnowledge(ctx, r.Target.ID)
		if e != nil || k.Knowledge.Version != r.Target.Version {
			t.Fatal("relation outside valid snapshot")
		}
	}
	path, e := s.GetPublishedPath(ctx, p.Package().Paths[0].ID)
	if e != nil || len(path.Knowledge) != 10 {
		t.Fatal(e)
	}
	domain, e := s.GetDomain(ctx, "elementary-mathematics")
	if e != nil || domain.ContentStatus != "published" || domain.PublishedKnowledgeCount != 10 || len(domain.Paths) != 1 {
		t.Fatal(domain, e)
	}
}

func revisedAsset(t *testing.T, v content.ValidatedPackage, unitVersion int) content.ValidatedPackage {
	t.Helper()
	p := v.Package()
	p.ID = "fractions-revised"
	p.Units[0].Version = unitVersion
	b, _ := v.AssetBytes(p.Assets[0].ID)
	b = []byte(strings.Replace(string(b), "<title>", "<title>Revised ", 1))
	p.Assets[0].SHA256 = fmt.Sprintf("%x", sha256.Sum256(b))
	root := t.TempDir()
	if e := os.WriteFile(filepath.Join(root, p.Assets[0].Path), b, 0600); e != nil {
		t.Fatal(e)
	}
	next, r := content.ValidateAndSeal(v.Catalogue(), p, root)
	if len(r.Errors) > 0 {
		t.Fatal(r.Errors)
	}
	return next
}
func TestUnitAssetDigestBindingIsImmutableAcrossPackages(t *testing.T) {
	_, s, ctx := setup(t)
	v := input(t, nil)
	if _, e := s.ImportDraft(ctx, v); e != nil {
		t.Fatal(e)
	}
	if _, e := s.ImportDraft(ctx, revisedAsset(t, v, 1)); !errors.Is(e, store.ErrImmutableConflict) {
		t.Fatal("same unit version accepted different asset digest", e)
	}
	if _, e := s.ImportDraft(ctx, revisedAsset(t, v, 2)); e != nil {
		t.Fatal("new unit version was rejected", e)
	}
}
func TestPublishedUnitRejectsMixedAssetDigest(t *testing.T) {
	db, s, ctx := setup(t)
	v := input(t, nil)
	if _, e := s.ImportDraft(ctx, v); e != nil {
		t.Fatal(e)
	}
	next := revisedAsset(t, v, 2)
	if _, e := s.ImportDraft(ctx, next); e != nil {
		t.Fatal(e)
	}
	if _, e := db.ExecContext(ctx, "UPDATE publication_snapshots SET status='published';INSERT INTO publication_heads SELECT true,snapshot_id FROM publication_members WHERE package_id='fractions-revised' LIMIT 1"); e != nil {
		t.Fatal(e)
	}
	k, e := s.GetPublishedKnowledge(ctx, "equivalent-fractions")
	if e != nil || len(k.Units) != 1 || k.Units[0].Version != 2 {
		t.Fatal(k, e)
	}
	if _, e := db.ExecContext(ctx, "UPDATE publication_members SET package_id=$1,package_version=$2,version=1 WHERE snapshot_id=(SELECT snapshot_id FROM publication_heads) AND kind='unit'", v.Package().ID, v.Package().Version); e != nil {
		t.Fatal(e)
	}
	k, e = s.GetPublishedKnowledge(ctx, "equivalent-fractions")
	if e != nil || len(k.Units) != 0 {
		t.Fatal("old unit rendered a replacement asset digest", e)
	}
}
