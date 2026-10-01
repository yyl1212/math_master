package publication

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"github.com/yyl1212/math_master/backend/internal/auth"
	"github.com/yyl1212/math_master/backend/internal/content"
	"sort"
)

func memberKey(m MemberIdentity) string { return m.Kind + "/" + m.ID }
func memberOrder(a, b MemberIdentity) bool {
	if a.Kind != b.Kind {
		return a.Kind < b.Kind
	}
	if a.ID != b.ID {
		return a.ID < b.ID
	}
	if a.Version != b.Version {
		return a.Version < b.Version
	}
	if a.PackageID != b.PackageID {
		return a.PackageID < b.PackageID
	}
	if a.PackageVersion != b.PackageVersion {
		return a.PackageVersion < b.PackageVersion
	}
	return a.SHA256 < b.SHA256
}
func bindingKey(b content.AssetBinding) string {
	return fmt.Sprintf("%s:%d:%s", b.Unit.ID, b.Unit.Version, b.AssetID)
}
func copyJSON[T any](v T) T {
	b, _ := json.Marshal(v)
	var out T
	_ = json.Unmarshal(b, &out)
	return out
}
func mathVersion(v int) bool { return v > 0 && v <= 2147483647 }
func CanonicalManifest(m Manifest) (Manifest, error) {
	m = copyJSON(m)
	if !mathVersion(m.CatalogueVersion) || !ValidSHA(m.CatalogueSHA256) || m.BaseHead != nil && !ValidID(*m.BaseHead) {
		return m, auth.ErrInvalidInput
	}
	if m.Members == nil {
		m.Members = []ManifestMember{}
	}
	if m.Bindings == nil {
		m.Bindings = []content.AssetBinding{}
	}
	sort.Slice(m.Members, func(i, j int) bool { return memberOrder(m.Members[i].Identity, m.Members[j].Identity) })
	for i, item := range m.Members {
		v, e := item.Identity, item.Evidence
		if v.Kind != "knowledge" && v.Kind != "unit" && v.Kind != "path" && v.Kind != "asset" || !ValidMathID(v.ID) || !ValidMathID(v.PackageID) || !mathVersion(v.Version) || !mathVersion(v.PackageVersion) || !ValidSHA(v.SHA256) || v.Kind == "asset" && v.Version != 1 {
			return m, ErrContentInvalid
		}
		if i > 0 && memberKey(m.Members[i-1].Identity) == memberKey(v) {
			return m, ErrVersionConflict
		}
		if !ValidID(e.SubmissionID) || !ValidID(e.DecisionID) || !ValidSHA(e.FrozenDigest) || e.InheritedFrom != nil && !ValidID(*e.InheritedFrom) {
			return m, ErrReviewRequired
		}
	}
	sort.Slice(m.Bindings, func(i, j int) bool {
		a, b := m.Bindings[i], m.Bindings[j]
		if a.Unit.ID != b.Unit.ID {
			return a.Unit.ID < b.Unit.ID
		}
		if a.Unit.Version != b.Unit.Version {
			return a.Unit.Version < b.Unit.Version
		}
		return a.AssetID < b.AssetID
	})
	for i, b := range m.Bindings {
		if !ValidMathID(b.Unit.ID) || !mathVersion(b.Unit.Version) || !ValidMathID(b.AssetID) || !ValidSHA(b.SHA256) {
			return m, ErrContentInvalid
		}
		if i > 0 && bindingKey(m.Bindings[i-1]) == bindingKey(b) {
			return m, ErrContentInvalid
		}
	}
	return m, nil
}
func ManifestBytes(m Manifest) ([]byte, error) {
	m, err := CanonicalManifest(m)
	if err != nil {
		return nil, err
	}
	return json.Marshal(struct {
		Purpose string `json:"purpose"`
		Manifest
	}{"math-master/publication-manifest/v1", m})
}
func ManifestDigest(m Manifest) (string, error) {
	raw, err := ManifestBytes(m)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", sha256.Sum256(raw)), nil
}
func SnapshotBindings(p content.Package) []content.AssetBinding {
	assets := map[string]string{}
	for _, a := range p.Assets {
		assets[a.ID] = a.SHA256
	}
	out := []content.AssetBinding{}
	for _, u := range p.Units {
		for _, id := range u.AssetIDs {
			out = append(out, content.AssetBinding{Unit: content.VersionRef{ID: u.ID, Version: u.Version}, AssetID: id, SHA256: assets[id]})
		}
	}
	return out
}
