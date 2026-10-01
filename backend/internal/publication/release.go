package publication

import (
	"encoding/json"
	"github.com/yyl1212/math_master/backend/internal/auth"
	"github.com/yyl1212/math_master/backend/internal/content"
	"sort"
)

type ReviewedBatch struct {
	Submission SubmissionView
	Members    []MemberIdentity
	Bindings   []content.AssetBinding
}
type releaseBody struct {
	knowledge *content.Knowledge
	unit      *content.Unit
	path      *content.Path
	asset     *content.Asset
}

func snapshotBodies(s content.Snapshot) map[string]releaseBody {
	out := map[string]releaseBody{}
	for _, v := range s.Knowledge {
		out["knowledge/"+v.ID] = releaseBody{knowledge: &v}
	}
	for _, v := range s.Units {
		out["unit/"+v.ID] = releaseBody{unit: &v}
	}
	for _, v := range s.Paths {
		out["path/"+v.ID] = releaseBody{path: &v}
	}
	for _, v := range s.Assets {
		out["asset/"+v.ID] = releaseBody{asset: &v}
	}
	return out
}
func (b releaseBody) digest() string {
	switch {
	case b.knowledge != nil:
		return content.Digest(*b.knowledge)
	case b.unit != nil:
		return content.Digest(*b.unit)
	case b.path != nil:
		return content.Digest(*b.path)
	case b.asset != nil:
		return b.asset.SHA256
	}
	return ""
}
func (b releaseBody) semanticDigest() string {
	if b.asset != nil {
		return content.Digest(*b.asset)
	}
	return b.digest()
}
func (b releaseBody) version() int {
	switch {
	case b.knowledge != nil:
		return b.knowledge.Version
	case b.unit != nil:
		return b.unit.Version
	case b.path != nil:
		return b.path.Version
	case b.asset != nil:
		return 1
	}
	return 0
}
func (b releaseBody) owner() content.VersionRef {
	if b.unit != nil {
		return b.unit.Knowledge
	}
	if b.asset != nil {
		return b.asset.Knowledge
	}
	return content.VersionRef{}
}
func snapshotFromBodies(version int, bodies map[string]releaseBody, bindings map[string]content.AssetBinding) content.Snapshot {
	out := content.Snapshot{CatalogueVersion: version, Knowledge: []content.Knowledge{}, Units: []content.Unit{}, Paths: []content.Path{}, Assets: []content.Asset{}, Bindings: []content.AssetBinding{}}
	keys := []string{}
	for key := range bodies {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	units := map[content.VersionRef]bool{}
	for _, key := range keys {
		b := bodies[key]
		switch {
		case b.knowledge != nil:
			out.Knowledge = append(out.Knowledge, *b.knowledge)
		case b.unit != nil:
			out.Units = append(out.Units, *b.unit)
			units[content.VersionRef{ID: b.unit.ID, Version: b.unit.Version}] = true
		case b.path != nil:
			out.Paths = append(out.Paths, *b.path)
		case b.asset != nil:
			out.Assets = append(out.Assets, *b.asset)
		}
	}
	for _, b := range bindings {
		if units[b.Unit] {
			out.Bindings = append(out.Bindings, b)
		}
	}
	sort.Slice(out.Bindings, func(i, j int) bool { return bindingKey(out.Bindings[i]) < bindingKey(out.Bindings[j]) })
	return out
}
func ValidateCandidateGraph(s content.Snapshot) error {
	known := map[content.VersionRef]content.Knowledge{}
	byID := map[string]content.Knowledge{}
	for _, k := range s.Knowledge {
		ref := content.VersionRef{ID: k.ID, Version: k.Version}
		if _, ok := byID[k.ID]; ok {
			return ErrContentInvalid
		}
		known[ref] = k
		byID[k.ID] = k
	}
	indegree := map[content.VersionRef]int{}
	dependents := map[content.VersionRef][]content.VersionRef{}
	for ref := range known {
		indegree[ref] = 0
	}
	for ref, k := range known {
		for _, rel := range k.Relations {
			if rel.Kind == "prerequisite" {
				if _, ok := known[rel.Target]; !ok {
					return ErrContentInvalid
				}
				indegree[ref]++
				dependents[rel.Target] = append(dependents[rel.Target], ref)
			}
		}
	}
	queue := []content.VersionRef{}
	for ref, n := range indegree {
		if n == 0 {
			queue = append(queue, ref)
		}
	}
	for i := 0; i < len(queue); i++ {
		for _, ref := range dependents[queue[i]] {
			indegree[ref]--
			if indegree[ref] == 0 {
				queue = append(queue, ref)
			}
		}
	}
	if len(queue) != len(known) {
		return ErrContentInvalid
	}
	assets := map[string]content.Asset{}
	for _, a := range s.Assets {
		if _, ok := known[a.Knowledge]; !ok {
			return ErrContentInvalid
		}
		assets[a.ID] = a
	}
	bindingMap := map[string]string{}
	for _, b := range s.Bindings {
		key := bindingKey(b)
		if _, ok := bindingMap[key]; ok {
			return ErrContentInvalid
		}
		bindingMap[key] = b.SHA256
	}
	usedBindings := 0
	unitCounts := map[content.VersionRef]int{}
	for _, u := range s.Units {
		if _, ok := known[u.Knowledge]; !ok {
			return ErrContentInvalid
		}
		unitCounts[u.Knowledge]++
		for _, id := range u.AssetIDs {
			a, ok := assets[id]
			if !ok || a.Knowledge != u.Knowledge || bindingMap[bindingKey(content.AssetBinding{Unit: content.VersionRef{ID: u.ID, Version: u.Version}, AssetID: id})] != a.SHA256 {
				return ErrContentInvalid
			}
			usedBindings++
		}
	}
	if usedBindings != len(bindingMap) {
		return ErrContentInvalid
	}
	for ref := range known {
		if unitCounts[ref] == 0 {
			return ErrContentInvalid
		}
	}
	for _, path := range s.Paths {
		positions := map[content.VersionRef]int{}
		for i, ref := range path.Nodes {
			if _, ok := known[ref]; !ok {
				return ErrContentInvalid
			}
			if _, ok := positions[ref]; ok {
				return ErrContentInvalid
			}
			positions[ref] = i
		}
		for i, ref := range path.Nodes {
			for _, rel := range known[ref].Relations {
				if rel.Kind == "prerequisite" {
					at, ok := positions[rel.Target]
					if !ok || at >= i {
						return ErrContentInvalid
					}
				}
			}
		}
	}
	return nil
}
func CandidateLimits(s content.Snapshot) error {
	edges := 0
	for _, k := range s.Knowledge {
		edges += len(k.Relations)
	}
	if len(s.Knowledge) > 1000 || len(s.Units) > 4000 || len(s.Paths) > 200 || len(s.Assets) > 1000 || edges > 16000 {
		return ErrContentLimitExceeded
	}
	raw, err := json.Marshal(s)
	if err != nil {
		return ErrContentInvalid
	}
	if len(raw) > 32<<20 {
		return ErrContentLimitExceeded
	}
	return nil
}
func CandidateDiff(before, after []ManifestMember, reasons map[string]string) Diff {
	old := map[string]MemberIdentity{}
	next := map[string]MemberIdentity{}
	for _, m := range before {
		old[memberKey(m.Identity)] = m.Identity
	}
	for _, m := range after {
		next[memberKey(m.Identity)] = m.Identity
	}
	keys := map[string]bool{}
	for key := range old {
		keys[key] = true
	}
	for key := range next {
		keys[key] = true
	}
	sorted := []string{}
	for key := range keys {
		sorted = append(sorted, key)
	}
	sort.Strings(sorted)
	diff := Diff{Changes: []Change{}}
	for _, key := range sorted {
		a, had := old[key]
		b, has := next[key]
		reason := reasons[key]
		switch {
		case !had && has:
			diff.Added++
			diff.Changes = append(diff.Changes, Change{Kind: b.Kind, ID: b.ID, Reason: "Reviewed member added.", After: &b})
		case had && !has:
			diff.Removed++
			if reason == "" {
				reason = "Member removed from this snapshot."
			}
			diff.Changes = append(diff.Changes, Change{Kind: a.Kind, ID: a.ID, Reason: reason, Before: &a})
		case had && has && a != b:
			diff.Replaced++
			diff.Changes = append(diff.Changes, Change{Kind: b.Kind, ID: b.ID, Reason: "Reviewed fixed member replaced.", Before: &a, After: &b})
		}
	}
	return diff
}
func BuildCandidate(base Candidate, batches []ReviewedBatch) (Candidate, error) {
	var out Candidate
	if len(batches) < 1 || len(batches) > 20 {
		return out, auth.ErrInvalidInput
	}
	publicationID := base.PublicationID
	base = copyJSON(base)
	base.PublicationID = publicationID
	batches = copyJSON(batches)
	return buildCandidate(base, batches)
}
func buildCandidate(base Candidate, batches []ReviewedBatch) (Candidate, error) {
	var out Candidate
	version, sha := base.Manifest.CatalogueVersion, base.Manifest.CatalogueSHA256
	current := map[string]ManifestMember{}
	bodies := snapshotBodies(base.Snapshot)
	bindings := map[string]content.AssetBinding{}
	var parent *string
	if base.PublicationID != "" {
		id := base.PublicationID
		parent = &id
	}
	for _, m := range base.Manifest.Members {
		if m.Evidence.SubmissionID == "" || m.Evidence.DecisionID == "" {
			return out, ErrReviewRequired
		}
		m.Evidence.InheritedFrom = parent
		current[memberKey(m.Identity)] = m
	}
	for _, b := range base.Snapshot.Bindings {
		bindings[bindingKey(b)] = b
	}
	chosen := map[string]ManifestMember{}
	chosenBodies := map[string]releaseBody{}
	chosenBindings := map[string]content.AssetBinding{}
	for _, batch := range batches {
		s := batch.Submission
		if s.Status != "approved" || s.Review == nil || s.Review.Decision != "approve" || s.Review.SubmissionID != s.ID || s.Review.FrozenDigest != s.Frozen.FrozenDigest || !s.Gate.ReadyToSubmit {
			return out, ErrReviewRequired
		}
		if version == 0 {
			version, sha = s.Frozen.CatalogueVersion, s.Frozen.CatalogueSHA256
		}
		if version != s.Frozen.CatalogueVersion || sha != s.Frozen.CatalogueSHA256 {
			return out, ErrVersionConflict
		}
		digest, err := FrozenDigest(s.Frozen)
		if err != nil || digest != s.Frozen.FrozenDigest {
			return out, ErrReviewRequired
		}
		if err = ValidateReviewInput(ReviewInput{Decision: s.Review.Decision, Checks: s.Review.Checks, IndependenceNote: s.Review.IndependenceNote, Note: s.Review.Note}); err != nil {
			return out, ErrReviewRequired
		}
		for _, id := range s.Frozen.AuthorIDs {
			if id == s.Review.ReviewerID {
				return out, ErrReviewRequired
			}
		}
		p := s.Frozen.Package
		sb := snapshotBodies(content.Snapshot{Knowledge: p.Knowledge, Units: p.Units, Paths: p.Paths, Assets: p.Assets})
		if len(batch.Members) != len(sb) {
			return out, ErrContentInvalid
		}
		for _, identity := range batch.Members {
			key := memberKey(identity)
			b, ok := sb[key]
			if !ok || b.digest() != identity.SHA256 || b.version() != identity.Version || identity.PackageID != p.ID || identity.PackageVersion != p.Version {
				return out, ErrContentInvalid
			}
			member := ManifestMember{Identity: identity, Evidence: MemberEvidence{SubmissionID: s.ID, DecisionID: s.Review.ID, FrozenDigest: s.Frozen.FrozenDigest}}
			if previous, ok := chosen[key]; ok {
				if previous.Identity.Version != identity.Version || previous.Identity.SHA256 != identity.SHA256 || chosenBodies[key].semanticDigest() != b.semanticDigest() {
					return out, ErrVersionConflict
				}
				if memberOrder(previous.Identity, identity) || previous.Identity == identity && previous.Evidence.SubmissionID < member.Evidence.SubmissionID {
					continue
				}
			}
			if old, ok := current[key]; ok && old.Identity.Version == identity.Version && old.Identity.SHA256 != identity.SHA256 {
				return out, ErrImmutableConflict
			}
			chosen[key] = member
			chosenBodies[key] = b
		}
		for _, b := range batch.Bindings {
			key := bindingKey(b)
			if old, ok := chosenBindings[key]; ok && old.SHA256 != b.SHA256 {
				return out, ErrImmutableConflict
			}
			if old, ok := bindings[key]; ok && old.SHA256 != b.SHA256 {
				return out, ErrImmutableConflict
			}
			chosenBindings[key] = b
		}
	}
	reasons := map[string]string{}
	for key, m := range chosen {
		if m.Identity.Kind != "knowledge" {
			continue
		}
		old, ok := current[key]
		if !ok || old.Identity.Version == m.Identity.Version {
			continue
		}
		for oldKey, b := range bodies {
			owner := b.owner()
			if owner.ID == m.Identity.ID && owner.Version != m.Identity.Version {
				delete(current, oldKey)
				delete(bodies, oldKey)
				reasons[oldKey] = "Removed with the replaced knowledge version."
			}
		}
	}
	for key, m := range chosen {
		current[key] = m
		bodies[key] = chosenBodies[key]
	}
	for key, b := range chosenBindings {
		bindings[key] = b
	}
	snapshot := snapshotFromBodies(version, bodies, bindings)
	if err := CandidateLimits(snapshot); err != nil {
		return out, err
	}
	if err := ValidateCandidateGraph(snapshot); err != nil {
		return out, err
	}
	manifest := Manifest{CatalogueVersion: version, CatalogueSHA256: sha, BaseHead: parent, Members: []ManifestMember{}, Bindings: snapshot.Bindings}
	for _, m := range current {
		manifest.Members = append(manifest.Members, m)
	}
	manifest, err := CanonicalManifest(manifest)
	if err != nil {
		return out, err
	}
	out = Candidate{Manifest: manifest, Diff: CandidateDiff(base.Manifest.Members, manifest.Members, reasons), Snapshot: snapshot}
	return out, nil
}
