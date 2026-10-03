package store_test

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/yyl1212/math_master/backend/internal/assessment"
	"github.com/yyl1212/math_master/backend/internal/auth"
	"github.com/yyl1212/math_master/backend/internal/question"
	"testing"
	"time"
)

type learningFixture struct {
	*questionFixture
	knowledge, blueprint question.Identity
	approvedSubmissionID string
	items                []assessment.ItemBinding
	mode                 *assessment.Mode
}

func newLearningFixture(t *testing.T, initialMigration ...int) *learningFixture {
	t.Helper()
	f := &learningFixture{questionFixture: newQuestionFixture(t, initialMigration...)}
	template := f.questionInput.QuestionPackage.Templates[0]
	template.ID = "lf-addition"
	template.Parameters = []question.Parameter{{Name: "left", Values: []string{"1", "2", "3", "4"}}, {Name: "right", Values: []string{"1", "2", "3", "4"}}}
	template.Constraints = []question.Constraint{}
	ref := question.Ref{ID: "workflow-fractions", Version: 1}
	template.Coverage = []question.ObjectiveCoverage{{Knowledge: ref, ObjectiveIndices: []int{0}}}
	f.questionInput.QuestionPackage.Templates = []question.Template{template}
	f.questionInput.QuestionPackage.FixedQuestions = []question.FixedQuestion{}
	f.questionInput.QuestionPackage.Blueprints = []question.Blueprint{{ID: "lf-five", Version: 1, Knowledge: ref, CoreObjectiveIndices: []int{0}, Sources: []question.BlueprintSource{{Kind: "template", Ref: question.Ref{ID: "lf-addition", Version: 1}}}, CoverageNote: "Original finite addition cases cover objective zero.", RuleVersion: 1, QuestionCount: 5, PassCount: 4}}
	sub := f.QApproved("author_a", "reviewer_a")
	f.approvedSubmissionID = sub.ID
	f.QActivate(f.QPrepare(sub.ID))
	for _, name := range []string{"learner_a", "learner_b"} {
		u, cookies, csrf := f.signup(name)
		proof, e := auth.DecodeContentProof(cookies, csrf, true)
		if e != nil {
			t.Fatal(e)
		}
		f.ids[name] = u.ID
		f.access[name] = question.Access{TokenHash: proof.TokenHash, CSRF: proof.CSRF, RequestID: "learning-fixture"}
	}
	f.knowledge = question.Identity{ID: ref.ID, Version: 1}
	if e := f.db.QueryRow(`SELECT sha256 FROM knowledge_versions WHERE id=$1 AND version=1`, ref.ID).Scan(&f.knowledge.SHA256); e != nil {
		t.Fatal(e)
	}
	f.blueprint = question.Identity{ID: "lf-five", Version: 1}
	if e := f.db.QueryRow(`SELECT sha256 FROM question_blueprints WHERE id=$1 AND version=1`, f.blueprint.ID).Scan(&f.blueprint.SHA256); e != nil {
		t.Fatal(e)
	}
	rows, e := f.db.Query(`SELECT i.id,i.version,i.sha256,i.body->'body',m.evidence FROM question_instances i JOIN question_publication_members m ON m.kind='instance' AND m.id=i.id AND m.version=i.version AND m.sha256=i.sha256 WHERE m.publication_id=$1 ORDER BY i.id LIMIT 5`, *f.QHead())
	if e != nil {
		t.Fatal(e)
	}
	defer rows.Close()
	var unit question.Identity
	unit.ID = template.Units[0].ID
	unit.Version = template.Units[0].Version
	if e = f.db.QueryRow(`SELECT sha256 FROM unit_versions WHERE id=$1 AND version=$2`, unit.ID, unit.Version).Scan(&unit.SHA256); e != nil {
		t.Fatal(e)
	}
	for rows.Next() {
		var id question.Identity
		var raw, evidence []byte
		var i question.Instance
		var approval question.MemberEvidence
		if e = rows.Scan(&id.ID, &id.Version, &id.SHA256, &raw, &evidence); e != nil {
			t.Fatal(e)
		}
		if json.Unmarshal(raw, &i) != nil || json.Unmarshal(evidence, &approval) != nil {
			t.Fatal("invalid fixed source")
		}
		f.items = append(f.items, assessment.ItemBinding{Position: len(f.items) + 1, Instance: id, Template: i.Template, Coverage: []int{0}, Units: []question.Identity{unit}, Assets: i.Body.Assets, Approval: approval})
	}
	if e = rows.Err(); e != nil || len(f.items) != 5 {
		t.Fatal("fixed items", e)
	}
	return f
}
func (f *learningFixture) seal(kind string) assessment.Seal {
	s := assessment.Seal{Kind: kind, Knowledge: f.knowledge, KnowledgePublicationID: *f.KHead(), QuestionPublicationID: *f.QHead(), RuleVersion: 1, Seed: fmt.Sprintf("%064x", 1), Items: f.items, Core: []int{0}}
	if kind == "assessment" {
		mode := assessment.ModeDiagnostic
		if f.mode != nil {
			mode = *f.mode
		}
		s.Mode = &mode
		s.Blueprint = &f.blueprint
	} else {
		s.Items = f.items[:1]
		s.Core = []int{}
	}
	return s
}
func (f *learningFixture) insertAssessmentBase(tx *sql.Tx, id, owner string, n int) error {
	raw, h, e := assessment.CanonicalSeal(f.seal("assessment"))
	if e != nil {
		return e
	}
	now := time.Now().UTC()
	_, e = tx.Exec(`INSERT INTO assessment_attempts(id,owner_user_id,knowledge_id,knowledge_version,knowledge_sha256,knowledge_publication_id,question_publication_id,blueprint_id,blueprint_version,blueprint_sha256,mode,rule_version,core,seed,seal,seal_bytes,seal_sha256,created_at,expires_at,creation_exposure_sequence) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$17,1,'[0]',$11,$12,$13,$14,$15,$16,coalesce((SELECT sequence FROM learner_exposure_state WHERE owner_user_id=$2),0))`, id, owner, f.knowledge.ID, f.knowledge.Version, f.knowledge.SHA256, *f.KHead(), *f.QHead(), f.blueprint.ID, f.blueprint.Version, f.blueprint.SHA256, f.seal("assessment").Seed, string(raw), raw, h, now, now.Add(24*time.Hour), string(*f.seal("assessment").Mode))
	if e != nil {
		return e
	}
	for _, i := range f.items[:n] {
		b, _ := json.Marshal(i)
		if _, e = tx.Exec(`INSERT INTO assessment_items(attempt_id,position,instance_id,instance_version,instance_sha256,binding) VALUES($1,$2,$3,$4,$5,$6)`, id, i.Position, i.Instance.ID, i.Instance.Version, i.Instance.SHA256, string(b)); e != nil {
			return e
		}
	}
	_, e = tx.Exec(`UPDATE assessment_attempts SET sealed=true WHERE id=$1`, id)
	return e
}
func (f *learningFixture) insertTwoActiveAssessments() error {
	tx, e := f.db.Begin()
	if e != nil {
		return e
	}
	defer tx.Rollback()
	for j := 0; j < 2; j++ {
		if e = f.insertAssessment(tx, f.ID(), f.ids["learner_a"], 5); e != nil {
			return e
		}
	}
	return tx.Commit()
}
func (f *learningFixture) learningTableCount() int {
	return f.count(`SELECT count(*) FROM information_schema.tables WHERE table_schema='public' AND table_name=ANY($1::text[])`, []string{"learning_records", "learning_events", "learning_path_enrollments", "learning_path_nodes", "learning_unlocks", "learning_qualification_events", "practice_attempts", "assessment_attempts", "assessment_items", "assessment_answers", "assessment_results", "learning_evidence_dependencies", "learner_answer_exposures", "learner_exposure_state", "learner_question_views", "learning_idempotency"})
}

func (f *learningFixture) insertDependencies(tx *sql.Tx, id, owner, kind string, s assessment.Seal) error {
	type ref struct {
		kind, id string
		v        *int
		sha      string
	}
	version := func(v int) *int { return &v }
	refs := []ref{{"knowledge", s.Knowledge.ID, version(s.Knowledge.Version), s.Knowledge.SHA256}}
	if kind != "learning-event" && s.Blueprint != nil {
		refs = append(refs, ref{"blueprint", s.Blueprint.ID, version(s.Blueprint.Version), s.Blueprint.SHA256})
	}
	for _, i := range s.Items {

		if kind != "learning-event" {
			refs = append(refs, ref{"instance", i.Instance.ID, version(i.Instance.Version), i.Instance.SHA256})
			if i.Template != nil {
				refs = append(refs, ref{"template", i.Template.ID, version(i.Template.Version), i.Template.SHA256})
			}
		}
		for _, u := range i.Units {
			refs = append(refs, ref{"unit", u.ID, version(u.Version), u.SHA256})
		}
		for _, a := range i.Assets {
			refs = append(refs, ref{"asset", a.ID, nil, a.SHA256})
		}
	}
	for _, r := range refs {
		if _, e := tx.Exec(`INSERT INTO learning_evidence_dependencies(evidence_kind,evidence_id,owner_user_id,kind,id,version,sha256) VALUES($1,$2,$3,$4,$5,$6,$7) ON CONFLICT DO NOTHING`, kind, id, owner, r.kind, r.id, r.v, r.sha); e != nil {
			return e
		}
	}
	return nil
}
func (f *learningFixture) insertAssessment(tx *sql.Tx, id, owner string, n int) error {
	if e := f.insertAssessmentBase(tx, id, owner, n); e != nil {
		return e
	}
	return f.insertDependencies(tx, id, owner, "assessment", f.seal("assessment"))
}
