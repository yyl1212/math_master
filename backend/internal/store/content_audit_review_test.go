package store_test

import (
	"bytes"
	"fmt"
	"github.com/yyl1212/math_master/backend/internal/content"
	"github.com/yyl1212/math_master/backend/internal/publication"
	"github.com/yyl1212/math_master/backend/internal/question"
	"os"
	"testing"
)

func TestContentAuditTemplateDependencyEligibility(t *testing.T) {
	f := newQuestionFixture(t)
	input := f.Input()
	input.Package.ID = "audit-template-dependency"
	input.Package.Paths = []content.Path{{ID: auditRoute.ID, Version: auditRoute.Version, DomainIDs: []string{"elementary-mathematics"}, Title: "Template dependency route", TitleZh: "模板依赖夹具", Nodes: []content.VersionRef{{ID: input.Package.Knowledge[0].ID, Version: 1}}}}
	extra := input.Package.Units[0]
	extra.ID = "template-only-unit"
	input.Package.Units = append(input.Package.Units, extra)
	publish := func(in publication.DraftInput) {
		d, e := f.repo.CreateDraft(f.ctx, f.Access("author_b", false), in)
		if e != nil {
			t.Fatal(e)
		}
		sub, e := f.repo.SubmitDraft(f.ctx, f.Access("author_b", false), d.ID, publication.SubmitInput{ExpectedRevision: d.Revision, ExpectedDigest: d.Gate.Digest})
		if e != nil {
			t.Fatal(e)
		}
		sub, e = f.repo.DecideReview(f.ctx, f.Access("reviewer_b", false), sub.ID, approvedReviewInput())
		if e != nil {
			t.Fatal(e)
		}
		f.Activate(f.Prepare(sub, f.KHead()), f.KHead())
	}
	publish(input)
	f.questionInput.QuestionPackage.Templates[0].Units = []question.Ref{{ID: extra.ID, Version: 1}}
	raw, e := os.ReadFile("../../../content/questions/elementary-foundations-numbers.v1.json")
	if e != nil {
		t.Fatal(e)
	}
	original, e := question.DecodePackage(bytes.NewReader(raw))
	if e != nil {
		t.Fatal(e)
	}
	for j, fixed := range original.FixedQuestions[:15] {
		fixed.ID = fmt.Sprintf("audit-fixed-%02d", j)
		fixed.Body.Knowledge = question.Ref{ID: input.Package.Knowledge[0].ID, Version: 1}
		fixed.Body.Units = []question.Ref{}
		fixed.Body.Assets = []question.AssetRef{}
		fixed.Body.Coverage = []question.ObjectiveCoverage{{Knowledge: fixed.Body.Knowledge, ObjectiveIndices: []int{0}}}
		f.questionInput.QuestionPackage.FixedQuestions = append(f.questionInput.QuestionPackage.FixedQuestions, fixed)
		for k := range f.questionInput.QuestionPackage.Blueprints {
			f.questionInput.QuestionPackage.Blueprints[k].Sources = append(f.questionInput.QuestionPackage.Blueprints[k].Sources, question.BlueprintSource{Kind: "instance", Ref: question.Ref{ID: fixed.ID, Version: fixed.Version}})
		}
	}
	f.QActivate(f.QPrepare(f.QApproved("author_a", "reviewer_a").ID))
	before, e := f.repo.ReadContentAudit(f.ctx, auditRoute)
	if e != nil {
		t.Fatal(e)
	}
	tpl := before.Bank.Templates[0]
	_, sha, e := question.CanonicalTemplate(tpl)
	if e != nil {
		t.Fatal(e)
	}
	// Replace only the additional unit with v2; fixed questions retain their exact primary knowledge and no unit dependency.
	input.Package.Version = 2
	input.Package.Units[len(input.Package.Units)-1].Version = 2
	publish(input)
	after, e := f.repo.ReadContentAudit(f.ctx, auditRoute)
	if e != nil {
		t.Fatal("valid current head should be readable", e)
	}
	fixed := 0
	for _, in := range after.Bank.Instances {
		if in.Template == nil {
			fixed++
		}
	}
	if fixed < 10 {
		t.Fatal("fixed coverage was lost", fixed)
	}
	eligible := map[question.Identity]bool{}
	for _, id := range after.EligibleInstances {
		eligible[id] = true
	}
	for _, in := range after.Bank.Instances {
		if in.Template != nil && in.Template.ID == tpl.ID && eligible[in.Identity] {
			t.Fatal("retired template dependency remained eligible")
		}
	}
	found := false
	for _, x := range after.Excluded {
		if x.Object.Kind == "template" && x.Object.ID == tpl.ID && x.Object.Version != nil && *x.Object.Version == tpl.Version && x.Object.SHA256 == sha {
			found = true
		}
	}
	if !found {
		t.Fatal("template with no current eligible instance was not excluded")
	}
}
