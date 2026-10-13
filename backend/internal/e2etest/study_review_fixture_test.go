package e2etest

import (
	"context"
	"github.com/yyl1212/math_master/backend/internal/knowledgeadmin"
	"github.com/yyl1212/math_master/backend/internal/studyreview"
	"strings"
	"testing"
)

func TestStudyReviewSceneResetAndAtomicity(t *testing.T) {
	db, repo, accounts, admin, root := fixtureStorageSetup(t)
	ctx := context.Background()
	if e := resetManagedKnowledge(ctx, db, repo, accounts, admin, root); e != nil {
		t.Fatal("original real scene cannot reset with review FK", e)
	}
	proof, e := fixtureAccess(ctx, accounts, "auth_learner", false)
	if e != nil {
		t.Fatal(e)
	}
	a := knowledgeadmin.Access{TokenHash: proof.TokenHash, CSRF: proof.CSRF, RequestID: "review-reset-test", IdempotencyKey: "begin"}
	k, e := repo.ReadCurrentKnowledge(ctx, knowledgeadmin.KnowledgeID("demo-rational-fraction"))
	if e != nil {
		t.Fatal(e)
	}
	r, e := repo.ApplyManagedStudy(ctx, a, k.ID, "begin", knowledgeadmin.ManagedStudyInput{Knowledge: k.Ref})
	if e != nil {
		t.Fatal(e)
	}
	a.IdempotencyKey = "complete"
	if _, e = repo.ApplyManagedStudy(ctx, a, k.ID, "complete", knowledgeadmin.ManagedStudyInput{Knowledge: k.Ref, ExpectedSequence: r.Record.Sequence}); e != nil {
		t.Fatal(e)
	}
	a.IdempotencyKey = "start"
	started, e := repo.StartReviewSession(ctx, a, studyreview.StartInput{Knowledge: k.Ref, Mode: studyreview.Recall, Target: studyreview.Statement})
	if e != nil {
		t.Fatal(e)
	}
	a.IdempotencyKey = "compare"
	compared, e := repo.CompareReviewSession(ctx, a, started.SessionID, studyreview.CompareInput{ExpectedSequence: 1, Body: "测试自制作答"})
	if e != nil {
		t.Fatal(e)
	}
	a.IdempotencyKey = "grade"
	if _, e = repo.GradeReviewSession(ctx, a, started.SessionID, studyreview.GradeInput{ExpectedSequence: compared.Sequence, Rating: studyreview.Familiar}); e != nil {
		t.Fatal(e)
	}
	before := fixtureStorageFingerprint(t, db)
	var reviewRows int
	if e = db.QueryRow(`SELECT count(*) FROM study_review_sessions`).Scan(&reviewRows); e != nil || reviewRows != 1 {
		t.Fatal(e)
	}
	if _, e = db.Exec(`CREATE TABLE review_reset_blocker(user_id uuid REFERENCES auth_users(id));INSERT INTO review_reset_blocker SELECT id FROM auth_users LIMIT 1`); e != nil {
		t.Fatal(e)
	}
	if e = resetFixtureStorage(ctx, db, true); e == nil {
		t.Fatal("foreign-key blocker ignored")
	}
	after := fixtureStorageFingerprint(t, db)
	for table, digest := range before {
		if after[table] != digest {
			t.Fatal("partial original reset", table)
		}
	}
	if e = db.QueryRow(`SELECT count(*) FROM study_review_sessions`).Scan(&reviewRows); e != nil || reviewRows != 1 {
		t.Fatal("partial review reset", e)
	}
	if _, e = db.Exec(`DROP TABLE review_reset_blocker`); e != nil {
		t.Fatal(e)
	}
	counted, count := fixtureStorageCountingDB(t, db)
	if e = resetFixtureStorage(ctx, counted, true); e != nil {
		t.Fatal(e)
	}
	if count.Load() != 1 {
		t.Fatal("reset split into storage round trips", count.Load())
	}
	for _, table := range []string{"study_review_sessions", "study_review_snapshots", "study_review_schedules", "study_review_events", "study_review_receipts"} {
		var rows int
		if e = db.QueryRow("SELECT count(*) FROM " + table).Scan(&rows); e != nil || rows != 0 {
			t.Fatal("new rows remain", table, e)
		}
	}
}

func TestStudyReviewStorageCounterIncludesNewTables(t *testing.T) {
	db, _, _, _, _ := fixtureStorageSetup(t)
	counted, count := fixtureStorageCountingDB(t, db)
	if _, e := counted.Exec(`TRUNCATE study_review_receipts`); e != nil {
		t.Fatal(e)
	}
	if count.Load() != 1 {
		t.Fatal("new reset round trip not counted", count.Load())
	}
}

func TestStudyReviewFixturePublishedAndLearned(t *testing.T) {
	db, repo, accounts, admin, root := fixtureStorageSetup(t)
	ctx := context.Background()
	if e := resetStudyReview(ctx, db, repo, accounts, admin, root); e != nil {
		t.Fatal(e)
	}
	proof, e := fixtureAccess(ctx, accounts, "auth_learner", false)
	if e != nil {
		t.Fatal(e)
	}
	a := knowledgeadmin.Access{TokenHash: proof.TokenHash, RequestID: "fixture-overview"}
	overview, e := repo.ReadReviewOverview(ctx, a)
	if e != nil || overview.Total != 2 || overview.Unplanned != 2 {
		t.Fatalf("real learned fixtures %+v %v", overview, e)
	}
	p, e := repo.ListReviewQueue(ctx, a, studyreview.QueueQuery{Limit: 20})
	if e != nil || len(p.Items) != 2 {
		t.Fatal(e)
	}
	proofs := 0
	for _, item := range p.Items {
		if item.ProofAvailable {
			proofs++
		}
	}
	if proofs != 1 {
		t.Fatal("proof eligibility", proofs)
	}
	fixtureStorageActors(t, db, 6)
}

func TestStudyReviewBoundaryFixture(t *testing.T) {
	db, repo, accounts, admin, root := fixtureStorageSetup(t)
	ctx := context.Background()
	if e := resetStudyReview(ctx, db, repo, accounts, admin, root); e != nil {
		t.Fatal(e)
	}
	for _, op := range []string{"boundary-raw", "boundary-wire", "boundary-oversize"} {
		if e := studyReviewChange(ctx, repo, accounts, op); e != nil {
			t.Fatal(op, e)
		}
		if op == "boundary-wire" {
			var title string
			if e := db.QueryRowContext(ctx, `SELECT point->>'title' FROM managed_knowledge WHERE external_id='demo-rational-fraction'`).Scan(&title); e != nil {
				t.Fatal(e)
			}
			if title != strings.Repeat("&", 512) {
				t.Fatal("wire case needs the largest legal escaped title envelope")
			}
		}
	}
}
