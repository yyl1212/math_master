package contentaudit

import (
	"context"
	"encoding/json"
	"github.com/yyl1212/math_master/backend/internal/content"
	"github.com/yyl1212/math_master/backend/internal/question"
	"os"
	"strconv"
	"strings"
	"testing"
)

func questionBatch(t *testing.T, names ...string) DraftFacts {
	t.Helper()
	cf, e := os.Open("../../../content/catalogue/domains.json")
	if e != nil {
		t.Fatal(e)
	}
	defer cf.Close()
	c, e := content.DecodeCatalogue(cf)
	if e != nil {
		t.Fatal(e)
	}
	p := editorialPackage(t)
	in := DraftInput{Catalogue: c, Content: p, AssetsRoot: "../../../content/assets", Questions: []question.QuestionPackage{}}
	for _, name := range names {
		f, e := os.Open("../../../content/questions/elementary-foundations-" + name + ".v1.json")
		if e != nil {
			t.Fatal(e)
		}
		q, e := question.DecodePackage(f)
		f.Close()
		if e != nil {
			t.Fatal(name, e)
		}
		in.Questions = append(in.Questions, q)
	}
	facts, e := CheckDraft(context.Background(), in)
	if e != nil {
		t.Fatal(e)
	}
	return facts
}
func checkQuestionBatch(t *testing.T, f DraftFacts, counts [][3]int) {
	t.Helper()
	keys := map[string]bool{}
	for j, s := range f.Sealed {
		want := counts[j]
		if len(s.Package.Templates) != want[0] || len(s.Package.FixedQuestions) != want[1] || len(s.Instances) != want[2]+want[1] {
			t.Fatal("batch quantity", j, len(s.Instances))
		}
		for _, g := range f.QuestionReports[j].Generation {
			if g.ValidInstances != 16 || g.RawCombinations != 16 || g.ExcludedCombinations != 0 {
				t.Fatal("exact finite parameter domain", g)
			}
		}
		for _, in := range s.Instances {
			if in.Origin == "template" && question.VerifyInstance(in) != nil {
				t.Fatal("generated verification")
			}
		}
		for _, q := range s.Package.FixedQuestions {
			if len(q.Body.Choices) != 4 || q.Body.CorrectChoiceID == nil || q.Body.Witness != nil {
				t.Fatal("fixed MC shape")
			}
			keys[*q.Body.CorrectChoiceID] = true
			texts := map[string]bool{}
			for _, c := range q.Body.Choices {
				if texts[c.Text] || c.Text == "" {
					t.Fatal("duplicate option")
				}
				texts[c.Text] = true
			}
			number, e := strconv.Atoi(q.ID[len(q.ID)-2:])
			if e != nil || number < 1 || number > 15 {
				t.Fatal("fixed ID")
			}
			goals := q.Body.Coverage[0].ObjectiveIndices
			want := []int{0}
			if number > 5 {
				want = []int{1}
			}
			if number > 10 {
				want = []int{0, 1}
			}
			a, _ := json.Marshal(goals)
			b, _ := json.Marshal(want)
			if string(a) != string(b) {
				t.Fatal("fixed goal grouping")
			}
		}
		for _, bp := range s.Package.Blueprints {
			instances := []question.Instance{}
			for _, in := range s.Instances {
				if in.Body.Knowledge == bp.Knowledge && matchesSource(bp, in) {
					instances = append(instances, in)
				}
			}
			w, ok, after, e := witness(context.Background(), bp.CoreObjectiveIndices, instances)
			if e != nil || !ok || !after || len(w) != 5 {
				t.Fatal("exposure cover", bp.ID, e)
			}
		}
		if f.QuestionReports[j].PackageBytes > question.MaxPackageBytes || f.QuestionReports[j].FrozenBytes > question.MaxEnvelopeBytes {
			t.Fatal("byte capacity")
		}
	}
	if len(keys) != 4 {
		t.Fatal("constant correct key")
	}
}
func TestContentAuditQuestionsNumbersOperations(t *testing.T) {
	f := questionBatch(t, "numbers", "operations")
	checkQuestionBatch(t, f, [][3]int{{1, 90, 16}, {10, 135, 160}})
	instances := map[string]question.Instance{}
	for _, s := range f.Sealed {
		for _, in := range s.Instances {
			instances[in.Identity.ID] = in
		}
	}
	goldens := map[string]string{"ef-natural-numbers-fixed-01": "0", "ef-zero-fixed-01": "0", "ef-rounding-estimation-fixed-01": "3", "ef-division-fixed-01": "4", "ef-addition-subtraction-inverse-fixed-06": "7"}
	for id, expected := range goldens {
		in, ok := instances[id]
		if !ok {
			t.Fatal("golden missing", id)
		}
		found := false
		for _, c := range in.Body.Choices {
			if c.ID == *in.Body.CorrectChoiceID && c.Text == expected {
				found = true
			}
		}
		if !found {
			t.Fatal("independent numerical golden", id, expected)
		}
	}
	zeroCase := instances["ef-division-fixed-06"]
	correct := ""
	for _, c := range zeroCase.Body.Choices {
		if c.ID == *zeroCase.Body.CorrectChoiceID {
			correct = c.Text
		}
	}
	if !strings.Contains(correct, "undefined") {
		t.Fatal("division zero condition")
	}
	var generated question.Instance
	for _, in := range f.Sealed[1].Instances {
		if in.Origin == "template" && in.Body.CorrectNumeric != nil {
			generated = in
			break
		}
	}
	value := *generated.Body.CorrectNumeric
	value.Numerator = "999999"
	generated.Body.CorrectNumeric = &value
	if question.VerifyInstance(generated) == nil {
		t.Fatal("wrong generated answer accepted")
	}
	var template question.Template
	for _, candidate := range f.Sealed[1].Package.Templates {
		if candidate.ID == "ef-divide-measuring" {
			template = candidate
			break
		}
	}
	template.Parameters[1].Values = []string{"0"}
	if _, r, e := question.Generate(context.Background(), template); e == nil && r.ValidInstances != 0 {
		t.Fatal("zero divisor accepted")
	}
	concept := instances["ef-natural-numbers-fixed-01"]
	concept.Body.Witness = &question.VerificationWitness{Engine: question.EngineSpec{Family: "rational_arithmetic", Operation: "add", GeneratorVersion: 1, VerifierVersion: 1}, Parameters: []question.ParameterValue{{Name: "left", Value: "1"}, {Name: "right", Value: "2"}}}
	raw, _ := json.Marshal(f.Sealed[0].Package)
	var bad question.QuestionPackage
	json.Unmarshal(raw, &bad)
	for i := range bad.FixedQuestions {
		if bad.FixedQuestions[i].ID == concept.Identity.ID {
			bad.FixedQuestions[i].Body.Witness = concept.Body.Witness
		}
	}
	_, badReport, badError := question.ValidateAndSeal(context.Background(), question.DraftInput{CatalogueVersion: f.References.CatalogueVersion, QuestionPackage: bad, SourceMap: []question.SourceLink{}}, f.References)
	if badError == nil || badReport.ReadyToSubmit {
		t.Fatal("fake concept witness resealed")
	}
	if question.VerifyInstance(concept) == nil {
		t.Fatal("fake concept witness accepted")
	}
}
