package store_test

import (
	"github.com/yyl1212/math_master/backend/internal/store"
	"github.com/yyl1212/math_master/backend/internal/study"
	"github.com/yyl1212/math_master/backend/internal/taxonomy"
	"strings"
	"sync"
	"testing"
)

func newCutoverFixture(t *testing.T) (*learningFixture, study.CutoverInput) {
	f := newLearningFixture(t)
	f.repo = store.NewWithTrustedCodeSHA(f.db, strings.Repeat("a", 40))
	pair := f.migrationPair()
	batch, e := f.repo.MigrateLegacyStudyBatch(f.ctx, 50, nil)
	if e != nil {
		t.Fatal(e)
	}
	return f, study.CutoverInput{ExpectedPair: pair, CodeSHA: strings.Repeat("a", 40), ExpectedMigrationBatchID: batch.BatchID, Reason: "Explicit isolated topic experience cutover.", BackupRecord: strings.Repeat("b", 64)}
}
func assertLegacyCutover(t *testing.T, f *learningFixture) {
	t.Helper()
	var mode string
	if e := f.db.QueryRow("SELECT experience_mode FROM topic_learning_state WHERE singleton").Scan(&mode); e != nil || mode != "legacy" || f.count("SELECT count(*) FROM topic_cutovers") != 0 {
		t.Fatal("unsafe mode changed", mode, e)
	}
}
func TestTopicCutoverRefusesPendingMigration(t *testing.T) {
	f, in := newCutoverFixture(t)
	if _, e := f.repo.StartLearning(f.ctx, f.Access("learner_a", false), f.knowledge.ID, f.startInput()); e != nil {
		t.Fatal(e)
	}
	if _, e := f.repo.ActivateTopicExperience(f.ctx, in); e == nil {
		t.Fatal("stale done batch trusted")
	}
	assertLegacyCutover(t, f)
}
func TestTopicCutoverRefusesPartialSchema(t *testing.T) {
	f, in := newCutoverFixture(t)
	f.exec("ALTER TABLE study_legacy_event_links DISABLE TRIGGER study_legacy_link_guard")
	if _, e := f.repo.ActivateTopicExperience(f.ctx, in); e == nil {
		t.Fatal("partial schema accepted")
	}
	assertLegacyCutover(t, f)
}
func TestTopicCutoverRefusesPairMismatch(t *testing.T) {
	f, in := newCutoverFixture(t)
	id := f.ID()
	in.ExpectedPair.TaxonomyHead = &id
	if _, e := f.repo.ActivateTopicExperience(f.ctx, in); e == nil {
		t.Fatal("wrong pair accepted")
	}
	assertLegacyCutover(t, f)
}
func TestTopicCutoverSingleWinnerAndReplay(t *testing.T) {
	f, in := newCutoverFixture(t)
	var wg sync.WaitGroup
	results := make(chan study.CutoverReport, 2)
	errs := make(chan error, 2)
	for n := 0; n < 2; n++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			out, e := f.repo.ActivateTopicExperience(f.ctx, in)
			results <- out
			errs <- e
		}()
	}
	wg.Wait()
	close(results)
	close(errs)
	for e := range errs {
		if e != nil {
			t.Fatal(e)
		}
	}
	activated := 0
	var winner study.CutoverReport
	for out := range results {
		if out.Activated {
			activated++
			winner = out
		}
	}
	if activated != 1 || f.count("SELECT count(*) FROM topic_cutovers") != 1 {
		t.Fatal("cutover duplicated", activated)
	}
	out, e := f.repo.ActivateTopicExperience(f.ctx, in)
	if e != nil || out.Activated || out.CutoverID == nil || *out.CutoverID != *winner.CutoverID || !out.RecordedAt.Equal(*winner.RecordedAt) {
		t.Fatal("replay rewrote audit", e)
	}
}
func TestTopicCutoverRetainsActiveAttemptBytes(t *testing.T) {
	f, in := newCutoverFixture(t)
	f.createDiagnostic("learner_a")
	var before, after string
	if e := f.db.QueryRow("SELECT md5(row_to_json(a)::text) FROM assessment_attempts a").Scan(&before); e != nil {
		t.Fatal(e)
	}
	if _, e := f.repo.ActivateTopicExperience(f.ctx, in); e != nil {
		t.Fatal(e)
	}
	if e := f.db.QueryRow("SELECT md5(row_to_json(a)::text) FROM assessment_attempts a").Scan(&after); e != nil || before != after {
		t.Fatal("active attempt changed", e)
	}
	if _, e := f.repo.StartLearning(f.ctx, f.Access("learner_a", false), f.knowledge.ID, f.startInput()); e == nil {
		t.Fatal("old write restored")
	}
}
func TestTopicCutoverTrustedCodeAndEmptyKnowledge(t *testing.T) {
	f, in := newCutoverFixture(t)
	in.CodeSHA = strings.Repeat("c", 40)
	if _, e := f.repo.ActivateTopicExperience(f.ctx, in); e == nil {
		t.Fatal("request supplied its own code version")
	}
	assertLegacyCutover(t, f)
	empty := newTopicWorkflowFixture(t)
	empty.repo = store.NewWithTrustedCodeSHA(empty.db, strings.Repeat("a", 40))
	p := empty.preparePair(empty.initialPair())
	empty.activatePair(p)
	batch, e := empty.repo.MigrateLegacyStudyBatch(empty.ctx, 50, nil)
	if e != nil {
		t.Fatal(e)
	}
	out, e := empty.repo.ActivateTopicExperience(empty.ctx, study.CutoverInput{ExpectedPair: taxonomy.PairRef{TaxonomyVersionID: empty.v.ID, TaxonomyHead: &p.ID}, CodeSHA: strings.Repeat("a", 40), ExpectedMigrationBatchID: batch.BatchID, Reason: "Explicit empty catalogue isolated cutover.", BackupRecord: strings.Repeat("b", 64)})
	if e != nil || out.Pair == nil || out.Pair.KnowledgeHead != nil || !out.Activated {
		t.Fatal("empty knowledge pair rejected", e)
	}
}
func TestTopicCutoverRefusesMissingConstraint(t *testing.T) {
	f, in := newCutoverFixture(t)
	f.exec("ALTER TABLE study_notes DROP CONSTRAINT study_notes_pkey")
	if _, e := f.repo.ActivateTopicExperience(f.ctx, in); e == nil {
		t.Fatal("missing primary protection accepted")
	}
	assertLegacyCutover(t, f)
}
func TestTopicCutoverInspectIsReadOnly(t *testing.T) {
	f, _ := newCutoverFixture(t)
	before := f.count("SELECT count(*) FROM study_events")
	report, e := f.repo.InspectTopicCutover(f.ctx)
	if e != nil || !report.SchemaReady || !report.MigrationDone || !report.HistoryReady || !report.CodeCompatible || report.Pair == nil || f.count("SELECT count(*) FROM study_events") != before || f.count("SELECT count(*) FROM topic_cutovers") != 0 {
		t.Fatal("inspection changed facts or lied", report, e)
	}
}
