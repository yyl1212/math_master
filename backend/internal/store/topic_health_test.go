package store_test

import (
	"testing"
)

func TestTopicSchemaHealthFailsClosedWithoutUserData(t *testing.T) {
	f := newStudyFixture(t)
	f.exec("ALTER TABLE study_legacy_event_links DISABLE TRIGGER study_legacy_link_guard")
	health, e := f.repo.ReadTopicSchemaHealth(f.ctx)
	if e != nil || health.SchemaReady {
		t.Fatal("partial schema marked ready", e)
	}
}
