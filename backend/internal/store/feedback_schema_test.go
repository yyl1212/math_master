package store_test

import (
	"github.com/yyl1212/math_master/backend/internal/feedback"
	"strings"
	"testing"
)

func TestFeedbackSchemaOriginalAndEvents(t *testing.T) {
	f := newFeedbackFixture(t)
	if countFeedbackTables(t, f.db) != 4 {
		t.Fatal("four tables")
	}
	id := f.seededFeedback()
	f.seedFeedbackReceipt(id)
	for _, q := range []string{
		`UPDATE feedback_tickets SET original_title='changed' WHERE id=$1`,
		`UPDATE feedback_tickets SET original_location='changed' WHERE id=$1`,
		`UPDATE feedback_tickets SET owner_user_id=gen_random_uuid() WHERE id=$1`,
		`UPDATE feedback_tickets SET source='{}' WHERE id=$1`,
		`UPDATE feedback_tickets SET target='{}' WHERE id=$1`,
		`UPDATE feedback_tickets SET category='typo' WHERE id=$1`,
		`UPDATE feedback_tickets SET created_at=clock_timestamp() WHERE id=$1`,
		`DELETE FROM feedback_tickets WHERE id=$1`,
		`UPDATE feedback_events SET message='changed' WHERE ticket_id=$1`,
		`DELETE FROM feedback_events WHERE ticket_id=$1`,
		`UPDATE feedback_idempotency SET receipt='{}' WHERE ticket_id=$1`,
		`DELETE FROM feedback_idempotency WHERE ticket_id=$1`,
		`UPDATE feedback_tickets SET status='processing',sequence=2 WHERE id=$1`,
	} {
		if _, e := f.db.Exec(q, id); e == nil {
			t.Fatalf("accepted mutation: %s", q)
		}
	}
}
func TestFeedbackSchemaCreationAndProjection(t *testing.T) {
	f := newFeedbackFixture(t)
	if countFeedbackTables(t, f.db) != 4 {
		t.Fatal("four tables")
	}
	tx, e := f.db.Begin()
	if e != nil {
		t.Fatal(e)
	}
	defer tx.Rollback()
	if e = f.insertFeedback(tx, f.ID(), f.ids["learner_a"], f.feedbackInput(), false); e != nil {
		t.Fatal(e)
	}
	if e = tx.Commit(); e == nil {
		t.Fatal("missing creation event")
	}
	id := f.seededFeedback()
	for _, seq := range []int{3, 2} {
		tx, e = f.db.Begin()
		if e != nil {
			t.Fatal(e)
		}
		_, e = tx.Exec(`INSERT INTO feedback_events(ticket_id,sequence,actor_user_id,kind,from_status,to_status,message,recorded_at,request_id) VALUES($1,$2,$3,'replied','new','new','reply',clock_timestamp(),'projection-test')`, id, seq, f.ids["learner_a"])
		if e == nil {
			e = tx.Commit()
		}
		tx.Rollback()
		if e == nil {
			t.Fatal("gap or missing projection", seq)
		}
	}
	_, e = f.db.Exec(`INSERT INTO feedback_events(ticket_id,sequence,actor_user_id,kind,from_status,to_status,message,resolution,effective_resolution,recorded_at,request_id) VALUES($1,2,$2,'transitioned','new','resolved','reason','{"kind":"clarified","withdrawal":null,"replacement":null,"duplicateOf":null}','{"kind":"clarified","withdrawal":null,"replacement":null,"duplicateOf":null}',clock_timestamp(),'invalid-transition')`, id, f.ids["reviewer_a"])
	if e == nil {
		t.Fatal("new to resolved")
	}
}
func TestFeedbackSchemaSourceProof(t *testing.T) {
	f := newFeedbackFixture(t)
	if countFeedbackTables(t, f.db) != 4 {
		t.Fatal("four tables")
	}
	run := func(t *testing.T, inOwner string, modify func()) {
		t.Helper()
		tx, e := f.db.Begin()
		if e != nil {
			t.Fatal(e)
		}
		defer tx.Rollback()
		in := f.feedbackInput()
		if modify != nil {
			modify()
		}
		if inOwner == "bad-sha" {
			c := *in.Target.Identity
			c.SHA256 = strings.Repeat("0", 64)
			in.Target.Identity = &c
			inOwner = f.ids["learner_a"]
		}
		if inOwner == "bad-pub" {
			p := f.ID()
			in.Source.PublicationID = &p
			inOwner = f.ids["learner_a"]
		}
		if inOwner == "bad-part" {
			c := *in.Target.Identity
			in.Target.Part = &feedback.Part{Kind: "unit", Unit: &c}
			inOwner = f.ids["learner_a"]
		}
		e = f.insertFeedback(tx, f.ID(), inOwner, in, true)
		if e == nil {
			e = tx.Commit()
		}
		if e == nil {
			t.Fatal("fake source accepted")
		}
	}
	for _, bad := range []string{"bad-sha", "bad-pub", "bad-part", f.ID()} {
		t.Run(bad[:7], func(t *testing.T) { run(t, bad, nil) })
	}
	// The schema also proves private item ownership, not just a frontend assertion.
	aid := f.ID()
	tx, _ := f.db.Begin()
	if e := f.insertAssessment(tx, aid, f.ids["learner_a"], 5); e != nil {
		t.Fatal(e)
	}
	if e := tx.Commit(); e != nil {
		t.Fatal(e)
	}
	in := f.feedbackInput()
	pos := 1
	in.Target.Kind = "instance"
	in.Target.Identity = &f.items[0].Instance
	in.Source = feedback.Source{Kind: "assessment", AttemptID: &aid, Position: &pos}
	tx, _ = f.db.Begin()
	defer tx.Rollback()
	e := f.insertFeedback(tx, f.ID(), f.ids["learner_b"], in, true)
	if e == nil {
		e = tx.Commit()
	}
	if e == nil {
		t.Fatal("private source owner")
	}
}
