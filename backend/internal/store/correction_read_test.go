package store_test

import (
	"encoding/base64"
	"errors"
	"github.com/yyl1212/math_master/backend/internal/assessment"
	"github.com/yyl1212/math_master/backend/internal/auth"
	"github.com/yyl1212/math_master/backend/internal/correction"
	"strings"
	"testing"
	"time"
)

func (f *correctionFixture) cReadableResult() (string, string, correction.PlanRef) {
	f.t.Helper()
	aid, b := f.cLegacyFailedFour()
	c := f.registerRule(nil, 1)
	f.cFinishRoots()
	p := f.cApproveAPI(c.ID)
	f.cRunAll()
	var id string
	if e := f.db.QueryRow(`SELECT id::text FROM correction_results WHERE evidence_id=$1 AND plan_id=$2 AND sealed`, aid, p.ID).Scan(&id); e != nil {
		f.t.Fatal(e)
	}
	_ = b
	return id, aid, p
}
func TestCorrectionOwnPrivacy(t *testing.T) {
	f := newCorrectionFixture(t)
	id, aid, p := f.cReadableResult()
	for _, detail := range []bool{false, true} {
		foreign, e := f.repo.ReadOwnCorrection(f.ctx, f.Access("learner_b", false), id, detail)
		if !errors.Is(e, auth.ErrNotFound) || foreign.ActorID != "" || foreign.Data.Result.ID != "" || len(foreign.Data.Items) != 0 {
			t.Fatal("foreign result", e)
		}
	}
	ref := correction.EvidenceRef{Kind: correction.AssessmentEvidence, ID: aid}
	if _, e := f.repo.ListOwnCorrections(f.ctx, f.Access("learner_b", false), ref, correction.Query{}); !errors.Is(e, auth.ErrNotFound) {
		t.Fatal("foreign evidence", e)
	}
	before := f.exposureSequence("learner_a")
	meta, e := f.repo.ReadOwnCorrection(f.ctx, f.Access("learner_a", false), id, false)
	if e != nil || meta.ActorID != f.ids["learner_a"] || len(meta.Data.Items) != 0 || meta.Data.PlanReason != nil || meta.Data.Result.Validity != "effective" || f.exposureSequence("learner_a") != before {
		t.Fatal("unsafe metadata", meta, e)
	}
	plan, e := f.repo.ReadCorrectionPlan(f.ctx, f.Access("reviewer_a", false), p, false)
	if e != nil || plan.Data.Reason != nil || plan.Data.DecisionReason != nil || len(plan.Data.Mappings) != 0 {
		t.Fatal("plan metadata", e)
	}
	if _, e = f.repo.ReadCorrectionPlan(f.ctx, f.Access("learner_a", false), p, false); !errors.Is(e, auth.ErrForbidden) {
		t.Fatal("management permission", e)
	}
	full, e := f.repo.ReadOwnCorrection(f.ctx, f.Access("learner_a", false), id, true)
	if e != nil || len(full.Data.Items) != 5 || full.Data.PlanReason == nil || f.exposureSequence("learner_a") <= before {
		t.Fatal("protected result", full, e)
	}
	for n, item := range full.Data.Items {
		if item.Position != n+1 || item.Original.ID == "" || item.Effective.ID == "" || item.Prompt == "" || item.Explanation == "" {
			t.Fatal("actual item", item)
		}
	}
	if f.count(`SELECT count(*) FROM correction_events WHERE kind='detail_exposed' AND owner_user_id=$1`, f.ids["learner_a"]) != 1 {
		t.Fatal("scope exposure absent")
	}
	f.registerRule(nil, 1)
	meta, e = f.repo.ReadOwnCorrection(f.ctx, f.Access("learner_a", false), id, false)
	if e != nil || meta.Data.Result.Validity != "restricted" {
		t.Fatal("new case validity", e)
	}
}
func TestCorrectionResultPagination(t *testing.T) {
	f := newCorrectionFixture(t)
	aid, b := f.cSubmittedBasis("learner_a")
	cid := f.cCase()
	p := f.cApproved(cid)
	b.HandledCaseIDs = []string{cid}
	b.PlanRefs = []correction.PlanRef{p}
	f.exec(`ALTER TABLE correction_results ALTER COLUMN created_at SET DEFAULT '2026-01-01 01:02:03.123456+00'`)
	wanted := map[string]bool{}
	for n := 0; n < 51; n++ {
		id := f.cSealResult(aid, cid, p, b)
		wanted[id] = true
	}
	q := correction.Query{Limit: 50}
	seen := map[string]bool{}
	ref := correction.EvidenceRef{Kind: correction.AssessmentEvidence, ID: aid}
	for {
		page, e := f.repo.ListOwnCorrections(f.ctx, f.Access("learner_a", false), ref, q)
		if e != nil {
			t.Fatal(e)
		}
		for _, m := range page.Data.Items {
			if seen[m.ID] || !wanted[m.ID] {
				t.Fatal("duplicate/foreign", m.ID)
			}
			seen[m.ID] = true
		}
		if page.Data.NextCursor == nil {
			break
		}
		q.Cursor = *page.Data.NextCursor
	}
	if len(seen) != 51 {
		t.Fatal("same microsecond lost rows", len(seen))
	}
	raw, _ := base64.RawURLEncoding.DecodeString(q.Cursor)
	bad := base64.RawURLEncoding.EncodeToString(append(raw, []byte(" {}")...))
	for _, query := range []correction.Query{{Limit: 51}, {Cursor: bad}, {Cursor: "bad"}} {
		if _, e := f.repo.ListOwnCorrections(f.ctx, f.Access("learner_a", false), ref, query); !errors.Is(e, auth.ErrInvalidInput) {
			t.Fatal("invalid cursor", e)
		}
	}
	page, e := f.repo.ListCorrectionPlans(f.ctx, f.Access("reviewer_a", false), cid, correction.Query{})
	if e != nil || len(page.Data.Items) != 1 {
		t.Fatal("plan list", e)
	}
	cases, e := f.repo.ListCorrectionCases(f.ctx, f.Access("reviewer_a", false), correction.Query{})
	if e != nil || len(cases.Data.Items) != 1 {
		t.Fatal("case list", e)
	}
	jobs, e := f.repo.ListCorrectionJobs(f.ctx, f.Access("reviewer_a", false), cid, correction.Query{Limit: 50})
	if e != nil || len(jobs.Data.Items) != 50 || jobs.Data.NextCursor == nil {
		t.Fatal("job list", e)
	}
	if strings.Contains(corrJSON(jobs), "leaseToken") || strings.Contains(corrJSON(page), "reason") {
		t.Fatal("private metadata leak")
	}
}
func TestCorrectionResultPaginationPlanVersionsShareClock(t *testing.T) {
	f := newCorrectionFixture(t)
	cid := f.cCase()
	p := f.cDraft(cid)
	if e := f.cPending(cid, p); e != nil {
		t.Fatal(e)
	}
	var at time.Time
	if e := f.db.QueryRow(`SELECT created_at FROM correction_plans WHERE id=$1 AND version=1`, p.ID).Scan(&at); e != nil {
		t.Fatal(e)
	}
	child := correction.PlanRef{ID: p.ID, Version: 2}
	in := correctionPlanInput()
	in.Parent = &p
	tx, e := f.db.Begin()
	if e != nil {
		t.Fatal(e)
	}
	defer tx.Rollback()
	if _, e = tx.Exec(`INSERT INTO correction_plans(id,version,case_id,parent_id,parent_version,creator_user_id,body,created_at,updated_at) VALUES($1,2,$2,$1,1,$3,$4,$5,$5)`, p.ID, cid, f.ids["author_a"], corrJSON(in), at); e != nil {
		t.Fatal(e)
	}
	if e = f.cEvent(tx, "plan", child.ID, "plan_created", f.ids["author_a"], cid, &child.Version, 1, map[string]any{}); e != nil {
		t.Fatal(e)
	}
	if e = tx.Commit(); e != nil {
		t.Fatal(e)
	}
	first, e := f.repo.ListCorrectionPlans(f.ctx, f.Access("reviewer_a", false), cid, correction.Query{Limit: 1})
	if e != nil || len(first.Data.Items) != 1 || first.Data.NextCursor == nil || first.Data.Items[0].Ref.Version != 2 {
		t.Fatal(first, e)
	}
	second, e := f.repo.ListCorrectionPlans(f.ctx, f.Access("reviewer_a", false), cid, correction.Query{Limit: 1, Cursor: *first.Data.NextCursor})
	if e != nil || len(second.Data.Items) != 1 || second.Data.Items[0].Ref.Version != 1 {
		t.Fatal("same ID/version clock pagination", second, e)
	}
}

func TestCorrectionOwnCurrentPracticeAndKnowledgeValidity(t *testing.T) {
	for _, kind := range []string{"practice-second-rule", "knowledge-update"} {
		t.Run(kind, func(t *testing.T) {
			f := newCorrectionFixture(t)
			var id string
			if kind == "knowledge-update" {
				id, _, _ = f.cReadableResult()
			} else {
				p := f.createPractice("learner_a")
				var raw string
				if e := f.db.QueryRow(`SELECT (body#>>'{body,body,correctNumeric,numerator}')||'/'||(body#>>'{body,body,correctNumeric,denominator}') FROM question_instances WHERE id=$1 AND version=$2`, p.Question.Instance.ID, p.Question.Instance.Version).Scan(&raw); e != nil {
					t.Fatal(e)
				}
				if _, e := f.repo.AnswerPractice(f.ctx, f.Access("learner_a", false), p.Summary.ID, assessment.Answer{Kind: "numeric", Raw: &raw}); e != nil {
					t.Fatal(e)
				}
				c := f.registerRule(nil, 1)
				f.cFinishRoots()
				f.cApproveAPI(c.ID)
				f.cRunAll()
				if e := f.db.QueryRow(`SELECT id::text FROM correction_results WHERE evidence_id=$1`, p.Summary.ID).Scan(&id); e != nil {
					t.Fatal(e)
				}
			}
			before, e := f.repo.ReadOwnCorrection(f.ctx, f.Access("learner_a", false), id, false)
			if e != nil || before.Data.Result.Validity != "effective" {
				t.Fatal("initial validity", e)
			}
			if kind == "knowledge-update" {
				in := newMaterial(f.Input())
				head := f.KHead()
				sub := f.ApprovedInput(in)
				f.Activate(f.Prepare(sub, head), head)
			} else {
				f.registerRule(nil, 1)
			}
			after, e := f.repo.ReadOwnCorrection(f.ctx, f.Access("learner_a", false), id, false)
			if e != nil || after.Data.Result.Validity != "restricted" {
				t.Fatal("stale correction appears current", e)
			}
		})
	}
}
