package question

import (
	"reflect"
	"sort"
)

const MaxBankTemplates = 200
const MaxBankInstances = 10000
const MaxBankBlueprints = 1000
const MaxBankBodyBytes = 32 << 20
const MaxManifestBytes = 8 << 20

func CheckBankLimits(templates, instances, blueprints, bodyBytes, manifestBytes int) error {
	for _, n := range []int{templates, instances, blueprints, bodyBytes, manifestBytes} {
		if n < 0 {
			return ErrInvalid
		}
	}
	if templates > MaxBankTemplates || instances > MaxBankInstances || blueprints > MaxBankBlueprints || bodyBytes > MaxBankBodyBytes || manifestBytes > MaxManifestBytes {
		return ErrLimitExceeded
	}
	return nil
}
func manifestKey(m MemberIdentity) string { return m.Kind + "/" + m.ID }
func CanonicalManifest(m Manifest) ([]byte, string, error) {
	m = canonicalValue(reflect.ValueOf(m)).Interface().(Manifest)
	if m.CatalogueVersion < 1 || m.CatalogueVersion > 2147483647 || !ValidSHA(m.CatalogueSHA256) || m.BaseKnowledgeHead != nil && !ValidID(*m.BaseKnowledgeHead) || m.BaseQuestionHead != nil && !ValidID(*m.BaseQuestionHead) {
		return nil, "", ErrInvalid
	}
	sort.Slice(m.Members, func(i, j int) bool { return manifestKey(m.Members[i].Identity) < manifestKey(m.Members[j].Identity) })
	for n, item := range m.Members {
		i, e := item.Identity, item.Evidence
		validID := ValidMathID(i.ID)
		if i.Kind == "instance" {
			validID = ValidInstanceID(i.ID)
		}
		if i.Kind != "template" && i.Kind != "instance" && i.Kind != "blueprint" || !validID || i.Version < 1 || i.Version > 2147483647 || !ValidSHA(i.SHA256) || !ValidMathID(i.PackageID) || i.PackageVersion < 1 || i.PackageVersion > 2147483647 {
			return nil, "", ErrInvalid
		}
		if n > 0 && manifestKey(i) == manifestKey(m.Members[n-1].Identity) {
			return nil, "", ErrVersionConflict
		}
		if !ValidID(e.SubmissionID) || !ValidID(e.DecisionID) || !ValidSHA(e.FrozenDigest) || e.InheritedFrom != nil && !ValidID(*e.InheritedFrom) {
			return nil, "", ErrReviewRequired
		}
	}
	sort.Slice(m.Resolved, func(i, j int) bool {
		return objectKey(m.Resolved[i].Kind, m.Resolved[i].Ref) < objectKey(m.Resolved[j].Kind, m.Resolved[j].Ref)
	})
	for n, r := range m.Resolved {
		if r.Kind != "knowledge" && r.Kind != "unit" && r.Kind != "asset" || !ValidMathID(r.ID) || !ValidSHA(r.SHA256) || r.Kind == "asset" && r.Version != 0 || r.Kind != "asset" && (r.Version < 1 || r.Version > 2147483647) {
			return nil, "", ErrInvalid
		}
		if n > 0 && objectKey(r.Kind, r.Ref) == objectKey(m.Resolved[n-1].Kind, m.Resolved[n-1].Ref) {
			return nil, "", ErrVersionConflict
		}
	}
	raw, sha, err := canonical("question-manifest-v1", m)
	if err == nil && len(raw) > MaxManifestBytes {
		return nil, "", ErrLimitExceeded
	}
	return raw, sha, err
}
