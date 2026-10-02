package question

import (
	"context"
	"sort"
)

type currentReferences struct {
	knowledge map[Ref]FixedKnowledge
	units     map[Ref]FixedUnit
	assets    map[string]string
}

func newCurrentReferences(refs ReferenceSnapshot) currentReferences {
	out := currentReferences{knowledge: map[Ref]FixedKnowledge{}, units: map[Ref]FixedUnit{}, assets: map[string]string{}}
	for _, k := range refs.Knowledge {
		out.knowledge[Ref{ID: k.Identity.ID, Version: k.Identity.Version}] = k
	}
	for _, u := range refs.Units {
		out.units[Ref{ID: u.Identity.ID, Version: u.Identity.Version}] = u
	}
	for _, a := range refs.Assets {
		out.assets[a.ID] = a.SHA256
	}
	return out
}
func (r currentReferences) bodyEligible(k Ref, coverage []ObjectiveCoverage, units []Ref, assets []AssetRef) bool {
	if _, ok := r.knowledge[k]; !ok {
		return false
	}
	for _, c := range coverage {
		know, ok := r.knowledge[c.Knowledge]
		if !ok {
			return false
		}
		for _, goal := range c.ObjectiveIndices {
			if goal < 0 || goal >= len(know.Objectives) {
				return false
			}
		}
	}
	for _, u := range units {
		if _, ok := r.units[u]; !ok {
			return false
		}
	}
	for _, a := range assets {
		if r.assets[a.ID] != a.SHA256 {
			return false
		}
	}
	return true
}

// Each instance is considered once globally. Mapping it to several nodes does not increase bank totals.
func ComputeCoverage(ctx context.Context, c Candidate, refs ReferenceSnapshot, head *string) (CoverageReport, error) {
	out := CoverageReport{KnowledgeHead: refs.KnowledgeHead, QuestionHead: head, Nodes: Page[CoverageNode]{Items: []CoverageNode{}}}
	if err := ctx.Err(); err != nil {
		return out, err
	}
	r := newCurrentReferences(refs)
	out.PublishedKnowledge = len(r.knowledge)
	availableTemplates := map[Ref]bool{}
	templateIDs := map[Ref]bool{}
	for _, t := range c.Templates {
		ref := Ref{ID: t.ID, Version: t.Version}
		templateIDs[ref] = true
		availableTemplates[ref] = r.bodyEligible(t.Knowledge, t.Coverage, t.Units, t.Assets)
		if availableTemplates[ref] {
			out.ApprovedTemplates++
		}
	}
	available := map[string]Instance{}
	all := map[Ref]Instance{}
	byTemplate := map[Ref][]Instance{}
	byKnowledge := map[Ref][]Instance{}
	for _, i := range c.Instances {
		if err := ctx.Err(); err != nil {
			return out, err
		}
		all[Ref{ID: i.Identity.ID, Version: i.Identity.Version}] = i
		b := i.Body
		valid := r.bodyEligible(b.Knowledge, b.Coverage, b.Units, b.Assets)
		if i.Origin == "template" {
			valid = valid && i.Template != nil
			if i.Template != nil {
				valid = valid && availableTemplates[Ref{ID: i.Template.ID, Version: i.Template.Version}]
			}
		}
		if !valid {
			continue
		}
		if _, exists := available[i.Identity.ID]; exists {
			out.DuplicateInstances++
			continue
		}
		available[i.Identity.ID] = i
		out.EffectiveInstances++
		if i.Origin == "fixed" {
			out.FixedQuestions++
		} else {
			ref := Ref{ID: i.Template.ID, Version: i.Template.Version}
			byTemplate[ref] = append(byTemplate[ref], i)
		}
		for _, mapping := range i.Body.Coverage {
			if len(mapping.ObjectiveIndices) > 0 {
				byKnowledge[mapping.Knowledge] = append(byKnowledge[mapping.Knowledge], i)
				if len(byKnowledge[mapping.Knowledge]) > 1000 {
					return out, ErrLimitExceeded
				}
			}
		}
	}
	identities := map[Ref]Identity{}
	for ref, k := range r.knowledge {
		identities[ref] = k.Identity
	}
	for _, known := range c.Manifest.Resolved {
		if known.Kind == "knowledge" {
			if _, ok := identities[known.Ref]; !ok {
				identities[known.Ref] = Identity{ID: known.ID, Version: known.Version, SHA256: known.SHA256}
			}
		}
	}
	blueprints := map[Ref][]Blueprint{}
	for _, b := range c.Blueprints {
		blueprints[b.Knowledge] = append(blueprints[b.Knowledge], b)
		if _, ok := identities[b.Knowledge]; !ok {
			identities[b.Knowledge] = Identity{ID: b.Knowledge.ID, Version: b.Knowledge.Version}
		}
	}
	nodeOf := func(ref Ref) CoverageNode {
		node := CoverageNode{Knowledge: identities[ref], CoreObjectiveIndices: []int{}, CoveredObjectiveIndices: []int{}, SupplementaryObjectiveIndices: []int{}, Reasons: []Issue{}}
		for _, i := range byKnowledge[ref] {
			node.EffectiveInstances++
			if i.Origin == "fixed" {
				node.FixedInstances++
			} else {
				node.GeneratedInstances++
			}
		}
		return node
	}
	for ref := range identities {
		if err := ctx.Err(); err != nil {
			return out, err
		}
		bs := blueprints[ref]
		if len(bs) == 0 {
			node := nodeOf(ref)
			node.Reasons = append(node.Reasons, Issue{Code: "BLUEPRINT_REQUIRED", Path: "/blueprints", Message: "No assessment blueprint is defined."})
			out.Nodes.Items = append(out.Nodes.Items, node)
			continue
		}
		for _, b := range bs {
			node := nodeOf(ref)
			_, sha, err := canonical("question-blueprint-v1", b)
			if err != nil {
				return out, err
			}
			node.Blueprint = &Identity{ID: b.ID, Version: b.Version, SHA256: sha}
			node.CoreObjectiveIndices = append([]int{}, b.CoreObjectiveIndices...)
			pool := map[string]Instance{}
			validSources := true
			for _, s := range b.Sources {
				if s.Kind == "template" {
					if !templateIDs[s.Ref] {
						validSources = false
					}
					for _, i := range byTemplate[s.Ref] {
						pool[i.Identity.ID] = i
					}
				} else if s.Kind == "instance" {
					i, exists := all[s.Ref]
					if !exists || i.Origin != "fixed" {
						validSources = false
					}
					if _, ok := available[i.Identity.ID]; ok && exists {
						pool[i.Identity.ID] = i
					}
				} else {
					validSources = false
				}
			}
			candidates := []CandidateCoverage{}
			covered := map[int]bool{}
			for _, i := range pool {
				goals := instanceObjectives(i, ref)
				if len(goals) == 0 {
					validSources = false
					continue
				}
				candidates = append(candidates, CandidateCoverage{InstanceID: i.Identity.ID, ObjectiveIndices: goals})
				for _, goal := range goals {
					covered[goal] = true
				}
			}
			node.AssessmentInstances = len(candidates)
			for goal := range covered {
				node.CoveredObjectiveIndices = append(node.CoveredObjectiveIndices, goal)
			}
			sort.Ints(node.CoveredObjectiveIndices)
			k, known := r.knowledge[ref]
			core := map[int]bool{}
			coreValid := known
			for _, goal := range b.CoreObjectiveIndices {
				if goal < 0 || goal >= len(k.Objectives) || core[goal] {
					coreValid = false
				}
				core[goal] = true
			}
			for goal := range k.Objectives {
				if !core[goal] {
					node.SupplementaryObjectiveIndices = append(node.SupplementaryObjectiveIndices, goal)
				}
			}
			_, feasible, err := FiveQuestionCover(b.CoreObjectiveIndices, candidates)
			if err != nil {
				return out, err
			}
			node.FiveQuestionFeasible = feasible
			node.Ready = known && coreValid && validSources && feasible && b.RuleVersion == 1 && b.QuestionCount == 5 && b.PassCount == 4
			if !known {
				node.Reasons = append(node.Reasons, Issue{Code: "REFERENCE_UNAVAILABLE", Path: "/knowledge", Message: "The fixed knowledge version is unavailable."})
			}
			if !node.Ready {
				node.Reasons = append(node.Reasons, Issue{Code: "FIVE_QUESTION_COVER_REQUIRED", Path: "/blueprints", Message: "Five distinct available questions must cover every core objective."})
			}
			out.Nodes.Items = append(out.Nodes.Items, node)
		}
	}
	sort.Slice(out.Nodes.Items, func(i, j int) bool {
		a, b := out.Nodes.Items[i], out.Nodes.Items[j]
		if a.Knowledge.ID != b.Knowledge.ID {
			return a.Knowledge.ID < b.Knowledge.ID
		}
		if a.Knowledge.Version != b.Knowledge.Version {
			return a.Knowledge.Version < b.Knowledge.Version
		}
		if a.Blueprint == nil {
			return b.Blueprint != nil
		}
		if b.Blueprint == nil {
			return false
		}
		return a.Blueprint.ID < b.Blueprint.ID
	})
	out.Nodes.Total = len(out.Nodes.Items)
	out.Nodes.Limit = out.Nodes.Total
	return out, ctx.Err()
}
