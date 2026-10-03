package store_test

import (
	"database/sql"
	"encoding/json"
	"github.com/yyl1212/math_master/backend/internal/feedback"
	"testing"
	"time"
)

type feedbackFixture struct{ *learningFixture }

func newFeedbackFixture(t *testing.T, initialMigration ...int) *feedbackFixture {
	t.Helper()
	return &feedbackFixture{newLearningFixture(t, initialMigration...)}
}
func (f *feedbackFixture) feedbackInput() feedback.CreateInput {
	head := f.KHead()
	return feedback.CreateInput{Target: feedback.Target{Kind: "knowledge", Identity: &f.knowledge}, Source: feedback.Source{Kind: "publication", PublicationID: head}, Category: "math_error", Title: "answer-sentinel", Message: "Original mathematical report\nwith a second line.", Location: "original location"}
}
func feedbackJSON(v any) string {
	b, e := json.Marshal(v)
	if e != nil {
		panic(e)
	}
	return string(b)
}
func (f *feedbackFixture) insertFeedback(tx *sql.Tx, id, owner string, in feedback.CreateInput, withEvent bool) error {
	var now time.Time
	if e := tx.QueryRow(`SELECT clock_timestamp()`).Scan(&now); e != nil {
		return e
	}
	_, e := tx.Exec(`INSERT INTO feedback_tickets(id,owner_user_id,target,source,original_title,original_location,category,status,sequence,created_at,updated_at) VALUES($1,$2,$3,$4,$5,$6,$7,'new',1,$8,$8)`, id, owner, feedbackJSON(in.Target), feedbackJSON(in.Source), in.Title, in.Location, in.Category, now)
	if e != nil || !withEvent {
		return e
	}
	_, e = tx.Exec(`INSERT INTO feedback_events(ticket_id,sequence,actor_user_id,kind,from_status,to_status,message,resolution,effective_resolution,recorded_at,request_id) VALUES($1,1,$2,'created',NULL,'new',$3,NULL,NULL,$4,'schema-fixture')`, id, owner, in.Message, now)
	return e
}
func (f *feedbackFixture) seededFeedback() string {
	f.t.Helper()
	id := f.ID()
	tx, e := f.db.Begin()
	if e != nil {
		f.t.Fatal(e)
	}
	defer tx.Rollback()
	if e = f.insertFeedback(tx, id, f.ids["learner_a"], f.feedbackInput(), true); e != nil {
		f.t.Fatal(e)
	}
	if e = tx.Commit(); e != nil {
		f.t.Fatal(e)
	}
	return id
}
func countFeedbackTables(t *testing.T, db *sql.DB) int {
	t.Helper()
	var n int
	e := db.QueryRow(`SELECT count(*) FROM information_schema.tables WHERE table_schema='public' AND table_name=ANY($1::text[])`, []string{"feedback_tickets", "feedback_events", "feedback_idempotency", "feedback_rate_limits"}).Scan(&n)
	if e != nil {
		t.Fatal(e)
	}
	return n
}
func (f *feedbackFixture) seedFeedbackReceipt(id string) {
	f.t.Helper()
	var m feedback.Metadata
	var raw []byte
	e := f.db.QueryRow(`SELECT id,target,category,status,sequence,created_at,updated_at FROM feedback_tickets WHERE id=$1`, id).Scan(&m.ID, &raw, &m.Category, &m.Status, &m.Sequence, &m.CreatedAt, &m.UpdatedAt)
	if e != nil {
		f.t.Fatal(e)
	}
	if json.Unmarshal(raw, &m.Target) != nil {
		f.t.Fatal("target")
	}
	m.Label = "knowledge workflow-fractions · v1"
	m.TargetValidity = "current"
	r := feedback.Receipt{Status: 201, Ticket: m}
	_, e = f.db.Exec(`INSERT INTO feedback_idempotency(actor_user_id,action,resource,key,request_sha256,ticket_id,event_sequence,receipt) VALUES($1,'create','tickets',$2,$3,$4,1,$5)`, f.ids["learner_a"], f.ID(), string(makeSHA('a')), id, feedbackJSON(r))
	if e != nil {
		f.t.Fatal(e)
	}
}
func makeSHA(ch byte) []byte {
	b := make([]byte, 64)
	for i := range b {
		b[i] = ch
	}
	return b
}
