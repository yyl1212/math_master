package store_test

import (
	"encoding/json"
	"fmt"
	"github.com/yyl1212/math_master/backend/internal/content"
	"github.com/yyl1212/math_master/backend/internal/publication"
	"github.com/yyl1212/math_master/backend/internal/question"
	"runtime"
	"strings"
	"syscall"
	"testing"
	"time"
)

func capacityMeasure(t *testing.T, name string, fn func()) {
	t.Helper()
	runtime.GC()
	var before, after runtime.MemStats
	runtime.ReadMemStats(&before)
	start := time.Now()
	fn()
	elapsed := time.Since(start)
	runtime.ReadMemStats(&after)
	var usage syscall.Rusage
	_ = syscall.Getrusage(syscall.RUSAGE_SELF, &usage)
	t.Logf("CAPACITY %s duration=%s allocated=%d heap=%d maxRSS=%d GOOS=%s", name, elapsed, after.TotalAlloc-before.TotalAlloc, after.HeapAlloc, usage.Maxrss, runtime.GOOS)
	if elapsed >= 8*time.Second {
		t.Fatalf("%s exceeded the unchanged 8s budget", name)
	}
}
func TestQuestionMaximumLegalWorkflow(t *testing.T) {
	f := newQuestionFixture(t)
	learning := f.Input()
	learning.Package.ID = "capacity-learning-package"
	for n := 1; n < 10; n++ {
		k := learning.Package.Knowledge[0]
		k.ID = fmt.Sprintf("capacity-knowledge-%d", n)
		learning.Package.Knowledge = append(learning.Package.Knowledge, k)
		u := learning.Package.Units[0]
		u.ID = fmt.Sprintf("capacity-unit-%d", n)
		u.Knowledge = content.VersionRef{ID: k.ID, Version: 1}
		u.AssetIDs = []string{}
		u.Angles = []content.Angle{{Kind: "formal", Body: "Original rational arithmetic and finite question fixture."}, {Kind: "visual", Body: "Equal rational parts form one whole in the original technical fixture."}}
		learning.Package.Units = append(learning.Package.Units, u)
	}
	draft, err := f.repo.CreateDraft(f.ctx, f.Access("author_b", false), learning)
	if err != nil {
		t.Fatal(err)
	}
	if !draft.Gate.ReadyToSubmit {
		t.Fatalf("capacity learning gate: %+v", draft.Gate)
	}
	frozen, err := f.repo.SubmitDraft(f.ctx, f.Access("author_b", false), draft.ID, publication.SubmitInput{ExpectedRevision: draft.Revision, ExpectedDigest: draft.Gate.Digest})
	if err != nil {
		t.Fatal(err)
	}
	approvedLearning, err := f.repo.DecideReview(f.ctx, f.Access("reviewer_b", false), frozen.ID, approvedReviewInput())
	if err != nil {
		t.Fatal(err)
	}
	f.Activate(f.Prepare(approvedLearning, f.KHead()), f.KHead())
	base := f.questionInput.QuestionPackage.Templates[0]
	base.Units = []question.Ref{}
	base.Assets = []question.AssetRef{}
	base.ExplanationTemplate += "\n\n" + strings.Repeat("Original maximum-capacity technical fixture. ", 44) + strings.Repeat(" ", 18)
	base.Parameters = []question.Parameter{{Name: "left", Values: []string{}}, {Name: "right", Values: []string{}}}
	for i := 1; i <= 31; i++ {
		base.Parameters[0].Values = append(base.Parameters[0].Values, fmt.Sprint(i))
	}
	for i := 1; i <= 32; i++ {
		base.Parameters[1].Values = append(base.Parameters[1].Values, fmt.Sprint(i))
	}
	capacityMeasure(t, "maximum-single-template-992", func() {
		instances, report, err := question.Generate(f.ctx, base)
		if err != nil || len(instances) != 992 || report.RawCombinations != 992 {
			t.Fatal("maximum legal family generation", err)
		}
	})
	ids := []string{}
	for batch := 0; batch < 10; batch++ {
		in := f.questionInput
		in.SourceMap = []question.SourceLink{}
		p := question.QuestionPackage{Kind: "question-bank", SchemaVersion: 1, ID: fmt.Sprintf("capacity-bank-%d", batch), Version: 1, Templates: []question.Template{}, FixedQuestions: []question.FixedQuestion{}, Blueprints: []question.Blueprint{}}
		for n := 0; n < 20; n++ {
			tpl := base
			if batch > 0 {
				tpl.Knowledge = question.Ref{ID: fmt.Sprintf("capacity-knowledge-%d", batch), Version: 1}
			}
			tpl.Coverage = []question.ObjectiveCoverage{{Knowledge: tpl.Knowledge, ObjectiveIndices: []int{0}}}
			tpl.ID = fmt.Sprintf("capacity-template-%d-%d", batch, n)
			tpl.Parameters = []question.Parameter{{Name: "left", Values: []string{}}, {Name: "right", Values: []string{"1", "2"}}}
			for v := 1; v <= 25; v++ {
				tpl.Parameters[0].Values = append(tpl.Parameters[0].Values, fmt.Sprint(v))
			}
			p.Templates = append(p.Templates, tpl)
		}
		for n := 0; n < 100; n++ {
			p.Blueprints = append(p.Blueprints, question.Blueprint{ID: fmt.Sprintf("capacity-blueprint-%d-%d", batch, n), Version: 1, Knowledge: p.Templates[0].Knowledge, CoreObjectiveIndices: []int{0}, Sources: []question.BlueprintSource{{Kind: "template", Ref: question.Ref{ID: p.Templates[n%20].ID, Version: 1}}}, CoverageNote: "Original maximum-volume technical fixture coverage.", RuleVersion: 1, QuestionCount: 5, PassCount: 4})
		}
		in.QuestionPackage = p
		f.questionInput = in
		var sub question.SubmissionView
		capacityMeasure(t, fmt.Sprintf("submit-1000-batch-%d", batch), func() {
			sub = f.QSubmitted("author_a")
			if len(sub.Frozen.InstanceIdentities) != 1000 {
				t.Fatal("submission did not keep maximum 1000 instances")
			}
		})
		approved, err := f.repo.DecideQuestionReview(f.ctx, f.Access("reviewer_a", false), sub.ID, approvedQuestionInput())
		if err != nil {
			t.Fatal(err)
		}
		ids = append(ids, approved.ID)
	}
	var prepared question.PublicationSummary
	capacityMeasure(t, "prepare-200-10000-1000", func() {
		prepared = f.QPrepare(ids...)
		if prepared.TemplateCount != 200 || prepared.InstanceCount != 10000 || prepared.BlueprintCount != 1000 {
			t.Fatal("not the maximum real candidate", prepared)
		}
	})
	capacityMeasure(t, "activate-200-10000-1000", func() { f.QActivate(prepared) })
	capacityMeasure(t, "coverage-200-10000-1000", func() {
		r, err := f.repo.ReadQuestionCoverage(f.ctx, f.Access("reviewer_a", false), question.CoverageQuery{Limit: 100})
		if err != nil || r.EffectiveInstances != 10000 || r.Nodes.Total != 1000 {
			t.Fatal("maximum real coverage", err, r.EffectiveInstances, r.Nodes.Total)
		}
	})
	var manifest, body int
	if err := f.db.QueryRow(`SELECT octet_length(manifest_bytes) FROM question_publications WHERE id=$1`, prepared.ID).Scan(&manifest); err != nil {
		t.Fatal(err)
	}
	if err := f.db.QueryRow(`SELECT sum(octet_length(CASE m.kind WHEN 'template' THEN t.body_bytes ELSE i.body_bytes END)) FROM question_publication_members m LEFT JOIN question_templates t ON m.kind='template' AND t.id=m.id AND t.version=m.version LEFT JOIN question_instances i ON m.kind='instance' AND i.id=m.id AND i.version=m.version WHERE m.publication_id=$1 AND m.kind IN('template','instance')`, prepared.ID).Scan(&body); err != nil {
		t.Fatal(err)
	}
	if manifest > question.MaxManifestBytes || body > question.MaxBankBodyBytes || body < 31<<20 {
		t.Fatal("actual canonical bank byte caps exceeded")
	}
	t.Logf("CAPACITY actual manifestBytes=%d templateInstanceBytes=%d", manifest, body)
	for _, q := range []string{`EXPLAIN (ANALYZE,BUFFERS,FORMAT JSON) SELECT kind,id,version,sha256,evidence FROM question_publication_members WHERE publication_id=$1 ORDER BY kind,id LIMIT 100`, `EXPLAIN (ANALYZE,BUFFERS,FORMAT JSON) SELECT id FROM question_publications WHERE id=$1 AND sealed`} {
		var plan []byte
		if err := f.db.QueryRow(q, prepared.ID).Scan(&plan); err != nil {
			t.Fatal(err)
		}
		var data any
		if json.Unmarshal(plan, &data) != nil {
			t.Fatal("invalid real SQL plan")
		}
		t.Logf("CAPACITY SQLPLAN %s", plan)
	}
}
