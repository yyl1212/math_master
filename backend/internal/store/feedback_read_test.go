package store_test

import (
	"encoding/base64"
	"errors"
	"github.com/yyl1212/math_master/backend/internal/auth"
	"github.com/yyl1212/math_master/backend/internal/feedback"
	"strings"
	"testing"
	"time"
)

func TestFeedbackMetadataRedaction(t *testing.T) {
	f := newFeedbackFixture(t)
	in := f.feedbackInput()
	in.Title = "title-answer-sentinel"
	in.Location = "location-answer-sentinel"
	in.Message = "body-answer-sentinel"
	m := f.feedbackCreate("reviewer_a", in)
	other := f.feedbackCreate("learner_b", in)
	m = f.feedbackTransition("reviewer_b", m, feedback.Closed, &feedback.Resolution{Kind: "duplicate", DuplicateOf: &other.ID})
	for _, review := range []bool{false, true} {
		one, e := f.repo.ReadFeedbackTicket(f.ctx, f.Access("reviewer_a", false), m.ID, review)
		if e != nil {
			t.Fatal(e)
		}
		list, e := f.repo.ListFeedbackTickets(f.ctx, f.Access("reviewer_a", false), review, feedback.ListQuery{})
		if e != nil {
			t.Fatal(e)
		}
		for _, raw := range []string{feedbackJSON(one), feedbackJSON(list)} {
			if strings.Contains(raw, "answer-sentinel") || strings.Contains(raw, "duplicateOf") {
				t.Fatal("metadata leak")
			}
		}
		discussion, e := f.repo.ReadFeedbackEvents(f.ctx, f.Access("reviewer_a", false), m.ID, review, feedback.ListQuery{})
		if e != nil || discussion.Data.Title != in.Title || discussion.Data.Location != in.Location || discussion.Data.Items[0].Message != in.Message {
			t.Fatal("discussion", e)
		}
		if strings.Contains(feedbackJSON(discussion), other.ID) || strings.Contains(feedbackJSON(discussion), f.ids["reviewer_b"]) {
			t.Fatal("owner privacy even review URL")
		}
	}
	discussion, e := f.repo.ReadFeedbackEvents(f.ctx, f.Access("reviewer_b", false), m.ID, true, feedback.ListQuery{})
	if e != nil || discussion.Data.Items[1].Resolution.DuplicateOf == nil || *discussion.Data.Items[1].Resolution.DuplicateOf != other.ID {
		t.Fatal("authorized duplicate link", e)
	}
}
func TestFeedbackReadAndCursorIsolation(t *testing.T) {
	f := newFeedbackFixture(t)
	in := f.feedbackInput()
	tx, _ := f.db.Begin()
	defer tx.Rollback()
	var now time.Time
	tx.QueryRow(`SELECT clock_timestamp()`).Scan(&now)
	for n := 0; n < 51; n++ {
		id := f.ID()
		_, e := tx.Exec(`INSERT INTO feedback_tickets(id,owner_user_id,target,source,original_title,original_location,category,status,sequence,created_at,updated_at) VALUES($1,$2,$3,$4,$5,$6,$7,'new',1,$8,$8)`, id, f.ids["learner_a"], feedbackJSON(in.Target), feedbackJSON(in.Source), in.Title, in.Location, in.Category, now)
		if e != nil {
			t.Fatal(e)
		}
		_, e = tx.Exec(`INSERT INTO feedback_events(ticket_id,sequence,actor_user_id,kind,to_status,message,recorded_at,request_id) VALUES($1,1,$2,'created','new',$3,$4,'cursor-fixture')`, id, f.ids["learner_a"], in.Message, now)
		if e != nil {
			t.Fatal(e)
		}
	}
	if e := tx.Commit(); e != nil {
		t.Fatal(e)
	}
	first, e := f.repo.ListFeedbackTickets(f.ctx, f.Access("learner_a", false), false, feedback.ListQuery{Limit: 50})
	if e != nil || len(first.Data.Items) != 50 || first.Data.NextCursor == nil {
		t.Fatal("first", e)
	}
	rawCursor, _ := base64.RawURLEncoding.DecodeString(*first.Data.NextCursor)
	trailing := base64.RawURLEncoding.EncodeToString(append(rawCursor, []byte(" {}")...))
	if _, e = f.repo.ListFeedbackTickets(f.ctx, f.Access("learner_a", false), false, feedback.ListQuery{Cursor: trailing}); !errors.Is(e, auth.ErrInvalidInput) {
		t.Fatal("trailing cursor JSON", e)
	}
	second, e := f.repo.ListFeedbackTickets(f.ctx, f.Access("learner_a", false), false, feedback.ListQuery{Limit: 50, Cursor: *first.Data.NextCursor})
	if e != nil || len(second.Data.Items) != 1 || second.Data.NextCursor != nil {
		t.Fatal("last", e)
	}
	seen := map[string]bool{}
	for _, m := range append(first.Data.Items, second.Data.Items...) {
		if seen[m.ID] {
			t.Fatal("duplicate cursor")
		}
		seen[m.ID] = true
	}
	_, e = f.repo.ListFeedbackTickets(f.ctx, f.Access("learner_b", false), false, feedback.ListQuery{Cursor: *first.Data.NextCursor})
	if !errors.Is(e, auth.ErrInvalidInput) {
		t.Fatal("cross actor cursor", e)
	}
	if _, e = f.repo.ReadFeedbackTicket(f.ctx, f.Access("learner_b", false), first.Data.Items[0].ID, false); !errors.Is(e, auth.ErrNotFound) {
		t.Fatal("foreign metadata", e)
	}
	for _, q := range []feedback.ListQuery{{Limit: 51}, {Cursor: "bad"}, {Status: statusPointer("unknown")}} {
		if _, e = f.repo.ListFeedbackTickets(f.ctx, f.Access("learner_a", false), false, q); !errors.Is(e, auth.ErrInvalidInput) {
			t.Fatal("invalid page", e)
		}
	}
	empty, e := f.repo.ListFeedbackTickets(f.ctx, f.Access("learner_b", false), false, feedback.ListQuery{})
	if e != nil || feedbackJSON(empty.Data) != "{\"items\":[],\"nextCursor\":null}" {
		t.Fatal("empty page", e)
	}
	m := first.Data.Items[0]
	for n := 0; n < 50; n++ {
		m = f.feedbackTransition("reviewer_a", m, m.Status, nil)
	}
	q := feedback.ListQuery{}
	seqs := map[int64]bool{}
	for {
		page, e := f.repo.ReadFeedbackEvents(f.ctx, f.Access("learner_a", false), m.ID, false, q)
		if e != nil {
			t.Fatal(e)
		}
		for _, event := range page.Data.Items {
			if seqs[event.Sequence] {
				t.Fatal("duplicate sequence")
			}
			seqs[event.Sequence] = true
		}
		if page.Data.NextCursor == nil {
			break
		}
		q.Cursor = *page.Data.NextCursor
	}
	if len(seqs) != 51 {
		t.Fatal("event pagination", len(seqs))
	}
	m = f.feedbackTransition("reviewer_a", m, feedback.Closed, &feedback.Resolution{Kind: "suggestion_recorded"})
	if m.Sequence != 52 {
		t.Fatal("long thread cannot close")
	}
}
func statusPointer(s feedback.Status) *feedback.Status { return &s }

func TestFeedbackMetadataReceiptLabelGuard(t *testing.T) {
	f := newFeedbackFixture(t)
	m := f.feedbackCreate("learner_a", f.feedbackInput())
	for _, mode := range []string{"label", "action"} {
		r := feedback.Receipt{Status: 201, Ticket: m}
		action, resource := "create", "tickets"
		if mode == "label" {
			r.Ticket.Label = "answer-sentinel"
		} else {
			action = "reply"
			resource = m.ID
			r.Status = 200
		}
		_, e := f.db.Exec(`INSERT INTO feedback_idempotency(actor_user_id,action,resource,key,request_sha256,ticket_id,event_sequence,receipt) VALUES($1,$2,$3,$4,$5,$6,1,$7)`, f.ids["learner_a"], action, resource, f.ID(), string(makeSHA('c')), m.ID, feedbackJSON(r))
		if e == nil {
			t.Fatal("unsafe original receipt", mode)
		}
	}
}

func TestFeedbackMetadataSiteReceipt(t *testing.T) {
	f := newFeedbackFixture(t)
	area := feedback.Area("home")
	in := feedback.CreateInput{Target: feedback.Target{Kind: "site", Area: &area}, Source: feedback.Source{Kind: "site"}, Category: "technical_issue", Title: "Site report", Message: "Details", Location: ""}
	var label string
	if e := f.db.QueryRow(`SELECT feedback_label($1::jsonb)`, feedbackJSON(in.Target)).Scan(&label); e != nil {
		t.Fatal(e)
	}
	m := f.feedbackCreate("learner_a", in)
	if m.Label != "Site · home" {
		t.Fatal(m.Label)
	}
}
