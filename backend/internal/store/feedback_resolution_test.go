package store_test

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"github.com/yyl1212/math_master/backend/internal/auth"
	"github.com/yyl1212/math_master/backend/internal/feedback"
	"github.com/yyl1212/math_master/backend/internal/publication"
	"github.com/yyl1212/math_master/backend/internal/question"
	"strings"
	"testing"
)

func (f *feedbackFixture) feedbackCreate(actor string, in feedback.CreateInput) feedback.Metadata {
	f.t.Helper()
	r, e := f.repo.CreateFeedback(f.ctx, f.Access(actor, false), in)
	if e != nil {
		f.t.Fatal(e)
	}
	return r.Data.Ticket
}
func (f *feedbackFixture) feedbackTransition(actor string, m feedback.Metadata, status feedback.Status, r *feedback.Resolution) feedback.Metadata {
	f.t.Helper()
	got, e := f.repo.TransitionFeedback(f.ctx, f.Access(actor, false), m.ID, feedback.TransitionInput{ExpectedSequence: m.Sequence, Status: status, Message: "Independent handling reason", Resolution: r})
	if e != nil {
		f.t.Fatal(e)
	}
	return got.Data.Ticket
}
func TestFeedbackTransitions(t *testing.T) {
	f := newFeedbackFixture(t)
	m := f.feedbackCreate("reviewer_a", f.feedbackInput())
	if _, e := f.repo.TransitionFeedback(f.ctx, f.Access("reviewer_a", false), m.ID, feedback.TransitionInput{ExpectedSequence: 1, Status: feedback.Processing, Message: "self"}); !errors.Is(e, auth.ErrForbidden) {
		t.Fatal("self", e)
	}
	m = f.feedbackTransition("reviewer_b", m, feedback.Processing, nil)
	m = f.feedbackTransition("reviewer_b", m, feedback.WaitingDetails, nil)
	m = f.feedbackTransition("reviewer_b", m, feedback.Resolved, &feedback.Resolution{Kind: "clarified"})
	same := f.feedbackTransition("reviewer_b", m, feedback.Resolved, nil)
	if same.ResolutionKind == nil || *same.ResolutionKind != "clarified" {
		t.Fatal("same-state basis")
	}
	replayKey := f.Access("reviewer_b", false)
	input := feedback.TransitionInput{ExpectedSequence: same.Sequence, Status: feedback.Processing, Message: "reopen"}
	old, e := f.repo.TransitionFeedback(f.ctx, replayKey, same.ID, input)
	if e != nil {
		t.Fatal(e)
	}
	m = old.Data.Ticket
	m = f.feedbackTransition("reviewer_b", m, feedback.Closed, &feedback.Resolution{Kind: "not_reproducible"})
	again, e := f.repo.TransitionFeedback(f.ctx, replayKey, same.ID, input)
	if e != nil || feedbackJSON(again) != feedbackJSON(old) {
		t.Fatal("transition replay", e)
	}
	own, e := f.repo.ReplyFeedback(f.ctx, f.Access("reviewer_a", false), m.ID, feedback.ReplyInput{ExpectedSequence: m.Sequence, Message: "Reopen with evidence"})
	if e != nil || own.Data.Ticket.Status != feedback.Processing || own.Data.Ticket.ResolutionKind != nil {
		t.Fatal("owner reopening", e)
	}
	m = f.feedbackCreate("learner_a", f.feedbackInput())
	start := make(chan struct{})
	done := make(chan error, 2)
	for _, actor := range []string{"reviewer_a", "reviewer_b"} {
		a := f.Access(actor, false)
		go func() {
			<-start
			_, e := f.repo.TransitionFeedback(f.ctx, a, m.ID, feedback.TransitionInput{ExpectedSequence: m.Sequence, Status: feedback.Processing, Message: "concurrent"})
			done <- e
		}()
	}
	close(start)
	success := 0
	for j := 0; j < 2; j++ {
		e := <-done
		if e == nil {
			success++
		} else if !errors.Is(e, feedback.ErrConflict) {
			t.Fatal(e)
		}
	}
	if success != 1 || f.count(`SELECT sequence FROM feedback_tickets WHERE id=$1`, m.ID) != 2 {
		t.Fatal("concurrent sequence")
	}
	m = f.feedbackCreate("learner_a", f.feedbackInput())
	m = f.feedbackTransition("reviewer_a", m, feedback.Processing, nil)
	if _, e = f.repo.TransitionFeedback(f.ctx, f.Access("reviewer_a", false), m.ID, feedback.TransitionInput{ExpectedSequence: m.Sequence, Status: feedback.Resolved, Message: "invalid basis", Resolution: &feedback.Resolution{Kind: "service_fixed"}}); !errors.Is(e, auth.ErrInvalidInput) {
		t.Fatal("service only site", e)
	}
	w := workflowWithdraw(f.workflowFixture, publication.WithdrawalTarget{Kind: "knowledge", ID: f.knowledge.ID, Version: 1}, f.KHead())
	m = f.feedbackTransition("reviewer_a", m, feedback.Resolved, &feedback.Resolution{Kind: "withdrawn", Withdrawal: &feedback.WithdrawalRef{Space: "content", ID: w.EventID}})
	if m.TargetValidity != "withdrawn" {
		t.Fatal(m)
	}
}
func TestFeedbackDuplicatePrivacy(t *testing.T) {
	f := newFeedbackFixture(t)
	a := f.feedbackCreate("learner_a", f.feedbackInput())
	b := f.feedbackCreate("learner_b", f.feedbackInput())
	r := &feedback.Resolution{Kind: "duplicate", DuplicateOf: &b.ID}
	closed := f.feedbackTransition("reviewer_a", a, feedback.Closed, r)
	if strings.Contains(feedbackJSON(closed), b.ID) || strings.Contains(feedbackJSON(closed), "duplicateOf") {
		t.Fatal("receipt privacy")
	}
	other := f.feedbackCreate("learner_a", f.feedbackInput())
	for _, id := range []string{other.ID, f.ID()} {
		_, e := f.repo.TransitionFeedback(f.ctx, f.Access("reviewer_a", false), other.ID, feedback.TransitionInput{ExpectedSequence: 1, Status: feedback.Closed, Message: "duplicate evidence", Resolution: &feedback.Resolution{Kind: "duplicate", DuplicateOf: &id}})
		if !errors.Is(e, auth.ErrInvalidInput) {
			t.Fatal("fake duplicate", e)
		}
	}
	changed := f.feedbackInput()
	changed.Category = "typo"
	target := f.feedbackCreate("learner_b", changed)
	_, e := f.repo.TransitionFeedback(f.ctx, f.Access("reviewer_a", false), other.ID, feedback.TransitionInput{ExpectedSequence: 1, Status: feedback.Closed, Message: "wrong category", Resolution: &feedback.Resolution{Kind: "duplicate", DuplicateOf: &target.ID}})
	if !errors.Is(e, auth.ErrInvalidInput) {
		t.Fatal("category", e)
	}
}
func TestFeedbackGeneratedReplacement(t *testing.T) {
	f := newFeedbackFixture(t)
	p := f.createPractice("learner_a")
	context, e := f.repo.ReadFeedbackContext(f.ctx, f.Access("learner_a", false), feedback.ContextQuery{Kind: "practice", ID: p.Summary.ID})
	if e != nil {
		t.Fatal(e)
	}
	in := f.feedbackInput()
	in.Target = context.Data.Target
	in.Source = context.Data.Source
	m := f.feedbackCreate("learner_a", in)
	m = f.feedbackTransition("reviewer_a", m, feedback.Processing, nil)
	w := f.QWithdraw(question.WithdrawalTarget{Kind: "instance", ID: p.Question.Instance.ID, Version: 1})
	pkg := &f.questionInput.QuestionPackage
	pkg.Version = 2
	pkg.Templates[0].ID = "lf-revised-addition"
	pkg.Blueprints[0].Version = 2
	pkg.Blueprints[0].Sources = []question.BlueprintSource{{Kind: "template", Ref: question.Ref{ID: "lf-revised-addition", Version: 1}}}
	sub := f.QApproved("author_b", "reviewer_b")
	pub := f.QActivate(f.QPrepare(sub.ID))
	var replacement question.Identity
	if e = f.db.QueryRow(`SELECT i.id,i.version,i.sha256 FROM question_instances i JOIN question_publication_members p ON p.publication_id=$1 AND p.kind='instance' AND p.id=i.id AND p.version=i.version AND p.sha256=i.sha256 WHERE i.template_id='lf-revised-addition' ORDER BY i.id LIMIT 1`, pub.ID).Scan(&replacement.ID, &replacement.Version, &replacement.SHA256); e != nil {
		t.Fatal(e)
	}
	if replacement.ID == p.Question.Instance.ID || !strings.HasPrefix(replacement.ID, "qi-") {
		t.Fatal("new generated id")
	}
	var before, after string
	q := `SELECT jsonb_build_array((SELECT coalesce(jsonb_agg(to_jsonb(x) ORDER BY x.id),'[]') FROM learning_events x),(SELECT coalesce(jsonb_agg(to_jsonb(x) ORDER BY x.attempt_id,x.position),'[]') FROM assessment_answers x),(SELECT coalesce(jsonb_agg(to_jsonb(x) ORDER BY x.attempt_id),'[]') FROM assessment_results x),(SELECT coalesce(jsonb_agg(to_jsonb(x) ORDER BY x.owner_user_id,x.knowledge_id),'[]') FROM learning_unlocks x))::text`
	if e = f.db.QueryRow(q).Scan(&before); e != nil {
		t.Fatal(e)
	}
	res := &feedback.Resolution{Kind: "revision_published", Withdrawal: &feedback.WithdrawalRef{Space: "question", ID: w.EventID}, Replacement: &feedback.Replacement{Kind: "instance", Identity: &replacement, PublicationID: pub.ID}}
	bad := *res.Replacement
	bad.PublicationID = f.ID()
	copyRes := *res
	copyRes.Replacement = &bad
	if _, e = f.repo.TransitionFeedback(f.ctx, f.Access("reviewer_a", false), m.ID, feedback.TransitionInput{ExpectedSequence: m.Sequence, Status: feedback.Resolved, Message: "unpublished", Resolution: &copyRes}); !errors.Is(e, auth.ErrInvalidInput) {
		t.Fatal("unpublished", e)
	}
	m = f.feedbackTransition("reviewer_a", m, feedback.Resolved, res)
	if m.Target.Identity.ID != p.Question.Instance.ID || m.TargetValidity != "withdrawn" {
		t.Fatal("original root changed")
	}
	if e = f.db.QueryRow(q).Scan(&after); e != nil || before != after {
		t.Fatal("learning facts changed", e)
	}
}

func TestFeedbackTransitionsPublishedContentParts(t *testing.T) {
	for _, kind := range []string{"knowledge", "unit", "asset"} {
		t.Run(kind, func(t *testing.T) {
			f := newFeedbackFixture(t)
			q := feedback.ContextQuery{Kind: "knowledge", ID: f.knowledge.ID}
			if kind != "knowledge" {
				q.PartKind = kind
				q.PartID = "workflow-fractions-unit"
				if kind == "asset" {
					q.PartID = f.Input().Package.Assets[0].ID
				}
			}
			context, e := f.repo.ReadFeedbackContext(f.ctx, f.Access("learner_a", false), q)
			if e != nil {
				t.Fatal(e)
			}
			in := f.feedbackInput()
			in.Target = context.Data.Target
			in.Source = context.Data.Source
			m := f.feedbackCreate("learner_a", in)
			m = f.feedbackTransition("reviewer_a", m, feedback.Processing, nil)
			target := publication.WithdrawalTarget{Kind: kind, ID: f.knowledge.ID, Version: 1}
			if kind == "unit" {
				target.ID = q.PartID
			}
			if kind == "asset" {
				target = publication.WithdrawalTarget{Kind: "asset", SHA256: in.Target.Part.Asset.SHA256}
			}
			w := workflowWithdraw(f.workflowFixture, target, f.KHead())
			v := newMaterial(f.Input())
			if kind == "asset" {
				svg := []byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 120 80"><rect x="10" y="10" width="100" height="60" fill="#517dd1"/></svg>`)
				sum := sha256.Sum256(svg)
				v.Package.Assets[0].SHA256 = hex.EncodeToString(sum[:])
				v.AssetBytes[0].Base64 = base64.StdEncoding.EncodeToString(svg)
			}
			head := f.KHead()
			sub := f.ApprovedInput(v)
			pub := f.Activate(f.Prepare(sub, head), head)
			replacement := feedback.Replacement{Kind: kind, PublicationID: pub.ID}
			if kind == "asset" {
				replacement.Asset = &feedback.AssetRef{ID: v.Package.Assets[0].ID, SHA256: v.Package.Assets[0].SHA256}
			} else {
				table := "knowledge_versions"
				id := f.knowledge.ID
				if kind == "unit" {
					table = "unit_versions"
					id = q.PartID
				}
				identity := question.Identity{ID: id, Version: 2}
				if e = f.db.QueryRow(`SELECT sha256 FROM `+table+` WHERE id=$1 AND version=2`, id).Scan(&identity.SHA256); e != nil {
					t.Fatal(e)
				}
				replacement.Identity = &identity
			}
			m = f.feedbackTransition("reviewer_a", m, feedback.Resolved, &feedback.Resolution{Kind: "revision_published", Withdrawal: &feedback.WithdrawalRef{Space: "content", ID: w.EventID}, Replacement: &replacement})
			if m.ResolutionKind == nil || *m.ResolutionKind != "revision_published" {
				t.Fatal("published part")
			}
		})
	}
}
