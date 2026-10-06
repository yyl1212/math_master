package study_test

import (
	"errors"
	"github.com/yyl1212/math_master/backend/internal/study"
	"strings"
	"testing"
)

func TestStudyStateTransitions(t *testing.T) {
	ref := study.KnowledgeRef{ID: "fractions", Version: 1, SHA256: strings.Repeat("a", 64)}
	for _, v := range []struct {
		s    study.State
		a    study.Action
		want study.State
		bad  bool
	}{{study.Unlearned, study.Begin, study.Learning, false}, {study.Learning, study.Complete, study.Completed, false}, {study.Completed, study.StartReview, study.Reviewing, false}, {study.Reviewing, study.FinishReview, study.Completed, false}, {study.Unlearned, study.Complete, "", true}, {study.Learning, study.StartReview, "", true}, {study.Completed, study.Begin, study.Completed, false}, {study.Reviewing, study.Begin, study.Reviewing, false}, {study.Completed, study.Complete, study.Completed, false}, {study.Unlearned, study.FinishReview, "", true}, {"mastered", study.Begin, "", true}, {study.Completed, "pass", "", true}} {
		t.Run(string(v.s)+"/"+string(v.a), func(t *testing.T) {
			got, e := study.ApplyState(v.s, v.a, ref, ref)
			if v.bad {
				if !errors.Is(e, study.ErrStateConflict) && !errors.Is(e, study.ErrInvalid) {
					t.Fatal("illegal transition accepted", got, e)
				}
			} else if e != nil || got != v.want {
				t.Fatal(got, e, "want", v.want)
			}
		})
	}
	newer := ref
	newer.Version = 2
	got, e := study.ApplyState(study.Completed, study.Begin, ref, newer)
	if e != nil || got != study.Completed {
		t.Fatal("ordinary reread reset completion", got, e)
	}
	if !study.MaterialChanged(&ref, &newer) || study.MaterialChanged(&ref, &ref) || study.MaterialChanged(nil, &newer) {
		t.Fatal("material update not projected independently")
	}
}
