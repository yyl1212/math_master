package store_test

import (
	"errors"
	"github.com/yyl1212/math_master/backend/internal/learning"
	"testing"
)

func (f *learningFixture) startInput() learning.StartInput {
	return learning.StartInput{Knowledge: f.knowledge, ExpectedKnowledgeHead: *f.KHead()}
}
func (f *learningFixture) completeInput() learning.CompleteInput { return f.startInput() }
func TestLearningCompleteRequiresStart(t *testing.T) {
	f := newLearningFixture(t)
	_, e := f.repo.CompleteLearning(f.ctx, f.Access("learner_a", false), f.knowledge.ID, f.completeInput())
	if !errors.Is(e, learning.ErrStateConflict) {
		t.Fatal(e)
	}
}
func TestLearningActionsExplicitAndIdempotent(t *testing.T) {
	f := newLearningFixture(t)
	key := f.Access("learner_a", false)
	in := f.startInput()
	got, e := f.repo.StartLearning(f.ctx, key, f.knowledge.ID, in)
	if e != nil || got.State != learning.Learning || got.StartedAt == nil || got.CompletedAt != nil || !got.EverUnlocked {
		t.Fatal(got, e)
	}
	first := *got.StartedAt
	for _, a := range []struct{ same bool }{{true}, {false}} {
		access := key
		if !a.same {
			access = f.Access("learner_a", false)
		}
		got, e = f.repo.StartLearning(f.ctx, access, f.knowledge.ID, in)
		if e != nil || !got.StartedAt.Equal(first) || f.count(`SELECT count(*) FROM learning_events`) != 1 {
			t.Fatal("start changed first proof", got, e)
		}
	}
	got, e = f.repo.CompleteLearning(f.ctx, f.Access("learner_a", false), f.knowledge.ID, f.completeInput())
	if e != nil || got.State != learning.Learned || !got.CompletionValid || got.Qualification != nil || got.CompletedAt == nil {
		t.Fatal(got, e)
	}
	completed := *got.CompletedAt
	got, e = f.repo.CompleteLearning(f.ctx, f.Access("learner_a", false), f.knowledge.ID, f.completeInput())
	if e != nil || !got.CompletedAt.Equal(completed) || f.count(`SELECT count(*) FROM learning_events`) != 2 {
		t.Fatal("repeat completion created fictitious learning", got, e)
	}
	bad := in
	bad.Knowledge.Version = 2
	if _, e = f.repo.StartLearning(f.ctx, f.Access("learner_a", false), f.knowledge.ID, bad); !errors.Is(e, learning.ErrVersionStale) {
		t.Fatal("old or forged version accepted", e)
	}
}
func TestLearningCompletionNewMaterialKeepsFirstTime(t *testing.T) {
	f := newLearningFixture(t)
	if _, e := f.repo.StartLearning(f.ctx, f.Access("learner_a", false), f.knowledge.ID, f.startInput()); e != nil {
		t.Fatal(e)
	}
	got, e := f.repo.CompleteLearning(f.ctx, f.Access("learner_a", false), f.knowledge.ID, f.completeInput())
	if e != nil {
		t.Fatal(e)
	}
	first := *got.CompletedAt
	v := f.Input()
	v.Package.Version = 2
	v.Package.Units[0].Version = 2
	v.Package.Units[0].Angles[0].Body += " Updated original explanation."
	old := f.KHead()
	sub := f.ApprovedInput(v)
	f.Activate(f.Prepare(sub, old), old)
	got, e = f.repo.StartLearning(f.ctx, f.Access("learner_a", false), f.knowledge.ID, f.startInput())
	if e != nil || got.CompletionValid || got.CompletedAt == nil || !got.CompletedAt.Equal(first) {
		t.Fatal("old first completion treated as new lecture proof", got, e)
	}
	got, e = f.repo.CompleteLearning(f.ctx, f.Access("learner_a", false), f.knowledge.ID, f.completeInput())
	if e != nil || !got.CompletionValid || !got.CompletedAt.Equal(first) || f.count(`SELECT count(*) FROM learning_events`) != 3 {
		t.Fatal("explicit refreshed completion missing", got, e)
	}
	if _, e = f.repo.CompleteLearning(f.ctx, f.Access("learner_a", false), f.knowledge.ID, f.completeInput()); e != nil || f.count(`SELECT count(*) FROM learning_events`) != 3 {
		t.Fatal("same lecture proof duplicated", e)
	}
}
