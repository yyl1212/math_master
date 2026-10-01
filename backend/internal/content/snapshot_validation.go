package content

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/yyl1212/math_master/backend/internal/catalogue"
)

// ValidateSnapshot validates content only. Publication evidence is checked separately
// against frozen approved submissions; a successful report is never an approval.
func ValidateSnapshot(ctx context.Context, c catalogue.Catalogue, s Snapshot, reader AssetReader) (WorkflowReport, error) {
	if err := ctx.Err(); err != nil {
		return WorkflowReport{}, err
	}
	edges := 0
	for _, k := range s.Knowledge {
		edges += len(k.Relations)
	}
	if s.CatalogueVersion != c.Version {
		return WorkflowReport{}, ErrValidation
	}
	if len(s.Knowledge) > 1000 || len(s.Units) > 4000 || len(s.Paths) > 200 || len(s.Assets) > 1000 || edges > 16000 {
		return WorkflowReport{}, ErrLimit
	}
	raw, err := json.Marshal(s)
	if err != nil {
		return WorkflowReport{}, ErrValidation
	}
	if len(raw) > MaxSnapshotBytes {
		return WorkflowReport{}, ErrLimit
	}
	s = clone(s)
	known := map[VersionRef]bool{}
	for _, k := range s.Knowledge {
		known[VersionRef{k.ID, k.Version}] = true
	}
	for i, k := range s.Knowledge {
		relations := []Relation{}
		for _, rel := range k.Relations {
			if rel.Kind == "prerequisite" || known[rel.Target] {
				relations = append(relations, rel)
			}
		}
		s.Knowledge[i].Relations = relations
	}
	p := Package{SchemaVersion: 1, ID: "snapshot-validation", Version: 1, Knowledge: s.Knowledge, Units: s.Units, Paths: s.Paths, Assets: s.Assets}
	if p.Knowledge == nil {
		p.Knowledge = []Knowledge{}
	}
	if p.Units == nil {
		p.Units = []Unit{}
	}
	if p.Paths == nil {
		p.Paths = []Path{}
	}
	if p.Assets == nil {
		p.Assets = []Asset{}
	}
	_, base := validateAndSeal(ctx, c, p, reader, MaxSnapshotBytes, 10<<20, true)
	r := reportFrom(base)
	if len(p.Knowledge) > 0 {
		completeness(p, &r)
	}
	pathOrder(p, &r)
	invalid := func(path string) {
		r.StructuralErrors = append(r.StructuralErrors, Issue{"ASSET_BINDING", path, "The immutable unit must bind this exact asset digest."})
	}
	assets := map[string]Asset{}
	for _, a := range p.Assets {
		assets[a.ID] = a
	}
	required := map[string]string{}
	for _, u := range p.Units {
		for _, id := range u.AssetIDs {
			required[fmt.Sprintf("%s:%d:%s", u.ID, u.Version, id)] = assets[id].SHA256
		}
	}
	seen := map[string]bool{}
	for i, b := range s.Bindings {
		key := fmt.Sprintf("%s:%d:%s", b.Unit.ID, b.Unit.Version, b.AssetID)
		sha, ok := required[key]
		if !ok || seen[key] || sha != b.SHA256 {
			invalid(fmt.Sprintf("/bindings/%d", i))
		}
		seen[key] = true
	}
	for key := range required {
		if !seen[key] {
			invalid("/bindings/" + key)
		}
	}
	views := map[VersionRef]KnowledgeView{}
	for _, k := range p.Knowledge {
		if err := ctx.Err(); err != nil {
			return r, err
		}
		ref := VersionRef{k.ID, k.Version}
		v := KnowledgeView{Knowledge: k, Units: []Unit{}, Assets: []AssetView{}}
		used := map[string]bool{}
		for _, u := range p.Units {
			if u.Knowledge == ref {
				v.Units = append(v.Units, u)
				for _, id := range u.AssetIDs {
					used[id] = true
				}
			}
		}
		for _, a := range p.Assets {
			if used[a.ID] {
				v.Assets = append(v.Assets, AssetView{a.ID, a.SHA256, a.Author, a.License, a.Attribution, a.Knowledge})
			}
		}
		b, err := json.Marshal(v)
		if err != nil {
			return r, ErrValidation
		}
		if len(b) > 10<<20 {
			return r, ErrLimit
		}
		views[ref] = v
	}
	for _, path := range p.Paths {
		if err := ctx.Err(); err != nil {
			return r, err
		}
		v := PathView{Path: path, Knowledge: []KnowledgeView{}}
		for _, ref := range path.Nodes {
			v.Knowledge = append(v.Knowledge, views[ref])
		}
		b, err := json.Marshal(v)
		if err != nil {
			return r, ErrValidation
		}
		if len(b) > 10<<20 {
			return r, ErrLimit
		}
	}
	r.totals()
	if !r.ReadyToSubmit {
		return r, ErrValidation
	}
	return r, nil
}
