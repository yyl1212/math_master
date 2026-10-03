package store_test

import (
	"github.com/yyl1212/math_master/backend/internal/correction"
	"testing"
)

func TestCorrectionTerminalSourceApproved(t *testing.T) {
	f := newCorrectionFixture(t)
	aid, b := f.cSubmittedBasis("learner_a")
	cid := f.cCase()
	p := f.cApproved(cid)
	b.HandledCaseIDs = []string{cid}
	b.PlanRefs = []correction.PlanRef{p}
	if _, e := f.cResult(aid, f.ids["learner_a"], cid, p, b, true, false, true); e != nil {
		t.Fatal("owned terminal job must accept independently approved actual basis", e)
	}
}

func TestCorrectionProcessDatabaseRejectsForeignScopeAndWithdrawnBasis(t *testing.T) {
	for _, kind := range []string{"foreign-scope", "withdrawn"} {
		t.Run(kind, func(t *testing.T) {
			f := newCorrectionFixture(t)
			f.publishLearningGraph()
			aid, b := f.cSubmittedBasis("learner_a")
			cid := f.cCase()
			if kind == "foreign-scope" {
				k := f.knowledge
				k.ID = "workflow-dependent"
				if e := f.db.QueryRow(`SELECT sha256 FROM knowledge_versions WHERE id=$1 AND version=1`, k.ID).Scan(&k.SHA256); e != nil {
					t.Fatal(e)
				}
				cid = f.registerRule(&k, 1).ID
			} else {
				f.legacyCorrectionWithdrawal(b.OriginalItems[0].Identity)
			}
			p := f.cApproved(cid)
			b.HandledCaseIDs = []string{cid}
			b.PlanRefs = []correction.PlanRef{p}
			if _, e := f.cResult(aid, f.ids["learner_a"], cid, p, b, true); e == nil {
				t.Fatal("database accepted forged handled scope or active withdrawn pass", kind)
			}
		})
	}
}
func TestCorrectionProcessDatabaseRequiresCompleteParentClosure(t *testing.T) {
	f := newCorrectionFixture(t)
	aid, b := f.cSubmittedBasis("learner_a")
	cid := f.cCase()
	p := f.cApproved(cid)
	b.HandledCaseIDs = []string{cid}
	b.PlanRefs = []correction.PlanRef{p}
	parent := f.cSealResult(aid, cid, p, b)
	b.ParentResultIDs = []string{parent}
	b.ParentResultID = &parent
	child := f.cSealResult(aid, cid, p, b)
	b.ParentResultIDs = []string{child}
	b.ParentResultID = &child
	if _, e := f.cResult(aid, f.ids["learner_a"], cid, p, b, true); e == nil {
		t.Fatal("missing transitive parent accepted")
	}
}

func TestCorrectionProcessDatabaseRequiresCompleteDependencies(t *testing.T) {
	for _, role := range []string{"audit", "effective"} {
		t.Run(role, func(t *testing.T) {
			f := newCorrectionFixture(t)
			aid, b := f.cSubmittedBasis("learner_a")
			cid := f.cCase()
			p := f.cApproved(cid)
			b.HandledCaseIDs = []string{cid}
			b.PlanRefs = []correction.PlanRef{p}
			if role == "audit" {
				b.AuditDeps = []correction.Dependency{}
			} else {
				b.EffectiveDeps = []correction.Dependency{}
			}
			if _, e := f.cResult(aid, f.ids["learner_a"], cid, p, b, true); e == nil {
				t.Fatal("complete mathematical dependency proof omitted", role)
			}
		})
	}
}
