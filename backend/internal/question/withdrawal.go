package question

import (
	"context"
	"reflect"
	"sort"
)

func ValidateWithdrawalTarget(t WithdrawalTarget) error {
	valid := ValidMathID(t.ID)
	if t.Kind == "instance" {
		valid = ValidInstanceID(t.ID)
	}
	if t.Kind != "template" && t.Kind != "instance" && t.Kind != "blueprint" || !valid || t.Version < 1 || t.Version > 2147483647 {
		return ErrInvalid
	}
	return nil
}
func WithdrawCandidate(ctx context.Context, base Candidate, target WithdrawalTarget) (Candidate, error) {
	if err := ctx.Err(); err != nil {
		return Candidate{}, err
	}
	if err := ValidateWithdrawalTarget(target); err != nil {
		return Candidate{}, err
	}
	c := canonicalValue(reflect.ValueOf(base)).Interface().(Candidate)
	removed := map[string]bool{}
	for _, m := range c.Manifest.Members {
		if m.Identity.Kind == target.Kind && m.Identity.ID == target.ID && m.Identity.Version == target.Version {
			removed[manifestKey(m.Identity)] = true
		}
	}
	if target.Kind == "template" {
		for _, i := range c.Instances {
			if i.Template != nil && i.Template.ID == target.ID && i.Template.Version == target.Version {
				removed["instance/"+i.Identity.ID] = true
			}
		}
	}
	for _, b := range c.Blueprints {
		for _, s := range b.Sources {
			if s.Ref.ID == target.ID && s.Ref.Version == target.Version && (target.Kind == "template" && s.Kind == "template" || target.Kind == "instance" && s.Kind == "instance") {
				removed["blueprint/"+b.ID] = true
			}
		}
	}
	c.Manifest.Members = []ManifestMember{}
	for _, m := range base.Manifest.Members {
		if !removed[manifestKey(m.Identity)] {
			c.Manifest.Members = append(c.Manifest.Members, canonicalValue(reflect.ValueOf(m)).Interface().(ManifestMember))
		}
	}
	c.Templates = []Template{}
	for _, t := range base.Templates {
		if !removed["template/"+t.ID] {
			c.Templates = append(c.Templates, canonicalValue(reflect.ValueOf(t)).Interface().(Template))
		}
	}
	c.Instances = []Instance{}
	for _, i := range base.Instances {
		if !removed["instance/"+i.Identity.ID] {
			c.Instances = append(c.Instances, canonicalValue(reflect.ValueOf(i)).Interface().(Instance))
		}
	}
	c.Blueprints = []Blueprint{}
	for _, b := range base.Blueprints {
		if !removed["blueprint/"+b.ID] {
			c.Blueprints = append(c.Blueprints, canonicalValue(reflect.ValueOf(b)).Interface().(Blueprint))
		}
	}
	used := candidateReferences(c)
	c.Manifest.Resolved = []KnownObject{}
	for _, r := range base.Manifest.Resolved {
		if used[objectKey(r.Kind, r.Ref)] {
			c.Manifest.Resolved = append(c.Manifest.Resolved, r)
		}
	}
	sort.Slice(c.Manifest.Members, func(i, j int) bool {
		return manifestKey(c.Manifest.Members[i].Identity) < manifestKey(c.Manifest.Members[j].Identity)
	})
	c.Diff, c.Changes = candidateDiff(base.Manifest.Members, c.Manifest.Members)
	for n := range c.Changes {
		c.Changes[n].Reason = "permanent withdrawal closure"
	}
	return c, ctx.Err()
}
