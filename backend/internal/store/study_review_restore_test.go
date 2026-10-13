package store_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/pressly/goose/v3"
	"github.com/yyl1212/math_master/backend/internal/store"
	"github.com/yyl1212/math_master/backend/internal/studyreview"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestReviewCapacitySchemaRestore(t *testing.T) {
	container := os.Getenv("STUDY_REVIEW_RESTORE_CONTAINER")
	if container == "" {
		t.Skip("requires owned PostgreSQL 17.11 Docker service for real restore")
	}
	f := newKnowledgeAdminFixture(t)
	ctx, cancel := context.WithTimeout(context.Background(), 270*time.Second)
	f.ctx = ctx
	defer cancel()
	if e := goose.DownToContext(f.ctx, f.db, "../../../db/migrations", 14); e != nil {
		t.Fatal(e)
	}
	k := completedReviewKnowledge(t, f)
	n := sharingNote(t, f, k, "原14数学笔记 $x^2$，恢复后必须逐字节保持。")
	sharingSubmit(t, f, k, n)
	var database string
	if e := f.db.QueryRowContext(f.ctx, "SELECT current_database()").Scan(&database); e != nil {
		t.Fatal(e)
	}
	revisionBytes, revisionErr := exec.Command("git", "rev-parse", "HEAD").Output()
	if revisionErr != nil {
		t.Fatal(revisionErr)
	}
	revision := strings.TrimSpace(string(revisionBytes))
	folder, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	run := func(action string) {
		t.Helper()
		cmd := exec.CommandContext(f.ctx, "python3", "../../../tools/verify/study-review-restore.py", action, container, database, folder, revision)
		b, e := cmd.CombinedOutput()
		if e != nil {
			t.Fatalf("restore %s: %s %v", action, b, e)
		}
		t.Log(action, string(b))
	}
	run("capture14")
	defer func() {
		cmd := exec.Command("python3", "../../../tools/verify/study-review-restore.py", "cleanup14", container, database, folder, revision)
		if b, e := cmd.CombinedOutput(); e != nil {
			t.Errorf("owned restore cleanup: %s %v", b, e)
		}
	}()
	var target struct {
		Database string `json:"database"`
		User     string `json:"user"`
		Port     int    `json:"port"`
	}
	b, e := os.ReadFile(filepath.Join(folder, "14-target.json"))
	if e != nil {
		t.Fatal(e)
	}
	if e = json.Unmarshal(b, &target); e != nil {
		t.Fatal(e)
	}
	restored, e := sql.Open("pgx", fmt.Sprintf("postgres://%s@127.0.0.1:%d/%s?sslmode=disable", target.User, target.Port, target.Database))
	if e != nil {
		t.Fatal(e)
	}
	if e = store.Up(f.ctx, restored, "../../../db/migrations"); e != nil {
		restored.Close()
		t.Fatal(e)
	}
	{
		repo := store.New(restored)
		h, e := repo.ReadReviewHealth(f.ctx)
		if e != nil || !h.Capability || !h.SchemaReady {
			t.Fatal("restored managed/review digest", h, e)
		}
		n, e := repo.ReadSchemaHealth(f.ctx)
		if e != nil || !n.Capability || !n.SchemaReady {
			t.Fatal("restored original sharing digest", n, e)
		}
	}
	if e = restored.Close(); e != nil {
		t.Fatal(e)
	}
	run("upgrade15")
	if e := store.Up(f.ctx, f.db, "../../../db/migrations"); e != nil {
		t.Fatal(e)
	}
	for i, mode := range []studyreview.Mode{studyreview.Read, studyreview.Recall} {
		r := startReview(t, f, k, mode, studyreview.Statement, []string{"restore-read", "restore-recall"}[i])
		body := ""
		if mode == studyreview.Recall {
			body = "本人默写 $a^2+b^2$"
		}
		c := compareReview(t, f, r, body, false)
		if _, e := f.repo.GradeReviewSession(f.ctx, f.Access("learner_a", "grade-restore-"+r.SessionID), r.SessionID, studyreview.GradeInput{ExpectedSequence: c.Sequence, Rating: studyreview.Familiar}); e != nil {
			t.Fatal(e)
		}
	}
	r := startReview(t, f, k, studyreview.Recall, studyreview.Statement, "restore-abandon")
	if _, e := f.repo.AbandonReviewSession(f.ctx, f.Access("learner_a", "restore-abandon-write"), r.SessionID, studyreview.AbandonInput{ExpectedSequence: r.Sequence}); e != nil {
		t.Fatal(e)
	}
	r = startReview(t, f, k, studyreview.Recall, studyreview.Statement, "restore-working")
	if _, e := f.repo.SaveReviewDraft(f.ctx, f.Access("learner_a", "restore-draft"), r.SessionID, studyreview.DraftInput{ExpectedSequence: r.Sequence, ExpectedDraftRevision: 0, Body: "未比对的本人草稿"}); e != nil {
		t.Fatal(e)
	}
	for _, table := range []string{"study_review_sessions", "study_review_snapshots", "study_review_schedules", "study_review_events", "study_review_receipts"} {
		if f.count("SELECT count(*) FROM "+table) == 0 {
			t.Fatal("missing restore data", table)
		}
	}
	run("capture15")
}
