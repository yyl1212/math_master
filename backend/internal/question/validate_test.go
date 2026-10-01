package question

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/yyl1212/math_master/backend/internal/content"
	"os"
	"strings"
	"testing"
	"time"
)

func sealFixture() (DraftInput, ReferenceSnapshot) {
	template := templateFixture()
	template.Parameters = []Parameter{{Name: "left", Values: integerValues(5)}, {Name: "right", Values: []string{"1"}}}
	blueprint := Blueprint{ID: "rational-assessment", Version: 1, Knowledge: template.Knowledge, CoreObjectiveIndices: []int{0}, Sources: []BlueprintSource{{Kind: "template", Ref: Ref{ID: template.ID, Version: 1}}}, CoverageNote: "Objective zero is core; recognizing order is supplementary.", RuleVersion: 1, QuestionCount: 5, PassCount: 4}
	input := DraftInput{CatalogueVersion: 1, QuestionPackage: QuestionPackage{Kind: "question-bank", SchemaVersion: 1, ID: "seal-fixture", Version: 1, Templates: []Template{template}, FixedQuestions: []FixedQuestion{}, Blueprints: []Blueprint{blueprint}}, SourceMap: []SourceLink{}}
	head := actorID
	refs := ReferenceSnapshot{KnowledgeHead: &head, CatalogueVersion: 1, CatalogueSHA256: strings.Repeat("b", 64), Knowledge: []FixedKnowledge{{Identity: Identity{ID: "fractions", Version: 1, SHA256: strings.Repeat("a", 64)}, Title: "Rational numbers", TitleZh: "有理数", Objectives: []string{"Compute rational sums.", "Recognize equality and order."}}}, Units: []FixedUnit{}, Assets: []content.AssetView{}}
	return input, refs
}
func cloneDraft(d DraftInput) DraftInput {
	b, _ := json.Marshal(d)
	var out DraftInput
	_ = json.Unmarshal(b, &out)
	return out
}
func numericFixed(id string) FixedQuestion {
	format := "rational"
	t := templateFixture()
	return FixedQuestion{ID: id, Version: 1, Body: QuestionBody{Type: "numeric", Knowledge: t.Knowledge, Coverage: t.Coverage, Units: []Ref{}, Prompt: "Compute 1 + 2.", Explanation: "Adding one and two gives three.", AnswerFormat: &format, Choices: []Choice{}, CorrectNumeric: &Rational{Numerator: "3", Denominator: "1"}, Witness: &VerificationWitness{Engine: t.Engine, Parameters: []ParameterValue{{Name: "left", Value: "1"}, {Name: "right", Value: "2"}}}, Assets: []AssetRef{}, Sources: t.Sources}}
}
func TestQuestionFixedReferences(t *testing.T) {
	input, refs := sealFixture()
	sealed, report, e := ValidateAndSeal(context.Background(), input, refs)
	if e != nil || !report.ReadyToSubmit || len(sealed.Instances) != 5 || len(sealed.Objectives) == 0 || sealed.Objectives[0].Text != "Compute rational sums." || sealed.Objectives[0].Knowledge.SHA256 != strings.Repeat("a", 64) {
		t.Fatal("valid fixed references", report, e)
	}
	if len(report.Coverage) != 1 || !report.Coverage[0].Ready || len(report.Coverage[0].SupplementaryObjectiveIndices) != 1 || report.Coverage[0].SupplementaryObjectiveIndices[0] != 1 {
		t.Fatal("supplementary goals not explicit")
	}
	for _, mutate := range []func(*DraftInput){
		func(d *DraftInput) { d.QuestionPackage.Templates[0].Coverage[0].ObjectiveIndices = []int{2} },
		func(d *DraftInput) {
			d.QuestionPackage.Templates[0].Knowledge.Version = 2
			d.QuestionPackage.Templates[0].Coverage[0].Knowledge.Version = 2
		},
		func(d *DraftInput) { d.QuestionPackage.Templates[0].Units = []Ref{{ID: "private-unit", Version: 1}} },
		func(d *DraftInput) {
			d.QuestionPackage.Templates[0].Assets = []AssetRef{{ID: "private-asset", SHA256: strings.Repeat("c", 64)}}
		},
		func(d *DraftInput) { d.QuestionPackage.Templates[0].Coverage[0].Knowledge.ID = "other-knowledge" },
		func(d *DraftInput) { d.QuestionPackage.Blueprints[0].Sources[0].Ref.ID = "other-package-template" },
		func(d *DraftInput) { d.QuestionPackage.Blueprints[0].CoreObjectiveIndices = []int{1} },
	} {
		bad := cloneDraft(input)
		mutate(&bad)
		_, r, e := ValidateAndSeal(context.Background(), bad, refs)
		if e == nil || r.ReadyToSubmit {
			t.Fatal("invalid reference ready", r)
		}
	}
	for n := 3; n <= 4; n++ {
		d := cloneDraft(input)
		r := refs
		r.Knowledge = append([]FixedKnowledge{}, refs.Knowledge...)
		for j := 0; j < n; j++ {
			ref := Ref{ID: fmt.Sprintf("extra-%d", j), Version: 1}
			d.QuestionPackage.Templates[0].Coverage = append(d.QuestionPackage.Templates[0].Coverage, ObjectiveCoverage{Knowledge: ref, ObjectiveIndices: []int{0}})
			r.Knowledge = append(r.Knowledge, FixedKnowledge{Identity: Identity{ID: ref.ID, Version: 1, SHA256: strings.Repeat("c", 64)}, Objectives: []string{"Original additional objective."}})
		}
		_, report, e := ValidateAndSeal(context.Background(), d, r)
		if (e == nil) != (n == 3) || (n == 4 && report.ReadyToSubmit) {
			t.Fatal("extra coverage bound", n, report, e)
		}
	}
	unavailable := refs
	unavailable.Knowledge = []FixedKnowledge{}
	r, e := ValidateEditable(context.Background(), input, unavailable)
	if e != nil || r.ReadyToSubmit || r.StructuralTotal != 0 || r.CompletenessTotal == 0 {
		t.Fatal("pending references cannot be saved safely", r, e)
	}
	updated := refs
	updated.Knowledge = append([]FixedKnowledge{}, refs.Knowledge...)
	updated.Knowledge[0].Identity.SHA256 = strings.Repeat("c", 64)
	_, r2, e := ValidateAndSeal(context.Background(), input, updated)
	if e != nil || r.Digest == r2.Digest || report.Digest == r2.Digest {
		t.Fatal("reference identity absent from digest", e)
	}
}
func TestQuestionSealLimits(t *testing.T) {
	input, refs := sealFixture()
	for _, kind := range []string{"template", "fixed", "blueprint"} {
		maximum := 50
		if kind == "fixed" {
			maximum = 200
		} else if kind == "blueprint" {
			maximum = 100
		}
		for _, n := range []int{maximum, maximum + 1} {
			d := cloneDraft(input)
			switch kind {
			case "template":
				d.QuestionPackage.Templates = []Template{}
				for j := 0; j < n; j++ {
					temp := cloneTemplate(input.QuestionPackage.Templates[0])
					if j != 0 {
						temp.ID = fmt.Sprintf("template-%d", j)
					}
					d.QuestionPackage.Templates = append(d.QuestionPackage.Templates, temp)
				}
			case "fixed":
				for j := 0; j < n; j++ {
					d.QuestionPackage.FixedQuestions = append(d.QuestionPackage.FixedQuestions, numericFixed(fmt.Sprintf("fixed-%d", j)))
				}
			case "blueprint":
				d.QuestionPackage.Blueprints = []Blueprint{}
				for j := 0; j < n; j++ {
					bp := input.QuestionPackage.Blueprints[0]
					bp.ID = fmt.Sprintf("blueprint-%d", j)
					d.QuestionPackage.Blueprints = append(d.QuestionPackage.Blueprints, bp)
				}
			}
			_, r, e := ValidateAndSeal(context.Background(), d, refs)
			if (e == nil) != (n == maximum) {
				t.Fatalf("%s %d: %v %v", kind, n, r, e)
			}
		}
	}
	for _, n := range []int{1000, 1001} {
		d := cloneDraft(input)
		d.QuestionPackage.Templates = []Template{}
		for j := 0; j < 50; j++ {
			temp := cloneTemplate(input.QuestionPackage.Templates[0])
			if j != 0 {
				temp.ID = fmt.Sprintf("large-template-%d", j)
			}
			count := 20
			if j == 49 && n == 1001 {
				count = 21
			}
			temp.Parameters[0].Values = integerValues(count)
			d.QuestionPackage.Templates = append(d.QuestionPackage.Templates, temp)
		}
		start := time.Now()
		_, r, e := ValidateAndSeal(context.Background(), d, refs)
		if (e == nil) != (n == 1000) {
			t.Fatal("package generation limit", n, r, e)
		}
		if elapsed := time.Since(start); elapsed >= 8*time.Second {
			t.Fatal("legal capacity exceeded 8 seconds")
		}
		t.Logf("%d generated full validation: %s", n, time.Since(start))
	}
	for _, n := range []int{8192, 8193} {
		d := cloneDraft(input)
		f := numericFixed("long-prompt")
		f.Body.Prompt = strings.Repeat("x", n-len(f.Body.Explanation))
		d.QuestionPackage.FixedQuestions = []FixedQuestion{f}
		_, r, e := ValidateAndSeal(context.Background(), d, refs)
		if (e == nil) != (n == 8192) {
			t.Fatal("text byte limit", n, r, e)
		}
	}
	for _, n := range []int{6, 7} {
		d := cloneDraft(input)
		f := numericFixed("concept-options")
		f.Body.Type = "single_choice"
		f.Body.AnswerFormat = nil
		f.Body.CorrectNumeric = nil
		f.Body.Witness = nil
		answer := "option-0"
		f.Body.CorrectChoiceID = &answer
		for j := 0; j < n; j++ {
			f.Body.Choices = append(f.Body.Choices, Choice{ID: fmt.Sprintf("option-%d", j), Text: fmt.Sprintf("Original option %d", j)})
		}
		d.QuestionPackage.FixedQuestions = []FixedQuestion{f}
		_, r, e := ValidateAndSeal(context.Background(), d, refs)
		if (e == nil) != (n == 6) {
			t.Fatal("choice bound", n, r, e)
		}
	}
	for _, n := range []int{8, 9} {
		d := cloneDraft(input)
		r := refs
		r.Assets = []content.AssetView{}
		for j := 0; j < n; j++ {
			id := fmt.Sprintf("asset-%d", j)
			sha := strings.Repeat("c", 64)
			d.QuestionPackage.Templates[0].Assets = append(d.QuestionPackage.Templates[0].Assets, AssetRef{ID: id, SHA256: sha})
			r.Assets = append(r.Assets, content.AssetView{ID: id, SHA256: sha, Knowledge: Ref{ID: "fractions", Version: 1}, Author: "Math Master", License: "CC0-1.0"})
		}
		_, report, e := ValidateAndSeal(context.Background(), d, r)
		if (e == nil) != (n == 8) {
			t.Fatal("asset bound", n, report, e)
		}
	}
	body := FrozenBody{AuthorIDs: []string{}, SourceMap: []SourceLink{{Note: ""}}}
	raw, _, _ := CanonicalFrozen(body, []Instance{})
	base := len(raw)
	for _, n := range []int{4194304, 4194305} {
		body.SourceMap[0].Note = strings.Repeat("x", n-base)
		size, e := frozenSize(body, []Instance{})
		if size != n || (e == nil) != (n == 4194304) {
			t.Fatal("exact frozen byte budget", size, e)
		}
	}
}
func TestFixedNumericWitness(t *testing.T) {
	input, refs := sealFixture()
	input.QuestionPackage.FixedQuestions = []FixedQuestion{numericFixed("fixed-witness")}
	if _, r, e := ValidateAndSeal(context.Background(), input, refs); e != nil || !r.ReadyToSubmit {
		t.Fatal("valid witness", r, e)
	}
	for _, mutate := range []func(*FixedQuestion){func(f *FixedQuestion) { f.Body.Witness = nil }, func(f *FixedQuestion) { f.Body.CorrectNumeric = &Rational{Numerator: "4", Denominator: "1"} }, func(f *FixedQuestion) {
		format := "percentage"
		f.Body.AnswerFormat = &format
		f.Body.CorrectNumeric = &Rational{Numerator: "1", Denominator: "3"}
	}, func(f *FixedQuestion) { f.Body.Witness.Engine.Family = "script" }} {
		d := cloneDraft(input)
		mutate(&d.QuestionPackage.FixedQuestions[0])
		_, r, e := ValidateAndSeal(context.Background(), d, refs)
		if e == nil || r.ReadyToSubmit {
			t.Fatal("invalid fixed witness ready", r)
		}
	}
	concept := numericFixed("concept-human-review")
	concept.Body.Type = "single_choice"
	concept.Body.AnswerFormat = nil
	concept.Body.CorrectNumeric = nil
	concept.Body.Witness = nil
	choice := "ratio"
	concept.Body.CorrectChoiceID = &choice
	concept.Body.Choices = []Choice{{ID: "ratio", Text: "A ratio"}, {ID: "shape", Text: "A shape"}}
	input.QuestionPackage.FixedQuestions = []FixedQuestion{concept}
	_, r, e := ValidateAndSeal(context.Background(), input, refs)
	if e != nil || r.HumanReviewTotal == 0 {
		t.Fatal("concept missing human review", r, e)
	}
	input.QuestionPackage.FixedQuestions = []FixedQuestion{}
	for j := 0; j < 101; j++ {
		fixed := numericFixed(fmt.Sprintf("missing-witness-%d", j))
		fixed.Body.Witness = nil
		input.QuestionPackage.FixedQuestions = append(input.QuestionPackage.FixedQuestions, fixed)
	}
	r, e = ValidateEditable(context.Background(), input, refs)
	display := len(r.StructuralErrors) + len(r.CompletenessErrors) + len(r.HumanReviewRequirements)
	if e != nil || display != 100 || r.CompletenessTotal < 101 || !r.Truncated || r.ReadyToSubmit {
		t.Fatal("display truncation hid failure", r, e)
	}
}

func TestQuestionSealOwnsFixedSnapshot(t *testing.T) {
	input, refs := sealFixture()
	sealed, _, e := ValidateAndSeal(context.Background(), input, refs)
	if e != nil {
		t.Fatal(e)
	}
	before, _ := json.Marshal(sealed)
	refs.Knowledge[0].Objectives[0] = "Changed outside the sealed package"
	*refs.KnowledgeHead = "22222222-2222-4222-8222-222222222222"
	input.QuestionPackage.Templates[0].Parameters[0].Values[0] = "999"
	after, _ := json.Marshal(sealed)
	if string(before) != string(after) {
		t.Fatal("sealed reference snapshot aliases mutable caller data")
	}
}
func TestFixedWitnessDoesNotEraseDuplicates(t *testing.T) {
	input, refs := sealFixture()
	fixed := numericFixed("duplicate-witness")
	fixed.Body.Witness.Parameters = append(fixed.Body.Witness.Parameters, ParameterValue{Name: "left", Value: "1"})
	input.QuestionPackage.FixedQuestions = []FixedQuestion{fixed}
	if _, report, e := ValidateAndSeal(context.Background(), input, refs); e == nil || report.ReadyToSubmit {
		t.Fatal("duplicate witness parameter normalized away")
	}
}
func TestQuestionCanonicalSealedPackage(t *testing.T) {
	input, refs := sealFixture()
	second := cloneTemplate(input.QuestionPackage.Templates[0])
	second.ID = "another-template"
	input.QuestionPackage.Templates = append(input.QuestionPackage.Templates, second)
	input.QuestionPackage.Blueprints[0].Sources = append(input.QuestionPackage.Blueprints[0].Sources, BlueprintSource{Kind: "template", Ref: Ref{ID: second.ID, Version: 1}})
	a, _, e := ValidateAndSeal(context.Background(), input, refs)
	if e != nil {
		t.Fatal(e)
	}
	input.QuestionPackage.Templates[0], input.QuestionPackage.Templates[1] = input.QuestionPackage.Templates[1], input.QuestionPackage.Templates[0]
	input.QuestionPackage.Blueprints[0].Sources[0], input.QuestionPackage.Blueprints[0].Sources[1] = input.QuestionPackage.Blueprints[0].Sources[1], input.QuestionPackage.Blueprints[0].Sources[0]
	b, _, e := ValidateAndSeal(context.Background(), input, refs)
	if e != nil {
		t.Fatal(e)
	}
	aa, _ := json.Marshal(a.Package)
	bb, _ := json.Marshal(b.Package)
	if string(aa) != string(bb) || a.PackageSHA != b.PackageSHA {
		t.Fatal("stored package differs from its canonical SHA bytes")
	}
}

func TestQuestionCoverageRequiresCurrentSources(t *testing.T) {
	input, refs := sealFixture()
	input.QuestionPackage.Templates[0].Coverage = append(input.QuestionPackage.Templates[0].Coverage, ObjectiveCoverage{Knowledge: Ref{ID: "unpublished-extra", Version: 1}, ObjectiveIndices: []int{0}})
	report, e := ValidateEditable(context.Background(), input, refs)
	if e != nil || len(report.Coverage) == 0 || report.Coverage[0].EffectiveInstances != 0 || report.Coverage[0].Ready {
		t.Fatal("unavailable extra reference still contributes ready questions", report, e)
	}
	input, refs = sealFixture()
	input.QuestionPackage.Blueprints[0].Sources = append(input.QuestionPackage.Blueprints[0].Sources, BlueprintSource{Kind: "template", Ref: Ref{ID: "absent-template", Version: 1}})
	report, e = ValidateEditable(context.Background(), input, refs)
	if e != nil || len(report.Coverage) == 0 || report.Coverage[0].Ready {
		t.Fatal("missing declared source still marked ready", report, e)
	}
}
func TestQuestionNodePoolWithoutBlueprint(t *testing.T) {
	input, refs := sealFixture()
	input.QuestionPackage.Blueprints = []Blueprint{}
	input.QuestionPackage.Templates = []Template{}
	for j := 0; j < 50; j++ {
		temp := templateFixture()
		temp.ID = fmt.Sprintf("pool-template-%d", j)
		temp.Parameters = []Parameter{{Name: "left", Values: integerValues(20)}, {Name: "right", Values: []string{"1"}}}
		input.QuestionPackage.Templates = append(input.QuestionPackage.Templates, temp)
	}
	input.QuestionPackage.FixedQuestions = []FixedQuestion{numericFixed("pool-overflow")}
	if _, report, e := ValidateAndSeal(context.Background(), input, refs); e == nil || report.ReadyToSubmit {
		t.Fatal("1001 node pool bypassed limit by omitting blueprint")
	}
}

func TestQuestionFrozenSupplementaryObjectives(t *testing.T) {
	input, refs := sealFixture()
	sealed, _, e := ValidateAndSeal(context.Background(), input, refs)
	if e != nil {
		t.Fatal(e)
	}
	found := false
	for _, goal := range sealed.Objectives {
		if goal.ObjectiveIndex == 1 && goal.Knowledge.ID == "fractions" && goal.Text == "Recognize equality and order." {
			found = true
		}
	}
	if !found {
		t.Fatal("supplementary target text absent from immutable review evidence")
	}
}

func TestNegativeQuestionFixtures(t *testing.T) {
	for _, name := range []string{"bad-answer", "coverage-trap"} {
		file, e := os.Open("../../../content/question-fixtures/" + name + ".json")
		if e != nil {
			t.Fatal(e)
		}
		pkg, e := DecodePackage(file)
		file.Close()
		if e != nil {
			t.Fatal(e)
		}
		input, refs := sealFixture()
		input.QuestionPackage = pkg
		refs.Knowledge[0].Identity.Version = 2
		refs.Knowledge[0].Objectives = []string{"Goal zero", "Goal one", "Goal two", "Goal three", "Goal four", "Goal five"}
		if _, report, e := ValidateAndSeal(context.Background(), input, refs); e == nil || report.ReadyToSubmit {
			t.Fatal("negative fixture accepted", name, report)
		}
	}
}
