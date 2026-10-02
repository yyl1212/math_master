package question

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
)

func candidateFixture(t *testing.T, mutations ...func(*DraftInput)) (ApprovedSubmission, ReferenceSnapshot) {
	t.Helper()
	in, refs := sealFixture()
	for _, mutate := range mutations {
		mutate(&in)
	}
	sealed, gate, err := ValidateAndSeal(context.Background(), in, refs)
	if err != nil {
		t.Fatal(err)
	}
	ids := []Identity{}
	for _, i := range sealed.Instances {
		ids = append(ids, i.Identity)
	}
	g, v := UsedEngineVersions(sealed.Package, sealed.Instances)
	frozen := FrozenBody{CatalogueVersion: in.CatalogueVersion, CatalogueSHA256: refs.CatalogueSHA256, QuestionPackage: sealed.Package, SourceMap: []SourceLink{}, AuthorIDs: []string{actorID}, Resolved: sealed.Resolved, Objectives: sealed.Objectives, Generation: sealed.Generation, InstanceIdentities: ids, Coverage: gate.Coverage, GeneratorVersions: g, VerifierVersions: v}
	_, frozen.FrozenDigest, err = CanonicalFrozen(frozen, sealed.Instances)
	if err != nil {
		t.Fatal(err)
	}
	subID := "22222222-2222-4222-8222-222222222222"
	d := ReviewDecision{ID: "33333333-3333-4333-8333-333333333333", SubmissionID: subID, ReviewerID: "44444444-4444-4444-8444-444444444444", FrozenDigest: frozen.FrozenDigest, Decision: "approve", Checks: ReviewChecks{true, true, true, true, true, true}, IndependenceNote: "I independently reviewed this immutable test payload.", Note: "All mathematics and frozen evidence reviewed."}
	return ApprovedSubmission{SubmissionID: subID, Frozen: frozen, Instances: sealed.Instances, Decision: d}, refs
}
func TestQuestionBuildCandidate(t *testing.T) {
	sub, refs := candidateFixture(t)
	ctx := context.Background()
	c, err := BuildCandidate(ctx, BaseManifest{}, []ApprovedSubmission{sub}, refs)
	if err != nil || len(c.Manifest.Members) != 7 || c.Diff.Added != 7 {
		t.Fatal("bad candidate", err)
	}
	_, sha, err := CanonicalManifest(c.Manifest)
	if err != nil || !ValidSHA(sha) {
		t.Fatal(err)
	}
	head := "55555555-5555-4555-8555-555555555555"
	base := BaseManifest{Head: &head, Manifest: &c.Manifest, Instances: c.Instances, Templates: c.Templates, Blueprints: c.Blueprints}
	next, err := BuildCandidate(ctx, base, []ApprovedSubmission{sub}, refs)
	if err != nil || next.Diff.Added != 0 {
		t.Fatal("same approved version replaced history", err)
	}
	for _, m := range next.Manifest.Members {
		if m.Evidence.InheritedFrom != nil || m.Evidence.SubmissionID != sub.SubmissionID {
			t.Fatal("reselected proof treated as inherited")
		}
	}
	additional, refs := candidateFixture(t, func(in *DraftInput) {
		in.QuestionPackage.ID = "additional-bank"
		in.QuestionPackage.Templates[0].ID = "additional-template"
		in.QuestionPackage.Blueprints = []Blueprint{}
	})
	additional.SubmissionID = "66666666-6666-4666-8666-666666666666"
	additional.Decision.SubmissionID = additional.SubmissionID
	additional.Decision.ID = "77777777-7777-4777-8777-777777777777"
	inherited, err := BuildCandidate(ctx, base, []ApprovedSubmission{additional}, refs)
	if err != nil {
		t.Fatal(err)
	}
	inheritedCount := 0
	for _, m := range inherited.Manifest.Members {
		if m.Evidence.InheritedFrom != nil {
			inheritedCount++
			if *m.Evidence.InheritedFrom != head || m.Evidence.SubmissionID != sub.SubmissionID {
				t.Fatal("inheritance proof lost")
			}
		}
	}
	if inheritedCount != 7 {
		t.Fatal("unselected evidence lost", inheritedCount)
	}
	for _, mutate := range []func(*ApprovedSubmission){func(s *ApprovedSubmission) { s.Decision.FrozenDigest = strings.Repeat("0", 64) }, func(s *ApprovedSubmission) { s.Decision.ReviewerID = actorID }, func(s *ApprovedSubmission) { s.Instances[0].Body.Explanation += " tampered" }, func(s *ApprovedSubmission) { s.Decision.Checks.Generation = false }} {
		original, refs := candidateFixture(t)
		mutate(&original)
		if _, err := BuildCandidate(ctx, BaseManifest{}, []ApprovedSubmission{original}, refs); err == nil {
			t.Fatal("invalid approved payload accepted")
		}
	}
	for _, n := range []int{1, 20} {
		subs := []ApprovedSubmission{}
		for i := 0; i < n; i++ {
			copy := sub
			copy.SubmissionID = fmt.Sprintf("%08x-2222-4222-8222-222222222222", i+1)
			copy.Decision.SubmissionID = copy.SubmissionID
			copy.Decision.ID = fmt.Sprintf("%08x-3333-4333-8333-333333333333", i+1)
			subs = append(subs, copy)
		}
		if _, err := BuildCandidate(ctx, BaseManifest{}, subs, refs); err != nil {
			t.Fatal("legal selected count", n, err)
		}
	}
	if _, err := BuildCandidate(ctx, BaseManifest{}, make([]ApprovedSubmission, 21), refs); !errors.Is(err, ErrLimitExceeded) {
		t.Fatal("selected cap omitted", err)
	}
	cancelled, cancel := context.WithCancel(ctx)
	cancel()
	if _, err := BuildCandidate(cancelled, BaseManifest{}, []ApprovedSubmission{sub}, refs); !errors.Is(err, context.Canceled) {
		t.Fatal("context ignored", err)
	}
}

func TestQuestionCandidateProvenanceDoesNotReplaceMath(t *testing.T) {
	sub, refs := candidateFixture(t)
	ctx := context.Background()
	c, err := BuildCandidate(ctx, BaseManifest{}, []ApprovedSubmission{sub}, refs)
	if err != nil {
		t.Fatal(err)
	}
	head := "55555555-5555-4555-8555-555555555555"
	base := BaseManifest{Head: &head, Manifest: &c.Manifest, Instances: c.Instances, Templates: c.Templates, Blueprints: c.Blueprints}
	sub.Frozen.QuestionPackage.ID = "new-provenance-package"
	_, sub.Frozen.FrozenDigest, err = CanonicalFrozen(sub.Frozen, sub.Instances)
	if err != nil {
		t.Fatal(err)
	}
	sub.Decision.FrozenDigest = sub.Frozen.FrozenDigest
	changed, err := BuildCandidate(ctx, base, []ApprovedSubmission{sub}, refs)
	if err != nil || changed.Diff != (DiffSummary{}) || len(changed.Changes) != 0 {
		t.Fatal("provenance was counted as a mathematical replacement", changed.Diff, err)
	}
	for _, m := range changed.Manifest.Members {
		if m.Identity.PackageID != "new-provenance-package" || m.Evidence.FrozenDigest != sub.Frozen.FrozenDigest || m.Evidence.InheritedFrom != nil {
			t.Fatal("fresh proof not bound to selected package")
		}
	}
}
