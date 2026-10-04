package correction

import (
	"errors"
	"github.com/yyl1212/math_master/backend/internal/question"
	"testing"
)

func TestCorrectionCumulativeConflict(t *testing.T) {
	root := sampleBasis()
	root.EffectiveItems = append([]question.Instance(nil), root.OriginalItems...)
	root.PlanRefs = []PlanRef{}
	p := sampleBasis()
	p.ParentResultID = new(string)
	*p.ParentResultID = testUUID
	p.ParentResultIDs = []string{testUUID}
	child := sampleBasis()
	child.OriginalItems = append([]question.Instance(nil), p.EffectiveItems...)
	child.EffectiveItems = append([]question.Instance(nil), p.EffectiveItems...)
	x := child.EffectiveItems[0]
	x.Identity.ID = "second-correction"
	x.Identity.Version = 2
	child.EffectiveItems[0] = x
	child.PlanRefs = []PlanRef{{ID: testUUID, Version: 2}}
	got, e := ComposeBasis(root, []Basis{p}, []Basis{child})
	if e != nil || got.EffectiveItems[0].Identity.ID != "second-correction" || got.OriginalItems[0].Identity != root.OriginalItems[0].Identity {
		t.Fatalf("explicit original-to-parent-to-child chain lost: %+v %v", got, e)
	}
	r, e := Evaluate(got)
	if e != nil || r.Status != CorrectedPassed {
		t.Fatalf("cumulative original answers not preserved: %+v %v", r, e)
	}
	conflict := sampleBasis()
	conflict.EffectiveItems = append([]question.Instance(nil), conflict.EffectiveItems...)
	bad := conflict.EffectiveItems[0]
	bad.Identity.ID = "branch"
	bad.Body.CorrectNumeric = &question.Rational{Numerator: "888", Denominator: "1"}
	conflict.EffectiveItems[0] = bad
	if _, e := ComposeBasis(root, []Basis{p}, []Basis{conflict}); !errors.Is(e, ErrConflict) {
		t.Fatalf("conflicting approved branches selected by time: %v", e)
	}
	if !RegisteredAlgorithm(1) || RegisteredAlgorithm(0) || RegisteredAlgorithm(2) {
		t.Fatal("unknown grading algorithm registered")
	}
}

func TestCorrectionCumulativeTemplateDependencies(t *testing.T) {
	b := sampleBasis()
	v := 1
	old := question.Identity{ID: "original-family", Version: 1, SHA256: b.OriginalItems[0].Identity.SHA256}
	fresh := question.Identity{ID: "corrected-family", Version: 2, SHA256: b.EffectiveItems[0].Identity.SHA256}
	for n := range b.OriginalItems {
		b.OriginalItems[n].Template = &old
		b.EffectiveItems[n].Template = &fresh
	}
	b.EffectiveItems[4] = b.OriginalItems[4]
	b.EffectiveDeps = []Dependency{{Kind: "template", ID: old.ID, Version: &v, SHA256: old.SHA256}}
	root := sampleBasis()
	root.OriginalItems = b.OriginalItems
	root.EffectiveItems = b.OriginalItems
	root.PlanRefs = []PlanRef{}
	root.EffectiveDeps = b.EffectiveDeps
	composed, e := ComposeBasis(root, nil, []Basis{b})
	if e != nil {
		t.Fatal(e)
	}
	hasOld := false
	for _, d := range composed.EffectiveDeps {
		if d.Kind == "template" && d.ID == old.ID {
			hasOld = true
		}
	}
	if !hasOld {
		t.Fatal("template still used at original fifth position was removed")
	}
	b.EffectiveItems[4] = sampleBasis().EffectiveItems[4]
	b.EffectiveItems[4].Template = &fresh
	composed, e = ComposeBasis(root, nil, []Basis{b})
	if e != nil {
		t.Fatal(e)
	}
	for _, d := range composed.EffectiveDeps {
		if d.Kind == "template" && d.ID == old.ID {
			t.Fatal("unused original template remains an effective dependency")
		}
	}
	hasAudit := false
	for _, d := range composed.AuditDeps {
		if d.Kind == "template" && d.ID == old.ID {
			hasAudit = true
		}
	}
	if !hasAudit {
		t.Fatal("original audit template disappeared")
	}
}
