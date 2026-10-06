package store_test

import (
	"github.com/yyl1212/math_master/backend/internal/study"
	"github.com/yyl1212/math_master/backend/internal/taxonomy"
	"testing"
)

type studyFixture struct {
	*topicWorkflowFixture
	pair taxonomy.PairRef
	ref  study.KnowledgeRef
}

func newStudyFixture(t *testing.T) *studyFixture {
	f := newTopicWorkflowFixture(t)
	p := f.preparePair(f.initialPair(), f.topicApproved("msc-00a00").ID)
	f.activatePair(p)
	var ref study.KnowledgeRef
	if e := f.db.QueryRow("SELECT id,version,sha256 FROM knowledge_versions WHERE id=$1 AND version=1", f.member.Knowledge.ID).Scan(&ref.ID, &ref.Version, &ref.SHA256); e != nil {
		t.Fatal(e)
	}
	return &studyFixture{f, taxonomy.PairRef{KnowledgeHead: p.KnowledgePublicationID, TaxonomyHead: &p.ID, TaxonomyVersionID: f.v.ID}, ref}
}
func (f *studyFixture) Access(name string) study.Access {
	a := f.workflowFixture.Access(name, false)
	return study.Access{TokenHash: a.TokenHash, CSRF: a.CSRF, IdempotencyKey: a.IdempotencyKey, RequestID: a.RequestID}
}
func (f *studyFixture) Ref() study.KnowledgeRef { return f.ref }
func (f *studyFixture) Pair() taxonomy.PairRef  { return f.pair }
func (f *studyFixture) Command(seq int64) study.CommandInput {
	return study.CommandInput{Knowledge: f.Ref(), ExpectedKnowledgeHead: *f.pair.KnowledgeHead, ExpectedSequence: seq}
}
func (f *studyFixture) CountEvents(name, kind string) int {
	return f.count("SELECT count(*) FROM study_events WHERE owner_user_id=$1 AND kind=$2", f.ids[name], kind)
}
func (f *studyFixture) ResetQuestionTablesForIsolation() {
	f.exec("ALTER TABLE question_heads RENAME TO disabled_old_question_heads")
	f.exec("ALTER TABLE assessment_attempts RENAME TO disabled_old_assessments")
}
