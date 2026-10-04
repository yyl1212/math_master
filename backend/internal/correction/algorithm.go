package correction

import (
	"encoding/json"
	"github.com/yyl1212/math_master/backend/internal/auth"
	"github.com/yyl1212/math_master/backend/internal/question"
	"reflect"
	"sort"
)

const AlgorithmV1 = 1
const AlgorithmV1Name = "correction-grade-v1"

func RegisteredAlgorithm(version int) bool { return version == AlgorithmV1 }

// ComposeBasis follows explicit approved instance edges; sibling conflicts have
// no ordering rule. The caller translates ErrConflict into awaiting_review.
func ComposeBasis(original Basis, parents []Basis, approved []Basis) (Basis, error) {
	raw, e := json.Marshal(original)
	if e != nil {
		return Basis{}, e
	}
	var out Basis
	if e = json.Unmarshal(raw, &out); e != nil {
		return Basis{}, e
	}
	n := len(original.OriginalItems)
	if n < 1 || n > 5 || len(original.OriginalAnswers) != n {
		return out, auth.ErrInvalidInput
	}
	out.EffectiveItems = append([]question.Instance{}, original.OriginalItems...)
	type edge struct {
		from question.Identity
		to   question.Instance
	}
	edges := make([]map[question.Identity]question.Instance, n)
	for i := range edges {
		edges[i] = map[question.Identity]question.Instance{}
	}
	all := append(append([]Basis{}, parents...), approved...)
	if len(original.PlanRefs) > 0 {
		all = append([]Basis{original}, all...)
	}
	if len(all) > 100 {
		return out, auth.ErrInvalidInput
	}
	cases := map[string]bool{}
	refs := map[PlanRef]bool{}
	results := map[string]bool{}
	knownDeps := append([]Dependency{}, original.EffectiveDeps...)
	audit := append([]Dependency{}, original.AuditDeps...)
	for _, id := range original.HandledCaseIDs {
		cases[id] = true
	}
	for _, id := range original.ParentResultIDs {
		results[id] = true
	}
	for _, p := range all {
		if !reflect.DeepEqual(p.OriginalSeal, original.OriginalSeal) || !reflect.DeepEqual(p.OriginalAnswers, original.OriginalAnswers) || len(p.OriginalItems) != n || len(p.EffectiveItems) != n || len(p.PlanRefs) == 0 {
			return out, ErrSourceStale
		}
		for _, id := range p.HandledCaseIDs {
			cases[id] = true
		}
		for _, r := range p.PlanRefs {
			if !ValidPlanRef(r) {
				return out, auth.ErrInvalidInput
			}
			refs[r] = true
		}
		for _, id := range p.ParentResultIDs {
			results[id] = true
		}
		if p.ParentResultID != nil {
			results[*p.ParentResultID] = true
			out.ParentResultID = p.ParentResultID
		}
		knownDeps = append(knownDeps, p.EffectiveDeps...)
		audit = append(audit, p.AuditDeps...)
		audit = append(audit, p.EffectiveDeps...)
		for pos := 0; pos < n; pos++ {
			from, to := p.OriginalItems[pos], p.EffectiveItems[pos]
			if from.Identity == to.Identity {
				if !reflect.DeepEqual(from, to) {
					return out, ErrSourceStale
				}
				continue
			}
			if existing, ok := edges[pos][from.Identity]; ok && !reflect.DeepEqual(existing, to) {
				return out, ErrConflict
			}
			edges[pos][from.Identity] = to
		}
	}
	for pos := 0; pos < n; pos++ {
		current := original.OriginalItems[pos]
		seen := map[question.Identity]bool{}
		visited := 0
		for {
			if seen[current.Identity] {
				return out, ErrConflict
			}
			seen[current.Identity] = true
			next, ok := edges[pos][current.Identity]
			if !ok {
				break
			}
			current = next
			visited++
		}
		if visited != len(edges[pos]) {
			return out, ErrSourceStale
		}
		out.EffectiveItems[pos] = current
	}
	out.HandledCaseIDs = sortedStrings(cases)
	out.ParentResultIDs = sortedStrings(results)
	out.PlanRefs = []PlanRef{}
	for r := range refs {
		out.PlanRefs = append(out.PlanRefs, r)
	}
	sort.Slice(out.PlanRefs, func(i, j int) bool {
		a, b := out.PlanRefs[i], out.PlanRefs[j]
		return a.ID < b.ID || a.ID == b.ID && a.Version < b.Version
	})
	audit = append(audit, original.EffectiveDeps...)
	out.AuditDeps = uniqueDependencies(audit)
	out.EffectiveDeps = effectiveDependencies(out, knownDeps)
	return out, nil
}
func sortedStrings(values map[string]bool) []string {
	out := []string{}
	for s := range values {
		out = append(out, s)
	}
	sort.Strings(out)
	return out
}
func dependencyKey(d Dependency) string { raw, _ := json.Marshal(d); return string(raw) }
func uniqueDependencies(deps []Dependency) []Dependency {
	seen := map[string]Dependency{}
	for _, d := range deps {
		seen[dependencyKey(d)] = d
	}
	keys := []string{}
	for k := range seen {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	out := []Dependency{}
	for _, k := range keys {
		out = append(out, seen[k])
	}
	return out
}
func effectiveDependencies(b Basis, known []Dependency) []Dependency {
	out := []Dependency{}
	addIdentity := func(kind string, i question.Identity) {
		v := i.Version
		out = append(out, Dependency{Kind: kind, ID: i.ID, Version: &v, SHA256: i.SHA256})
	}
	for _, d := range known {
		if d.Kind == "knowledge" && d.ID == b.OriginalSeal.Knowledge.ID && d.Version != nil && *d.Version == b.OriginalSeal.Knowledge.Version && d.SHA256 == b.OriginalSeal.Knowledge.SHA256 || d.Kind == "blueprint" && b.OriginalSeal.Blueprint != nil && d.ID == b.OriginalSeal.Blueprint.ID && d.Version != nil && *d.Version == b.OriginalSeal.Blueprint.Version && d.SHA256 == b.OriginalSeal.Blueprint.SHA256 {
			out = append(out, d)
		}
	}
	for _, i := range b.EffectiveItems {
		addIdentity("instance", i.Identity)
		if i.Template != nil {
			addIdentity("template", *i.Template)
		}
		for _, a := range i.Body.Assets {
			out = append(out, Dependency{Kind: "asset", ID: a.ID, SHA256: a.SHA256})
		}
		for _, u := range i.Body.Units {
			for _, d := range known {
				if d.Kind == "unit" && d.ID == u.ID && d.Version != nil && *d.Version == u.Version {
					out = append(out, d)
				}
			}
		}
	}
	return uniqueDependencies(out)
}
