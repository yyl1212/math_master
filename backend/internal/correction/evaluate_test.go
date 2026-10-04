package correction

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/yyl1212/math_master/backend/internal/assessment"
	"github.com/yyl1212/math_master/backend/internal/question"
	"strings"
	"testing"
)

func sampleInstance(n int, answer string) question.Instance {
	format := "rational"
	return question.Instance{Identity: question.Identity{ID: fmt.Sprintf("item-%d", n), Version: 1, SHA256: strings.Repeat("a", 64)}, Origin: "fixed", Parameters: []question.ParameterValue{}, Body: question.QuestionBody{Type: "numeric", Knowledge: question.Ref{ID: "addition", Version: 1}, Coverage: []question.ObjectiveCoverage{{Knowledge: question.Ref{ID: "addition", Version: 1}, ObjectiveIndices: []int{0}}}, Units: []question.Ref{}, Prompt: fmt.Sprintf("Compute %d + 1.", n), Explanation: "Original explanation.", AnswerFormat: &format, Choices: []question.Choice{}, CorrectNumeric: &question.Rational{Numerator: answer, Denominator: "1"}, Assets: []question.AssetRef{}}}
}
func sampleBasis() Basis {
	b := Basis{OriginalSeal: assessment.Seal{Kind: "assessment", RuleVersion: 1, Knowledge: question.Identity{ID: "addition", Version: 1, SHA256: strings.Repeat("b", 64)}, Core: []int{0}, Items: []assessment.ItemBinding{}}, OriginalAnswers: []assessment.Answer{}, OriginalItems: []question.Instance{}, EffectiveItems: []question.Instance{}, HandledCaseIDs: []string{testUUID}, PlanRefs: []PlanRef{{ID: testUUID, Version: 1}}}
	for n := 1; n <= 5; n++ {
		i := sampleInstance(n, "99")
		raw := fmt.Sprint(n + 1)
		b.OriginalItems = append(b.OriginalItems, i)
		e := i
		e.Identity.ID = fmt.Sprintf("corrected-%d", n)
		e.Identity.SHA256 = strings.Repeat("c", 64)
		e.Body.CorrectNumeric = &question.Rational{Numerator: raw, Denominator: "1"}
		b.EffectiveItems = append(b.EffectiveItems, e)
		b.OriginalAnswers = append(b.OriginalAnswers, assessment.Answer{Kind: "numeric", Raw: &raw})
		b.OriginalSeal.Items = append(b.OriginalSeal.Items, assessment.ItemBinding{Position: n, Instance: i.Identity, Coverage: []int{0}})
	}
	return b
}
func TestCorrectionFiveOriginalAnswers(t *testing.T) {
	for _, score := range []int{5, 4, 3} {
		t.Run(fmt.Sprint(score), func(t *testing.T) {
			b := sampleBasis()
			for n := score; n < 5; n++ {
				b.OriginalAnswers[n] = assessment.Answer{Kind: "skipped"}
			}
			old, _ := json.Marshal(struct {
				Seal    assessment.Seal
				Answers []assessment.Answer
			}{b.OriginalSeal, b.OriginalAnswers})
			got, e := Evaluate(b)
			if e != nil || got.Score == nil || *got.Score != score || got.Passed == nil || *got.Passed != (score >= 4) {
				t.Fatalf("original answers score %d: %+v %v", score, got, e)
			}
			want := CorrectedFailed
			if score >= 4 {
				want = CorrectedPassed
			}
			if got.Status != want {
				t.Fatalf("status: %s", got.Status)
			}
			for n := score; n < 5; n++ {
				if got.Correct[n] {
					t.Fatal("skipped marked correct")
				}
			}
			after, _ := json.Marshal(struct {
				Seal    assessment.Seal
				Answers []assessment.Answer
			}{b.OriginalSeal, b.OriginalAnswers})
			if !bytes.Equal(old, after) {
				t.Fatal("original evidence rewritten")
			}
		})
	}
	b := sampleBasis()
	b.EffectiveItems = b.EffectiveItems[:4]
	got, e := Evaluate(b)
	if e != nil || got.Status != RetakeRequired || got.Score != nil || got.Passed != nil {
		t.Fatalf("shrunk five-item denominator: %+v %v", got, e)
	}
}
func TestCorrectionFixedCoverage(t *testing.T) {
	b := sampleBasis()
	b.EffectiveItems[4].Body.Coverage = []question.ObjectiveCoverage{}
	r, e := Evaluate(b)
	if e != nil || r.Status != RetakeRequired || r.Reason != CoverageChanged || r.Score != nil {
		t.Fatalf("coverage mismatch grants pass: %+v %v", r, e)
	}
	b = sampleBasis()
	b.EffectiveItems[0].Body.Knowledge.Version = 2
	r, e = Evaluate(b)
	if e != nil || r.Reason != KnowledgeChanged || r.Passed != nil {
		t.Fatalf("knowledge migration: %+v %v", r, e)
	}
}
func TestCorrectionPracticeOnly(t *testing.T) {
	b := sampleBasis()
	b.OriginalSeal.Kind = "practice"
	b.OriginalSeal.Core = []int{}
	b.OriginalSeal.Items = b.OriginalSeal.Items[:1]
	b.OriginalItems = b.OriginalItems[:1]
	b.EffectiveItems = b.EffectiveItems[:1]
	b.OriginalAnswers = b.OriginalAnswers[:1]
	r, e := Evaluate(b)
	if e != nil || len(r.Correct) != 1 || !r.Correct[0] || r.Score != nil || r.Passed != nil {
		t.Fatalf("practice manufactures qualification: %+v %v", r, e)
	}
}

func TestCorrectionFiveChoiceAndExactRational(t *testing.T) {
	b := sampleBasis()
	choice := "a"
	wrong := "b"
	b.OriginalItems[0].Body.Type = "single_choice"
	b.OriginalItems[0].Body.AnswerFormat = nil
	b.OriginalItems[0].Body.CorrectNumeric = nil
	b.OriginalItems[0].Body.Choices = []question.Choice{{ID: "a", Text: "2"}, {ID: "b", Text: "99"}}
	b.OriginalItems[0].Body.CorrectChoiceID = &wrong
	b.EffectiveItems[0].Body = b.OriginalItems[0].Body
	b.EffectiveItems[0].Body.CorrectChoiceID = &choice
	b.OriginalAnswers[0] = assessment.Answer{Kind: "choice", ChoiceID: &choice}
	raw := "6/2"
	b.OriginalAnswers[1] = assessment.Answer{Kind: "numeric", Raw: &raw}
	r, e := Evaluate(b)
	if e != nil || r.Score == nil || *r.Score != 5 {
		t.Fatalf("exact rational and original choice ID: %+v %v", r, e)
	}
	b.EffectiveItems[0].Body.Choices = []question.Choice{{ID: "b", Text: "99"}, {ID: "a", Text: "2"}}
	r, e = Evaluate(b)
	if e != nil || r.Status != RetakeRequired || r.Passed != nil {
		t.Fatalf("changed choice order grants pass: %+v %v", r, e)
	}
}
func TestCorrectionFixedUnapprovedAndUnknownRule(t *testing.T) {
	b := sampleBasis()
	b.PlanRefs = []PlanRef{}
	r, e := Evaluate(b)
	if e != nil || r.Status != AwaitingReview || r.Reason != NoApprovedBasis || r.Score != nil {
		t.Fatalf("unapproved answer basis scored: %+v %v", r, e)
	}
	b.HandledCaseIDs = []string{}
	r, e = Evaluate(b)
	if e != nil || r.Status != CheckedUnaffected || r.Passed != nil {
		t.Fatalf("unaffected metadata fabricates grading: %+v %v", r, e)
	}
	b = sampleBasis()
	b.OriginalSeal.RuleVersion = 2
	r, e = Evaluate(b)
	if e != nil || r.Status != RetakeRequired || r.Score != nil {
		t.Fatalf("unsupported original rule grants pass: %+v %v", r, e)
	}
}
