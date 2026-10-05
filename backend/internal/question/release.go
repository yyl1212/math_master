package question

import (
	"context"
	"reflect"
	"sort"
)

func candidateDiff(before, after []ManifestMember) (DiffSummary, []Change) {
	prior := map[string]MemberIdentity{}
	current := map[string]MemberIdentity{}
	for _, m := range before {
		prior[manifestKey(m.Identity)] = m.Identity
	}
	for _, m := range after {
		current[manifestKey(m.Identity)] = m.Identity
	}
	diff := DiffSummary{}
	changes := []Change{}
	for key, m := range current {
		old, found := prior[key]
		if !found {
			diff.Added++
			copy := m
			changes = append(changes, Change{Kind: m.Kind, ID: m.ID, Reason: "approved member added", After: &copy})
		} else if old.Version != m.Version || old.SHA256 != m.SHA256 {
			diff.Replaced++
			copy := m
			previous := old
			changes = append(changes, Change{Kind: m.Kind, ID: m.ID, Reason: "approved fixed version replaced", Before: &previous, After: &copy})
		}
	}
	for key, m := range prior {
		if _, found := current[key]; !found {
			diff.Removed++
			copy := m
			changes = append(changes, Change{Kind: m.Kind, ID: m.ID, Reason: "previous version retired", Before: &copy})
		}
	}
	sort.Slice(changes, func(i, j int) bool { return changes[i].Kind+"/"+changes[i].ID < changes[j].Kind+"/"+changes[j].ID })
	return diff, changes
}
func BuildCandidate(ctx context.Context, base BaseManifest, subs []ApprovedSubmission, refs ReferenceSnapshot) (Candidate, error) {
	empty := Candidate{}
	if err := ctx.Err(); err != nil {
		return empty, err
	}
	if len(subs) == 0 {
		return empty, ErrInvalid
	}
	if len(subs) > 20 {
		return empty, ErrLimitExceeded
	}
	if refs.KnowledgeHead == nil || !ValidID(*refs.KnowledgeHead) {
		return empty, ErrNotReady
	}
	base = canonicalValue(reflect.ValueOf(base)).Interface().(BaseManifest)
	subs = canonicalValue(reflect.ValueOf(subs)).Interface().([]ApprovedSubmission)
	refs = canonicalValue(reflect.ValueOf(refs)).Interface().(ReferenceSnapshot)
	c := Candidate{Manifest: Manifest{CatalogueVersion: refs.CatalogueVersion, CatalogueSHA256: refs.CatalogueSHA256, BaseKnowledgeHead: refs.KnowledgeHead, BaseQuestionHead: base.Head, Members: []ManifestMember{}, Resolved: []KnownObject{}}, Templates: []Template{}, Blueprints: []Blueprint{}, Instances: []Instance{}, Changes: []Change{}}
	members := map[string]ManifestMember{}
	templates := map[string]Template{}
	blueprints := map[string]Blueprint{}
	instances := map[string]Instance{}
	known := map[string]KnownObject{}
	before := []ManifestMember{}
	if base.Manifest != nil {
		if base.Head == nil || !ValidID(*base.Head) {
			return empty, ErrInvalid
		}
		if _, _, err := CanonicalManifest(*base.Manifest); err != nil {
			return empty, err
		}
		prior := Candidate{Manifest: *base.Manifest, Templates: base.Templates, Blueprints: base.Blueprints, Instances: base.Instances}
		if err := CheckCandidate(ctx, prior, refs, false); err != nil {
			return empty, err
		}
		before = base.Manifest.Members
		for _, m := range before {
			m.Evidence.InheritedFrom = base.Head
			members[manifestKey(m.Identity)] = m
		}
		for _, r := range base.Manifest.Resolved {
			known[objectKey(r.Kind, r.Ref)] = r
		}
		for _, t := range base.Templates {
			templates[t.ID] = t
		}
		for _, b := range base.Blueprints {
			blueprints[b.ID] = b
		}
		for _, i := range base.Instances {
			instances[i.Identity.ID] = i
		}
	} else if base.Head != nil || len(base.Instances) > 0 || len(base.Templates) > 0 || len(base.Blueprints) > 0 {
		return empty, ErrInvalid
	}
	selected := map[string]MemberIdentity{}
	selectedIDs := map[string]bool{}
	for _, s := range subs {
		if err := ctx.Err(); err != nil {
			return empty, err
		}
		if selectedIDs[s.SubmissionID] || !ValidID(s.SubmissionID) {
			return empty, ErrInvalid
		}
		selectedIDs[s.SubmissionID] = true
		d, f := s.Decision, s.Frozen
		raw, sha, err := CanonicalFrozen(f, s.Instances)
		if err != nil {
			return empty, err
		}
		if len(raw) > MaxEnvelopeBytes {
			return empty, ErrLimitExceeded
		}
		if sha != f.FrozenDigest || d.FrozenDigest != sha || d.SubmissionID != s.SubmissionID || d.Decision != "approve" || !ValidID(d.ID) || !ValidID(d.ReviewerID) {
			return empty, ErrReviewRequired
		}
		if err := ValidateReviewInput(ReviewInput{Decision: d.Decision, Checks: d.Checks, IndependenceNote: d.IndependenceNote, GenerationNote: d.GenerationNote, Note: d.Note}, len(f.QuestionPackage.Templates) > 0); err != nil {
			return empty, ErrReviewRequired
		}
		for _, author := range f.AuthorIDs {
			if author == d.ReviewerID && !s.AdministratorReview {
				return empty, ErrReviewRequired
			}
		}
		if f.CatalogueVersion != refs.CatalogueVersion || f.CatalogueSHA256 != refs.CatalogueSHA256 {
			return empty, ErrNotReady
		}
		if len(f.InstanceIdentities) != len(s.Instances) {
			return empty, ErrInvalid
		}
		for n, i := range s.Instances {
			if i.Identity != f.InstanceIdentities[n] {
				return empty, ErrInvalid
			}
		}
		evidence := MemberEvidence{SubmissionID: s.SubmissionID, DecisionID: d.ID, FrozenDigest: sha}
		add := func(kind, id string, version int, sha string) error {
			identity := MemberIdentity{Kind: kind, ID: id, Version: version, SHA256: sha, PackageID: f.QuestionPackage.ID, PackageVersion: f.QuestionPackage.Version}
			key := manifestKey(identity)
			if old, ok := selected[key]; ok && (old.Version != version || old.SHA256 != sha) {
				return ErrVersionConflict
			}
			selected[key] = identity
			if old, ok := members[key]; ok && old.Identity.Version == version {
				if old.Identity.SHA256 != sha {
					return ErrVersionConflict
				}
				// Explicit selection carries fresh evidence even when the math identity is unchanged.
				members[key] = ManifestMember{Identity: identity, Evidence: evidence}
				return nil
			}
			members[key] = ManifestMember{Identity: identity, Evidence: evidence}
			return nil
		}
		for _, t := range f.QuestionPackage.Templates {
			_, sha, err := canonical("question-template-v1", t)
			if err != nil {
				return empty, err
			}
			if err = add("template", t.ID, t.Version, sha); err != nil {
				return empty, err
			}
			templates[t.ID] = t
		}
		for _, b := range f.QuestionPackage.Blueprints {
			_, sha, err := canonical("question-blueprint-v1", b)
			if err != nil {
				return empty, err
			}
			if err = add("blueprint", b.ID, b.Version, sha); err != nil {
				return empty, err
			}
			blueprints[b.ID] = b
		}
		for _, i := range s.Instances {
			if err := add("instance", i.Identity.ID, i.Identity.Version, i.Identity.SHA256); err != nil {
				return empty, err
			}
			instances[i.Identity.ID] = i
		}
		for _, r := range f.Resolved {
			key := objectKey(r.Kind, r.Ref)
			if old, ok := known[key]; ok && old.SHA256 != r.SHA256 {
				return empty, ErrVersionConflict
			}
			known[key] = r
		}
	}
	// Generated instances leave the current bank with their exact old template. Their stored bodies remain immutable.
	for id, i := range instances {
		if i.Origin == "template" {
			if i.Template == nil {
				return empty, ErrInvalid
			}
			t, ok := members["template/"+i.Template.ID]
			if !ok || t.Identity.Version != i.Template.Version || t.Identity.SHA256 != i.Template.SHA256 {
				delete(instances, id)
				delete(members, "instance/"+id)
			}
		}
	}
	for _, m := range members {
		c.Manifest.Members = append(c.Manifest.Members, m)
	}
	sort.Slice(c.Manifest.Members, func(i, j int) bool {
		return manifestKey(c.Manifest.Members[i].Identity) < manifestKey(c.Manifest.Members[j].Identity)
	})
	for _, t := range templates {
		c.Templates = append(c.Templates, t)
	}
	for _, b := range blueprints {
		c.Blueprints = append(c.Blueprints, b)
	}
	for _, i := range instances {
		c.Instances = append(c.Instances, i)
	}
	sort.Slice(c.Templates, func(i, j int) bool { return c.Templates[i].ID < c.Templates[j].ID })
	sort.Slice(c.Blueprints, func(i, j int) bool { return c.Blueprints[i].ID < c.Blueprints[j].ID })
	sort.Slice(c.Instances, func(i, j int) bool { return c.Instances[i].Identity.ID < c.Instances[j].Identity.ID })
	for key := range candidateReferences(c) {
		r, ok := known[key]
		if !ok {
			return empty, ErrNotReady
		}
		c.Manifest.Resolved = append(c.Manifest.Resolved, r)
	}
	sort.Slice(c.Manifest.Resolved, func(i, j int) bool {
		return objectKey(c.Manifest.Resolved[i].Kind, c.Manifest.Resolved[i].Ref) < objectKey(c.Manifest.Resolved[j].Kind, c.Manifest.Resolved[j].Ref)
	})
	if err := CheckCandidate(ctx, c, refs, true); err != nil {
		return empty, err
	}
	c.Diff, c.Changes = candidateDiff(before, c.Manifest.Members)
	return c, nil
}
func candidateReferences(c Candidate) map[string]bool {
	out := map[string]bool{}
	add := func(k Ref, coverage []ObjectiveCoverage, units []Ref, assets []AssetRef) {
		out[objectKey("knowledge", k)] = true
		for _, m := range coverage {
			out[objectKey("knowledge", m.Knowledge)] = true
		}
		for _, u := range units {
			out[objectKey("unit", u)] = true
		}
		for _, a := range assets {
			out[objectKey("asset", Ref{ID: a.ID})] = true
		}
	}
	for _, t := range c.Templates {
		add(t.Knowledge, t.Coverage, t.Units, t.Assets)
	}
	for _, i := range c.Instances {
		b := i.Body
		add(b.Knowledge, b.Coverage, b.Units, b.Assets)
	}
	for _, b := range c.Blueprints {
		out[objectKey("knowledge", b.Knowledge)] = true
	}
	return out
}

// Verify the materialized bodies; prepare/activate never regenerate a published finite parameter space.
func CheckCandidate(ctx context.Context, c Candidate, refs ReferenceSnapshot, requireReady bool) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	raw, _, err := CanonicalManifest(c.Manifest)
	if err != nil {
		return err
	}
	if err = CheckBankLimits(len(c.Templates), len(c.Instances), len(c.Blueprints), 0, len(raw)); err != nil {
		return err
	}
	members := map[string]MemberIdentity{}
	for _, m := range c.Manifest.Members {
		members[manifestKey(m.Identity)] = m.Identity
	}
	if len(members) != len(c.Templates)+len(c.Instances)+len(c.Blueprints) {
		return ErrInvalid
	}
	bodies := 0
	seen := map[string]bool{}
	check := func(kind, id string, version int, sha string) error {
		key := kind + "/" + id
		if seen[key] {
			return ErrVersionConflict
		}
		seen[key] = true
		m, ok := members[key]
		if !ok || m.Version != version || m.SHA256 != sha {
			return ErrInvalid
		}
		return nil
	}
	for _, t := range c.Templates {
		if err := ctx.Err(); err != nil {
			return err
		}
		normalized, _, err := normalizeTemplate(t)
		if err != nil {
			return err
		}
		raw, sha, err := canonical("question-template-v1", normalized)
		if err != nil {
			return err
		}
		bodies += len(raw)
		if err = check("template", t.ID, t.Version, sha); err != nil {
			return err
		}
	}
	instanceMap := map[Ref]Instance{}
	templateInstances := map[Ref][]Instance{}
	nodeCounts := map[Ref]int{}
	for _, i := range c.Instances {
		if err := ctx.Err(); err != nil {
			return err
		}
		if err = VerifyInstance(i); err != nil {
			return err
		}
		raw, sha, err := CanonicalInstance(i)
		if err != nil {
			return err
		}
		bodies += len(raw)
		if err = check("instance", i.Identity.ID, i.Identity.Version, sha); err != nil {
			return err
		}
		instanceMap[Ref{ID: i.Identity.ID, Version: i.Identity.Version}] = i
		if i.Origin == "template" {
			if i.Template == nil {
				return ErrInvalid
			}
			m, ok := members["template/"+i.Template.ID]
			if !ok || m.Version != i.Template.Version || m.SHA256 != i.Template.SHA256 {
				return ErrInvalid
			}
			r := Ref{ID: i.Template.ID, Version: i.Template.Version}
			templateInstances[r] = append(templateInstances[r], i)
		}
		for _, coverage := range i.Body.Coverage {
			nodeCounts[coverage.Knowledge]++
			if nodeCounts[coverage.Knowledge] > 1000 {
				return ErrLimitExceeded
			}
		}
	}
	if err = CheckBankLimits(len(c.Templates), len(c.Instances), len(c.Blueprints), bodies, len(raw)); err != nil {
		return err
	}
	knowledge := map[Ref]FixedKnowledge{}
	current := map[string]KnownObject{}
	for _, k := range refs.Knowledge {
		r := Ref{ID: k.Identity.ID, Version: k.Identity.Version}
		knowledge[r] = k
		current[objectKey("knowledge", r)] = KnownObject{Kind: "knowledge", Ref: r, SHA256: k.Identity.SHA256}
	}
	for _, u := range refs.Units {
		r := Ref{ID: u.Identity.ID, Version: u.Identity.Version}
		current[objectKey("unit", r)] = KnownObject{Kind: "unit", Ref: r, SHA256: u.Identity.SHA256}
	}
	for _, a := range refs.Assets {
		r := Ref{ID: a.ID}
		current[objectKey("asset", r)] = KnownObject{Kind: "asset", Ref: r, SHA256: a.SHA256}
	}
	expected := candidateReferences(c)
	if len(expected) != len(c.Manifest.Resolved) {
		return ErrInvalid
	}
	for _, r := range c.Manifest.Resolved {
		key := objectKey(r.Kind, r.Ref)
		if !expected[key] {
			return ErrInvalid
		}
		if requireReady {
			now, ok := current[key]
			if !ok || now.SHA256 != r.SHA256 {
				return ErrNotReady
			}
		}
	}
	if requireReady && (c.Manifest.CatalogueVersion != refs.CatalogueVersion || c.Manifest.CatalogueSHA256 != refs.CatalogueSHA256) {
		return ErrNotReady
	}
	for _, b := range c.Blueprints {
		_, sha, err := canonical("question-blueprint-v1", b)
		if err != nil {
			return err
		}
		if err = check("blueprint", b.ID, b.Version, sha); err != nil {
			return err
		}
		if b.RuleVersion != 1 || b.QuestionCount != 5 || b.PassCount != 4 {
			return ErrInvalid
		}
		pool := map[string]Instance{}
		sources := map[string]bool{}
		for _, s := range b.Sources {
			key := objectKey(s.Kind, s.Ref)
			if sources[key] {
				return ErrInvalid
			}
			sources[key] = true
			if s.Kind == "template" {
				m, ok := members["template/"+s.Ref.ID]
				if !ok || m.Version != s.Ref.Version {
					return ErrNotReady
				}
				for _, i := range templateInstances[s.Ref] {
					pool[i.Identity.ID] = i
				}
			} else if s.Kind == "instance" {
				i, ok := instanceMap[s.Ref]
				if !ok || i.Origin != "fixed" {
					return ErrNotReady
				}
				pool[i.Identity.ID] = i
			} else {
				return ErrInvalid
			}
		}
		if requireReady {
			k, ok := knowledge[b.Knowledge]
			if !ok {
				return ErrNotReady
			}
			for _, core := range b.CoreObjectiveIndices {
				if core < 0 || core >= len(k.Objectives) {
					return ErrNotReady
				}
			}
			candidates := []CandidateCoverage{}
			for _, i := range pool {
				goals := instanceObjectives(i, b.Knowledge)
				if len(goals) == 0 {
					return ErrNotReady
				}
				candidates = append(candidates, CandidateCoverage{InstanceID: i.Identity.ID, ObjectiveIndices: goals})
			}
			_, ready, err := FiveQuestionCover(b.CoreObjectiveIndices, candidates)
			if err != nil {
				return err
			}
			if !ready {
				return ErrNotReady
			}
		}
	}
	return ctx.Err()
}

// References for a current, bounded bank are resolved in one batch by Store.
func CandidateDraft(c Candidate) DraftInput {
	fixed := make([]FixedQuestion, 0, len(c.Instances))
	for _, i := range c.Instances {
		fixed = append(fixed, FixedQuestion{ID: i.Identity.ID, Version: i.Identity.Version, Body: i.Body})
	}
	return DraftInput{CatalogueVersion: c.Manifest.CatalogueVersion, QuestionPackage: QuestionPackage{Kind: "question-bank", SchemaVersion: 1, ID: "current-bank", Version: 1, Templates: c.Templates, FixedQuestions: fixed, Blueprints: c.Blueprints}, SourceMap: []SourceLink{}}
}
