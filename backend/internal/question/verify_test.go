package question

import (
	"context"
	"encoding/json"
	"testing"
)

// Rehash each mutation so an integrity check cannot conceal reuse of the answer algorithm.
func TestVerifierRejectsMutatedGeneratedAnswer(t *testing.T) {
	for _, family := range []string{"rational_arithmetic", "rational_comparison", "missing_operand"} {
		for _, typ := range []string{"numeric", "single_choice"} {
			if family == "rational_comparison" && typ == "numeric" {
				continue
			}
			temp := templateFixture()
			temp.Type = typ
			if typ == "single_choice" {
				temp.AnswerFormat = nil
				temp.Distractors = []Distractor{PlusOne, MinusOne}
			}
			if family == "missing_operand" {
				side := "right"
				temp.Engine.Family = family
				temp.Engine.UnknownSide = &side
				temp.Parameters = []Parameter{{Name: "known", Values: []string{"2"}}, {Name: "result", Values: []string{"5"}}}
				temp.PromptTemplate = "Known {{known}} and result {{result}}."
			}
			if family == "rational_comparison" {
				temp.Engine.Family = family
				temp.Engine.Operation = "compare"
				temp.Distractors = []Distractor{}
			}
			generated, _, e := Generate(context.Background(), temp)
			if e != nil {
				t.Fatal(e)
			}
			raw, _ := json.Marshal(generated[0])
			var bad Instance
			_ = json.Unmarshal(raw, &bad)
			if typ == "numeric" {
				bad.Body.CorrectNumeric = &Rational{Numerator: "999", Denominator: "1"}
			} else {
				for _, choice := range bad.Body.Choices {
					if choice.ID != *bad.Body.CorrectChoiceID {
						s := choice.ID
						bad.Body.CorrectChoiceID = &s
						break
					}
				}
			}
			_, bad.Identity.SHA256, _ = CanonicalInstance(bad)
			if e := VerifyInstance(bad); e == nil {
				t.Fatalf("mutated %s %s answer accepted", family, typ)
			}
		}
	}
}

func TestVerifierFixedConceptIntegrity(t *testing.T) {
	choice := "one"
	i := Instance{Identity: Identity{ID: "concept-fraction", Version: 1}, Origin: "fixed", Parameters: []ParameterValue{}, Body: QuestionBody{Type: "single_choice", Knowledge: Ref{ID: "fractions", Version: 1}, Coverage: []ObjectiveCoverage{}, Units: []Ref{}, Prompt: "Which description identifies a fraction?", Explanation: "A fraction expresses a ratio.", Choices: []Choice{{ID: "one", Text: "A ratio"}, {ID: "two", Text: "A geometric shape"}}, CorrectChoiceID: &choice, Assets: []AssetRef{}, Sources: nil}}
	_, i.Identity.SHA256, _ = CanonicalInstance(i)
	if e := VerifyInstance(i); e != nil {
		t.Fatal(e)
	}
	bad := i
	bad.Identity.SHA256 = "bad"
	if VerifyInstance(bad) == nil {
		t.Fatal("concept skipped byte integrity")
	}
	bad = i
	bad.Origin = "template"
	if VerifyInstance(bad) == nil {
		t.Fatal("concept inherited generated provenance without template")
	}
}

func TestVerifierIntegerIdentities(t *testing.T) {
	for _, operation := range []string{"add", "subtract", "multiply", "divide"} {
		temp := templateFixture()
		temp.Engine.Operation = operation
		temp.Parameters = []Parameter{{Name: "left", Values: []string{"-2/3", "0", "3/5"}}, {Name: "right", Values: []string{"-1/7", "2/5"}}}
		instances, _, e := Generate(context.Background(), temp)
		if e != nil {
			t.Fatal(operation, e)
		}
		for _, i := range instances {
			if VerifyInstance(i) != nil {
				t.Fatal("valid integer identity", operation)
			}
			bad := i
			bad.Body.CorrectNumeric = &Rational{Numerator: "999", Denominator: "1"}
			_, bad.Identity.SHA256, _ = CanonicalInstance(bad)
			if VerifyInstance(bad) == nil {
				t.Fatal("wrong integer identity", operation)
			}
		}
	}
}
