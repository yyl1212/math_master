package question

import (
	"context"
	"strings"
	"testing"
)

func TestQuestionCurrentCoverage(t *testing.T) {
	sub, refs := candidateFixture(t)
	ctx := context.Background()
	c, err := BuildCandidate(ctx, BaseManifest{}, []ApprovedSubmission{sub}, refs)
	if err != nil {
		t.Fatal(err)
	}
	head := "55555555-5555-4555-8555-555555555555"
	initial, err := ComputeCoverage(ctx, c, refs, &head)
	if err != nil || initial.EffectiveInstances != 5 || !initial.Nodes.Items[0].Ready {
		t.Fatal("initial coverage", initial, err)
	}
	reduced, err := WithdrawCandidate(ctx, c, WithdrawalTarget{Kind: "instance", ID: c.Instances[0].Identity.ID, Version: 1})
	if err != nil {
		t.Fatal(err)
	}
	after, err := ComputeCoverage(ctx, reduced, refs, &head)
	if err != nil || after.EffectiveInstances != 4 || after.Nodes.Items[0].Ready {
		t.Fatal("coverage retained a removed fifth question", after, err)
	}
	unavailable := refs
	unavailable.Knowledge = []FixedKnowledge{}
	noOffer, err := ComputeCoverage(ctx, c, unavailable, &head)
	if err != nil || noOffer.EffectiveInstances != 0 || noOffer.Nodes.Items[0].Ready {
		t.Fatal("unpublished exact knowledge still offered", err)
	}
	// One fixed identity covering two nodes counts once in global totals and once in each node.
	extra := Ref{ID: "second-coverage-node", Version: 1}
	refs.Knowledge = append(refs.Knowledge, FixedKnowledge{Identity: Identity{ID: extra.ID, Version: 1, SHA256: strings.Repeat("c", 64)}, Objectives: []string{"A supplementary objective."}})
	fixed := numericFixed("shared-fixed-question")
	fixed.Body.Coverage = append(fixed.Body.Coverage, ObjectiveCoverage{Knowledge: extra, ObjectiveIndices: []int{0}})
	i := Instance{Identity: Identity{ID: fixed.ID, Version: 1}, Origin: "fixed", Parameters: []ParameterValue{}, Body: fixed.Body}
	_, i.Identity.SHA256, _ = CanonicalInstance(i)
	c.Templates = []Template{}
	c.Blueprints = []Blueprint{}
	c.Instances = []Instance{i}
	c.Manifest.Resolved = append(c.Manifest.Resolved, KnownObject{Kind: "knowledge", Ref: extra, SHA256: strings.Repeat("c", 64)})
	c.Manifest.Members = []ManifestMember{{Identity: MemberIdentity{Kind: "instance", ID: i.Identity.ID, Version: 1, SHA256: i.Identity.SHA256, PackageID: "shared-coverage-package", PackageVersion: 1}, Evidence: MemberEvidence{SubmissionID: sub.SubmissionID, DecisionID: sub.Decision.ID, FrozenDigest: sub.Frozen.FrozenDigest}}}
	report, err := ComputeCoverage(ctx, c, refs, &head)
	if err != nil || report.EffectiveInstances != 1 || report.FixedQuestions != 1 || report.PublishedKnowledge != 2 || report.Nodes.Total != 2 || report.Nodes.Items[0].EffectiveInstances != 1 || report.Nodes.Items[1].EffectiveInstances != 1 {
		t.Fatal("multi-node totals double counted", report, err)
	}
}
