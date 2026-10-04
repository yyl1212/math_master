package contentaudit

import (
	"bytes"
	"context"
	"fmt"
	"github.com/yyl1212/math_master/backend/internal/content"
	"github.com/yyl1212/math_master/backend/internal/question"
	"path/filepath"
)

func LoadDraft(ctx context.Context, root string) (DraftInput, error) {
	var out DraftInput
	load := func(path string, limit int) ([]byte, error) {
		if e := ctx.Err(); e != nil {
			return nil, e
		}
		p, e := fixedPath(root, path)
		if e != nil {
			return nil, e
		}
		return readBounded(ctx, p, limit)
	}
	b, e := load("content/catalogue/domains.json", content.MaxPackageBytes)
	if e != nil {
		return out, e
	}
	out.Catalogue, e = content.DecodeCatalogue(bytes.NewReader(b))
	if e != nil {
		return out, e
	}
	b, e = load("content/packages/elementary-foundations.v1.json", content.MaxWorkflowPackageBytes)
	if e != nil {
		return out, e
	}
	out.Content, e = content.DecodePackage(bytes.NewReader(b))
	if e != nil {
		return out, e
	}
	out.AssetsRoot = filepath.Join(root, "content/assets")
	out.Questions = []question.QuestionPackage{}
	for _, name := range []string{"numbers", "operations", "fractions", "decimals", "ratios"} {
		b, e = load("content/questions/elementary-foundations-"+name+".v1.json", question.MaxPackageBytes)
		if e != nil {
			return out, e
		}
		p, e := question.DecodePackage(bytes.NewReader(b))
		if e != nil {
			return out, e
		}
		out.Questions = append(out.Questions, p)
	}
	return out, nil
}
func CheckDraft(ctx context.Context, in DraftInput) (DraftFacts, error) {
	return checkDraftWithReader(ctx, in, func(ctx context.Context, a content.Asset) ([]byte, error) {
		p, e := fixedPath(in.AssetsRoot, a.Path)
		if e != nil {
			return nil, e
		}
		return readBounded(ctx, p, 1<<20)
	})
}
func checkDraftWithReader(ctx context.Context, in DraftInput, reader content.AssetReader) (DraftFacts, error) {
	var out DraftFacts
	if e := ctx.Err(); e != nil {
		return out, e
	}
	v, report := content.ValidateWorkflow(ctx, in.Catalogue, in.Content, reader)
	if e := ctx.Err(); e != nil {
		return out, e
	}
	if !report.ReadyToSubmit || !v.Verify() {
		return out, fmt.Errorf("%w: content structure/completeness %d/%d", ErrInvalid, report.StructuralTotal, report.CompletenessTotal)
	}
	if len(in.Content.Paths) != 1 {
		return out, ErrInvalid
	}
	p := v.Package()
	out.Content = content.Snapshot{CatalogueVersion: in.Catalogue.Version, Knowledge: p.Knowledge, Units: p.Units, Paths: p.Paths, Assets: p.Assets, Bindings: []content.AssetBinding{}}
	out.Path = p.Paths[0]
	out.References = question.ReferenceSnapshot{CatalogueVersion: in.Catalogue.Version, CatalogueSHA256: v.CatalogueSHA256(), Knowledge: []question.FixedKnowledge{}, Units: []question.FixedUnit{}, Assets: []content.AssetView{}}
	for _, k := range p.Knowledge {
		out.References.Knowledge = append(out.References.Knowledge, question.FixedKnowledge{Identity: question.Identity{ID: k.ID, Version: k.Version, SHA256: content.Digest(k)}, Title: k.Title, TitleZh: k.TitleZh, Objectives: k.Objectives})
	}
	assets := map[string]content.Asset{}
	for _, a := range p.Assets {
		assets[a.ID] = a
		out.References.Assets = append(out.References.Assets, content.AssetView{ID: a.ID, SHA256: a.SHA256, Author: a.Author, License: a.License, Attribution: a.Attribution, Knowledge: a.Knowledge})
	}
	for _, u := range p.Units {
		out.References.Units = append(out.References.Units, question.FixedUnit{Identity: question.Identity{ID: u.ID, Version: u.Version, SHA256: content.Digest(u)}, Knowledge: u.Knowledge, AssetIDs: u.AssetIDs})
		for _, id := range u.AssetIDs {
			a := assets[id]
			out.Content.Bindings = append(out.Content.Bindings, content.AssetBinding{Unit: content.VersionRef{ID: u.ID, Version: u.Version}, AssetID: id, SHA256: a.SHA256})
		}
	}
	out.Sealed = []question.SealedPackage{}
	out.QuestionReports = []question.ValidationReport{}
	seen := map[string]bool{}
	for _, p := range in.Questions {
		if seen[p.ID] {
			return out, ErrInvalid
		}
		seen[p.ID] = true
		s, r, e := question.ValidateAndSeal(ctx, question.DraftInput{CatalogueVersion: in.Catalogue.Version, QuestionPackage: p, SourceMap: []question.SourceLink{}}, out.References)
		if canceled := ctx.Err(); canceled != nil {
			return out, canceled
		}
		if e != nil || !r.ReadyToSubmit {
			return out, fmt.Errorf("%w: question %s structural/completeness %d/%d", ErrInvalid, p.ID, r.StructuralTotal, r.CompletenessTotal)
		}
		out.Sealed = append(out.Sealed, s)
		out.QuestionReports = append(out.QuestionReports, r)
	}
	return out, ctx.Err()
}

// Read-only convenience for operator evidence preparation; it never writes a source file.
func ReadSourceMap(path string) (SourceMap, error) {
	var m SourceMap
	raw, e := readBounded(context.Background(), path, MaxSourceMapBytes)
	if e != nil {
		return m, e
	}
	e = DecodeSourceMap(bytes.NewReader(raw), &m)
	return m, e
}
