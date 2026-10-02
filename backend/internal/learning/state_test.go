package learning

import (
	"github.com/yyl1212/math_master/backend/internal/assessment"
	"github.com/yyl1212/math_master/backend/internal/question"
	"strings"
	"testing"
	"time"
)

func TestLearningStateReviewPrecedence(t *testing.T) {
	for _, tt := range []struct {
		facts StateFacts
		want  State
	}{{StateFacts{}, "unlearned"}, {StateFacts{Started: true}, "learning"}, {StateFacts{Completed: true}, "learned"}, {StateFacts{EffectivePass: true}, "mastered"}, {StateFacts{Completed: true, EffectivePass: true, LatestReviewFailed: true}, "needs-review"}, {StateFacts{Completed: true, HadInvalidatedPass: true}, "needs-review"}, {StateFacts{HadInvalidatedPass: true, EffectivePass: true}, "mastered"}} {
		if got := ResolveState(tt.facts); got != tt.want {
			t.Fatal(got, tt.want)
		}
	}
}
func TestLearningStateAlternativeEvidenceAndDiagnostic(t *testing.T) {
	k := question.Identity{ID: "fractions", Version: 1, SHA256: strings.Repeat("a", 64)}
	yes := true
	four := 4
	now := time.Now()
	event := "11111111-1111-4111-8111-111111111111"
	good := assessment.AttemptFact{ID: event, Knowledge: k, Mode: "node", Outcome: "passed", Score: &four, Passed: &yes, SubmittedAt: now, Validity: "effective"}
	bad := good
	bad.ID = "22222222-2222-4222-8222-222222222222"
	bad.Validity = "restricted"
	bad.SubmittedAt = now.Add(time.Minute)
	f := EvidenceFacts{Knowledge: k, Current: true, CompletionEventID: &event, Passes: []assessment.AttemptFact{good, bad}}
	got := EvaluateEvidence(f)
	if !got.Qualified || got.Qualification == nil || got.Qualification.EvidenceAttemptID != good.ID || got.Qualification.Kind != "normal" {
		t.Fatal(got)
	}
	f.CompletionEventID = nil
	if EvaluateEvidence(f).Qualified {
		t.Fatal("ordinary pass fabricated reading")
	}
	good.Mode = "diagnostic"
	f.Passes = []assessment.AttemptFact{good}
	got = EvaluateEvidence(f)
	if !got.Qualified || got.Qualification.CompletedEventID != nil || got.Qualification.Kind != "diagnostic" {
		t.Fatal(got)
	}
	f.Current = false
	if EvaluateEvidence(f).Qualified {
		t.Fatal("old version granted")
	}
	f.Current = true
	f.Knowledge.Version = 2
	if EvaluateEvidence(f).Qualified {
		t.Fatal("wrong version granted")
	}
}
