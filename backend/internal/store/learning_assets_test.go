package store_test

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"github.com/yyl1212/math_master/backend/internal/assessment"
	"github.com/yyl1212/math_master/backend/internal/auth"
	"github.com/yyl1212/math_master/backend/internal/content"
	"github.com/yyl1212/math_master/backend/internal/learning"
	"github.com/yyl1212/math_master/backend/internal/publication"
	"github.com/yyl1212/math_master/backend/internal/question"
	"strings"
	"testing"
)

func TestLearningAssetOwnedFixedReferenceAndWithdrawal(t *testing.T) {
	f := newLearningFixture(t)
	v := f.createDiagnostic("learner_a")
	sha := v.Questions[0].Assets[0].SHA256
	svg, e := f.repo.ReadLearningAsset(f.ctx, f.Access("learner_a", false), v.Summary.ID, sha)
	if e != nil || len(svg) == 0 || content.ValidateSVG(svg) != nil {
		t.Fatal("approved original SVG", e)
	}
	for _, a := range []struct{ actor, id, digest string }{{"learner_b", v.Summary.ID, sha}, {"learner_a", f.ID(), sha}, {"learner_a", v.Summary.ID, strings.Repeat("0", 64)}} {
		if _, e = f.repo.ReadLearningAsset(f.ctx, f.Access(a.actor, false), a.id, a.digest); !errors.Is(e, auth.ErrNotFound) {
			t.Fatal("asset ownership/fixed reference", e)
		}
	}
	if f.exposureSequence("learner_a") != 0 {
		t.Fatal("safe diagram exposed an answer")
	}
	f.questionInput.QuestionPackage.Version = 2
	sub := f.QApproved("author_a", "reviewer_a")
	f.QActivate(f.QPrepare(sub.ID))
	historical, e := f.repo.ReadLearningAsset(f.ctx, f.Access("learner_a", false), v.Summary.ID, sha)
	if e != nil || !bytes.Equal(svg, historical) {
		t.Fatal("ordinary publication replacement broke historical SVG", e)
	}
	workflowWithdraw(f.workflowFixture, publication.WithdrawalTarget{Kind: "asset", SHA256: sha}, f.KHead())
	if _, e = f.repo.ReadLearningAsset(f.ctx, f.Access("learner_a", false), v.Summary.ID, sha); !errors.Is(e, auth.ErrNotFound) {
		t.Fatal("permanently withdrawn asset remained accessible", e)
	}
}
func TestLearningAssetImplicitUnitDependencyNeverReactivates(t *testing.T) {
	f := newLearningFixture(t)
	p := &f.questionInput.QuestionPackage
	p.Version = 2
	p.Templates[0].Version = 2
	p.Templates[0].Assets = []question.AssetRef{}
	p.Templates[0].ExplanationTemplate = "The exact sum is {{answer}}."
	p.Blueprints[0].Version = 2
	p.Blueprints[0].Sources[0].Ref.Version = 2
	sub := f.QApproved("author_a", "reviewer_a")
	f.QActivate(f.QPrepare(sub.ID))
	f.blueprint.Version = 2
	if e := f.db.QueryRow(`SELECT sha256 FROM question_blueprints WHERE id=$1 AND version=2`, f.blueprint.ID).Scan(&f.blueprint.SHA256); e != nil {
		t.Fatal(e)
	}
	v := f.createDiagnostic("learner_a")
	if len(v.Questions[0].Assets) != 0 {
		t.Fatal("fixture did not omit direct question asset")
	}
	key := f.Access("learner_a", false)
	in := f.answers(v, 5)
	if _, e := f.repo.SubmitAssessment(f.ctx, key, v.Summary.ID, in); e != nil {
		t.Fatal(e)
	}
	oldSVG := f.Input().Package.Assets[0].SHA256
	if f.count(`SELECT count(*) FROM learning_evidence_dependencies WHERE evidence_kind='assessment' AND evidence_id=$1 AND kind='asset' AND sha256=$2`, v.Summary.ID, oldSVG) != 1 {
		t.Fatal("fixed unit's asset absent from evidence index")
	}
	workflowWithdraw(f.workflowFixture, publication.WithdrawalTarget{Kind: "asset", SHA256: oldSVG}, f.KHead())
	updated := f.Input()
	updated.Package.Version = 2
	updated.Package.Units[0].Version = 2
	updated.Package.Units[0].Angles[0].Body += " Corrected original illustration."
	svg, e := base64.StdEncoding.DecodeString(updated.AssetBytes[0].Base64)
	if e != nil {
		t.Fatal(e)
	}
	svg = bytes.Replace(svg, []byte("</svg>"), []byte("<title>Corrected original unit illustration</title></svg>"), 1)
	oldAlias := updated.Package.Assets[0].ID
	newAlias := "lf-corrected-illustration"
	updated.Package.Assets[0].ID = newAlias
	updated.AssetBytes[0].ID = newAlias
	updated.Package.Units[0].AssetIDs = []string{newAlias}
	for j := range updated.Package.Units[0].Angles {
		updated.Package.Units[0].Angles[j].Body = strings.ReplaceAll(updated.Package.Units[0].Angles[j].Body, "asset:"+oldAlias, "asset:"+newAlias)
	}
	updated.Package.Assets[0].SHA256 = fmt.Sprintf("%x", sha256.Sum256(svg))
	updated.AssetBytes[0].Base64 = base64.StdEncoding.EncodeToString(svg)
	head := f.KHead()
	approved := f.ApprovedInput(updated)
	f.Activate(f.Prepare(approved, head), head)
	d, e := f.repo.ReadLearningKnowledge(f.ctx, f.Access("learner_a", false), f.knowledge.ID, 1)
	if e != nil || d.State.Qualification != nil || d.State.State != learning.NeedsReview {
		t.Fatal("withdrawn original unit asset reactivated old proof", d, e)
	}
	r, e := f.repo.ReadAssessmentResult(f.ctx, f.Access("learner_a", false), v.Summary.ID)
	if e != nil || r.Score == nil || *r.Score != 5 || r.Validity != assessment.Restricted || r.Items[0].Explanation != nil || !containsReason(r.Reasons, assessment.AssetWithdrawn) {
		t.Fatal(r, e)
	}
}
