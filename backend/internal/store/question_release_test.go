package store_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"github.com/yyl1212/math_master/backend/internal/auth"
	"github.com/yyl1212/math_master/backend/internal/publication"
	"github.com/yyl1212/math_master/backend/internal/question"
	"strings"
	"testing"
)

func TestQuestionDualHeadActivation(t *testing.T) {
	f := newQuestionFixture(t)
	sub := f.QApproved("author_a", "reviewer_a")
	p := f.QPrepare(sub.ID)
	activate := func(v question.PublicationSummary) question.ActivateInput {
		return question.ActivateInput{ExpectedKnowledgeHead: v.BaseKnowledgeHead, ExpectedQuestionHead: v.BaseQuestionHead, ExpectedManifestSHA: v.ManifestSHA, Reason: "Activate fixed evidence after checking both current heads."}
	}
	bad := activate(p)
	bad.ExpectedManifestSHA = strings.Repeat("0", 64)
	if _, err := f.repo.ActivateQuestionRelease(f.ctx, f.Access("admin_a", true), p.ID, bad); !errors.Is(err, question.ErrPublicationStale) {
		t.Fatal("wrong manifest accepted", err)
	}
	a := f.Access("admin_a", false)
	f.exec(`UPDATE auth_sessions SET reauthenticated_at=clock_timestamp()-interval '301 seconds' WHERE token_hash=$1`, a.TokenHash[:])
	if _, err := f.repo.ActivateQuestionRelease(f.ctx, a, p.ID, activate(p)); !errors.Is(err, auth.ErrReauthRequired) {
		t.Fatal("expired reauth accepted", err)
	}
	f.exec(`CREATE FUNCTION question_test_activate_audit() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN IF NEW.action='activateRelease' THEN RAISE EXCEPTION 'fixture audit failure'; END IF; RETURN NEW; END $$`)
	f.exec(`CREATE TRIGGER question_test_activate_audit BEFORE INSERT ON question_events FOR EACH ROW EXECUTE FUNCTION question_test_activate_audit()`)
	if _, err := f.repo.ActivateQuestionRelease(f.ctx, f.Access("admin_a", true), p.ID, activate(p)); !errors.Is(err, auth.ErrUnavailable) || f.QHead() != nil {
		t.Fatal("audit failure switched head", err)
	}
	f.exec(`DROP TRIGGER question_test_activate_audit ON question_events`)
	a = f.Access("admin_a", true)
	first, err := f.repo.ActivateQuestionRelease(f.ctx, a, p.ID, activate(p))
	if err != nil {
		t.Fatal(err)
	}
	stale := f.QPrepare(sub.ID)
	second := f.QActivate(f.QPrepare(sub.ID))
	if _, err = f.repo.ActivateQuestionRelease(f.ctx, f.Access("admin_a", true), stale.ID, activate(stale)); !errors.Is(err, question.ErrPublicationStale) {
		t.Fatal("changed question head accepted", err)
	}
	replay, err := f.repo.ActivateQuestionRelease(f.ctx, a, p.ID, activate(p))
	if err != nil || replay.ID != first.ID || *f.QHead() != second.ID {
		t.Fatal("replay restored historical head", err)
	}
	stale = f.QPrepare(sub.ID)
	f.Activate(f.Prepare(f.Approved("author_b", "reviewer_b"), f.KHead()), f.KHead())
	if _, err = f.repo.ActivateQuestionRelease(f.ctx, f.Access("admin_a", true), stale.ID, activate(stale)); !errors.Is(err, question.ErrPublicationStale) {
		t.Fatal("changed knowledge head accepted", err)
	}
}
func TestQuestionEvidenceEligibility(t *testing.T) {
	f := newQuestionFixture(t)
	sub := f.QApproved("author_a", "reviewer_a")
	f.exec(`DELETE FROM auth_user_roles WHERE user_id=$1 AND role='reviewer'`, f.ids["reviewer_a"])
	input := question.PrepareInput{SubmissionIDs: []string{sub.ID}, ExpectedKnowledgeHead: f.KHead(), Reason: "A new selection must have a currently eligible reviewer."}
	if _, err := f.repo.PrepareQuestionRelease(f.ctx, f.Access("admin_a", false), input); !errors.Is(err, question.ErrReviewRequired) {
		t.Fatal("revoked new reviewer accepted", err)
	}
	f.exec(`INSERT INTO auth_user_roles VALUES($1,'reviewer')`, f.ids["reviewer_a"])
	f.QActivate(f.QPrepare(sub.ID))
	instances, _, err := question.Generate(f.ctx, f.questionInput.QuestionPackage.Templates[0])
	if err != nil {
		t.Fatal(err)
	}
	f.questionInput.QuestionPackage.ID = "additional-fixed-bank"
	f.questionInput.QuestionPackage.Templates = []question.Template{}
	f.questionInput.QuestionPackage.Blueprints = []question.Blueprint{}
	f.questionInput.QuestionPackage.FixedQuestions = []question.FixedQuestion{{ID: "additional-fixed", Version: 1, Body: instances[0].Body}}
	other := f.QApproved("author_b", "reviewer_b")
	f.exec(`DELETE FROM auth_user_roles WHERE user_id=$1 AND role='reviewer'`, f.ids["reviewer_a"])
	inherited := f.QActivate(f.QPrepare(other.ID))
	page, err := f.repo.ListQuestionMembers(f.ctx, f.Access("admin_a", false), inherited.ID, question.ListQuery{Limit: 100})
	if err != nil {
		t.Fatal(err)
	}
	for _, m := range page.Items {
		if m.Identity.ID == "additional-fixed" {
			if m.Evidence.InheritedFrom != nil || m.Evidence.SubmissionID != other.ID {
				t.Fatal("new selected evidence lost")
			}
			continue
		}
		if m.Evidence.InheritedFrom == nil || m.Evidence.SubmissionID != sub.ID {
			t.Fatal("published evidence lost")
		}
	}
}
func TestQuestionReplacementPreservesHistoricalFacts(t *testing.T) {
	f := newQuestionFixture(t)
	old := f.QApproved("author_a", "reviewer_a")
	f.QActivate(f.QPrepare(old.ID))
	oldInstance := old.Frozen.InstanceIdentities[0]
	var before []byte
	if err := f.db.QueryRow(`SELECT body_bytes FROM question_instances WHERE id=$1 AND version=$2`, oldInstance.ID, oldInstance.Version).Scan(&before); err != nil {
		t.Fatal(err)
	}
	var input question.DraftInput
	raw, _ := json.Marshal(f.questionInput)
	_ = json.Unmarshal(raw, &input)
	input.QuestionPackage.ID = "template-replacement-bank"
	input.QuestionPackage.Templates[0].Version = 2
	input.QuestionPackage.Blueprints = []question.Blueprint{}
	f.questionInput = input
	replacement := f.QApproved("author_b", "reviewer_b")
	if _, err := f.repo.PrepareQuestionRelease(f.ctx, f.Access("admin_a", false), question.PrepareInput{SubmissionIDs: []string{replacement.ID}, ExpectedKnowledgeHead: f.KHead(), ExpectedQuestionHead: f.QHead(), Reason: "An old blueprint may not silently use a replacement template."}); !errors.Is(err, question.ErrNotReady) {
		t.Fatal("old blueprint silently repaired", err)
	}
	_ = json.Unmarshal(raw, &input)
	input.QuestionPackage.ID = "complete-replacement-bank"
	input.QuestionPackage.Templates[0].Version = 2
	input.QuestionPackage.Blueprints[0].Version = 2
	for j := range input.QuestionPackage.Blueprints[0].Sources {
		if input.QuestionPackage.Blueprints[0].Sources[j].Ref.ID == input.QuestionPackage.Templates[0].ID {
			input.QuestionPackage.Blueprints[0].Sources[j].Ref.Version = 2
		}
	}
	f.questionInput = input
	approved := f.QApproved("author_b", "reviewer_b")
	p := f.QActivate(f.QPrepare(approved.ID))
	if f.count(`SELECT count(*) FROM question_publication_members m JOIN question_instances i ON i.id=m.id AND i.version=m.version WHERE m.publication_id=$1 AND m.kind='instance' AND i.template_id=$2 AND i.template_version=1`, p.ID, input.QuestionPackage.Templates[0].ID) != 0 {
		t.Fatal("old generated instances remained")
	}
	var after []byte
	if err := f.db.QueryRow(`SELECT body_bytes FROM question_instances WHERE id=$1 AND version=$2`, oldInstance.ID, oldInstance.Version).Scan(&after); err != nil || !bytes.Equal(before, after) || f.count(`SELECT count(*) FROM question_withdrawals`) != 0 {
		t.Fatal("replacement deleted history or blacklisted it", err)
	}
}
func TestQuestionPublicationPagination(t *testing.T) {
	f := newQuestionFixture(t)
	sub := f.QApproved("author_a", "reviewer_a")
	head := f.QActivate(f.QPrepare(sub.ID))
	for n := 0; n < 100; n++ {
		f.QPrepare(sub.ID)
	}
	page, err := f.repo.ListQuestionPublications(f.ctx, f.Access("admin_a", false), question.ListQuery{Limit: 100})
	if err != nil || page.Total != 101 || page.Head == nil || *page.Head != head.ID {
		t.Fatal("publication history/head lost", err)
	}
	for _, p := range page.Items {
		if p.ID == head.ID {
			t.Fatal("fixture head unexpectedly on first page")
		}
		raw, _ := json.Marshal(p)
		if len(raw) > question.MaxResponseBytes || bytes.Contains(raw, []byte(`"manifest":`)) {
			t.Fatal("summary leaked body")
		}
	}
	got, err := f.repo.ReadQuestionPublication(f.ctx, f.Access("admin_a", false), head.ID)
	if err != nil || got.Status != "published" {
		t.Fatal("head cannot be read by ID", err)
	}
	members := 0
	for offset := 0; ; {
		p, err := f.repo.ListQuestionMembers(f.ctx, f.Access("admin_a", false), head.ID, question.ListQuery{Limit: 7, Offset: offset})
		if err != nil {
			t.Fatal(err)
		}
		members += len(p.Items)
		offset += p.Limit
		if offset >= p.Total {
			break
		}
	}
	changes := 0
	for offset := 0; ; {
		p, err := f.repo.ListQuestionChanges(f.ctx, f.Access("admin_a", false), head.ID, question.ListQuery{Limit: 7, Offset: offset})
		if err != nil {
			t.Fatal(err)
		}
		changes += len(p.Items)
		offset += p.Limit
		if offset >= p.Total {
			break
		}
	}
	if members != 32 || changes != 32 {
		t.Fatal("inexact member/change pages", members, changes)
	}
}
func TestQuestionCandidateCapacity(t *testing.T) {
	for _, v := range [][5]int{{200, 10000, 1000, 32 << 20, 8 << 20}, {0, 0, 0, 0, 0}} {
		if err := question.CheckBankLimits(v[0], v[1], v[2], v[3], v[4]); err != nil {
			t.Fatal("legal boundary rejected", v, err)
		}
	}
	for _, v := range [][5]int{{201, 0, 0, 0, 0}, {0, 10001, 0, 0, 0}, {0, 0, 1001, 0, 0}, {0, 0, 0, (32 << 20) + 1, 0}, {0, 0, 0, 0, (8 << 20) + 1}} {
		if err := question.CheckBankLimits(v[0], v[1], v[2], v[3], v[4]); !errors.Is(err, question.ErrLimitExceeded) {
			t.Fatal("bank capacity accepted", v, err)
		}
	}
	f := newQuestionFixture(t)
	for _, n := range []int{0, 21} {
		ids := []string{}
		for i := 0; i < n; i++ {
			ids = append(ids, f.ID())
		}
		if _, err := f.repo.PrepareQuestionRelease(f.ctx, f.Access("admin_a", false), question.PrepareInput{SubmissionIDs: ids, ExpectedKnowledgeHead: f.KHead(), Reason: "Capacity test for an approved submission selection."}); err == nil {
			t.Fatal("invalid submission count accepted")
		}
	}
}

func TestQuestionDualHeadActivationProofs(t *testing.T) {
	for _, scenario := range []string{"admin role", "admin credential", "selected reviewer"} {
		t.Run(scenario, func(t *testing.T) {
			f := newQuestionFixture(t)
			sub := f.QApproved("author_a", "reviewer_a")
			p := f.QPrepare(sub.ID)
			a := f.Access("admin_a", true)
			want := auth.ErrForbidden
			switch scenario {
			case "admin role":
				f.exec(`DELETE FROM auth_user_roles WHERE user_id=$1 AND role='admin'`, f.ids["admin_a"])
			case "admin credential":
				f.exec(`UPDATE auth_users SET credential_version=credential_version+1 WHERE id=$1`, f.ids["admin_a"])
				want = auth.ErrAuthenticationRequired
			case "selected reviewer":
				f.exec(`DELETE FROM auth_user_roles WHERE user_id=$1 AND role='reviewer'`, f.ids["reviewer_a"])
				want = question.ErrReviewRequired
			}
			if _, err := f.repo.ActivateQuestionRelease(f.ctx, a, p.ID, question.ActivateInput{ExpectedKnowledgeHead: p.BaseKnowledgeHead, ExpectedQuestionHead: p.BaseQuestionHead, ExpectedManifestSHA: p.ManifestSHA, Reason: "An activation must recheck current account and reviewer proofs."}); !errors.Is(err, want) || f.QHead() != nil {
				t.Fatal("stale proof committed", scenario, err)
			}
		})
	}
}
func TestQuestionKnowledgePublicationRace(t *testing.T) {
	for _, first := range []string{"knowledge", "question"} {
		t.Run(first, func(t *testing.T) {
			f := newQuestionFixture(t)
			sub := f.QApproved("author_a", "reviewer_a")
			p := f.QPrepare(sub.ID)
			kbase := f.KHead()
			content := f.Prepare(f.Approved("author_b", "reviewer_b"), kbase)
			gate := holdAdminLock(t, f.db)
			knowledgeResult := make(chan error, 1)
			questionResult := make(chan error, 1)
			ka := f.Access("admin_a", true)
			qa := f.Access("admin_a", true)
			publishKnowledge := func() {
				_, err := f.repo.ActivateRelease(f.ctx, ka, content.ID, publication.ActivateInput{ExpectedHead: kbase, ExpectedManifestSHA: content.ManifestSHA, Reason: "Publish content during a synchronized question activation race."})
				knowledgeResult <- err
			}
			publishQuestion := func() {
				_, err := f.repo.ActivateQuestionRelease(f.ctx, qa, p.ID, question.ActivateInput{ExpectedKnowledgeHead: p.BaseKnowledgeHead, ExpectedQuestionHead: p.BaseQuestionHead, ExpectedManifestSHA: p.ManifestSHA, Reason: "Activate questions during a synchronized content publication race."})
				questionResult <- err
			}
			if first == "knowledge" {
				go publishKnowledge()
				waitForAdminLockWaiters(t, f.db, 1)
				go publishQuestion()
			} else {
				go publishQuestion()
				waitForAdminLockWaiters(t, f.db, 1)
				go publishKnowledge()
			}
			waitForAdminLockWaiters(t, f.db, 2)
			if err := gate.Commit(); err != nil {
				t.Fatal(err)
			}
			if err := <-knowledgeResult; err != nil {
				t.Fatal(err)
			}
			err := <-questionResult
			if first == "knowledge" {
				if !errors.Is(err, question.ErrPublicationStale) || f.QHead() != nil {
					t.Fatal("question activation ignored earlier knowledge publication", err)
				}
			} else {
				if err != nil || f.QHead() == nil || *f.QHead() != p.ID {
					t.Fatal("question did not activate against actual initial heads", err)
				}
			}
		})
	}
}

func TestQuestionReSelectedEvidenceRequiresReviewer(t *testing.T) {
	f := newQuestionFixture(t)
	sub := f.QApproved("author_a", "reviewer_a")
	f.QActivate(f.QPrepare(sub.ID))
	p := f.QPrepare(sub.ID)
	f.exec(`DELETE FROM auth_user_roles WHERE user_id=$1 AND role='reviewer'`, f.ids["reviewer_a"])
	before := *f.QHead()
	if _, err := f.repo.ActivateQuestionRelease(f.ctx, f.Access("admin_a", true), p.ID, question.ActivateInput{ExpectedKnowledgeHead: p.BaseKnowledgeHead, ExpectedQuestionHead: p.BaseQuestionHead, ExpectedManifestSHA: p.ManifestSHA, Reason: "A reselected exact version still needs an eligible current reviewer."}); !errors.Is(err, question.ErrReviewRequired) || *f.QHead() != before {
		t.Fatal("reselection bypassed current reviewer eligibility", err)
	}
}
