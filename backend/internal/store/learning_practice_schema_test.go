package store_test

import (
	"github.com/yyl1212/math_master/backend/internal/assessment"
	"testing"
)

func TestLearningSchemaPracticeTerminalImmutable(t *testing.T) {
	f := newLearningFixture(t)
	s := f.seal("practice")
	raw, sha, _ := assessment.CanonicalSeal(s)
	id := f.ID()
	tx, err := f.db.Begin()
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	if _, err = tx.Exec(`INSERT INTO practice_attempts(id,owner_user_id,knowledge_id,knowledge_version,knowledge_sha256,knowledge_publication_id,question_publication_id,seal,seal_bytes,seal_sha256,created_at,expires_at) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,statement_timestamp(),statement_timestamp()+interval '24 hours')`, id, f.ids["learner_a"], s.Knowledge.ID, s.Knowledge.Version, s.Knowledge.SHA256, s.KnowledgePublicationID, s.QuestionPublicationID, string(raw), raw, sha); err != nil {
		t.Fatal(err)
	}
	if err = f.insertDependencies(tx, id, f.ids["learner_a"], "practice", s); err != nil {
		t.Fatal(err)
	}
	if err = tx.Commit(); err != nil {
		t.Fatal(err)
	}
	if _, err := f.db.Exec(`UPDATE practice_attempts SET state='revealed',terminal_at=clock_timestamp() WHERE id=$1`, id); err != nil {
		t.Fatal("valid explicit reveal rejected", err)
	}
	if _, err := f.db.Exec(`UPDATE practice_attempts SET state='active',terminal_at=NULL WHERE id=$1`, id); err == nil {
		t.Fatal("terminal practice reopened")
	}
}
