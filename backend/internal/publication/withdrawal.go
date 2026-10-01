package publication

import (
	"github.com/yyl1212/math_master/backend/internal/auth"
	"github.com/yyl1212/math_master/backend/internal/content"
)

func ValidateWithdrawalTarget(t WithdrawalTarget) error {
	switch t.Kind {
	case "knowledge", "unit", "path":
		if !ValidMathID(t.ID) || !mathVersion(t.Version) || t.SHA256 != "" {
			return auth.ErrInvalidInput
		}
	case "asset":
		if t.ID != "" || t.Version != 0 || !ValidSHA(t.SHA256) {
			return auth.ErrInvalidInput
		}
	default:
		return auth.ErrInvalidInput
	}
	return nil
}

// WithdrawCandidate only removes previously published members. P4 must use the
// same publication lock when granting learning evidence; P5 can extend the
// withdrawal transaction before scheduling evidence re-evaluation.
func WithdrawCandidate(base Candidate, target WithdrawalTarget) (Candidate, error) {
	var out Candidate
	if err := ValidateWithdrawalTarget(target); err != nil {
		return out, err
	}
	id := base.PublicationID
	base = copyJSON(base)
	base.PublicationID = id
	if err := CandidateLimits(base.Snapshot); err != nil {
		return out, err
	}
	paused := map[content.VersionRef]bool{}
	reverse := map[content.VersionRef][]content.VersionRef{}
	for _, k := range base.Snapshot.Knowledge {
		ref := content.VersionRef{ID: k.ID, Version: k.Version}
		if target.Kind == "knowledge" && target.ID == k.ID && target.Version == k.Version {
			paused[ref] = true
		}
		for _, r := range k.Relations {
			if r.Kind == "prerequisite" {
				reverse[r.Target] = append(reverse[r.Target], ref)
			}
		}
	}
	for _, u := range base.Snapshot.Units {
		if target.Kind == "unit" && target.ID == u.ID && target.Version == u.Version {
			paused[u.Knowledge] = true
		}
	}
	if target.Kind == "asset" {
		for _, a := range base.Snapshot.Assets {
			if a.SHA256 == target.SHA256 {
				paused[a.Knowledge] = true
			}
		}
		for _, b := range base.Snapshot.Bindings {
			if b.SHA256 == target.SHA256 {
				for _, u := range base.Snapshot.Units {
					if b.Unit.ID == u.ID && b.Unit.Version == u.Version {
						paused[u.Knowledge] = true
					}
				}
			}
		}
	}
	queue := []content.VersionRef{}
	for ref := range paused {
		queue = append(queue, ref)
	}
	for i := 0; i < len(queue); i++ {
		for _, dependent := range reverse[queue[i]] {
			if !paused[dependent] {
				paused[dependent] = true
				queue = append(queue, dependent)
			}
		}
	}
	bodies := snapshotBodies(base.Snapshot)
	reasons := map[string]string{}
	for key, b := range bodies {
		remove, reason := false, ""
		if b.knowledge != nil && paused[content.VersionRef{ID: b.knowledge.ID, Version: b.knowledge.Version}] {
			remove = true
			reason = "Knowledge paused because its fixed content or a prerequisite is unavailable."
		}
		if b.unit != nil && paused[b.unit.Knowledge] || b.asset != nil && paused[b.asset.Knowledge] {
			remove = true
			reason = "Owned content removed with its paused knowledge version."
		}
		if b.path != nil {
			if target.Kind == "path" && b.path.ID == target.ID && b.path.Version == target.Version {
				remove = true
				reason = "Problem path version withdrawn."
			}
			for _, ref := range b.path.Nodes {
				if paused[ref] {
					remove = true
					reason = "Path contains a paused knowledge version."
				}
			}
		}
		if remove {
			delete(bodies, key)
			reasons[key] = reason
		}
	}
	bindings := map[string]content.AssetBinding{}
	for _, b := range base.Snapshot.Bindings {
		bindings[bindingKey(b)] = b
	}
	snapshot := snapshotFromBodies(base.Manifest.CatalogueVersion, bodies, bindings)
	var parent *string
	if id != "" {
		parent = &id
	}
	manifest := Manifest{CatalogueVersion: base.Manifest.CatalogueVersion, CatalogueSHA256: base.Manifest.CatalogueSHA256, BaseHead: parent, Members: []ManifestMember{}, Bindings: snapshot.Bindings}
	for _, m := range base.Manifest.Members {
		if _, ok := bodies[memberKey(m.Identity)]; ok {
			m.Evidence.InheritedFrom = parent
			manifest.Members = append(manifest.Members, m)
		}
	}
	manifest, err := CanonicalManifest(manifest)
	if err != nil {
		return out, err
	}
	if err = ValidateCandidateGraph(snapshot); err != nil {
		return out, err
	}
	out = Candidate{Manifest: manifest, Snapshot: snapshot, Diff: CandidateDiff(base.Manifest.Members, manifest.Members, reasons)}
	return out, nil
}
