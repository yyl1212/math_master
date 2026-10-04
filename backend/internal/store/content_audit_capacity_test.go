package store_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/yyl1212/math_master/backend/internal/content"
	"github.com/yyl1212/math_master/backend/internal/publication"
	"github.com/yyl1212/math_master/backend/internal/question"
	"github.com/yyl1212/math_master/backend/internal/testutil"
	"strings"
	"testing"
	"time"
)

func TestContentAuditCapacity(t *testing.T) {
	t.Run("maximum-content-head", auditMaximumContent)
	t.Run("bank-budget-guards", func(t *testing.T) {
		if e := question.CheckBankLimits(200, 10000, 1000, 32<<20, 8<<20); e != nil {
			t.Fatal(e)
		}
		for _, sizes := range [][2]int{{(32 << 20) + 1, 8 << 20}, {32 << 20, (8 << 20) + 1}} {
			if e := question.CheckBankLimits(200, 10000, 1000, sizes[0], sizes[1]); !errors.Is(e, question.ErrLimitExceeded) {
				t.Fatal("oversize bank accepted", e)
			}
		}
	})
	f := newQuestionFixture(t)
	// Many sequential, individually bounded commands need their own overall fixture
	// lifecycle, just like TestWorkflowCapacityEnvelope. Never widen a command.
	whole, stop := context.WithTimeout(context.Background(), 4*time.Minute)
	defer stop()
	f.ctx = whole
	learning := f.Input()
	learning.Package.ID = "capacity-learning-package"
	learning.Package.Paths = []content.Path{{ID: "audit-route", Version: 1, DomainIDs: []string{"elementary-mathematics"}, Title: "Capacity route", TitleZh: "容量路线", Nodes: []content.VersionRef{{ID: "workflow-fractions", Version: 1}}}}
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
	capacityMeasure(t, "audit-max-bank", func() {
		facts, e := f.repo.ReadContentAudit(f.ctx, auditRoute)
		if e != nil || len(facts.Bank.Templates) != 20 || len(facts.Bank.Instances) != 1000 || len(facts.Bank.Blueprints) != 100 || len(facts.EligibleInstances) != 1000 {
			t.Fatal("maximum bank audit", e, len(facts.Bank.Instances), len(facts.EligibleInstances))
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

func auditMaximumContent(t *testing.T) {
	capacity, err := testutil.CapacityContent("../content/testdata", "../../../content/catalogue/domains.json")
	if err != nil {
		t.Fatal(err)
	}
	s := capacity.Snapshot
	raw, _ := json.Marshal(s)
	if len(raw) != 32<<20 || len(s.Knowledge) != 1000 || len(s.Units) != 4000 || len(s.Paths) != 200 || len(s.Assets) != 1000 || capacity.SVGBytes != 10<<20 {
		t.Fatal("capacity fixture is not simultaneously maximal")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	start := time.Now()
	report, err := content.ValidateSnapshot(ctx, capacity.Catalogue, s, capacity.Reader)
	cancel()
	if err != nil || !report.ReadyToSubmit {
		t.Fatalf("legal maximum rejected: %v; structural=%d completeness=%d", err, report.StructuralTotal, report.CompletenessTotal)
	}
	t.Logf("maximum pure validation: %s, canonical=%d SVG=%d", time.Since(start), len(raw), capacity.SVGBytes)
	f := newWorkflowFixture(t)
	whole, stop := context.WithTimeout(context.Background(), 4*time.Minute)
	defer stop()
	f.ctx = whole
	var submissions []string
	var longest time.Duration
	timed := func(start time.Time) {
		if elapsed := time.Since(start); elapsed > longest {
			longest = elapsed
		}
	}
	for _, input := range capacity.Inputs {
		start = time.Now()
		draft, err := f.repo.CreateDraft(f.ctx, f.Access("author_a", false), input)
		timed(start)
		if err != nil {
			t.Fatal("maximum package create", err)
		}
		start = time.Now()
		submission, err := f.repo.SubmitDraft(f.ctx, f.Access("author_a", false), draft.ID, publication.SubmitInput{ExpectedRevision: draft.Revision, ExpectedDigest: draft.Gate.Digest})
		timed(start)
		if err != nil {
			t.Fatal("maximum package submit", err)
		}
		start = time.Now()
		submission, err = f.repo.DecideReview(f.ctx, f.Access("reviewer_a", false), submission.ID, publication.ReviewInput{Decision: "approve", Checks: publication.ReviewChecks{Mathematics: true, Explanations: true, Relationships: true, Sources: true, Illustrations: true}, IndependenceNote: "Independent isolated capacity fixture reviewer.", Note: "Technical capacity fixture checks only; no production mathematical approval."})
		timed(start)
		if err != nil {
			t.Fatal("maximum package review", err)
		}
		submissions = append(submissions, submission.ID)
	}
	var head *string
	var final publication.PublicationView
	for at := 0; at < len(submissions); at += 20 {
		end := at + 20
		if end > len(submissions) {
			end = len(submissions)
		}
		start = time.Now()
		prepared, err := f.repo.PrepareRelease(f.ctx, f.Access("admin_a", false), publication.PrepareInput{SubmissionIDs: submissions[at:end], ExpectedHead: head, Reason: "Prepare maximal isolated capacity fixture batches."})
		timed(start)
		if err != nil {
			t.Fatal("maximum candidate prepare", err)
		}
		t.Logf("prepare group %d: %s", at/20+1, time.Since(start))
		start = time.Now()
		final, err = f.repo.ActivateRelease(f.ctx, f.Access("admin_a", true), prepared.ID, publication.ActivateInput{ExpectedHead: head, ExpectedManifestSHA: prepared.ManifestSHA, Reason: "Activate isolated capacity fixture only."})
		timed(start)
		if err != nil {
			t.Fatal("maximum candidate activate", err)
		}
		id := final.ID
		head = &id
	}

	if len(final.Manifest.Members) != 6200 {
		t.Fatal("1000-node head incomplete")
	}
	start = time.Now()
	facts, e := f.repo.ReadContentAudit(f.ctx, content.VersionRef{ID: s.Paths[0].ID, Version: s.Paths[0].Version})
	if e != nil || len(facts.Content.Knowledge) != len(s.Paths[0].Nodes) {
		t.Fatal("maximum content audit", e)
	}
	t.Logf("maximum content audit: %s; global knowledge=1000/canonical=33554432; scoped=%d", time.Since(start), len(facts.Content.Knowledge))
}
