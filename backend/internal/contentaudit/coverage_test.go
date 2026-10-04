package contentaudit

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/yyl1212/math_master/backend/internal/content"
	"github.com/yyl1212/math_master/backend/internal/question"
	"strings"
	"testing"
	"time"
)

func testRequest(mode Mode) Request {
	return Request{Mode: mode, Route: content.VersionRef{ID: "target", Version: 1}, CodeSHA: strings.Repeat("a", 40), At: time.Date(2026, 10, 4, 0, 0, 0, 0, time.UTC)}
}
func auditFixture(n int) (DraftFacts, SourceBundle) {
	f := DraftFacts{Content: content.Snapshot{CatalogueVersion: 1}, Path: content.Path{ID: "target", Version: 1, Nodes: []content.VersionRef{}}, References: question.ReferenceSnapshot{CatalogueVersion: 1, CatalogueSHA256: strings.Repeat("b", 64)}, Sealed: []question.SealedPackage{{Package: question.QuestionPackage{}, Instances: []question.Instance{}}}, QuestionReports: []question.ValidationReport{{Coverage: []question.CoverageNode{}}}}
	s := SourceBundle{SnapshotID: strings.Repeat("c", 64), ReportSHA: strings.Repeat("d", 64), Report: SourceReport{SchemaVersion: 1, PolicyVersion: 1, Ready: true, Issues: []SourceIssue{}}, Mapping: SourceMap{SchemaVersion: 1, PolicyVersion: 1, Objects: []SourceObject{}, Sources: []MappedSource{{ID: "checked-source", Use: "fact_check"}}}}
	add := func(kind, id string, v int, sha string) {
		s.Mapping.Objects = append(s.Mapping.Objects, SourceObject{Kind: kind, ID: id, Version: &v, SHA256: sha, SourceIDs: []string{"checked-source"}, Origin: "original"})
	}
	for j := 0; j < n; j++ {
		ref := content.VersionRef{ID: fmt.Sprintf("node-%02d", j), Version: 1}
		k := content.Knowledge{ID: ref.ID, Version: 1, Objectives: []string{"Model equal units.", "Explain a representation."}}
		f.Content.Knowledge = append(f.Content.Knowledge, k)
		f.Path.Nodes = append(f.Path.Nodes, ref)
		add("knowledge", k.ID, 1, content.Digest(k))
		u := content.Unit{ID: ref.ID + "-unit", Version: 1, Knowledge: ref, Angles: []content.Angle{{Kind: "intuitive", Body: "Equal groups."}, {Kind: "formal", Body: "Addition combines counts."}}, Examples: []string{"2+3=5."}, Counterexamples: []string{"Different unit sizes cannot be counted as equal units."}}
		f.Content.Units = append(f.Content.Units, u)
		add("unit", u.ID, 1, content.Digest(u))
		bp := question.Blueprint{ID: ref.ID + "-assessment", Version: 1, Knowledge: ref, CoreObjectiveIndices: []int{0, 1}, Sources: []question.BlueprintSource{}, RuleVersion: 1, QuestionCount: 5, PassCount: 4}
		bi := question.Identity{ID: bp.ID, Version: 1, SHA256: strings.Repeat("e", 64)}
		f.Sealed[0].Package.Blueprints = append(f.Sealed[0].Package.Blueprints, bp)
		f.QuestionReports[0].Coverage = append(f.QuestionReports[0].Coverage, question.CoverageNode{Knowledge: question.Identity{ID: ref.ID, Version: 1, SHA256: content.Digest(k)}, Blueprint: &bi})
		add("blueprint", bp.ID, 1, bi.SHA256)
		for i := 0; i < 10; i++ {
			key := "a"
			body := question.QuestionBody{Type: "single_choice", Knowledge: ref, Coverage: []question.ObjectiveCoverage{{Knowledge: ref, ObjectiveIndices: []int{0, 1}}}, Prompt: fmt.Sprintf("Represent %d groups in model %d.", i, j), Explanation: "Combine equal units.", CorrectChoiceID: &key, Choices: []question.Choice{{ID: "a", Text: "5"}, {ID: "b", Text: "6"}, {ID: "c", Text: "7"}, {ID: "d", Text: "8"}}, Units: []question.Ref{}, Assets: []question.AssetRef{}, Sources: []content.Source{}}
			id := ref.ID + fmt.Sprintf("-fixed-%02d", i)
			in := question.Instance{Identity: question.Identity{ID: id, Version: 1}, Origin: "fixed", Body: body, Parameters: []question.ParameterValue{}}
			_, hash, _ := question.CanonicalInstance(in)
			in.Identity.SHA256 = hash
			f.Sealed[0].Instances = append(f.Sealed[0].Instances, in)
			bp.Sources = append(bp.Sources, question.BlueprintSource{Kind: "instance", Ref: question.Ref{ID: id, Version: 1}})
			add("instance", id, 1, hash)
		}
		f.Sealed[0].Package.Blueprints[j] = bp
		if j < 20 {
			format := "rational"
			tpl := question.Template{ID: ref.ID + "-add", Version: 1, Knowledge: ref, Coverage: []question.ObjectiveCoverage{{Knowledge: ref, ObjectiveIndices: []int{0}}}, Type: "numeric", AnswerFormat: &format, PromptTemplate: fmt.Sprintf("Model %d: add {{left}} and {{right}}.", j), ExplanationTemplate: "Combine equal units.", Engine: question.EngineSpec{Family: "rational_arithmetic", Operation: "add", GeneratorVersion: 1, VerifierVersion: 1}, Parameters: []question.Parameter{{Name: "left", Values: []string{"1", "2"}}, {Name: "right", Values: []string{"3"}}}, Constraints: []question.Constraint{}, Distractors: []question.Distractor{}, Units: []question.Ref{}, Assets: []question.AssetRef{}, Sources: []content.Source{}}
			_, hash, _ := question.CanonicalTemplate(tpl)
			f.Sealed[0].Package.Templates = append(f.Sealed[0].Package.Templates, tpl)
			add("template", tpl.ID, 1, hash)
		}
	}
	f.Content.Paths = []content.Path{f.Path}
	add("path", f.Path.ID, 1, content.Digest(f.Path))
	return f, s
}
func TestContentAuditRouteCounts(t *testing.T) {
	f, s := auditFixture(29)
	for j := 0; j < 300; j++ {
		f.Content.Knowledge = append(f.Content.Knowledge, content.Knowledge{ID: fmt.Sprintf("foreign-%d", j), Version: 1})
	}
	f.Content.Knowledge = append(f.Content.Knowledge, content.Knowledge{ID: "node-00", Version: 2})
	f.Sealed[0].Instances = append(f.Sealed[0].Instances, f.Sealed[0].Instances[0])
	r, e := EvaluateDraft(context.Background(), testRequest(Draft), f, s)
	if e != nil || r.DraftCounts.Knowledge != 29 || r.DraftCounts.EffectiveInstances != 290 || r.DraftCounts.Duplicates != 1 || r.Conclusion != NotReady {
		t.Fatal(r, e)
	}
}
func TestContentAuditPrimaryBinding(t *testing.T) {
	f, s := auditFixture(2)
	for i := range f.Sealed[0].Instances {
		f.Sealed[0].Instances[i].Body.Knowledge = f.Path.Nodes[0]
		f.Sealed[0].Instances[i].Body.Coverage = append(f.Sealed[0].Instances[i].Body.Coverage, question.ObjectiveCoverage{Knowledge: f.Path.Nodes[1], ObjectiveIndices: []int{0, 1}})
	}
	r, e := EvaluateDraft(context.Background(), testRequest(Draft), f, s)
	if e != nil || r.Nodes[1].AssessmentInstances != 0 {
		t.Fatal(r.Nodes, e)
	}
}
func TestContentAuditExposureWitness(t *testing.T) {
	f, s := auditFixture(1)
	r, e := EvaluateDraft(context.Background(), testRequest(Draft), f, s)
	if e != nil || !r.Nodes[0].AfterPracticeWitness || len(r.Nodes[0].FiveWitness) != 5 {
		t.Fatal(r, e)
	}
	tpl := question.Identity{ID: "one-template", Version: 1, SHA256: strings.Repeat("f", 64)}
	for i := range f.Sealed[0].Instances {
		f.Sealed[0].Instances[i].Origin = "generated"
		f.Sealed[0].Instances[i].Template = &tpl
	}
	f.Sealed[0].Package.Blueprints[0].Sources = []question.BlueprintSource{{Kind: "template", Ref: question.Ref{ID: tpl.ID, Version: 1}}}
	r, e = EvaluateDraft(context.Background(), testRequest(Draft), f, s)
	if e != nil || r.Nodes[0].AfterPracticeWitness {
		t.Fatal(r, e)
	}
}
func TestContentAuditConclusions(t *testing.T) {
	f, s := auditFixture(30)
	r, e := EvaluateDraft(context.Background(), testRequest(Draft), f, s)
	if e != nil || r.Conclusion != DraftReady || r.FormalCounts.EffectiveInstances != 0 {
		t.Fatal(r, e)
	}
	p := PublishedFacts{CatalogueVersion: 1, CatalogueSHA: f.References.CatalogueSHA256, Path: f.Path, PathSHA: content.Digest(f.Path), Content: f.Content, Bank: question.Candidate{Templates: f.Sealed[0].Package.Templates, Blueprints: f.Sealed[0].Package.Blueprints, Instances: f.Sealed[0].Instances}, Approvals: []ApprovalFact{}, EligibleInstances: []question.Identity{}, Excluded: []Exclusion{}}
	for _, tpl := range p.Bank.Templates {
		generated, _, e := question.Generate(context.Background(), tpl)
		if e != nil {
			t.Fatal(e)
		}
		p.Bank.Instances = append(p.Bank.Instances, generated...)
	}
	for _, i := range p.Bank.Instances {
		p.EligibleInstances = append(p.EligibleInstances, i.Identity)
	}
	r, e = EvaluatePublished(context.Background(), testRequest(Published), p, s, AcceptanceEvidence{})
	if e != nil || r.Conclusion != AwaitingReview {
		t.Fatal(r, e)
	}
	p.FixtureOnly = true
	r, e = EvaluatePublished(context.Background(), testRequest(Published), p, s, AcceptanceEvidence{})
	if e != nil || r.Conclusion == Accepted || r.FormalCounts.EffectiveInstances != 0 {
		t.Fatal(r, e)
	}
	var j, m bytes.Buffer
	if e = WriteReport(&j, &m, r); e != nil {
		t.Fatal(e)
	}
	for _, secret := range []string{"Combine equal units", "checked-source", "ReviewerID", "sourceRoot", "CorrectChoiceID"} {
		if strings.Contains(j.String()+m.String(), secret) {
			t.Fatal("private content leaked", secret)
		}
	}
	var decoded Report
	if json.Unmarshal(j.Bytes(), &decoded) != nil || decoded.Conclusion != r.Conclusion || !strings.Contains(m.String(), string(r.Conclusion)) {
		t.Fatal("report parity")
	}
}
func TestContentAuditCandidateBoundary(t *testing.T) {
	f, s := auditFixture(1)
	original := f.Sealed[0].Instances[0]
	f.Sealed[0].Instances = nil
	bp := &f.Sealed[0].Package.Blueprints[0]
	bp.Sources = nil
	for i := 0; i < 1001; i++ {
		in := original
		in.Identity.ID = fmt.Sprintf("capacity-%d", i)
		in.Identity.SHA256 = fmt.Sprintf("%064x", i+1)
		in.Body.Prompt = fmt.Sprintf("Represent %d equal items.", i)
		f.Sealed[0].Instances = append(f.Sealed[0].Instances, in)
		bp.Sources = append(bp.Sources, question.BlueprintSource{Kind: "instance", Ref: question.Ref{ID: in.Identity.ID, Version: 1}})
	}
	f.Sealed[0].Instances = f.Sealed[0].Instances[:1000]
	r, e := EvaluateDraft(context.Background(), testRequest(Draft), f, s)
	if e != nil || r.Nodes[0].AssessmentInstances != 1000 {
		t.Fatal(r, e)
	}
	f.Sealed[0].Instances = append(f.Sealed[0].Instances, question.Instance{Identity: question.Identity{ID: "capacity-extra", Version: 1}, Origin: "fixed", Body: original.Body})
	bp.Sources = append(bp.Sources, question.BlueprintSource{Kind: "instance", Ref: question.Ref{ID: "capacity-extra", Version: 1}})
	if _, e = EvaluateDraft(context.Background(), testRequest(Draft), f, s); e == nil {
		t.Fatal("1001 candidates accepted")
	}
}
