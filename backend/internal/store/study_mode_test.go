package store_test

import (
	"errors"
	"github.com/yyl1212/math_master/backend/internal/study"
	"github.com/yyl1212/math_master/backend/internal/taxonomy"
	"testing"
)

func TestStudyExperienceModeCapabilities(t *testing.T) {
	f := newStudyFixture(t)
	mode, e := f.repo.ReadExperienceMode(f.ctx)
	if e != nil || mode != taxonomy.ModeLegacy {
		t.Fatal(mode, e)
	}
	f.exec("UPDATE topic_learning_state SET experience_mode='topics'")
	mode, e = f.repo.ReadExperienceMode(f.ctx)
	if e != nil || mode != taxonomy.ModeTopics {
		t.Fatal(mode, e)
	}
	f.exec("ALTER TABLE study_events RENAME TO broken_study_events")
	if _, e = f.repo.ReadExperienceMode(f.ctx); !errors.Is(e, study.ErrNotConfigured) {
		t.Fatal("topics damaged schema downgraded", e)
	}
}
