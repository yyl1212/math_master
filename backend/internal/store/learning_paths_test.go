package store_test

import (
	"encoding/json"
	"errors"
	"github.com/yyl1212/math_master/backend/internal/assessment"
	"github.com/yyl1212/math_master/backend/internal/content"
	"github.com/yyl1212/math_master/backend/internal/learning"
	"github.com/yyl1212/math_master/backend/internal/publication"
	"github.com/yyl1212/math_master/backend/internal/question"
	"testing"
)

func learningGraphInput(f *learningFixture) publication.DraftInput {
	v := workflowWithdrawalInput(f.workflowFixture)
	v.Package.ID = "lf-graph-package"
	target := v.Package.Knowledge[1]
	target.ID = "lf-target"
	target.Relations = []content.Relation{{Kind: "prerequisite", Target: content.VersionRef{ID: "workflow-dependent", Version: 1}}, {Kind: "prerequisite", Target: content.VersionRef{ID: f.knowledge.ID, Version: 1}}}
	v.Package.Knowledge = append(v.Package.Knowledge, target)
	v.Package.Units = append(v.Package.Units, content.Unit{ID: "lf-target-unit", Version: 1, Knowledge: content.VersionRef{ID: target.ID, Version: 1}, Angles: []content.Angle{{Kind: "formal", Body: "Original formal target explanation."}, {Kind: "intuitive", Body: "Original intuitive target explanation."}}, Examples: []string{"An original target example."}, Counterexamples: []string{}, AssetIDs: []string{}})
	v.Package.Paths[0].Nodes = append(v.Package.Paths[0].Nodes, content.VersionRef{ID: target.ID, Version: 1})

	return v
}
func (f *learningFixture) publishLearningGraph() question.Identity {
	f.t.Helper()
	v := learningGraphInput(f)
	old := f.KHead()
	sub := f.ApprovedInput(v)
	f.Activate(f.Prepare(sub, old), old)
	p := &f.questionInput.QuestionPackage
	p.Version = 2
	for _, id := range []string{"workflow-dependent", "lf-target"} {
		t := p.Templates[0]
		t.ID = "lf-" + id
		t.Knowledge = question.Ref{ID: id, Version: 1}
		t.Coverage = []question.ObjectiveCoverage{{Knowledge: t.Knowledge, ObjectiveIndices: []int{0}}}
		t.Units = []question.Ref{{ID: id + "-unit", Version: 1}}
		t.Assets = []question.AssetRef{}
		p.Templates = append(p.Templates, t)
		bp := p.Blueprints[0]
		bp.ID = "lf-" + id + "-five"
		bp.Knowledge = t.Knowledge
		bp.Sources = []question.BlueprintSource{{Kind: "template", Ref: question.Ref{ID: t.ID, Version: 1}}}
		p.Blueprints = append(p.Blueprints, bp)
	}
	qsub := f.QApproved("author_a", "reviewer_a")
	f.QActivate(f.QPrepare(qsub.ID))
	f.setLearningTarget("workflow-fractions", "lf-five", 0)
	var path question.Identity
	path.ID = "workflow-path"
	path.Version = 1
	if e := f.db.QueryRow(`SELECT sha256 FROM path_versions WHERE id=$1 AND version=1`, path.ID).Scan(&path.SHA256); e != nil {
		f.t.Fatal(e)
	}
	return path
}
func (f *learningFixture) setLearningTarget(kid, bpid string, offset int) {
	f.t.Helper()
	f.knowledge = question.Identity{ID: kid, Version: 1}
	f.blueprint = question.Identity{ID: bpid, Version: 1}
	if e := f.db.QueryRow(`SELECT sha256 FROM knowledge_versions WHERE id=$1 AND version=1`, kid).Scan(&f.knowledge.SHA256); e != nil {
		f.t.Fatal(e)
	}
	if e := f.db.QueryRow(`SELECT sha256 FROM question_blueprints WHERE id=$1 AND version=1`, bpid).Scan(&f.blueprint.SHA256); e != nil {
		f.t.Fatal(e)
	}
	rows, e := f.db.Query(`SELECT i.id,i.version,i.sha256,i.body->'body',m.evidence FROM question_instances i JOIN question_publication_members m ON m.kind='instance' AND m.id=i.id AND m.version=i.version AND m.sha256=i.sha256 WHERE m.publication_id=$1 AND i.knowledge_id=$2 AND i.knowledge_version=1 ORDER BY i.id LIMIT 5 OFFSET $3`, *f.QHead(), kid, offset)
	if e != nil {
		f.t.Fatal(e)
	}
	f.items = []assessment.ItemBinding{}
	for rows.Next() {
		var id question.Identity
		var raw, proof []byte
		var i question.Instance
		var a question.MemberEvidence
		if e = rows.Scan(&id.ID, &id.Version, &id.SHA256, &raw, &proof); e != nil {
			f.t.Fatal(e)
		}
		if json.Unmarshal(raw, &i) != nil || json.Unmarshal(proof, &a) != nil {
			f.t.Fatal("fixed target decode")
		}
		binding := assessment.ItemBinding{Position: len(f.items) + 1, Instance: id, Template: i.Template, Coverage: []int{0}, Assets: i.Body.Assets, Approval: a, Units: []question.Identity{}}
		for _, r := range i.Body.Units {
			u := question.Identity{ID: r.ID, Version: r.Version}
			if e = f.db.QueryRow(`SELECT sha256 FROM unit_versions WHERE id=$1 AND version=$2`, r.ID, r.Version).Scan(&u.SHA256); e != nil {
				f.t.Fatal(e)
			}
			binding.Units = append(binding.Units, u)
		}
		f.items = append(f.items, binding)
	}
	e = rows.Err()
	rows.Close()
	if e != nil || len(f.items) != 5 {
		f.t.Fatal("actual target has no fixed five", e)
	}
}
func TestLearningPathsAllPrerequisitesAndFixedDenominator(t *testing.T) {
	f := newLearningFixture(t)
	path := f.publishLearningGraph()
	key := f.Access("learner_a", false)
	in := learning.EnrollInput{Path: path, ExpectedKnowledgeHead: *f.KHead()}
	joined, e := f.repo.EnrollLearningPath(f.ctx, key, path.ID, in)
	if e != nil || joined.Summary.TotalNodes != 3 || joined.Summary.UnlockedNodes != 1 || joined.Summary.CompletedNodes != 0 {
		t.Fatal(joined, e)
	}
	f.setLearningTarget("workflow-dependent", "lf-workflow-dependent-five", 0)
	if _, e = f.repo.StartLearning(f.ctx, f.Access("learner_a", false), f.knowledge.ID, f.startInput()); !errors.Is(e, learning.ErrPrerequisitesUnmet) {
		t.Fatal("unqualified parent entered middle", e)
	}
	f.setLearningTarget("workflow-fractions", "lf-five", 0)
	root := f.passedFixture("learner_a", assessment.ModeDiagnostic)
	if _, e = f.repo.LearningApplyForTest(f.ctx, f.Access("learner_a", false), root); e != nil {
		t.Fatal(e)
	}
	f.setLearningTarget("lf-target", "lf-lf-target-five", 0)
	if _, e = f.repo.StartLearning(f.ctx, f.Access("learner_a", false), f.knowledge.ID, f.startInput()); !errors.Is(e, learning.ErrPrerequisitesUnmet) {
		t.Fatal("one of two prerequisites was enough", e)
	}
	target := f.passedFixture("learner_b", assessment.ModeDiagnostic)
	if _, e = f.repo.LearningApplyForTest(f.ctx, f.Access("learner_b", false), target); e != nil {
		t.Fatal(e)
	}
	b, e := f.repo.EnrollLearningPath(f.ctx, f.Access("learner_b", false), path.ID, in)
	if e != nil || b.Summary.PassedNodes != 1 || b.Summary.CompletedNodes != 0 || b.Summary.UnlockedNodes != 2 {
		t.Fatal("target diagnosis fabricated ancestor knowledge", b, e)
	}
	before := f.count(`SELECT count(*) FROM learning_path_nodes`)
	workflowWithdraw(f.workflowFixture, publication.WithdrawalTarget{Kind: "knowledge", ID: "workflow-dependent", Version: 1}, f.KHead())
	old, e := f.repo.EnrollLearningPath(f.ctx, key, path.ID, in)
	if e != nil || old.Summary.ID != joined.Summary.ID || old.Summary.TotalNodes != 3 || f.count(`SELECT count(*) FROM learning_path_nodes`) != before {
		t.Fatal("withdrawal shrank old route", old, e)
	}
}

func TestLearningPathsExplicitNewVersion(t *testing.T) {
	f := newLearningFixture(t)
	path := f.publishLearningGraph()
	key := f.Access("learner_a", false)
	in := learning.EnrollInput{Path: path, ExpectedKnowledgeHead: *f.KHead()}
	first, e := f.repo.EnrollLearningPath(f.ctx, key, path.ID, in)
	if e != nil {
		t.Fatal(e)
	}
	v := learningGraphInput(f)
	v.Package.Version = 2
	v.Package.Paths[0].Version = 2
	v.Package.Paths[0].Nodes = append(v.Package.Paths[0].Nodes, content.VersionRef{ID: "workflow-related", Version: 1})
	old := f.KHead()
	sub := f.ApprovedInput(v)
	f.Activate(f.Prepare(sub, old), old)
	unchanged, e := f.repo.EnrollLearningPath(f.ctx, key, path.ID, in)
	if e != nil || unchanged.Summary.ID != first.Summary.ID || unchanged.Summary.TotalNodes != 3 || !unchanged.Summary.NewVersionAvailable {
		t.Fatal("old route silently upgraded", unchanged, e)
	}
	path.Version = 2
	if e = f.db.QueryRow(`SELECT sha256 FROM path_versions WHERE id=$1 AND version=2`, path.ID).Scan(&path.SHA256); e != nil {
		t.Fatal(e)
	}
	next, e := f.repo.EnrollLearningPath(f.ctx, f.Access("learner_a", false), path.ID, learning.EnrollInput{Path: path, ExpectedKnowledgeHead: *f.KHead()})
	if e != nil || next.Summary.ID == first.Summary.ID || next.Summary.TotalNodes != 4 || f.count(`SELECT count(*) FROM learning_path_nodes`) != 7 {
		t.Fatal("explicit new version did not produce another fixed route", next, e)
	}
}

func TestLearningPathsCountsReadingAndPassSeparately(t *testing.T) {
	f := newLearningFixture(t)
	path := f.publishLearningGraph()
	f.passedFixture("learner_a", assessment.ModeNode)
	mode := assessment.ModeReview
	f.mode = &mode
	f.submitSkippedFixture("learner_a")
	joined, e := f.repo.EnrollLearningPath(f.ctx, f.Access("learner_a", false), path.ID, learning.EnrollInput{Path: path, ExpectedKnowledgeHead: *f.KHead()})
	if e != nil || joined.Summary.PassedNodes != 1 || joined.Summary.CompletedNodes != 0 || joined.Summary.UnlockedNodes != 1 {
		t.Fatal("valid pass was confused with completion or latest review state", joined, e)
	}
}
