package question

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/yyl1212/math_master/backend/internal/content"
	"os"
	"testing"
)

func templateFixture() Template {
	format := "rational"
	return Template{ID: "rational-addition", Version: 1, Knowledge: Ref{ID: "fractions", Version: 1}, Coverage: []ObjectiveCoverage{{Knowledge: Ref{ID: "fractions", Version: 1}, ObjectiveIndices: []int{0}}}, Units: []Ref{}, Type: "numeric", AnswerFormat: &format, PromptTemplate: "Calculate {{left}} + {{right}}.", ExplanationTemplate: "Adding these rational numbers gives {{answer}}.", Engine: EngineSpec{Family: "rational_arithmetic", Operation: "add", GeneratorVersion: 1, VerifierVersion: 1}, Parameters: []Parameter{{Name: "left", Values: []string{"1/2", "2/4", "0.50"}}, {Name: "right", Values: []string{"1", "2"}}}, Constraints: []Constraint{}, Distractors: []Distractor{}, Assets: []AssetRef{}, Sources: []content.Source{{Kind: "original", Author: "Math Master", Title: "Original rational arithmetic exercises", URL: "", AccessedAt: "", License: "CC0-1.0", Attribution: ""}}}
}
func cloneTemplate(v Template) Template {
	raw, _ := json.Marshal(v)
	var c Template
	_ = json.Unmarshal(raw, &c)
	return c
}
func integerValues(n int) []string {
	v := []string{}
	for i := 0; i < n; i++ {
		v = append(v, fmt.Sprint(i+1))
	}
	return v
}

// This catches silent skipping, counting aliases as new questions, unbounded Cartesian products and unstable identities.
func TestGeneratorFiniteSpace(t *testing.T) {
	template := templateFixture()
	instances, report, e := Generate(context.Background(), template)
	if e != nil || len(instances) != 2 || report.RawCombinations != 2 || report.ValidInstances != 2 || report.ExcludedCombinations != 0 {
		t.Fatal("exact dedup", report, e)
	}
	want := map[string]bool{"3/2": false, "5/2": false}
	for _, i := range instances {
		if i.Body.CorrectNumeric == nil {
			t.Fatal("no answer")
		}
		k := i.Body.CorrectNumeric.Numerator + "/" + i.Body.CorrectNumeric.Denominator
		if _, ok := want[k]; !ok {
			t.Fatal("wrong answer", k)
		}
		want[k] = true
		if len(i.Identity.ID) != 67 || i.Identity.Version != 1 || VerifyInstance(i) != nil {
			t.Fatal("invalid instance")
		}
	}
	for _, found := range want {
		if !found {
			t.Fatal("missing combination")
		}
	}
	again, _, e := Generate(context.Background(), template)
	raw, _ := json.Marshal(instances)
	rawAgain, _ := json.Marshal(again)
	if e != nil || string(raw) != string(rawAgain) {
		t.Fatal("unstable repeated generation")
	}
	reversed := cloneTemplate(template)
	reversed.Parameters[0], reversed.Parameters[1] = reversed.Parameters[1], reversed.Parameters[0]
	again, _, e = Generate(context.Background(), reversed)
	rawAgain, _ = json.Marshal(again)
	if e != nil || string(raw) != string(rawAgain) {
		t.Fatal("parameter relation order changed identities")
	}
	// The generic budget is 4 parameters / 1000 raw combinations; supported families have exactly two named operands.
	for _, c := range []struct {
		counts []int
		want   int
		valid  bool
	}{{[]int{2, 2, 2, 2}, 16, true}, {[]int{1, 1, 1, 1, 1}, 0, false}, {[]int{32}, 32, true}, {[]int{33}, 0, false}, {[]int{10, 10, 10}, 1000, true}, {[]int{7, 11, 13}, 0, false}} {
		params := []Parameter{}
		for j, n := range c.counts {
			params = append(params, Parameter{Name: fmt.Sprintf("param-%d", j), Values: integerValues(n)})
		}
		_, count, e := normalizeParameters(params)
		if (e == nil) != c.valid || (e == nil && count != c.want) {
			t.Fatal("space budget", c.counts, count, e)
		}
	}
	maxTemplate := cloneTemplate(template)
	maxTemplate.Parameters[0].Values = integerValues(31)
	maxTemplate.Parameters[1].Values = integerValues(32)
	maxInstances, maxReport, e := Generate(context.Background(), maxTemplate)
	if e != nil || len(maxInstances) != 992 || maxReport.RawCombinations != 992 {
		t.Fatal("largest family space", maxReport, e)
	}
	maxTemplate.Parameters[0].Values = integerValues(32)
	if _, _, e := Generate(context.Background(), maxTemplate); !errors.Is(e, ErrLimitExceeded) {
		t.Fatal("1024 combinations accepted")
	}
	divided := cloneTemplate(template)
	divided.Engine.Operation = "divide"
	divided.Parameters = []Parameter{{Name: "left", Values: []string{"1", "2"}}, {Name: "right", Values: []string{"0", "1"}}}
	if _, invalidReport, e := Generate(context.Background(), divided); e == nil || invalidReport.RawCombinations != 4 || invalidReport.ValidInstances != 2 || invalidReport.ExcludedCombinations != 0 {
		t.Fatal("undeclared division-by-zero silently skipped or incomplete counts", invalidReport, e)
	}
	divided.Constraints = []Constraint{NonzeroDivisor}
	out, r, e := Generate(context.Background(), divided)
	if e != nil || len(out) != 2 || r.RawCombinations != 4 || r.ExcludedCombinations != 2 || r.ValidInstances != 2 || len(r.ConstraintCounts) != 1 || r.ConstraintCounts[0].Count != 2 {
		t.Fatal("explicit exclusions", r, e)
	}
	divided.Parameters[1].Values = []string{"0"}
	if _, r, e := Generate(context.Background(), divided); e == nil || r.ExcludedCombinations != 2 || r.ValidInstances != 0 {
		t.Fatal("zero valid instances", r, e)
	}
	bad := cloneTemplate(template)
	bad.Engine.GeneratorVersion = 2
	if _, _, e := Generate(context.Background(), bad); e == nil {
		t.Fatal("unregistered engine")
	}
	bad = cloneTemplate(template)
	bad.Constraints = []Constraint{NonzeroDivisor}
	if _, _, e := Generate(context.Background(), bad); e == nil {
		t.Fatal("inapplicable constraint")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, _, e := Generate(ctx, template); !errors.Is(e, context.Canceled) {
		t.Fatal("cancelled generation", e)
	}
}
func TestMissingOperandUniqueSolution(t *testing.T) {
	for _, c := range []struct {
		op, side, known, result, n, d string
		valid                         bool
	}{
		{"add", "left", "2", "5", "3", "1", true}, {"add", "right", "2", "5", "3", "1", true},
		{"subtract", "left", "2", "5", "7", "1", true}, {"subtract", "right", "2", "5", "-3", "1", true},
		{"multiply", "left", "2", "5", "5", "2", true}, {"multiply", "right", "2", "5", "5", "2", true},
		{"multiply", "left", "0", "0", "", "", false}, {"multiply", "right", "0", "1", "", "", false},
		{"divide", "left", "2", "5", "10", "1", true}, {"divide", "right", "2", "5", "2", "5", true},
		{"divide", "left", "0", "1", "", "", false}, {"divide", "right", "0", "0", "", "", false}, {"divide", "right", "1", "0", "", "", false}, {"divide", "right", "0", "1", "", "", false},
	} {
		temp := templateFixture()
		temp.Engine = EngineSpec{Family: "missing_operand", Operation: c.op, UnknownSide: &c.side, GeneratorVersion: 1, VerifierVersion: 1}
		temp.Parameters = []Parameter{{Name: "known", Values: []string{c.known}}, {Name: "result", Values: []string{c.result}}}
		temp.PromptTemplate = "Find the missing operand with known {{known}} and result {{result}}."
		temp.ExplanationTemplate = "The unique missing operand is {{answer}}."
		instances, _, e := Generate(context.Background(), temp)
		if (e == nil) != c.valid {
			t.Fatalf("%v: %v", c, e)
		}
		if e == nil && (len(instances) != 1 || *instances[0].Body.CorrectNumeric != (Rational{c.n, c.d}) || VerifyInstance(instances[0]) != nil) {
			t.Fatalf("wrong solution %v", c)
		}
	}
}
func TestTemplateChoiceEquivalence(t *testing.T) {
	temp := templateFixture()
	temp.Type = "single_choice"
	temp.AnswerFormat = nil
	temp.Distractors = []Distractor{PlusOne, MinusOne}
	temp.Parameters = []Parameter{{Name: "left", Values: []string{"1"}}, {Name: "right", Values: []string{"1"}}}
	instances, _, e := Generate(context.Background(), temp)
	if e != nil || len(instances[0].Body.Choices) != 3 || VerifyInstance(instances[0]) != nil {
		t.Fatal("choice generation", e)
	}
	mutated := instances[0]
	mutated.Body.Choices = append([]Choice{}, mutated.Body.Choices...)
	mutated.Body.Choices[0].Text = "1/2"
	mutated.Body.Choices[1].Text = "2/4"
	_, mutated.Identity.SHA256, _ = CanonicalInstance(mutated)
	if e := VerifyInstance(mutated); e == nil {
		t.Fatal("equivalent numeric choices accepted")
	}
	temp.Parameters[0].Values = []string{"0"}
	temp.Parameters[1].Values = []string{"0"}
	temp.Distractors = []Distractor{Negate}
	if _, _, e := Generate(context.Background(), temp); e == nil {
		t.Fatal("distractor equals answer")
	}
	temp.Distractors = []Distractor{Reciprocal}
	if _, _, e := Generate(context.Background(), temp); e == nil {
		t.Fatal("undefined reciprocal silently removed")
	}
	temp.Distractors = []Distractor{PlusOne, PlusOne}
	if _, _, e := Generate(context.Background(), temp); e == nil {
		t.Fatal("duplicate distractor")
	}
}

func TestTechnicalQuestionPackage(t *testing.T) {
	file, e := os.Open("../../../content/questions/elementary-rationals.v1.json")
	if e != nil {
		t.Fatal(e)
	}
	defer file.Close()
	pkg, e := DecodePackage(file)
	if e != nil {
		t.Fatal(e)
	}
	counts := map[string]int{"rational_arithmetic": 10, "rational_comparison": 8, "missing_operand": 10}
	for _, temp := range pkg.Templates {
		instances, report, e := Generate(context.Background(), temp)
		if e != nil || len(instances) != counts[temp.Engine.Family] || report.ValidInstances != counts[temp.Engine.Family] {
			t.Fatal(temp.ID, report, e)
		}
		for _, i := range instances {
			if e := VerifyInstance(i); e != nil {
				t.Fatal(e)
			}
		}
	}
}
