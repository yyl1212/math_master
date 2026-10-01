package store_test

import (
	"errors"
	"fmt"
	"github.com/yyl1212/math_master/backend/internal/auth"
	"github.com/yyl1212/math_master/backend/internal/publication"
	"github.com/yyl1212/math_master/backend/internal/question"
	"testing"
)

func (f *questionFixture) QWithdraw(target question.WithdrawalTarget) question.WithdrawalResult {
	f.t.Helper()
	out, err := f.repo.WithdrawQuestionVersion(f.ctx, f.Access("admin_a", true), question.WithdrawalInput{Target: target, ExpectedKnowledgeHead: f.KHead(), ExpectedQuestionHead: f.QHead(), Reason: "Permanently withdraw this exact mathematical version in an isolated test."})
	if err != nil {
		f.t.Fatal(err)
	}
	return out
}
func singlePoolQuestionFixture(t *testing.T) *questionFixture {
	f := newQuestionFixture(t)
	p := &f.questionInput.QuestionPackage
	p.ID = "single-pool-bank"
	p.Templates = p.Templates[:1]
	p.Templates[0].Parameters[1].Values = p.Templates[0].Parameters[1].Values[:1]
	p.Blueprints[0].Sources = []question.BlueprintSource{{Kind: "template", Ref: question.Ref{ID: p.Templates[0].ID, Version: 1}}}
	return f
}
func TestQuestionPermanentWithdrawalFacts(t *testing.T) {
	for _, kind := range []string{"template", "instance", "blueprint"} {
		t.Run(kind, func(t *testing.T) {
			f := newQuestionFixture(t)
			sub := f.QApproved("author_a", "reviewer_a")
			f.QActivate(f.QPrepare(sub.ID))
			target := question.WithdrawalTarget{Kind: kind, Version: 1}
			wantTemplates, wantInstances, wantBlueprints := 3, 28, 1
			switch kind {
			case "template":
				target.ID = f.questionInput.QuestionPackage.Templates[0].ID
				wantTemplates = 2
				wantInstances = 18
				wantBlueprints = 0
			case "instance":
				target.ID = sub.Frozen.InstanceIdentities[0].ID
				wantInstances = 27
			case "blueprint":
				target.ID = f.questionInput.QuestionPackage.Blueprints[0].ID
				wantBlueprints = 0
			}
			preview, err := f.repo.PreviewQuestionWithdrawal(f.ctx, f.Access("admin_a", false), question.WithdrawalPreviewInput{Target: target}, question.ListQuery{Limit: 1})
			if err != nil || !question.ValidSHA(preview.ImpactDigest) || preview.Changes.Limit != 1 || preview.CurrentQuestionHead == nil {
				t.Fatal("invalid fixed preview", err)
			}
			result := f.QWithdraw(target)
			if result.Publication.TemplateCount != wantTemplates || result.Publication.InstanceCount != wantInstances || result.Publication.BlueprintCount != wantBlueprints {
				t.Fatal("incorrect withdrawal closure", result.Publication)
			}
			if f.count(`SELECT count(*) FROM question_instances`) != 28 || f.count(`SELECT count(*) FROM question_templates`) != 3 || f.count(`SELECT count(*) FROM question_blueprints`) != 1 || f.count(`SELECT count(*) FROM question_withdrawals`) != 1 {
				t.Fatal("historical bodies lost")
			}
		})
	}
	t.Run("explicit fixed source", func(t *testing.T) {
		f := newQuestionFixture(t)
		p := &f.questionInput.QuestionPackage
		instances, _, err := question.Generate(f.ctx, p.Templates[0])
		if err != nil {
			t.Fatal(err)
		}
		p.ID = "fixed-withdrawal-bank"
		p.Templates = []question.Template{}
		p.FixedQuestions = []question.FixedQuestion{}
		p.Blueprints[0].Sources = []question.BlueprintSource{}
		for n, i := range instances[:5] {
			id := fmt.Sprintf("fixed-withdrawal-%d", n)
			p.FixedQuestions = append(p.FixedQuestions, question.FixedQuestion{ID: id, Version: 1, Body: i.Body})
			p.Blueprints[0].Sources = append(p.Blueprints[0].Sources, question.BlueprintSource{Kind: "instance", Ref: question.Ref{ID: id, Version: 1}})
		}
		sub := f.QApproved("author_a", "reviewer_a")
		f.QActivate(f.QPrepare(sub.ID))
		result := f.QWithdraw(question.WithdrawalTarget{Kind: "instance", ID: p.FixedQuestions[0].ID, Version: 1})
		if result.Publication.InstanceCount != 4 || result.Publication.BlueprintCount != 0 {
			t.Fatal("fixed dependency retained")
		}
	})
}
func TestQuestionCoverageAfterSingleInstanceWithdrawal(t *testing.T) {
	f := singlePoolQuestionFixture(t)
	sub := f.QApproved("author_a", "reviewer_a")
	f.QActivate(f.QPrepare(sub.ID))
	report, err := f.repo.ReadQuestionCoverage(f.ctx, f.Access("reviewer_a", false), question.CoverageQuery{})
	if err != nil || report.EffectiveInstances != 5 || len(report.Nodes.Items) != 1 || !report.Nodes.Items[0].Ready {
		t.Fatal("five question fixture not ready", report, err)
	}
	f.QWithdraw(question.WithdrawalTarget{Kind: "instance", ID: sub.Frozen.InstanceIdentities[0].ID, Version: 1})
	report, err = f.repo.ReadQuestionCoverage(f.ctx, f.Access("reviewer_a", false), question.CoverageQuery{})
	if err != nil || report.EffectiveInstances != 4 || report.Nodes.Items[0].Ready || report.Nodes.Items[0].FiveQuestionFeasible || report.Nodes.Items[0].AssessmentInstances != 4 {
		t.Fatal("old ready or repeated questions retained", report, err)
	}
	empty := f.QWithdraw(question.WithdrawalTarget{Kind: "template", ID: f.questionInput.QuestionPackage.Templates[0].ID, Version: 1})
	if empty.Publication.TemplateCount != 0 || empty.Publication.InstanceCount != 0 || empty.Publication.BlueprintCount != 0 || empty.Publication.Status != "published" || *f.QHead() != empty.Publication.ID {
		t.Fatal("empty bank not allowed")
	}
	report, err = f.repo.ReadQuestionCoverage(f.ctx, f.Access("reviewer_a", false), question.CoverageQuery{})
	if err != nil || report.EffectiveInstances != 0 || report.Nodes.Items[0].Ready {
		t.Fatal("empty bank offered assessment", err)
	}
}
func TestQuestionKnowledgeWithdrawalStopsOffer(t *testing.T) {
	for _, kind := range []string{"knowledge", "unit", "asset"} {
		t.Run(kind, func(t *testing.T) {
			f := newQuestionFixture(t)
			sub := f.QApproved("author_a", "reviewer_a")
			f.QActivate(f.QPrepare(sub.ID))
			p := f.Input().Package
			target := publication.WithdrawalTarget{Kind: kind, Version: 1}
			want := 0
			switch kind {
			case "knowledge":
				target.ID = p.Knowledge[0].ID
				want = 0
			case "unit":
				target.ID = p.Units[0].ID
			case "asset":
				target.ID = ""
				target.Version = 0
				target.SHA256 = p.Assets[0].SHA256
			}
			if _, err := f.repo.WithdrawVersion(f.ctx, f.Access("admin_a", true), publication.WithdrawalInput{Target: target, ExpectedHead: f.KHead(), Reason: "Current exact knowledge, unit, or asset must immediately stop bound questions."}); err != nil {
				t.Fatal(err)
			}
			report, err := f.repo.ReadQuestionCoverage(f.ctx, f.Access("reviewer_a", false), question.CoverageQuery{})
			if err != nil || report.EffectiveInstances != want {
				t.Fatal("current content withdrawal did not stop bound questions", kind, report.EffectiveInstances, err)
			}
			if f.count(`SELECT count(*) FROM question_withdrawals`) != 0 || f.count(`SELECT count(*) FROM question_instances`) != 28 {
				t.Fatal("content withdrawal rewrote question history")
			}
		})
	}
	t.Run("unrelated head change", func(t *testing.T) {
		f := newQuestionFixture(t)
		sub := f.QApproved("author_a", "reviewer_a")
		f.QActivate(f.QPrepare(sub.ID))
		head := f.KHead()
		f.Activate(f.Prepare(f.Approved("author_b", "reviewer_b"), head), head)
		report, err := f.repo.ReadQuestionCoverage(f.ctx, f.Access("reviewer_a", false), question.CoverageQuery{})
		if err != nil || report.EffectiveInstances != 28 {
			t.Fatal("head change alone stopped fixed identities", err)
		}
	})
}
func TestQuestionWithdrawalReplayRace(t *testing.T) {
	f := newQuestionFixture(t)
	sub := f.QApproved("author_a", "reviewer_a")
	f.QActivate(f.QPrepare(sub.ID))
	prepared := f.QPrepare(sub.ID)
	target := question.WithdrawalTarget{Kind: "instance", ID: sub.Frozen.InstanceIdentities[0].ID, Version: 1}
	a := f.Access("admin_a", true)
	input := question.WithdrawalInput{Target: target, ExpectedKnowledgeHead: f.KHead(), ExpectedQuestionHead: f.QHead(), Reason: "Withdraw an exact instance and preserve the original idempotent result."}
	first, err := f.repo.WithdrawQuestionVersion(f.ctx, a, input)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = f.repo.ActivateQuestionRelease(f.ctx, f.Access("admin_a", true), prepared.ID, question.ActivateInput{ExpectedKnowledgeHead: prepared.BaseKnowledgeHead, ExpectedQuestionHead: prepared.BaseQuestionHead, ExpectedManifestSHA: prepared.ManifestSHA, Reason: "A prepared old bank cannot restore a permanently withdrawn instance."}); !errors.Is(err, question.ErrPublicationStale) {
		t.Fatal("old prepared bank restored target", err)
	}
	if _, err = f.repo.PrepareQuestionRelease(f.ctx, f.Access("admin_a", false), question.PrepareInput{SubmissionIDs: []string{sub.ID}, ExpectedKnowledgeHead: f.KHead(), ExpectedQuestionHead: f.QHead(), Reason: "An approved old submission cannot restore a blacklisted exact version."}); err == nil {
		t.Fatal("old submission restored target")
	}
	duplicate := input
	duplicate.ExpectedQuestionHead = f.QHead()
	if _, err = f.repo.WithdrawQuestionVersion(f.ctx, f.Access("admin_a", true), duplicate); !errors.Is(err, question.ErrImmutableConflict) {
		t.Fatal("different key duplicate accepted", err)
	}
	second := f.QWithdraw(question.WithdrawalTarget{Kind: "blueprint", ID: f.questionInput.QuestionPackage.Blueprints[0].ID, Version: 1})
	replay, err := f.repo.WithdrawQuestionVersion(f.ctx, a, input)
	if err != nil || replay.Publication.ID != first.Publication.ID || *f.QHead() != second.Publication.ID {
		t.Fatal("replay switched old head", err)
	}
	stale := question.WithdrawalInput{Target: question.WithdrawalTarget{Kind: "template", ID: f.questionInput.QuestionPackage.Templates[0].ID, Version: 1}, ExpectedKnowledgeHead: f.KHead(), ExpectedQuestionHead: &first.Publication.ID, Reason: "Both current heads must match at an exact withdrawal."}
	if _, err = f.repo.WithdrawQuestionVersion(f.ctx, f.Access("admin_a", true), stale); !errors.Is(err, question.ErrPublicationStale) {
		t.Fatal("stale question head accepted", err)
	}
	stale.ExpectedQuestionHead = f.QHead()
	oldK := f.KHead()
	f.Activate(f.Prepare(f.Approved("author_b", "reviewer_b"), oldK), oldK)
	if _, err = f.repo.WithdrawQuestionVersion(f.ctx, f.Access("admin_a", true), stale); !errors.Is(err, question.ErrPublicationStale) {
		t.Fatal("stale knowledge head accepted", err)
	}
	if _, err = f.repo.ReadQuestionCoverage(f.ctx, f.Access("author_a", false), question.CoverageQuery{}); !errors.Is(err, auth.ErrForbidden) {
		t.Fatal("editor read management coverage", err)
	}
}
func TestQuestionCoverageCounts(t *testing.T) {
	f := newQuestionFixture(t)
	sub := f.QApproved("author_a", "reviewer_a")
	f.QActivate(f.QPrepare(sub.ID))
	report, err := f.repo.ReadQuestionCoverage(f.ctx, f.Access("reviewer_a", false), question.CoverageQuery{Limit: 1})
	if err != nil || report.PublishedKnowledge != 1 || report.ApprovedTemplates != 3 || report.FixedQuestions != 0 || report.EffectiveInstances != 28 || report.DuplicateInstances != 0 || report.Nodes.Total != 1 || report.Nodes.Items[0].GeneratedInstances != 28 || report.Nodes.Items[0].FixedInstances != 0 {
		t.Fatal("counts conflate templates and instances", report, err)
	}
}
