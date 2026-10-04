package contentaudit

import (
	"context"
	"github.com/yyl1212/math_master/backend/internal/content"
	"github.com/yyl1212/math_master/backend/internal/question"
	"os"
	"testing"
)

func TestContentAuditOfflineDraftReferences(t *testing.T) {
	cfile, e := os.Open("../../../content/catalogue/domains.json")
	if e != nil {
		t.Fatal(e)
	}
	defer cfile.Close()
	c, e := content.DecodeCatalogue(cfile)
	if e != nil {
		t.Fatal(e)
	}
	pfile, e := os.Open("../content/testdata/workflow-ready.json")
	if e != nil {
		t.Fatal(e)
	}
	defer pfile.Close()
	p, e := content.DecodePackage(pfile)
	if e != nil {
		t.Fatal(e)
	}
	p.Paths = []content.Path{{ID: "target", Version: 1, Title: "Fractions", TitleZh: "分数", DomainIDs: []string{"elementary-mathematics"}, Nodes: []content.VersionRef{{ID: "fractions", Version: 1}}}}
	format := "rational"
	ref := question.Ref{ID: "fractions", Version: 1}
	tpl := question.Template{ID: "offline-add", Version: 1, Knowledge: ref, Coverage: []question.ObjectiveCoverage{{Knowledge: ref, ObjectiveIndices: []int{0}}}, Units: []question.Ref{{ID: "fractions-unit", Version: 1}}, Type: "numeric", AnswerFormat: &format, PromptTemplate: "Add {{left}} and {{right}}.", ExplanationTemplate: "Equal units combine to {{answer}}.", Engine: question.EngineSpec{Family: "rational_arithmetic", Operation: "add", GeneratorVersion: 1, VerifierVersion: 1}, Parameters: []question.Parameter{{Name: "left", Values: []string{"1", "2", "3", "4", "5", "6"}}, {Name: "right", Values: []string{"1"}}}, Constraints: []question.Constraint{}, Distractors: []question.Distractor{}, Assets: []question.AssetRef{}, Sources: p.Knowledge[0].Sources}
	bp := question.Blueprint{ID: "offline-assessment", Version: 1, Knowledge: ref, CoreObjectiveIndices: []int{0}, Sources: []question.BlueprintSource{{Kind: "template", Ref: question.Ref{ID: tpl.ID, Version: 1}}}, CoverageNote: "Exact current fractions objective.", RuleVersion: 1, QuestionCount: 5, PassCount: 4}
	input := DraftInput{Catalogue: c, Content: p, Questions: []question.QuestionPackage{{Kind: "question-bank", SchemaVersion: 1, ID: "offline-bank", Version: 1, Templates: []question.Template{tpl}, FixedQuestions: []question.FixedQuestion{}, Blueprints: []question.Blueprint{bp}}}, AssetsRoot: "../content/testdata"}
	facts, e := CheckDraft(context.Background(), input)
	if e != nil || facts.References.KnowledgeHead != nil || len(facts.Sealed[0].Instances) != 6 || facts.References.Knowledge[0].Identity.SHA256 != content.Digest(p.Knowledge[0]) {
		t.Fatal(facts, e)
	}
	input.Content.Units[0].Angles[0].Body = ""
	if _, e = CheckDraft(context.Background(), input); e == nil {
		t.Fatal("incomplete original workflow accepted")
	}
}
