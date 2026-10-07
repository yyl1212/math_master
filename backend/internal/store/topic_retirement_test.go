package store_test

import (
	"errors"
	"github.com/yyl1212/math_master/backend/internal/correction"
	"github.com/yyl1212/math_master/backend/internal/learning"
	"github.com/yyl1212/math_master/backend/internal/question"
	"github.com/yyl1212/math_master/backend/internal/study"
	"testing"
	"time"
)

func TestTopicRetirementWritesAndReplay(t *testing.T) {
	f := newStudyFixture(t)
	a := f.workflowFixture.Access("author_a", false)
	in := learning.StartInput{Knowledge: question.Identity{ID: f.Ref().ID, Version: f.Ref().Version, SHA256: f.Ref().SHA256}, ExpectedKnowledgeHead: *f.Pair().KnowledgeHead}
	if _, e := f.repo.StartLearning(f.ctx, a, f.Ref().ID, in); e != nil {
		t.Fatal("legacy start", e)
	}
	before := f.count("SELECT count(*) FROM learning_events")
	f.exec("UPDATE topic_learning_state SET experience_mode='topics' WHERE singleton")
	if _, e := f.repo.StartLearning(f.ctx, a, f.Ref().ID, in); !errors.Is(e, study.ErrModuleRetired) {
		t.Fatal("old replay accepted", e)
	}
	if f.count("SELECT count(*) FROM learning_events") != before {
		t.Fatal("retirement changed facts")
	}
}
func TestTopicRetirementConcurrentCutover(t *testing.T) {
	f := newStudyFixture(t)
	tx, e := f.db.Begin()
	if e != nil {
		t.Fatal(e)
	}
	defer tx.Rollback()
	if _, e = tx.Exec("SELECT experience_mode FROM topic_learning_state WHERE singleton FOR UPDATE"); e != nil {
		t.Fatal(e)
	}
	a := f.workflowFixture.Access("author_a", false)
	in := learning.StartInput{Knowledge: question.Identity{ID: f.Ref().ID, Version: f.Ref().Version, SHA256: f.Ref().SHA256}, ExpectedKnowledgeHead: *f.Pair().KnowledgeHead}
	done := make(chan error, 1)
	go func() { _, e := f.repo.StartLearning(f.ctx, a, f.Ref().ID, in); done <- e }()
	waiting := false
	deadline := time.Now().Add(750 * time.Millisecond)
	for time.Now().Before(deadline) {
		if f.count(`SELECT count(*) FROM pg_stat_activity WHERE datname=current_database() AND wait_event_type='Lock' AND query LIKE '%topic_learning_state%FOR SHARE%'`) > 0 {
			waiting = true
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	if !waiting {
		t.Fatal("old write did not wait on experience configuration")
	}
	if _, e = tx.Exec("UPDATE topic_learning_state SET experience_mode='topics' WHERE singleton"); e != nil {
		t.Fatal(e)
	}
	if e = tx.Commit(); e != nil {
		t.Fatal(e)
	}
	if e = <-done; !errors.Is(e, study.ErrModuleRetired) {
		t.Fatal("old write crossed cutover", e)
	}
	if f.count("SELECT count(*) FROM learning_events") != 0 {
		t.Fatal("old write committed after cutover")
	}
}
func TestTopicRetirementMissingNewCapabilityFailsClosed(t *testing.T) {
	f := newStudyFixture(t)
	f.exec("ALTER TABLE topic_learning_state RENAME TO damaged_topic_learning_state")
	a := f.workflowFixture.Access("author_a", false)
	in := learning.StartInput{Knowledge: question.Identity{ID: f.Ref().ID, Version: f.Ref().Version, SHA256: f.Ref().SHA256}, ExpectedKnowledgeHead: *f.Pair().KnowledgeHead}
	if _, e := f.repo.StartLearning(f.ctx, a, f.Ref().ID, in); !errors.Is(e, study.ErrNotConfigured) {
		t.Fatal("ever-enabled schema fell back to old writes", e)
	}
	if f.count("SELECT count(*) FROM learning_events") != 0 {
		t.Fatal("old event despite missing capability")
	}
}
func TestTopicRetirementDirectQuestionAndCorrectionPreflight(t *testing.T) {
	f := newStudyFixture(t)
	f.exec("UPDATE topic_learning_state SET experience_mode='topics' WHERE singleton")
	if _, e := f.repo.CreateQuestionDraft(f.ctx, f.workflowFixture.Access("author_a", false), questionDraftInput(t)); !errors.Is(e, study.ErrModuleRetired) {
		t.Fatal("direct question write bypassed", e)
	}
	if _, e := f.repo.CorrectionPreflight(f.ctx, f.workflowFixture.Access("admin_a", false), correction.CreateCaseAction); !errors.Is(e, study.ErrModuleRetired) {
		t.Fatal("direct grading preflight bypassed", e)
	}
}
