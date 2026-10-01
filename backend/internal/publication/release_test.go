package publication

import (
	"encoding/json"
	"errors"
	"github.com/yyl1212/math_master/backend/internal/content"
	"strings"
	"testing"
)

func releasePackage() content.Package {
	ref := content.VersionRef{ID: "root", Version: 1}
	k := content.Knowledge{ID: ref.ID, Version: 1, DomainIDs: []string{"elementary-mathematics"}, TopicIDs: []string{}, Type: "definition", Title: "Root", TitleZh: "基础", Statement: "A technical root definition.", Scope: "Technical fixture.", Objectives: []string{"Understand the root."}, Conditions: []string{}, System: "Rational arithmetic", Sources: []content.Source{{Kind: "original", Author: "Technical author", Title: "Fixture", License: "CC0-1.0", Attribution: "Original technical fixture."}}, Relations: []content.Relation{}}
	u := content.Unit{ID: "root-unit", Version: 1, Knowledge: ref, Angles: []content.Angle{{Kind: "formal", Body: "A formal explanation."}, {Kind: "intuitive", Body: "An intuitive explanation."}}, Examples: []string{"An example."}, Counterexamples: []string{}, AssetIDs: []string{"root-asset"}}
	a := content.Asset{ID: "root-asset", Path: "root.svg", SHA256: strings.Repeat("a", 64), Author: "Technical author", License: "CC0-1.0", Attribution: "Original technical illustration.", Knowledge: ref}
	return content.Package{SchemaVersion: 1, ID: "release-package", Version: 1, Knowledge: []content.Knowledge{k}, Units: []content.Unit{u}, Paths: []content.Path{}, Assets: []content.Asset{a}}
}
func reviewedBatch(t *testing.T, p content.Package, id string) ReviewedBatch {
	t.Helper()
	raw, _ := json.Marshal(p)
	var cloned content.Package
	_ = json.Unmarshal(raw, &cloned)
	p = cloned
	assets := []content.AssetView{}
	members := []MemberIdentity{}
	bindings := []content.AssetBinding{}
	add := func(kind, id string, version int, sha string) {
		members = append(members, MemberIdentity{Kind: kind, ID: id, Version: version, SHA256: sha, PackageID: p.ID, PackageVersion: p.Version})
	}
	for _, k := range p.Knowledge {
		add("knowledge", k.ID, k.Version, content.Digest(k))
	}
	for _, u := range p.Units {
		add("unit", u.ID, u.Version, content.Digest(u))
		for _, assetID := range u.AssetIDs {
			for _, a := range p.Assets {
				if a.ID == assetID {
					bindings = append(bindings, content.AssetBinding{Unit: content.VersionRef{ID: u.ID, Version: u.Version}, AssetID: assetID, SHA256: a.SHA256})
				}
			}
		}
	}
	for _, path := range p.Paths {
		add("path", path.ID, path.Version, content.Digest(path))
	}
	for _, a := range p.Assets {
		add("asset", a.ID, 1, a.SHA256)
		assets = append(assets, content.AssetView{ID: a.ID, SHA256: a.SHA256, Author: a.Author, License: a.License, Attribution: a.Attribution, Knowledge: a.Knowledge})
	}
	frozen := FrozenBody{CatalogueVersion: 1, CatalogueSHA256: strings.Repeat("b", 64), Package: p, SourceMap: []SourceLink{}, AuthorIDs: []string{"11111111-1111-4111-8111-111111111111"}, Assets: assets}
	digest, err := FrozenDigest(frozen)
	if err != nil {
		t.Fatal(err)
	}
	frozen.FrozenDigest = digest
	decisionID := "88888888-8888-4888-8888-888888888888"
	if id[0] == '5' {
		decisionID = "99999999-9999-4999-8999-999999999999"
	}
	sub := SubmissionView{ID: id, OwnerID: frozen.AuthorIDs[0], Status: "approved", Frozen: frozen, Gate: GateReport{ReadyToSubmit: true}, Review: &ReviewDecision{ID: decisionID, SubmissionID: id, ReviewerID: "22222222-2222-4222-8222-222222222222", FrozenDigest: digest, Decision: "approve", Checks: ReviewChecks{true, true, true, true, true}, IndependenceNote: "Independent technical fixture accounts.", Note: "Technical fixture approval only."}}
	return ReviewedBatch{Submission: sub, Members: members, Bindings: bindings}
}
func TestReleaseMerge(t *testing.T) {
	p := releasePackage()
	first := reviewedBatch(t, p, "44444444-4444-4444-8444-444444444444")
	base, err := BuildCandidate(Candidate{}, []ReviewedBatch{first})
	if err != nil || base.Manifest.BaseHead != nil || base.Diff.Added != 3 {
		t.Fatal(base, err)
	}
	other := reviewedBatch(t, p, "55555555-5555-4555-8555-555555555555")
	other.Submission.Frozen.CatalogueVersion = 2
	other.Submission.Frozen.FrozenDigest, _ = FrozenDigest(other.Submission.Frozen)
	other.Submission.Review.FrozenDigest = other.Submission.Frozen.FrozenDigest
	if _, err = BuildCandidate(Candidate{}, []ReviewedBatch{first, other}); !errors.Is(err, ErrVersionConflict) {
		t.Fatal("different catalogue merged", err)
	}
	p.Version = 2
	p.Knowledge[0].Version = 2
	p.Units[0].Version = 2
	p.Units[0].Knowledge.Version = 2
	p.Assets[0].Knowledge.Version = 2
	other = reviewedBatch(t, p, "55555555-5555-4555-8555-555555555555")
	if _, err = BuildCandidate(Candidate{}, []ReviewedBatch{first, other}); !errors.Is(err, ErrVersionConflict) {
		t.Fatal("conflicting selected versions merged", err)
	}
	base.PublicationID = "66666666-6666-4666-8666-666666666666"
	p.Units[0].ID = "new-root-unit"
	p.Assets[0].ID = "new-root-asset"
	p.Units[0].AssetIDs = []string{"new-root-asset"}
	other = reviewedBatch(t, p, "55555555-5555-4555-8555-555555555555")
	next, err := BuildCandidate(base, []ReviewedBatch{other})
	if err != nil || next.Manifest.BaseHead == nil || *next.Manifest.BaseHead != base.PublicationID || next.Diff.Removed != 2 || next.Diff.Replaced != 1 {
		t.Fatal("old knowledge-owned members retained", next, err)
	}
	old := releasePackage()
	child := old.Knowledge[0]
	child.ID = "dependent"
	child.Relations = []content.Relation{{Kind: "prerequisite", Target: content.VersionRef{ID: "root", Version: 1}}}
	old.Knowledge = append(old.Knowledge, child)
	u := old.Units[0]
	u.ID = "dependent-unit"
	u.Knowledge.ID = "dependent"
	u.AssetIDs = []string{}
	old.Units = append(old.Units, u)
	old.Paths = []content.Path{{ID: "path", Version: 1, DomainIDs: old.Knowledge[0].DomainIDs, Title: "Root before dependent", TitleZh: "路线", Nodes: []content.VersionRef{{ID: "root", Version: 1}, {ID: "dependent", Version: 1}}}}
	before, err := BuildCandidate(Candidate{}, []ReviewedBatch{reviewedBatch(t, old, "44444444-4444-4444-8444-444444444444")})
	if err != nil {
		t.Fatal(err)
	}
	before.PublicationID = base.PublicationID
	if _, err = BuildCandidate(before, []ReviewedBatch{other}); !errors.Is(err, ErrContentInvalid) {
		t.Fatal("old exact prerequisite/path survived replacement", err)
	}
}
func TestReleaseRejectsOldUnitBinding(t *testing.T) {
	p := releasePackage()
	base, err := BuildCandidate(Candidate{}, []ReviewedBatch{reviewedBatch(t, p, "44444444-4444-4444-8444-444444444444")})
	if err != nil {
		t.Fatal(err)
	}
	base.PublicationID = "66666666-6666-4666-8666-666666666666"
	p.Version = 2
	p.Knowledge[0].Version = 2
	p.Units[0].Knowledge.Version = 2
	p.Assets[0].Knowledge.Version = 2
	p.Assets[0].SHA256 = strings.Repeat("c", 64)
	if _, err = BuildCandidate(base, []ReviewedBatch{reviewedBatch(t, p, "55555555-5555-4555-8555-555555555555")}); !errors.Is(err, ErrImmutableConflict) && !errors.Is(err, ErrContentInvalid) {
		t.Fatal("old immutable unit acquired new bytes", err)
	}
	p.Units[0].Version = 2
	batch := reviewedBatch(t, p, "55555555-5555-4555-8555-555555555555")
	batch.Submission.Status = "pending"
	batch.Submission.Review = nil
	if _, err = BuildCandidate(base, []ReviewedBatch{batch}); !errors.Is(err, ErrReviewRequired) {
		t.Fatal("unreviewed new unit prepared", err)
	}
}

func TestReleaseReplacesSVGWithNewOwnerAndUnitVersions(t *testing.T) {
	p := releasePackage()
	base, err := BuildCandidate(Candidate{}, []ReviewedBatch{reviewedBatch(t, p, "44444444-4444-4444-8444-444444444444")})
	if err != nil {
		t.Fatal(err)
	}
	base.PublicationID = "66666666-6666-4666-8666-666666666666"
	p.Version = 2
	p.Knowledge[0].Version = 2
	p.Units[0].Version = 2
	p.Units[0].Knowledge.Version = 2
	p.Assets[0].Knowledge.Version = 2
	p.Assets[0].SHA256 = strings.Repeat("c", 64)
	next, err := BuildCandidate(base, []ReviewedBatch{reviewedBatch(t, p, "55555555-5555-4555-8555-555555555555")})
	if err != nil || next.Snapshot.Assets[0].ID != "root-asset" || next.Snapshot.Assets[0].SHA256 != p.Assets[0].SHA256 || next.Snapshot.Bindings[0].Unit.Version != 2 || next.Snapshot.Bindings[0].SHA256 != p.Assets[0].SHA256 {
		t.Fatal("legal same-ID illustration revision rejected", err)
	}
	unchangedOwner := releasePackage()
	unchangedOwner.Version = 3
	unchangedOwner.Units[0].Version = 3
	unchangedOwner.Assets[0].SHA256 = strings.Repeat("d", 64)
	if _, err := BuildCandidate(base, []ReviewedBatch{reviewedBatch(t, unchangedOwner, "55555555-5555-4555-8555-555555555555")}); !errors.Is(err, ErrImmutableConflict) {
		t.Fatal("changed bytes kept old knowledge owner", err)
	}
	if base.Snapshot.Assets[0].SHA256 == p.Assets[0].SHA256 || base.Snapshot.Bindings[0].Unit.Version != 1 {
		t.Fatal("old immutable bytes changed")
	}
}
