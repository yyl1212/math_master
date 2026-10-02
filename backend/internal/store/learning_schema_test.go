package store_test

import (
	"database/sql"
	"encoding/json"
	"github.com/yyl1212/math_master/backend/internal/assessment"
	"github.com/yyl1212/math_master/backend/internal/content"
	"github.com/yyl1212/math_master/backend/internal/learning"
	"github.com/yyl1212/math_master/backend/internal/question"
	"strings"
	"testing"
	"time"
)

func TestLearningSchemaTableCount(t *testing.T) {
	f := newLearningFixture(t)
	if f.learningTableCount() != 16 {
		t.Fatal("expected 16 learning tables")
	}
	if n := f.count(`SELECT count(*) FROM question_instances`); n != 16 {
		t.Fatal("fixture must really publish 16", n)
	}
}
func TestLearningSchemaActiveUnique(t *testing.T) {
	f := newLearningFixture(t)
	if f.insertTwoActiveAssessments() == nil {
		t.Fatal("must reject second active")
	}
}
func TestLearningSchemaFixedFiveAndSeal(t *testing.T) {
	f := newLearningFixture(t)
	tx, e := f.db.Begin()
	if e != nil {
		t.Fatal(e)
	}
	id := f.ID()
	if e = f.insertAssessment(tx, id, f.ids["learner_a"], 5); e != nil {
		tx.Rollback()
		t.Fatal("valid sealed source rejected", e)
	}
	if e = tx.Commit(); e != nil {
		t.Fatal(e)
	}
	reject := func(q string, args ...any) {
		t.Helper()
		if _, e = f.db.Exec(q, args...); e == nil {
			t.Fatal("invalid change accepted", q)
		}
	}
	reject(`UPDATE assessment_attempts SET seal_sha256=$2 WHERE id=$1`, id, strings.Repeat("b", 64))
	reject(`UPDATE assessment_items SET position=4 WHERE attempt_id=$1 AND position=5`, id)
	reject(`DELETE FROM assessment_items WHERE attempt_id=$1 AND position=5`, id)
	reject(`UPDATE assessment_attempts SET state='submitted',terminal_at=clock_timestamp(),submission_exposure_sequence=0 WHERE id=$1`, id)
	reject(`UPDATE assessment_attempts SET state='active',created_at=created_at+interval '1 hour' WHERE id=$1`, id)
	reject(`INSERT INTO assessment_answers(attempt_id,owner_user_id,position,instance_id,instance_version,instance_sha256,answer) VALUES($1,$2,1,$3,1,$4,'{"kind":"skipped"}')`, id, f.ids["learner_b"], f.items[0].Instance.ID, f.items[0].Instance.SHA256)
	f.exec(`UPDATE assessment_attempts SET state='abandoned',terminal_at=clock_timestamp() WHERE id=$1`, id)
	reject(`UPDATE assessment_attempts SET state='active',terminal_at=NULL WHERE id=$1`, id)
	for _, n := range []int{4, 5} {
		tx, e = f.db.Begin()
		if e != nil {
			t.Fatal(e)
		}
		owner := f.ids["learner_b"]
		if n == 5 {
			owner = "99999999-9999-4999-8999-999999999999"
		}
		err := f.insertAssessment(tx, f.ID(), owner, n)
		if err == nil {
			err = tx.Commit()
		} else {
			tx.Rollback()
		}
		if err == nil {
			t.Fatal("incomplete or foreign owner committed", n)
		}
	}
}
func TestLearningSchemaNullableResultAndExposureCounter(t *testing.T) {
	f := newLearningFixture(t)
	owner := f.ids["learner_a"]
	f.exec(`INSERT INTO learner_exposure_state(owner_user_id,sequence) VALUES($1,3)`, owner)
	if _, e := f.db.Exec(`UPDATE learner_exposure_state SET sequence=2 WHERE owner_user_id=$1`, owner); e == nil {
		t.Fatal("sequence went backwards")
	}
	// A prepublication identity must be recordable without an existing question FK.
	f.exec(`INSERT INTO learner_answer_exposures(owner_user_id,kind,id,version,sha256,sequence,exposed_at) VALUES($1,'template','future-original-template',1,$2,3,clock_timestamp())`, owner, strings.Repeat("a", 64))
	if _, e := f.db.Exec(`UPDATE learner_answer_exposures SET sequence=2 WHERE owner_user_id=$1`, owner); e == nil {
		t.Fatal("last sequence went backwards")
	}
	tx, e := f.db.Begin()
	if e != nil {
		t.Fatal(e)
	}
	defer tx.Rollback()
	id := f.ID()
	if e = f.insertAssessment(tx, id, owner, 5); e != nil {
		t.Fatal(e)
	}
	for _, i := range f.items {
		answer, _ := json.Marshal(assessment.Answer{Kind: "skipped"})
		if _, e = tx.Exec(`INSERT INTO assessment_answers(attempt_id,owner_user_id,position,instance_id,instance_version,instance_sha256,answer) VALUES($1,$2,$3,$4,$5,$6,$7)`, id, owner, i.Position, i.Instance.ID, i.Instance.Version, i.Instance.SHA256, string(answer)); e != nil {
			t.Fatal(e)
		}
	}
	_, e = tx.Exec(`INSERT INTO assessment_results(attempt_id,owner_user_id,rule_version,outcome,score,passed,original_reasons,progress,submitted_at) VALUES($1,$2,1,'affected',4,true,'[]','{}',clock_timestamp())`, id, owner)
	if e == nil {
		t.Fatal("affected result cannot carry a score")
	}
}
func TestLearningSchemaPracticeGlobalUnique(t *testing.T) {
	f := newLearningFixture(t)
	s := f.seal("practice")
	raw, sha, _ := assessment.CanonicalSeal(s)
	insert := func(tx *sql.Tx) error {
		_, e := tx.Exec(`INSERT INTO practice_attempts(id,owner_user_id,knowledge_id,knowledge_version,knowledge_sha256,knowledge_publication_id,question_publication_id,seal,seal_bytes,seal_sha256,created_at,expires_at) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,statement_timestamp(),statement_timestamp()+interval '24 hours')`, f.ID(), f.ids["learner_a"], s.Knowledge.ID, s.Knowledge.Version, s.Knowledge.SHA256, s.KnowledgePublicationID, s.QuestionPublicationID, string(raw), raw, sha)

		if e != nil {
			return e
		}
		return f.insertDependencies(tx, (func() string {
			var id string
			tx.QueryRow(`SELECT id::text FROM practice_attempts WHERE owner_user_id=$1 AND state='active'`, f.ids["learner_a"]).Scan(&id)
			return id
		})(), f.ids["learner_a"], "practice", s)
	}
	tx, e := f.db.Begin()
	if e != nil {
		t.Fatal(e)
	}
	defer tx.Rollback()
	if e = insert(tx); e != nil {
		t.Fatal(e)
	}
	if e = tx.Commit(); e != nil {
		t.Fatal("valid practice source", e)
	}
	tx, e = f.db.Begin()
	if e != nil {
		t.Fatal(e)
	}
	defer tx.Rollback()
	if e = insert(tx); e == nil {
		t.Fatal("two active practice attempts accepted")
	}
}

func TestLearningSchemaEventsAndRoutesImmutable(t *testing.T) {
	f := newLearningFixture(t)
	owner := f.ids["learner_a"]
	now := time.Now().UTC()
	id := f.ID()
	event := learning.EventSeal{ID: id, ActorID: owner, Knowledge: f.knowledge, KnowledgePublicationID: *f.KHead(), Units: f.items[0].Units, Assets: f.items[0].Assets, Kind: "started", RecordedAt: now}
	raw, sha, e := learning.CanonicalLearningEvent(event)
	if e != nil {
		t.Fatal(e)
	}
	tx, e := f.db.Begin()
	if e != nil {
		t.Fatal(e)
	}
	defer tx.Rollback()
	if _, e = tx.Exec(`INSERT INTO learning_events(id,owner_user_id,knowledge_id,knowledge_version,knowledge_sha256,knowledge_publication_id,kind,seal,seal_bytes,seal_sha256,recorded_at) VALUES($1,$2,$3,$4,$5,$6,'started',$7,$8,$9,$10)`, id, owner, f.knowledge.ID, 1, f.knowledge.SHA256, *f.KHead(), string(raw), raw, sha, now); e != nil {
		t.Fatal(e)
	}
	if _, e = tx.Exec(`INSERT INTO learning_records(owner_user_id,knowledge_id,knowledge_version,knowledge_sha256,started_event_id,started_at) VALUES($1,$2,1,$3,$4,$5)`, owner, f.knowledge.ID, f.knowledge.SHA256, id, now); e != nil {
		t.Fatal(e)
	}

	if e = f.insertDependencies(tx, id, owner, "learning-event", f.seal("practice")); e != nil {
		t.Fatal(e)
	}
	if e = tx.Commit(); e != nil {
		t.Fatal("valid learning event", e)
	}
	reject := func(q string, args ...any) {
		t.Helper()
		if _, e = f.db.Exec(q, args...); e == nil {
			t.Fatal("immutable evidence changed", q)
		}
	}
	reject(`UPDATE learning_records SET started_at=started_at+interval '1 second' WHERE owner_user_id=$1`, owner)
	reject(`UPDATE learning_events SET kind='completed' WHERE id=$1`, id)
	reject(`DELETE FROM learning_events WHERE id=$1`, id)
	// The base workflow fixture publishes no route. Publish an original route through the real review flow.
	v := f.Input()
	v.Package.ID = "lf-route-package"
	v.Package.Paths = []content.Path{{ID: "lf-route", Version: 1, DomainIDs: v.Package.Knowledge[0].DomainIDs, Title: "Original one-node route", TitleZh: "原创单节点路线", Nodes: []content.VersionRef{{ID: f.knowledge.ID, Version: 1}}}}
	sub := f.ApprovedInput(v)
	head := f.KHead()
	f.Activate(f.Prepare(sub, head), head)
	var pathID, pathSHA string
	var version, total int
	if e = f.db.QueryRow(`SELECT m.id,m.version,p.sha256,(SELECT count(*) FROM path_nodes n WHERE n.path_id=m.id AND n.path_version=m.version) FROM publication_members m JOIN path_versions p ON p.id=m.id AND p.version=m.version WHERE m.snapshot_id=$1 AND m.kind='path' AND m.availability='active' LIMIT 1`, *f.KHead()).Scan(&pathID, &version, &pathSHA, &total); e != nil {
		t.Fatal(e)
	}
	tx, e = f.db.Begin()
	if e != nil {
		t.Fatal(e)
	}
	defer tx.Rollback()
	enrollment := f.ID()
	if _, e = tx.Exec(`INSERT INTO learning_path_enrollments(id,owner_user_id,path_id,path_version,path_sha256,knowledge_publication_id,total_nodes) VALUES($1,$2,$3,$4,$5,$6,$7)`, enrollment, owner, pathID, version, pathSHA, *f.KHead(), total); e != nil {
		t.Fatal(e)
	}
	if _, e = tx.Exec(`INSERT INTO learning_path_nodes(enrollment_id,position,knowledge_id,knowledge_version,knowledge_sha256) SELECT $1,n.position,k.id,k.version,k.sha256 FROM path_nodes n JOIN knowledge_versions k ON k.id=n.knowledge_id AND k.version=n.knowledge_version WHERE n.path_id=$2 AND n.path_version=$3`, enrollment, pathID, version); e != nil {
		t.Fatal(e)
	}
	if e = tx.Commit(); e != nil {
		t.Fatal("valid fixed route", e)
	}
	reject(`UPDATE learning_path_enrollments SET total_nodes=total_nodes+1 WHERE id=$1`, enrollment)
	reject(`DELETE FROM learning_path_nodes WHERE enrollment_id=$1`, enrollment)
}

func TestLearningSchemaResultProgressSealedBeforeCommit(t *testing.T) {
	f := newLearningFixture(t)
	owner := f.ids["learner_a"]
	tx, e := f.db.Begin()
	if e != nil {
		t.Fatal(e)
	}
	defer tx.Rollback()
	id := f.ID()
	if e = f.insertAssessment(tx, id, owner, 5); e != nil {
		t.Fatal(e)
	}
	for _, i := range f.items {
		if _, e = tx.Exec(`INSERT INTO assessment_answers(attempt_id,owner_user_id,position,instance_id,instance_version,instance_sha256,answer) VALUES($1,$2,$3,$4,$5,$6,'{"kind":"skipped"}')`, id, owner, i.Position, i.Instance.ID, i.Instance.Version, i.Instance.SHA256); e != nil {
			t.Fatal(e)
		}
	}
	var now time.Time
	if e = tx.QueryRow(`SELECT clock_timestamp()`).Scan(&now); e != nil {
		t.Fatal(e)
	}
	progress, _ := json.Marshal(assessment.ProgressUpdate{Knowledge: f.knowledge, NewlyUnlocked: []question.Identity{}})
	if _, e = tx.Exec(`INSERT INTO assessment_results(attempt_id,owner_user_id,rule_version,outcome,score,passed,original_reasons,progress,submitted_at,sealed) VALUES($1,$2,1,'failed',0,false,'[]',$3,$4,false)`, id, owner, string(progress), now); e != nil {
		t.Fatal("persist original result", e)
	}
	if _, e = tx.Exec(`UPDATE assessment_results SET progress=$2,sealed=true WHERE attempt_id=$1`, id, string(progress)); e != nil {
		t.Fatal("finalize atomic progress", e)
	}
	if _, e = tx.Exec(`UPDATE assessment_attempts SET state='submitted',terminal_at=$2,submission_exposure_sequence=0 WHERE id=$1`, id, now); e != nil {
		t.Fatal(e)
	}
	if e = tx.Commit(); e != nil {
		t.Fatal("complete sealed result", e)
	}
	if _, e = f.db.Exec(`UPDATE assessment_results SET score=1 WHERE attempt_id=$1`, id); e == nil {
		t.Fatal("fixed original score changed")
	}
}

func TestLearningSchemaMissingDependenciesRejected(t *testing.T) {
	f := newLearningFixture(t)
	tx, e := f.db.Begin()
	if e != nil {
		t.Fatal(e)
	}
	defer tx.Rollback()
	if e = f.insertAssessmentBase(tx, f.ID(), f.ids["learner_a"], 5); e != nil {
		t.Fatal("valid mathematical seal", e)
	}
	if e = tx.Commit(); e == nil {
		t.Fatal("sealed five-question evidence committed without its complete restriction index")
	}
}
