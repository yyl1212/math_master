package store_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/yyl1212/math_master/backend/internal/assessment"
	"github.com/yyl1212/math_master/backend/internal/auth"
	"github.com/yyl1212/math_master/backend/internal/learning"
	"github.com/yyl1212/math_master/backend/internal/publication"
	"github.com/yyl1212/math_master/backend/internal/question"
	"github.com/yyl1212/math_master/backend/internal/testutil"
	"runtime"
	"syscall"
	"testing"
	"time"
)

type learningCapacityFixture struct {
	*questionFixture
	knowledge, blueprint question.Identity
	paths                []question.Identity
	submissions          []string
}

func learningCapacityMeasure(t *testing.T, name string, fn func() error) {
	t.Helper()
	runtime.GC()
	var before, after runtime.MemStats
	runtime.ReadMemStats(&before)
	started := time.Now()
	err := fn()
	elapsed := time.Since(started)
	runtime.ReadMemStats(&after)
	var usage syscall.Rusage
	_ = syscall.Getrusage(syscall.RUSAGE_SELF, &usage)
	rss := usage.Maxrss
	if runtime.GOOS == "linux" {
		rss *= 1024
	}
	t.Logf("LEARNING_CAPACITY %s elapsed=%s allocs=%d bytes=%d heap=%d maxRSS=%d unit=bytes GOOS=%s err=%v", name, elapsed, after.Mallocs-before.Mallocs, after.TotalAlloc-before.TotalAlloc, after.HeapAlloc, rss, runtime.GOOS, err)
	if err != nil || elapsed >= 8*time.Second {
		t.Fatalf("%s violated the unchanged 8s deadline: %v", name, err)
	}
}
func (f *learningCapacityFixture) publish(in publication.DraftInput) string {
	f.t.Helper()
	d, e := f.repo.CreateDraft(f.ctx, f.Access("author_a", false), in)
	if e != nil {
		f.t.Fatal(e)
	}
	if !d.Gate.ReadyToSubmit {
		f.t.Fatalf("capacity content gate: %+v", d.Gate)
	}
	sub, e := f.repo.SubmitDraft(f.ctx, f.Access("author_a", false), d.ID, publication.SubmitInput{ExpectedRevision: d.Revision, ExpectedDigest: d.Gate.Digest})
	if e != nil {
		f.t.Fatal(e)
	}
	sub, e = f.repo.DecideReview(f.ctx, f.Access("reviewer_a", false), sub.ID, approvedReviewInput())
	if e != nil {
		f.t.Fatal(e)
	}
	return sub.ID
}
func (f *learningCapacityFixture) publishQuestions(in question.DraftInput) question.SubmissionView {
	f.t.Helper()
	d, e := f.repo.CreateQuestionDraft(f.ctx, f.Access("author_a", false), in)
	if e != nil {
		f.t.Fatal(e)
	}
	gate, e := f.repo.ValidateQuestionDraft(f.ctx, f.Access("author_a", false), d.ID, question.ValidateInput{ExpectedRevision: d.Revision})
	if e != nil || !gate.ReadyToSubmit {
		f.t.Fatalf("capacity question gate package=%s: structural=%v completeness=%v err=%v", in.QuestionPackage.ID, gate.StructuralErrors, gate.CompletenessErrors, e)
	}
	sub, e := f.repo.SubmitQuestionDraft(f.ctx, f.Access("author_a", false), d.ID, question.SubmitInput{ExpectedRevision: d.Revision, ExpectedDigest: gate.Digest})
	if e != nil {
		f.t.Fatal(e)
	}
	sub, e = f.repo.DecideQuestionReview(f.ctx, f.Access("reviewer_a", false), sub.ID, approvedQuestionInput())
	if e != nil {
		f.t.Fatal(e)
	}
	return sub
}
func (f *learningCapacityFixture) activateContent(ids []string) {
	for at := 0; at < len(ids); at += 20 {
		end := at + 20
		if end > len(ids) {
			end = len(ids)
		}
		var head *string
		if f.count(`SELECT count(*) FROM publication_heads`) > 0 {
			head = f.KHead()
		}
		p, e := f.repo.PrepareRelease(f.ctx, f.Access("admin_a", true), publication.PrepareInput{SubmissionIDs: ids[at:end], ExpectedHead: head, Reason: "Prepare actual maximum legal original learning fixture."})
		if e != nil {
			f.t.Fatal(e)
		}
		f.Activate(p, head)
	}
}
func learningCapacityCompareApproval(t *testing.T, f *learningCapacityFixture) {
	t.Helper()
	if _, e := f.db.ExecContext(f.ctx, learningApprovalReference); e != nil {
		t.Fatal(e)
	}
	var count int
	var equal, approved bool
	e := f.db.QueryRowContext(f.ctx, `WITH proofs AS MATERIALIZED (
 SELECT learning_content_approved($1,'knowledge',k.id,k.version,k.sha256) actual,
 learning_approval_reference($1,'knowledge',k.id,k.version,k.sha256) original
 FROM publication_members m JOIN knowledge_versions k ON k.id=m.id AND k.version=m.version
 WHERE m.snapshot_id=$1 AND m.kind='knowledge' AND m.availability='active'
 ) SELECT count(*),bool_and(actual=original),bool_and(actual AND original) FROM proofs`, *f.KHead()).Scan(&count, &equal, &approved)
	if e != nil || count != 1000 || !equal || !approved {
		t.Fatal("maximum exact approval proof changed", count, equal, approved, e)
	}
	t.Logf("LEARNING_CAPACITY approvalProofDifferential=1000 allEqual=true allApproved=true")
}

func finishLearningCapacitySetup(t *testing.T, f *learningCapacityFixture, stopSetup context.CancelFunc) {
	t.Helper()
	deadline, bounded := t.Deadline()
	if !bounded {
		t.Fatal("capacity validation requires a bounded whole-Go-test deadline")
	}
	stopSetup()
	validation, stopValidation := context.WithDeadline(context.Background(), deadline)
	t.Cleanup(stopValidation)
	f.ctx = validation
}
func newLearningCapacityFixture(t *testing.T, buildLong ...bool) *learningCapacityFixture {
	t.Helper()
	f := &learningCapacityFixture{questionFixture: &questionFixture{workflowFixture: newWorkflowFixture(t)}}
	setup, stop := context.WithTimeout(context.Background(), 4*time.Minute)
	t.Cleanup(stop)
	f.ctx = setup
	capacity, e := testutil.LearningCapacity("../content/testdata", "../../../content/catalogue/domains.json", "../../../content/questions/elementary-rationals.v1.json")
	if e != nil {
		t.Fatal(e)
	}
	ids := []string{}
	for _, in := range capacity.Content {
		ids = append(ids, f.publish(in))
	}
	f.activateContent(ids)
	paths := capacity.Paths
	if len(buildLong) == 0 || buildLong[0] {
		f.activateContent([]string{f.publish(capacity.LongRoutes)})
	} else {
		paths = nil
	}
	for _, in := range capacity.Questions {
		f.questionInput = in
		sub := f.publishQuestions(in)
		f.submissions = append(f.submissions, sub.ID)
	}
	f.QActivate(f.QPrepare(f.submissions...))
	root := capacity.Root
	for _, name := range []string{"learner_a", "learner_b"} {
		u, c, csrf := f.signup(name)
		proof, e := auth.DecodeContentProof(c, csrf, true)
		if e != nil {
			t.Fatal(e)
		}
		f.ids[name] = u.ID
		f.access[name] = question.Access{TokenHash: proof.TokenHash, CSRF: proof.CSRF, RequestID: "learning-capacity"}
	}
	f.knowledge = question.Identity{ID: root.ID, Version: 1}
	if e = f.db.QueryRow(`SELECT sha256 FROM knowledge_versions WHERE id=$1 AND version=1`, root.ID).Scan(&f.knowledge.SHA256); e != nil {
		t.Fatal(e)
	}
	f.blueprint = question.Identity{ID: "lc-blueprint-0-0", Version: 1}
	if e = f.db.QueryRow(`SELECT sha256 FROM question_blueprints WHERE id=$1 AND version=1`, f.blueprint.ID).Scan(&f.blueprint.SHA256); e != nil {
		t.Fatal(e)
	}
	for _, p := range paths {
		ref := question.Identity{ID: p.ID, Version: p.Version}
		if e = f.db.QueryRow(`SELECT sha256 FROM path_versions WHERE id=$1 AND version=$2`, ref.ID, ref.Version).Scan(&ref.SHA256); e != nil {
			t.Fatal(e)
		}
		f.paths = append(f.paths, ref)
	}
	for kind, want := range map[string]int{"knowledge": 1000, "unit": 4000, "path": 200, "asset": 1000} {
		if got := f.count(`SELECT count(*) FROM publication_members WHERE snapshot_id=$1 AND availability='active' AND kind=$2`, *f.KHead(), kind); got != want {
			t.Fatal(kind, got, want)
		}
	}
	for kind, want := range map[string]int{"template": 200, "instance": 10000, "blueprint": 1000} {
		if got := f.count(`SELECT count(*) FROM question_publication_members WHERE publication_id=$1 AND kind=$2`, *f.QHead(), kind); got != want {
			t.Fatal(kind, got, want)
		}
	}
	t.Logf("LEARNING_CAPACITY actual published knowledge=1000 units=4000 paths=200 assets=1000 templates=200 instances=10000 blueprints=1000 rootCandidates=1000 core=8 directSuccessors=999 actualMaxPathNodes=%d", f.count(`SELECT coalesce(max(n),0) FROM (SELECT count(*) n FROM path_nodes GROUP BY path_id,path_version) sizes`))
	learningCapacityCompareApproval(t, f)
	finishLearningCapacitySetup(t, f, stop)
	return f
}
func learningCapacityCheckPlans(t *testing.T, plans map[string]string, candidateRows int) {
	t.Helper()
	for name, raw := range plans {
		var data []struct {
			Plan struct {
				Rows  int `json:"Actual Rows"`
				Hits  int `json:"Shared Hit Blocks"`
				Reads int `json:"Shared Read Blocks"`
			} `json:"Plan"`
			Time float64 `json:"Execution Time"`
		}
		if json.Unmarshal([]byte(raw), &data) != nil || len(data) != 1 {
			t.Fatal("invalid actual SQL plan")
		}
		want := candidateRows
		if name == "private-five-bodies" {
			want = 5
		}
		if data[0].Plan.Rows != want || data[0].Time >= 8000 {
			t.Fatalf("unexpected bounded actual SQL result %s rows=%d timeMs=%f", name, data[0].Plan.Rows, data[0].Time)
		}
		t.Logf("LEARNING_CAPACITY SQLSUMMARY %s actualRows=%d executionMs=%f sharedHits=%d sharedReads=%d", name, data[0].Plan.Rows, data[0].Time, data[0].Plan.Hits, data[0].Plan.Reads)
		t.Logf("LEARNING_CAPACITY SQLPLAN %s %s", name, raw)
	}
}
func (f *learningCapacityFixture) assessmentInput() assessment.CreateInput {
	return assessment.CreateInput{Knowledge: f.knowledge, Blueprint: f.blueprint, Mode: assessment.ModeNode, ExpectedKnowledgeHead: *f.KHead(), ExpectedQuestionHead: *f.QHead()}
}
func TestLearningCapacityMaxPool(t *testing.T) {
	f := newLearningCapacityFixture(t)
	a := f.Access("learner_a", false)
	learningCapacityMeasure(t, "1000-safe-blueprint-options", func() error {
		v, e := f.repo.ReadLearningKnowledge(f.ctx, a, f.knowledge.ID, 1)
		if e == nil && (len(v.Blueprints) != 1000 || !v.Blueprints[0].Ready) {
			return fmt.Errorf("wrong safe options %d", len(v.Blueprints))
		}
		raw, _ := json.Marshal(v)
		t.Logf("LEARNING_CAPACITY optionsWireBytes=%d", len(raw))
		if len(raw) > 4<<20 {
			return fmt.Errorf("response budget")
		}
		return e
	})
	var approvalPlan []byte
	if e := f.db.QueryRowContext(f.ctx, `EXPLAIN (ANALYZE,BUFFERS,FORMAT JSON) SELECT learning_content_approved($1,'knowledge',k.id,k.version,k.sha256) FROM path_nodes pn JOIN knowledge_versions k ON k.id=pn.knowledge_id AND k.version=pn.knowledge_version WHERE pn.path_id=$2 AND pn.path_version=2`, *f.KHead(), f.paths[0].ID).Scan(&approvalPlan); e != nil {
		t.Fatal(e)
	}
	t.Logf("LEARNING_CAPACITY approval100Plan=%s", approvalPlan)
	saved := []string{}
	for n, p := range f.paths {

		learningCapacityMeasure(t, fmt.Sprintf("enroll-route-%d", n), func() error {
			v, e := f.repo.EnrollLearningPath(f.ctx, f.Access("learner_a", false), p.ID, learning.EnrollInput{Path: p, ExpectedKnowledgeHead: *f.KHead()})
			if e == nil {
				want := 16
				if n < 20 {
					want = 100
				}
				if v.Summary.TotalNodes != want {
					return fmt.Errorf("changed fixed denominator")
				}
				saved = append(saved, v.Summary.ID)
			}
			return e
		})
	}
	learningCapacityMeasure(t, "page-100-routes-3280-fixed-nodes", func() error {
		p, e := f.repo.ListLearningPaths(f.ctx, a, learning.ListQuery{Limit: 100})
		if e == nil {
			if len(p.Items) != 100 || p.Total != 100 {
				return fmt.Errorf("missing selected paths")
			}
			nodes, long := 0, 0
			for _, route := range p.Items {
				nodes += route.TotalNodes
				if route.TotalNodes == 100 {
					long++
				} else if route.TotalNodes != 16 {
					return fmt.Errorf("changed short fixed denominator")
				}
			}
			if long != 20 || nodes != 3280 {
				return fmt.Errorf("wrong actual route page long=%d fixedNodes=%d", long, nodes)
			}
		}
		return e
	})
	input := learning.StartInput{Knowledge: f.knowledge, ExpectedKnowledgeHead: *f.KHead()}
	if _, e := f.repo.StartLearning(f.ctx, f.Access("learner_a", false), f.knowledge.ID, input); e != nil {
		t.Fatal(e)
	}
	if _, e := f.repo.CompleteLearning(f.ctx, f.Access("learner_a", false), f.knowledge.ID, input); e != nil {
		t.Fatal(e)
	}
	var first assessment.AttemptView
	learningCapacityMeasure(t, "create-five-only-sparse-1000-eight-core", func() error {
		var e error
		first, e = f.repo.CreateAssessment(f.ctx, f.Access("learner_a", false), f.assessmentInput())
		if e == nil && len(first.Questions) != 5 {
			return fmt.Errorf("not five")
		}
		return e
	})
	lf := &learningFixture{questionFixture: f.questionFixture, knowledge: f.knowledge, blueprint: f.blueprint}
	var result assessment.ResultView
	learningCapacityMeasure(t, "submit-and-999-direct-successors", func() error {
		var e error
		result, e = f.repo.SubmitAssessment(f.ctx, f.Access("learner_a", false), first.Summary.ID, lf.answers(first, 5))
		if e == nil && (!*result.Passed || !result.Progress.QualificationGranted || len(result.Progress.NewlyUnlocked) != 999) {
			return fmt.Errorf("incorrect grant count %d", len(result.Progress.NewlyUnlocked))
		}
		return e
	})
	learningCapacityMeasure(t, "overview-1000-current", func() error {
		v, e := f.repo.ReadLearningOverview(f.ctx, a)
		if e == nil && (v.CompletedCount != 1 || v.EffectivePassedCount != 1 || v.HistoricalUnlockedCount != 1000) {
			return fmt.Errorf("incorrect actual counts %+v", v)
		}
		return e
	})
	learningCapacityMeasure(t, "100-node-page", func() error {
		v, e := f.repo.ListLearningPathNodes(f.ctx, a, saved[0], learning.ListQuery{Limit: 100})
		if e == nil && (len(v.Items) != 100 || v.Total != 100) {
			return fmt.Errorf("wrong fixed page")
		}
		return e
	})
	// A second real pass supplies independent replacement evidence; no result is inserted or forged.
	in := f.assessmentInput()
	in.Mode = assessment.ModeReview
	second, e := f.repo.CreateAssessment(f.ctx, f.Access("learner_a", false), in)
	if e != nil {
		t.Fatal(e)
	}
	if _, e = f.repo.SubmitAssessment(f.ctx, f.Access("learner_a", false), second.Summary.ID, lf.answers(second, 5)); e != nil {
		t.Fatal(e)
	}
	if _, e = f.repo.WithdrawQuestionVersion(f.ctx, f.Access("admin_a", true), question.WithdrawalInput{Target: question.WithdrawalTarget{Kind: "instance", ID: first.Questions[0].Instance.ID, Version: 1}, ExpectedKnowledgeHead: f.KHead(), ExpectedQuestionHead: f.QHead(), Reason: "Withdraw one exact source; another actual passed assessment remains."}); e != nil {
		t.Fatal(e)
	}
	learningCapacityMeasure(t, "alternative-pass-antijoin", func() error {
		v, e := f.repo.ReadLearningKnowledge(f.ctx, a, f.knowledge.ID, 1)
		if e == nil && v.State.Qualification == nil {
			return fmt.Errorf("alternative actual pass lost")
		}
		return e
	})
	learningCapacityMeasure(t, "historical-approved-svg", func() error {
		v, e := f.repo.ReadLearningAsset(f.ctx, a, second.Summary.ID, second.Questions[0].Assets[0].SHA256)
		if e == nil && len(v) == 0 {
			return fmt.Errorf("missing approved historical asset")
		}
		return e
	})
	for n := 0; n < 101; n++ {
		p, e := f.repo.CreatePractice(f.ctx, f.Access("learner_b", false), assessment.PracticeCreateInput{Knowledge: f.knowledge, ExpectedKnowledgeHead: *f.KHead(), ExpectedQuestionHead: *f.QHead()})
		if e != nil {
			t.Fatal(e)
		}
		if _, e = f.repo.AbandonPractice(f.ctx, f.Access("learner_b", false), p.Summary.ID); e != nil {
			t.Fatal(e)
		}
	}
	for _, offset := range []int{0, 100} {
		learningCapacityMeasure(t, fmt.Sprintf("owned-history-101-offset-%d", offset), func() error {
			v, e := f.repo.ListLearningHistory(f.ctx, f.Access("learner_b", false), learning.ListQuery{Limit: 100, Offset: offset})
			want := 100
			if offset == 100 {
				want = 1
			}
			if e == nil && (v.Total != 101 || len(v.Items) != want) {
				return fmt.Errorf("wrong actual history page")
			}
			return e
		})
	}
	learningCapacityMeasure(t, "batch-template-exposure", func() error {
		_, e := f.repo.ReadQuestionSubmission(f.ctx, f.Access("admin_a", false), f.submissions[0])
		return e
	})
	if n := f.count(`SELECT count(*) FROM learner_answer_exposures WHERE owner_user_id=$1 AND kind='template'`, f.ids["admin_a"]); n != 20 {
		t.Fatal("template exposure expanded instances", n)
	}
	plans, e := f.repo.LearningCapacityPlansForTest(f.ctx, a, f.knowledge, &f.blueprint, second.Summary.ID)
	if e != nil {
		t.Fatal(e)
	}
	learningCapacityCheckPlans(t, plans, 999)
	learningCapacityLocks(t, f)
}

func learningCapacityLocks(t *testing.T, f *learningCapacityFixture) {
	a := f.Access("learner_a", false)
	t.Run("maximum-shared-two-slots-and-exclusive-lock", func(t *testing.T) {
		pub := publication.NewService(f.repo)
		service, e := learning.NewService(f.repo, pub.AcquireValidation)
		if e != nil {
			t.Fatal(e)
		}
		r1, e := service.AcquireValidation(f.ctx)
		if e != nil {
			t.Fatal(e)
		}
		r2, e := pub.AcquireValidation(f.ctx)
		if e != nil {
			r1()
			t.Fatal(e)
		}
		if _, e = service.AcquireValidation(f.ctx); e == nil {
			t.Fatal("third heavy operation queued")
		}
		done := make(chan error, 2)
		startedShared := time.Now()
		for _, name := range []string{"learner_a", "learner_b"} {
			actor, abandon := f.Access(name, false), f.Access(name, false)
			go func() {
				in := f.assessmentInput()
				in.Mode = assessment.ModeDiagnostic
				v, e := f.repo.CreateAssessment(f.ctx, actor, in)
				if e == nil && len(v.Questions) != 5 {
					e = fmt.Errorf("real heavy operation did not create five")
				}
				if e == nil {
					_, e = f.repo.AbandonAssessment(f.ctx, abandon, v.Summary.ID)
				}
				done <- e
			}()
		}
		errorsFound := []error{<-done, <-done}
		r1()
		r2()
		for _, e := range errorsFound {
			if e != nil {
				t.Fatal(e)
			}
		}
		t.Logf("LEARNING_CAPACITY twoRealHeavyCreations=true differentUsers=true twoSharedSlots=true elapsed=%s", time.Since(startedShared))
		gate, e := f.db.BeginTx(f.ctx, nil)
		if e != nil {
			t.Fatal(e)
		}
		defer gate.Rollback()
		if _, e = gate.ExecContext(f.ctx, `SELECT pg_advisory_xact_lock(1296127048)`); e != nil {
			t.Fatal(e)
		}
		started := time.Now()
		_, e = f.repo.ReadLearningOverview(f.ctx, a)
		if !errors.Is(e, auth.ErrUnavailable) || time.Since(started) > 2*time.Second {
			t.Fatal("exclusive lock did not fail within shared 1s lock budget", e)
		}
		t.Logf("LEARNING_CAPACITY exclusiveLockWait=%s no-writing-read=true", time.Since(started))
	})
}

func TestLearningCapacitySourceVolume(t *testing.T) {
	f := newLearningCapacityFixture(t, false)
	a := f.Access("learner_a", false)
	t.Run("safe-options", func(t *testing.T) {
		learningCapacityMeasure(t, "published-1000-options-1000-candidates", func() error {
			v, e := f.repo.ReadLearningKnowledge(f.ctx, a, f.knowledge.ID, 1)
			if e == nil && (len(v.Blueprints) != 1000 || !v.Blueprints[0].Ready) {
				return fmt.Errorf("wrong safe options %d", len(v.Blueprints))
			}
			raw, _ := json.Marshal(v)
			t.Logf("LEARNING_CAPACITY optionsWireBytes=%d", len(raw))
			if len(raw) > 4<<20 {
				return fmt.Errorf("response budget")
			}
			return e
		})
	})
	t.Run("overview-empty", func(t *testing.T) {
		learningCapacityMeasure(t, "overview-1000-unread", func() error {
			v, e := f.repo.ReadLearningOverview(f.ctx, a)
			if e == nil && (v.StartedCount != 0 || v.CompletedCount != 0 || v.EffectivePassedCount != 0 || v.HistoricalUnlockedCount != 0 || len(v.AvailablePaths) != 200) {
				return fmt.Errorf("incorrect empty counts")
			}
			return e
		})
	})
	read := learning.StartInput{Knowledge: f.knowledge, ExpectedKnowledgeHead: *f.KHead()}
	if _, e := f.repo.StartLearning(f.ctx, f.Access("learner_a", false), f.knowledge.ID, read); e != nil {
		t.Fatal(e)
	}
	if _, e := f.repo.CompleteLearning(f.ctx, f.Access("learner_a", false), f.knowledge.ID, read); e != nil {
		t.Fatal(e)
	}
	in := f.assessmentInput()
	var attempt assessment.AttemptView
	learningCapacityMeasure(t, "five-body-projection", func() error {
		var e error
		attempt, e = f.repo.CreateAssessment(f.ctx, f.Access("learner_a", false), in)
		if e == nil && len(attempt.Questions) != 5 {
			return fmt.Errorf("not five actual questions")
		}
		return e
	})
	plans, e := f.repo.LearningCapacityPlansForTest(f.ctx, a, f.knowledge, &f.blueprint, attempt.Summary.ID)
	if e != nil {
		t.Fatal(e)
	}
	learningCapacityCheckPlans(t, plans, 1000)
	t.Run("submit-999-successors", func(t *testing.T) {
		lf := &learningFixture{questionFixture: f.questionFixture, knowledge: f.knowledge, blueprint: f.blueprint}
		learningCapacityMeasure(t, "node-pass-and-999-successors", func() error {
			v, e := f.repo.SubmitAssessment(f.ctx, f.Access("learner_a", false), attempt.Summary.ID, lf.answers(attempt, 5))
			if e == nil && (!*v.Passed || !v.Progress.QualificationGranted || len(v.Progress.NewlyUnlocked) != 999) {
				return fmt.Errorf("incorrect direct unlock count %d", len(v.Progress.NewlyUnlocked))
			}
			return e
		})
	})
	t.Run("overview-after-node-pass", func(t *testing.T) {
		learningCapacityMeasure(t, "overview-1000-with-actual-pass", func() error {
			v, e := f.repo.ReadLearningOverview(f.ctx, a)
			if e == nil && (v.CompletedCount != 1 || v.EffectivePassedCount != 1 || v.HistoricalUnlockedCount != 1000) {
				return fmt.Errorf("incorrect actual reading and pass counts")
			}
			return e
		})
	})

	learningCapacityMeasure(t, "maximum-template-exposure-20-families", func() error {
		_, e := f.repo.ReadQuestionSubmission(f.ctx, f.Access("admin_a", false), f.submissions[0])
		return e
	})
	if got := f.count(`SELECT count(*) FROM learner_answer_exposures WHERE owner_user_id=$1 AND kind='template'`, f.ids["admin_a"]); got != 20 {
		t.Fatalf("expanded template exposures: %d", got)
	}
	secondInput := f.assessmentInput()
	secondInput.Mode = assessment.ModeReview
	second, e := f.repo.CreateAssessment(f.ctx, f.Access("learner_a", false), secondInput)
	if e != nil {
		t.Fatal(e)
	}
	lf := &learningFixture{questionFixture: f.questionFixture, knowledge: f.knowledge, blueprint: f.blueprint}
	if _, e = f.repo.SubmitAssessment(f.ctx, f.Access("learner_a", false), second.Summary.ID, lf.answers(second, 5)); e != nil {
		t.Fatal(e)
	}
	if _, e = f.repo.WithdrawQuestionVersion(f.ctx, f.Access("admin_a", true), question.WithdrawalInput{Target: question.WithdrawalTarget{Kind: "instance", ID: attempt.Questions[0].Instance.ID, Version: 1}, ExpectedKnowledgeHead: f.KHead(), ExpectedQuestionHead: f.QHead(), Reason: "One exact source withdrawn; a second actual pass remains."}); e != nil {
		t.Fatal(e)
	}
	learningCapacityMeasure(t, "maximum-alternative-pass-antijoin", func() error {
		v, e := f.repo.ReadLearningKnowledge(f.ctx, a, f.knowledge.ID, 1)
		if e == nil && (v.State.Qualification == nil || v.State.Qualification.EvidenceAttemptID != second.Summary.ID) {
			return fmt.Errorf("missing exact replacement pass")
		}
		return e
	})
	learningCapacityMeasure(t, "maximum-owned-historical-svg", func() error {
		v, e := f.repo.ReadLearningAsset(f.ctx, a, second.Summary.ID, second.Questions[0].Assets[0].SHA256)
		if e == nil && len(v) == 0 {
			return fmt.Errorf("empty historical SVG")
		}
		return e
	})
	for n := 0; n < 101; n++ {
		v, e := f.repo.CreatePractice(f.ctx, f.Access("learner_b", false), assessment.PracticeCreateInput{Knowledge: f.knowledge, ExpectedKnowledgeHead: *f.KHead(), ExpectedQuestionHead: *f.QHead()})
		if e != nil {
			t.Fatal(e)
		}
		if _, e = f.repo.AbandonPractice(f.ctx, f.Access("learner_b", false), v.Summary.ID); e != nil {
			t.Fatal(e)
		}
	}
	for _, offset := range []int{0, 100} {
		learningCapacityMeasure(t, fmt.Sprintf("maximum-owned-history-101-offset-%d", offset), func() error {
			v, e := f.repo.ListLearningHistory(f.ctx, f.Access("learner_b", false), learning.ListQuery{Limit: 100, Offset: offset})
			want := 100
			if offset == 100 {
				want = 1
			}
			if e == nil && (v.Total != 101 || len(v.Items) != want) {
				return fmt.Errorf("incorrect real history page")
			}
			return e
		})
	}
	learningCapacityLocks(t, f)

}

func TestLearningFixturePhaseDeadlines(t *testing.T) {
	setup, cancelSetup := context.WithCancel(context.Background())
	f := &learningCapacityFixture{questionFixture: &questionFixture{workflowFixture: &workflowFixture{authFixture: &authFixture{ctx: setup}}}}
	finishLearningCapacitySetup(t, f, cancelSetup)
	if !errors.Is(setup.Err(), context.Canceled) {
		t.Fatal("setup must be closed after publication")
	}
	if f.ctx.Err() != nil {
		t.Fatal("expired setup must not cancel validation", f.ctx.Err())
	}
	deadline, bounded := f.ctx.Deadline()
	wholeDeadline, boundedGo := t.Deadline()
	if !bounded || !boundedGo || !deadline.Equal(wholeDeadline) {
		t.Fatal("validation must stay inside the existing whole-Go-test deadline")
	}
}
