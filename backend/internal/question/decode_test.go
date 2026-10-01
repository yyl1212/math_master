package question

import (
	"bytes"
	"encoding/json"
	"errors"
	"github.com/yyl1212/math_master/backend/internal/content"
	"strings"
	"testing"
)

const contractJSON = `{"kind":"question-bank","schemaVersion":1,"id":"contract-fixture","version":1,"templates":[],"fixedQuestions":[],"blueprints":[]}`

// A lax decoder accepting aliases, lossy strings, integer exponent syntax or oversized input breaks this contract.
func TestQuestionContractBoundary(t *testing.T) {
	if p, e := DecodePackage(strings.NewReader(contractJSON)); e != nil || p.ID != "contract-fixture" {
		t.Fatalf("valid contract: %v; schema diagnostic: %v", e, contractError)
	}
	invalid := []string{
		strings.Replace(contractJSON, `"question-bank"`, `"content"`, 1),
		strings.Replace(contractJSON, `"schemaVersion":1`, `"schemaVersion":2`, 1),
		strings.Replace(contractJSON, `"version":1`, `"version":1e0`, 1),
		strings.Replace(contractJSON, `"version":1`, `"version":1.0`, 1),
		strings.Replace(contractJSON, `"version":1`, `"version":2147483648`, 1),
		strings.Replace(contractJSON, `"id":`, `"ID":`, 1),
		strings.Replace(contractJSON, `"id":"contract-fixture"`, `"id":"contract-fixture","id":"other"`, 1),
		strings.Replace(contractJSON, `"id":"contract-fixture"`, `"id":"contract-fixture","ID":"other"`, 1),
		strings.Replace(contractJSON, `"templates":[]`, `"templates":null`, 1),
		strings.Replace(contractJSON, `"id":"contract-fixture"`, `"id":"\ud800"`, 1),
		strings.Replace(contractJSON, `"id":"contract-fixture"`, `"id":"\u0000"`, 1),
		strings.Replace(contractJSON, `"blueprints":[]`, `"blueprints":[],"approved":true`, 1),
		contractJSON + ` {}`, `{"x":` + strings.Repeat("[", 33) + `0` + strings.Repeat("]", 33) + `}`,
		string(append([]byte(contractJSON[:5]), 0xff)),
	}
	for i, raw := range invalid {
		t.Run(string(rune('a'+i)), func(t *testing.T) {
			if _, e := DecodePackage(strings.NewReader(raw)); e == nil {
				t.Fatal("accepted invalid contract")
			}
		})
	}
	for _, n := range []int{2097152, 2097153} {
		raw := contractJSON + strings.Repeat(" ", n-len(contractJSON))
		_, e := DecodePackage(strings.NewReader(raw))
		if (e == nil) != (n == 2097152) {
			t.Fatalf("package bytes %d: %v", n, e)
		}
	}
	envelope := `{"catalogueVersion":1,"questionPackage":` + contractJSON + `,"sourceMap":[]}`
	if _, e := DecodeDraft(strings.NewReader(envelope)); e != nil {
		t.Fatal(e)
	}
	for _, n := range []int{4194304, 4194305} {
		_, e := DecodeDraft(bytes.NewReader([]byte(envelope + strings.Repeat(" ", n-len(envelope)))))
		if (e == nil) != (n == 4194304) {
			t.Fatalf("envelope bytes %d: %v", n, e)
		}
	}
	missing := strings.Replace(envelope, `"sourceMap":[]`, `"SourceMap":[]`, 1)
	if _, e := DecodeDraft(strings.NewReader(missing)); e == nil {
		t.Fatal("accepted wrong field casing")
	}
	if _, e := DecodeDraft(strings.NewReader(strings.Replace(envelope, `"catalogueVersion":1`, `"catalogueVersion":1e0`, 1))); e == nil {
		t.Fatal("integer lexeme")
	}
	_, e := DecodePackage(strings.NewReader(contractJSON + strings.Repeat(" ", 2097153)))
	if !errors.Is(e, ErrLimitExceeded) {
		t.Fatal("size sentinel")
	}
}

func TestQuestionNestedContract(t *testing.T) {
	format := "rational"
	template := Template{ID: "template-fixture", Version: 1, Knowledge: Ref{ID: "fractions", Version: 1}, Coverage: []ObjectiveCoverage{}, Units: []Ref{}, Type: "numeric", AnswerFormat: &format, PromptTemplate: "Compute {{left}} + {{right}}", ExplanationTemplate: "The result is {{answer}}.", Engine: EngineSpec{Family: "rational_arithmetic", Operation: "add", GeneratorVersion: 1, VerifierVersion: 1}, Parameters: []Parameter{{Name: "left", Values: []string{"1"}}, {Name: "right", Values: []string{"2"}}}, Constraints: []Constraint{}, Distractors: []Distractor{}, Assets: []AssetRef{}, Sources: []content.Source{{Kind: "original", Author: "Math Master", Title: "Original exercise", URL: "", AccessedAt: "", License: "CC0-1.0", Attribution: ""}}}
	p := QuestionPackage{Kind: "question-bank", SchemaVersion: 1, ID: "contract-fixture", Version: 1, Templates: []Template{template}, FixedQuestions: []FixedQuestion{}, Blueprints: []Blueprint{}}
	raw, _ := json.Marshal(p)
	if _, e := DecodePackage(bytes.NewReader(raw)); e != nil {
		t.Fatal("valid nested contract", e)
	}
	for _, transform := range []func(*Template){
		func(t *Template) { t.Engine.Family = "arbitrary_script" },
		func(t *Template) { t.Engine.Family = "rational_comparison"; t.Engine.Operation = "compare" },
		func(t *Template) { t.Engine.UnknownSide = func() *string { s := "left"; return &s }() },
		func(t *Template) { t.Parameters[0].Name = "script" },
	} {
		var copy QuestionPackage
		_ = json.Unmarshal(raw, &copy)
		transform(&copy.Templates[0])
		bad, _ := json.Marshal(copy)
		if _, e := DecodePackage(bytes.NewReader(bad)); e == nil {
			t.Fatal("invalid family contract accepted")
		}
	}
	for _, depth := range []int{32, 33} {
		raw := strings.Repeat("[", depth) + "0" + strings.Repeat("]", depth)
		_, e := readJSON(strings.NewReader(raw), 1024)
		if (e == nil) != (depth == 32) {
			t.Fatal("depth boundary", depth, e)
		}
	}
}
