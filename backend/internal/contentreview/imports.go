package contentreview

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"sort"
	"strings"

	"github.com/yyl1212/math_master/backend/internal/content"
	"github.com/yyl1212/math_master/backend/internal/contentaudit"
	"github.com/yyl1212/math_master/backend/internal/publication"
	"github.com/yyl1212/math_master/backend/internal/question"
)

func linkKey(l publication.SourceLink) string { l.LegacyID = ""; l.Note = ""; return content.Digest(l) }

func BuildImports(ctx context.Context, in PrepareInput, scope ReviewScope) (Imports, error) {
	out := Imports{Questions: []question.DraftInput{}}
	verified, e := BuildScope(ctx, in)
	if e != nil {
		return out, e
	}
	if content.Digest(verified) != content.Digest(scope) {
		return out, ErrInvalid
	}
	facts := verified.facts
	bindings := map[string][]content.VersionRef{}
	bind := func(key string, refs ...content.VersionRef) { bindings[key] = append(bindings[key], refs...) }
	for _, k := range facts.Content.Knowledge {
		bind(ObjectKey(object("knowledge", k.ID, k.Version, content.Digest(k))), content.VersionRef{ID: k.ID, Version: k.Version})
	}
	for _, u := range facts.Content.Units {
		bind(ObjectKey(object("unit", u.ID, u.Version, content.Digest(u))), u.Knowledge)
	}
	bindings[ObjectKey(object("path", facts.Path.ID, facts.Path.Version, content.Digest(facts.Path)))] = []content.VersionRef{}
	for _, a := range facts.Content.Assets {
		bind(ObjectKey(contentauditObjectAsset(a)), a.Knowledge)
	}
	for _, s := range facts.Sealed {
		for _, t := range s.Package.Templates {
			_, hash, e := question.CanonicalTemplate(t)
			if e != nil {
				return out, e
			}
			key := ObjectKey(object("template", t.ID, t.Version, hash))
			bind(key, t.Knowledge)
			for _, c := range t.Coverage {
				bind(key, c.Knowledge)
			}
		}
		for _, i := range s.Instances {
			if i.Origin == "fixed" {
				key := ObjectKey(instanceObject(i.Identity))
				bind(key, i.Body.Knowledge)
				for _, c := range i.Body.Coverage {
					bind(key, c.Knowledge)
				}
			}
		}
	}
	for p, r := range facts.QuestionReports {
		for _, n := range r.Coverage {
			if n.Blueprint == nil {
				return out, ErrInvalid
			}
			for _, b := range facts.Sealed[p].Package.Blueprints {
				if b.ID == n.Blueprint.ID && b.Version == n.Blueprint.Version {
					bind(ObjectKey(object("blueprint", b.ID, b.Version, n.Blueprint.SHA256)), b.Knowledge)
				}
			}
		}
	}
	sources := map[string]contentaudit.MappedSource{}
	for _, s := range in.Sources.Mapping.Sources {
		sources[s.ID] = s
	}
	used := map[content.VersionRef]map[string]contentaudit.MappedSource{}
	for _, o := range in.Sources.Mapping.Objects {
		refs, ok := bindings[ObjectKey(contentaudit.ObjectIdentity{Kind: o.Kind, ID: o.ID, Version: o.Version, SHA256: o.SHA256})]
		if !ok {
			return out, ErrInvalid
		}
		for _, ref := range refs {
			if used[ref] == nil {
				used[ref] = map[string]contentaudit.MappedSource{}
			}
			for _, id := range o.SourceIDs {
				used[ref][id] = sources[id]
			}
		}
	}
	// Route citations aggregate the same per-knowledge records; they are not assigned
	// to unrelated nodes. Every route citation must already have a precise association.
	assigned := map[string]bool{}
	for _, records := range used {
		for id := range records {
			assigned[id] = true
		}
	}
	for _, o := range in.Sources.Mapping.Objects {
		if o.Kind == "path" {
			for _, id := range o.SourceIDs {
				if !assigned[id] {
					return out, ErrInvalid
				}
			}
		}
	}
	links := []publication.SourceLink{}
	for ref, records := range used {
		groups := map[string][]contentaudit.MappedSource{}
		base := map[string]publication.SourceLink{}
		for _, s := range records {
			l := publication.SourceLink{Knowledge: ref, BatchSHA256: in.Sources.SnapshotID, RelativePath: s.Path, SHA256: s.FileSHA256}
			key := linkKey(l)
			base[key] = l
			groups[key] = append(groups[key], s)
		}
		for key, group := range groups {
			sort.Slice(group, func(i, j int) bool { return group[i].ID < group[j].ID })
			type description struct {
				SourceID   string   `json:"sourceId"`
				RecordID   string   `json:"recordId"`
				LegacyIDs  []string `json:"legacyIds"`
				Use        string   `json:"use"`
				Conditions string   `json:"conditions"`
			}
			notes := []description{}
			legacy := map[string]bool{}
			for _, s := range group {
				ids := append([]string{}, s.LegacyIDs...)
				sort.Strings(ids)
				for _, id := range ids {
					legacy[id] = true
				}
				notes = append(notes, description{s.ID, s.RecordID, ids, s.Use, s.ConditionsNote})
			}
			old := []string{}
			for id := range legacy {
				old = append(old, id)
			}
			sort.Strings(old)
			note, e := json.Marshal(notes)
			if e != nil {
				return out, ErrInvalid
			}
			l := base[key]
			l.LegacyID = strings.Join(old, "; ")
			l.Note = string(note)
			links = append(links, l)
		}
	}
	sort.Slice(links, func(i, j int) bool {
		a, b := links[i], links[j]
		if a.Knowledge.ID != b.Knowledge.ID {
			return a.Knowledge.ID < b.Knowledge.ID
		}
		return linkKey(a) < linkKey(b)
	})
	out.Knowledge = publication.DraftInput{CatalogueVersion: in.Selected.Input.Catalogue.Version, Package: in.Selected.Input.Content, AssetBytes: []publication.AssetInput{}, SourceMap: links}
	for _, a := range out.Knowledge.Package.Assets {
		out.Knowledge.AssetBytes = append(out.Knowledge.AssetBytes, publication.AssetInput{ID: a.ID, Base64: base64.StdEncoding.EncodeToString(in.Selected.Assets[a.ID])})
	}
	assets, e := publication.DraftAssets(out.Knowledge)
	if e != nil {
		return out, e
	}
	v, gate := content.ValidateWorkflow(ctx, in.Selected.Input.Catalogue, out.Knowledge.Package, func(ctx context.Context, a content.Asset) ([]byte, error) {
		if e := ctx.Err(); e != nil {
			return nil, e
		}
		return assets[a.ID], nil
	})
	if !v.Verify() || !gate.ReadyToSubmit {
		return out, ErrInvalid
	}
	for p, s := range facts.Sealed {
		known := map[content.VersionRef]bool{}
		for _, t := range s.Package.Templates {
			known[t.Knowledge] = true
			for _, c := range t.Coverage {
				known[c.Knowledge] = true
			}
		}
		for _, i := range s.Instances {
			known[i.Body.Knowledge] = true
			for _, c := range i.Body.Coverage {
				known[c.Knowledge] = true
			}
		}
		for _, b := range s.Package.Blueprints {
			known[b.Knowledge] = true
		}
		input := question.DraftInput{CatalogueVersion: out.Knowledge.CatalogueVersion, QuestionPackage: in.Selected.Input.Questions[p], SourceMap: []question.SourceLink{}}
		for _, l := range links {
			if known[l.Knowledge] {
				input.SourceMap = append(input.SourceMap, l)
			}
		}
		raw, e := json.Marshal(input)
		if e != nil {
			return out, e
		}
		if _, e = question.DecodeDraft(bytes.NewReader(raw)); e != nil {
			return out, e
		}
		sealed, gate, e := question.ValidateAndSeal(ctx, input, facts.References)
		if e != nil {
			return out, e
		}
		if !gate.ReadyToSubmit || sealed.PackageSHA != s.PackageSHA || content.Digest(sealed.Instances) != content.Digest(s.Instances) {
			return out, ErrInvalid
		}
		out.Questions = append(out.Questions, input)
	}
	sort.Slice(out.Questions, func(i, j int) bool { return out.Questions[i].QuestionPackage.ID < out.Questions[j].QuestionPackage.ID })
	return out, ctx.Err()
}
func contentauditObjectAsset(a content.Asset) contentaudit.ObjectIdentity {
	return contentaudit.ObjectIdentity{Kind: "asset", ID: a.ID, SHA256: a.SHA256}
}
