package store_test

import (
	"encoding/json"
	"github.com/yyl1212/math_master/backend/internal/publication"
	"github.com/yyl1212/math_master/backend/internal/study"
	"github.com/yyl1212/math_master/backend/internal/taxonomy"
	"testing"
	"time"
)

func (f *learningFixture) explicitLegacy(actor string) {
	f.t.Helper()
	if _, e := f.repo.StartLearning(f.ctx, f.Access(actor, false), f.knowledge.ID, f.startInput()); e != nil {
		f.t.Fatal(e)
	}
	if _, e := f.repo.CompleteLearning(f.ctx, f.Access(actor, false), f.knowledge.ID, f.completeInput()); e != nil {
		f.t.Fatal(e)
	}
}
func (f *learningFixture) migrationPair() taxonomy.PairRef {
	f.t.Helper()
	v, e := f.repo.InstallTaxonomyBatch(f.ctx, taxonomyFixtureBatch())
	if e != nil {
		f.t.Fatal(e)
	}
	pair := taxonomy.PairRef{KnowledgeHead: f.KHead(), TaxonomyVersionID: v.ID}
	p, e := f.repo.PrepareTopicRelease(f.ctx, f.Access("admin_a", false), taxonomy.PrepareInput{SubmissionIDs: []string{}, ExpectedPair: pair, Reason: "Original migration classification only."})
	if e != nil {
		f.t.Fatal(e)
	}
	if _, e = f.repo.ActivateTopicRelease(f.ctx, f.Access("admin_a", true), p.ID, taxonomy.ActivateInput{ExpectedPair: pair, ManifestSHA: p.ManifestSHA, Reason: "Original migration protected fixture."}); e != nil {
		f.t.Fatal(e)
	}
	pair.TaxonomyHead = &p.ID
	return pair
}
func TestStudyMigrationOnlyExplicitFacts(t *testing.T) {
	f := newLearningFixture(t)
	a := f.createDiagnostic("learner_a")
	if _, e := f.repo.SubmitAssessment(f.ctx, f.Access("learner_a", false), a.Summary.ID, f.answers(a, 5)); e != nil {
		t.Fatal(e)
	}
	before := f.count("SELECT count(*) FROM learning_events")
	if before != 0 {
		t.Fatal("diagnostic fabricated reading")
	}
	report, e := f.repo.MigrateLegacyStudyBatch(f.ctx, 50, nil)
	if e != nil || !report.Done || report.Processed != 0 || f.count("SELECT count(*) FROM study_records") != 0 {
		t.Fatal("passed/diagnostic/unlock imported as completion", report, e)
	}
	f.explicitLegacy("learner_b")
	report, e = f.repo.MigrateLegacyStudyBatch(f.ctx, 50, nil)
	if e != nil || report.CreatedEvents != 2 || f.count("SELECT count(*) FROM study_events WHERE kind='completed'") != 1 {
		t.Fatal("explicit completion missing", report, e)
	}
	if f.count("SELECT count(*) FROM assessment_results WHERE passed") != 1 {
		t.Fatal("original score altered")
	}
}
func TestStudyMigrationRetainsTimeAndVersion(t *testing.T) {
	f := newLearningFixture(t)
	f.explicitLegacy("learner_a")
	r, e := f.repo.MigrateLegacyStudyBatch(f.ctx, 1, nil)
	if e != nil || r.Processed != 1 || r.Done || r.Cursor == nil {
		t.Fatal("first bounded batch", r, e)
	}
	r, e = f.repo.MigrateLegacyStudyBatch(f.ctx, 1, r.Cursor)
	if e != nil || !r.Done || r.CreatedEvents != 1 {
		t.Fatal("cursor lost facts", r, e)
	}
	if f.count(`SELECT count(*) FROM study_events s JOIN learning_events e ON e.id=s.origin_event_id WHERE s.owner_user_id=e.owner_user_id AND s.knowledge_id=e.knowledge_id AND s.knowledge_version=e.knowledge_version AND s.knowledge_sha256=e.knowledge_sha256 AND s.recorded_at=e.recorded_at AND s.kind=e.kind AND s.source_kind='legacy' AND s.taxonomy_head IS NULL AND s.taxonomy_version_id IS NULL`) != 2 {
		t.Fatal("source identity/time/classification invented")
	}
}
func TestStudyMigrationRerunIdempotent(t *testing.T) {
	f := newLearningFixture(t)
	f.explicitLegacy("learner_a")
	a, e := f.repo.MigrateLegacyStudyBatch(f.ctx, 50, nil)
	if e != nil || a.CreatedEvents != 2 {
		t.Fatal(a, e)
	}
	b, e := f.repo.MigrateLegacyStudyBatch(f.ctx, 50, nil)
	if e != nil || b.CreatedEvents != 0 || b.LinkedEvents != 2 || f.count("SELECT count(*) FROM study_events") != 2 {
		t.Fatal("migration replay duplicated", b, e)
	}
}
func TestStudyMigrationExistingStudyNotOverwritten(t *testing.T) {
	f := newLearningFixture(t)
	f.explicitLegacy("learner_a")
	pair := f.migrationPair()
	a := f.Access("learner_a", false)
	access := study.Access{TokenHash: a.TokenHash, CSRF: a.CSRF, IdempotencyKey: a.IdempotencyKey, RequestID: a.RequestID}
	ref := study.KnowledgeRef{ID: f.knowledge.ID, Version: f.knowledge.Version, SHA256: f.knowledge.SHA256}
	d, e := f.repo.BeginStudy(f.ctx, access, ref.ID, study.CommandInput{Knowledge: ref, ExpectedKnowledgeHead: *pair.KnowledgeHead})
	if e != nil {
		t.Fatal(e)
	}
	a = f.Access("learner_a", false)
	access.IdempotencyKey = a.IdempotencyKey
	if _, e = f.repo.SaveStudyNote(f.ctx, access, ref.ID, study.NoteInput{Knowledge: ref, Body: "Native private note preserved."}); e != nil {
		t.Fatal(e)
	}
	before, _ := json.Marshal(d.Record)
	if _, e = f.repo.MigrateLegacyStudyBatch(f.ctx, 50, nil); e != nil {
		t.Fatal(e)
	}
	d, e = f.repo.ReadStudyKnowledge(f.ctx, access, ref.ID)
	after, _ := json.Marshal(d.Record)
	if e != nil || string(after) != string(before) {
		t.Fatal("new record overwritten", e)
	}
	note, e := f.repo.ReadStudyNote(f.ctx, access, ref.ID)
	if e != nil || note.Body != "Native private note preserved." {
		t.Fatal("new note overwritten", e)
	}
}
func TestStudyMigrationWithdrawnKnowledgeHistory(t *testing.T) {
	f := newLearningFixture(t)
	f.explicitLegacy("learner_a")
	workflowWithdraw(f.workflowFixture, publication.WithdrawalTarget{Kind: "knowledge", ID: f.knowledge.ID, Version: 1}, f.KHead())
	r, e := f.repo.MigrateLegacyStudyBatch(f.ctx, 50, nil)
	if e != nil || r.CreatedEvents != 2 || f.count("SELECT count(*) FROM study_records WHERE state='completed'") != 1 {
		t.Fatal("withdrawn history discarded", r, e)
	}
}
func TestStudyMigrationEqualTimestampCursor(t *testing.T) {
	f := newLearningFixture(t)
	f.db.SetMaxOpenConns(1)
	f.db.SetMaxIdleConns(1)
	fixed := time.Now().UTC().Add(time.Minute).Format(time.RFC3339Nano)
	f.exec("CREATE FUNCTION public.clock_timestamp() RETURNS timestamptz LANGUAGE sql AS $$ SELECT '" + fixed + "'::timestamptz $$")
	f.exec("SET search_path TO public,pg_catalog")
	f.explicitLegacy("learner_a")
	f.explicitLegacy("learner_b")
	if f.count("SELECT count(DISTINCT recorded_at) FROM learning_events") != 1 {
		t.Fatal("fixture did not tie source timestamps")
	}
	var cursor *study.LegacyCursor
	total := 0
	for n := 0; n < 4; n++ {
		r, e := f.repo.MigrateLegacyStudyBatch(f.ctx, 1, cursor)
		if e != nil {
			t.Fatal(e)
		}
		total += r.CreatedEvents
		cursor = r.Cursor
	}
	inspection, e := f.repo.InspectStudyMigration(f.ctx)
	if e != nil || !inspection.MigrationDone {
		t.Fatal("equal-clock final batch not recognized", inspection, e)
	}
	if total != 4 || f.count("SELECT count(*) FROM study_events") != 4 {
		t.Fatal("equal timestamps lost rows")
	}
}

func legacyFactsDigest(f *learningFixture) string {
	f.t.Helper()
	var digest string
	if e := f.db.QueryRow(`SELECT md5(COALESCE((SELECT string_agg(row_to_json(e)::text,'' ORDER BY e.id) FROM learning_events e),'')||COALESCE((SELECT string_agg(row_to_json(r)::text,'' ORDER BY r.attempt_id) FROM assessment_results r),'')||COALESCE((SELECT string_agg(row_to_json(a)::text,'' ORDER BY a.attempt_id,a.position) FROM assessment_answers a),''))`).Scan(&digest); e != nil {
		f.t.Fatal(e)
	}
	return digest
}
func TestStudyMigrationOriginalBytesAndInspection(t *testing.T) {
	f := newLearningFixture(t)
	f.explicitLegacy("learner_a")
	a := f.createDiagnostic("learner_b")
	if _, e := f.repo.SubmitAssessment(f.ctx, f.Access("learner_b", false), a.Summary.ID, f.answers(a, 4)); e != nil {
		t.Fatal(e)
	}
	before := legacyFactsDigest(f)
	if _, e := f.repo.MigrateLegacyStudyBatch(f.ctx, 50, nil); e != nil {
		t.Fatal(e)
	}
	if before != legacyFactsDigest(f) {
		t.Fatal("original events/answers/results changed")
	}
	report, e := f.repo.InspectStudyMigration(f.ctx)
	if e != nil || !report.SchemaReady || !report.MigrationDone || report.InvalidLinks != 0 || report.UnmappedLegacyEvents != 0 || report.MigrationBatchID == nil {
		t.Fatal("migration verification incomplete", report, e)
	}
}
func TestStudyMigrationNativeBeginBetweenBatchesWins(t *testing.T) {
	f := newLearningFixture(t)
	f.explicitLegacy("learner_a")
	pair := f.migrationPair()
	r, e := f.repo.MigrateLegacyStudyBatch(f.ctx, 1, nil)
	if e != nil {
		t.Fatal(e)
	}
	a := f.Access("learner_a", false)
	access := study.Access{TokenHash: a.TokenHash, CSRF: a.CSRF, IdempotencyKey: a.IdempotencyKey, RequestID: a.RequestID}
	ref := study.KnowledgeRef{ID: f.knowledge.ID, Version: 1, SHA256: f.knowledge.SHA256}
	native, e := f.repo.BeginStudy(f.ctx, access, ref.ID, study.CommandInput{Knowledge: ref, ExpectedKnowledgeHead: *pair.KnowledgeHead})
	if e != nil {
		t.Fatal(e)
	}
	if _, e = f.repo.MigrateLegacyStudyBatch(f.ctx, 1, r.Cursor); e != nil {
		t.Fatal(e)
	}
	after, e := f.repo.ReadStudyKnowledge(f.ctx, access, ref.ID)
	if e != nil || after.Record.State != native.Record.State || after.Record.LastReadAt == nil || !after.Record.LastReadAt.Equal(*native.Record.LastReadAt) || after.Record.Sequence != native.Record.Sequence {
		t.Fatal("semantic native begin overwritten", e)
	}
}
func TestStudyMigrationCapabilitiesAndRollback(t *testing.T) {
	f := newLearningFixture(t)
	f.explicitLegacy("learner_a")
	before := legacyFactsDigest(f)
	f.exec("ALTER TABLE study_legacy_event_links DISABLE TRIGGER study_legacy_link_guard")
	if _, e := f.repo.MigrateLegacyStudyBatch(f.ctx, 50, nil); e == nil {
		t.Fatal("disabled source guard accepted")
	}
	if f.count("SELECT count(*) FROM study_events") != 0 || f.count("SELECT count(*) FROM study_migration_batches") != 0 || before != legacyFactsDigest(f) {
		t.Fatal("partial migration despite guard failure")
	}
	f.exec("ALTER TABLE study_legacy_event_links ENABLE TRIGGER study_legacy_link_guard")
	f.exec(`CREATE FUNCTION fail_legacy_import() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'isolated migration failure'; END $$`)
	f.exec(`CREATE TRIGGER fail_legacy_import BEFORE INSERT ON study_legacy_event_links FOR EACH ROW EXECUTE FUNCTION fail_legacy_import()`)
	if _, e := f.repo.MigrateLegacyStudyBatch(f.ctx, 50, nil); e == nil {
		t.Fatal("source failure committed")
	}
	if f.count("SELECT count(*) FROM study_records") != 0 || f.count("SELECT count(*) FROM study_events") != 0 || f.count("SELECT count(*) FROM study_migration_batches") != 0 || before != legacyFactsDigest(f) {
		t.Fatal("migration not atomic")
	}
}
