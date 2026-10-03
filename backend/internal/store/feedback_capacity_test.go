package store_test

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/yyl1212/math_master/backend/internal/assessment"
	"github.com/yyl1212/math_master/backend/internal/feedback"
	"github.com/yyl1212/math_master/backend/internal/store"
)

func TestFeedbackCapacity(t *testing.T) {
	t.Run("historical_volume", func(t *testing.T) {
		started := time.Now()
		f := newFeedbackFixture(t)
		ctx, cancel := context.WithTimeout(f.ctx, 4*time.Minute-time.Since(started))
		defer cancel()
		tx, err := f.db.BeginTx(ctx, nil)
		if err != nil {
			t.Fatal(err)
		}
		defer tx.Rollback()
		// These are historical query fixtures, not 1,000 quota-bypassing HTTP writes.
		// Every immutable-source, event and deferred projection guard remains enabled.
		_, err = tx.ExecContext(ctx, `CREATE TEMP TABLE feedback_history_ids ON COMMIT DROP AS
   SELECT gen_random_uuid() AS id,n,CASE WHEN n%2=1 THEN $1::uuid ELSE $2::uuid END AS owner,
    date_trunc('microseconds',clock_timestamp()-interval '2 days') AS recorded_at,
    CASE WHEN n=1 THEN 100 WHEN n BETWEEN 2 AND 91 THEN 9 ELSE 10 END AS events
   FROM generate_series(1,1000) n`, f.ids["learner_a"], f.ids["learner_b"])
		if err != nil {
			t.Fatal(err)
		}
		// Give every ticket and every event the same database microsecond.
		_, err = tx.ExecContext(ctx, `UPDATE feedback_history_ids SET recorded_at=(SELECT recorded_at FROM feedback_history_ids WHERE n=1)`)
		if err != nil {
			t.Fatal(err)
		}
		in := f.feedbackInput()
		_, err = tx.ExecContext(ctx, `INSERT INTO feedback_tickets(id,owner_user_id,target,source,original_title,original_location,category,status,sequence,created_at,updated_at)
   SELECT id,owner,$1,$2,'title-answer-sentinel','location-answer-sentinel','math_error','new',1,recorded_at,recorded_at FROM feedback_history_ids`, feedbackJSON(in.Target), feedbackJSON(in.Source))
		if err != nil {
			t.Fatal(err)
		}
		_, err = tx.ExecContext(ctx, `DO $$ DECLARE r record;s integer;msg text; BEGIN
   FOR r IN SELECT * FROM feedback_history_ids ORDER BY n LOOP
    msg:=CASE WHEN r.n=1 THEN repeat('中',4000) ELSE 'history-answer-sentinel' END;
    INSERT INTO feedback_events(ticket_id,sequence,actor_user_id,kind,to_status,message,recorded_at,request_id)
     VALUES(r.id,1,r.owner,'created','new',msg,r.recorded_at,'capacity-history');
    FOR s IN 2..r.events LOOP
     INSERT INTO feedback_events(ticket_id,sequence,actor_user_id,kind,from_status,to_status,message,recorded_at,request_id)
      VALUES(r.id,s,r.owner,'replied','new','new',msg,r.recorded_at,'capacity-history');
     UPDATE feedback_tickets SET sequence=s,updated_at=r.recorded_at WHERE id=r.id;
    END LOOP;
   END LOOP;END $$`)
		if err != nil {
			t.Fatal(err)
		}
		var longID string
		if err = tx.QueryRowContext(ctx, `SELECT id::text FROM feedback_history_ids WHERE n=1`).Scan(&longID); err != nil {
			t.Fatal(err)
		}
		if err = tx.Commit(); err != nil {
			t.Fatal(err)
		}
		initDuration := time.Since(started)
		if initDuration >= 4*time.Minute {
			t.Fatal("initialization deadline", initDuration)
		}
		cancel()
		deadline, bounded := t.Deadline()
		if !bounded {
			t.Fatal("capacity verification requires the original Go test deadline")
		}
		validation, stop := context.WithDeadline(f.ctx, deadline)
		defer stop()
		f.ctx = validation
		tickets, events := f.count(`SELECT count(*) FROM feedback_tickets`), f.count(`SELECT count(*) FROM feedback_events`)
		if tickets != 1000 || events != 10000 {
			t.Fatal("historical volume", tickets, events)
		}
		var largest time.Duration
		measured := func(name string, began time.Time, out any, err error, metadata bool) {
			t.Helper()
			elapsed := time.Since(began)
			if elapsed > largest {
				largest = elapsed
			}
			if err != nil {
				t.Fatal(name, err)
			}
			raw, e := json.Marshal(out)
			if e != nil || len(raw) > feedback.MaxResponseBytes {
				t.Fatal(name, "response bound", len(raw), e)
			}
			if metadata && (strings.Contains(string(raw), "answer-sentinel") || strings.Contains(string(raw), "message\"") || strings.Contains(string(raw), "location\"")) {
				t.Fatal(name, "metadata text leak")
			}
			t.Logf("%s: %s, %d bytes", name, elapsed, len(raw))
		}
		owners := map[string]string{}
		rows, err := f.db.Query(`SELECT id::text,owner_user_id::text FROM feedback_tickets`)
		if err != nil {
			t.Fatal(err)
		}
		for rows.Next() {
			var id, owner string
			if err = rows.Scan(&id, &owner); err != nil {
				t.Fatal(err)
			}
			owners[id] = owner
		}
		err = rows.Err()
		rows.Close()
		if err != nil {
			t.Fatal(err)
		}
		var crossCursor string
		for _, scenario := range []struct {
			actor  string
			review bool
			want   int
		}{{"learner_a", false, 500}, {"learner_b", false, 500}, {"reviewer_a", true, 1000}} {
			seen := map[string]bool{}
			q := feedback.ListQuery{Limit: 50}
			for {
				began := time.Now()
				page, e := f.repo.ListFeedbackTickets(f.ctx, f.Access(scenario.actor, false), scenario.review, q)
				measured("list "+scenario.actor, began, page, e, true)
				if len(page.Data.Items) != 50 {
					t.Fatal("bounded page", len(page.Data.Items))
				}
				for _, m := range page.Data.Items {
					if seen[m.ID] || !scenario.review && owners[m.ID] != f.ids[scenario.actor] {
						t.Fatal("pagination duplicate or foreign ticket")
					}
					seen[m.ID] = true
				}
				if page.Data.NextCursor == nil {
					break
				}
				if crossCursor == "" {
					crossCursor = *page.Data.NextCursor
				}
				q.Cursor = *page.Data.NextCursor
			}
			if len(seen) != scenario.want {
				t.Fatal("pagination count", scenario.actor, len(seen))
			}
		}
		began := time.Now()
		foreign, e := f.repo.ListFeedbackTickets(f.ctx, f.Access("learner_b", false), false, feedback.ListQuery{Limit: 50, Cursor: crossCursor})
		measured("cross-actor boundary", began, foreign, e, true)
		for _, m := range foreign.Data.Items {
			if owners[m.ID] != f.ids["learner_b"] {
				t.Fatal("cursor expanded permissions")
			}
		}
		var priorTime time.Time
		seq := int64(0)
		q := feedback.ListQuery{Limit: 50}
		for {
			began = time.Now()
			page, e := f.repo.ReadFeedbackEvents(f.ctx, f.Access("learner_a", false), longID, false, q)
			measured("discussion", began, page, e, false)
			if len(page.Data.Items) != 50 || len(feedbackJSON(page)) < 400000 {
				t.Fatal("largest discussion page", len(page.Data.Items))
			}
			for _, event := range page.Data.Items {
				seq++
				if event.Sequence != seq || !priorTime.IsZero() && !event.RecordedAt.Equal(priorTime) {
					t.Fatal("same-microsecond pagination")
				}
				priorTime = event.RecordedAt
			}
			if page.Data.NextCursor == nil {
				break
			}
			q.Cursor = *page.Data.NextCursor
		}
		if seq != 100 {
			t.Fatal("discussion count", seq)
		}
		key := f.Access("learner_a", false)
		reply := feedback.ReplyInput{ExpectedSequence: 100, Message: "Additional evidence on a long thread"}
		began = time.Now()
		original, e := f.repo.ReplyFeedback(f.ctx, key, longID, reply)
		measured("reply beyond 50", began, original, e, true)
		start := make(chan struct{})
		done := make(chan error, 2)
		for _, actor := range []string{"reviewer_a", "reviewer_b"} {
			access := f.Access(actor, false)
			go func() {
				<-start
				_, e := f.repo.TransitionFeedback(f.ctx, access, longID, feedback.TransitionInput{ExpectedSequence: 101, Status: feedback.Processing, Message: "Concurrent handling"})
				done <- e
			}()
		}
		began = time.Now()
		close(start)
		successes := 0
		for n := 0; n < 2; n++ {
			e := <-done
			if e == nil {
				successes++
			} else if !errors.Is(e, feedback.ErrConflict) {
				t.Fatal(e)
			}
		}
		measured("concurrent transition", began, struct{}{}, nil, true)
		if successes != 1 {
			t.Fatal("status race", successes)
		}
		began = time.Now()
		resolved, e := f.repo.TransitionFeedback(f.ctx, f.Access("reviewer_a", false), longID, feedback.TransitionInput{ExpectedSequence: 102, Status: feedback.Resolved, Message: "Explanation clarified", Resolution: &feedback.Resolution{Kind: "clarified"}})
		measured("resolve long thread", began, resolved, e, true)
		began = time.Now()
		reopened, e := f.repo.ReplyFeedback(f.ctx, f.Access("learner_a", false), longID, feedback.ReplyInput{ExpectedSequence: 103, Message: "A remaining ambiguity"})
		measured("reopen", began, reopened, e, true)
		eventCount, rateCount := f.count(`SELECT count(*) FROM feedback_events`), f.count(`SELECT count(*) FROM feedback_rate_limits`)
		began = time.Now()
		replay, e := f.repo.ReplyFeedback(f.ctx, key, longID, reply)
		measured("original receipt after advancement", began, replay, e, true)
		if feedbackJSON(replay) != feedbackJSON(original) || replay.Data.Ticket.Sequence != 101 || f.count(`SELECT sequence FROM feedback_tickets WHERE id=$1`, longID) != 104 || f.count(`SELECT count(*) FROM feedback_events`) != eventCount || f.count(`SELECT count(*) FROM feedback_rate_limits`) != rateCount {
			t.Fatal("replay changed current projection or quota")
		}
		if reopened.Data.Ticket.Status != feedback.Processing || reopened.Data.Ticket.ResolutionKind != nil {
			t.Fatal("reopen state")
		}
		began = time.Now()
		closed, e := f.repo.TransitionFeedback(f.ctx, f.Access("reviewer_a", false), longID, feedback.TransitionInput{ExpectedSequence: 104, Status: feedback.Closed, Message: "Recorded explanation", Resolution: &feedback.Resolution{Kind: "suggestion_recorded"}})
		measured("close long thread", began, closed, e, true)
		m := f.instanceFeedback()
		exposureBefore := f.exposureSequence("reviewer_b")
		began = time.Now()
		delivered, e := f.repo.ReadFeedbackEvents(f.ctx, f.Access("reviewer_b", false), m.ID, true, feedback.ListQuery{Limit: 50})
		measured("private discussion with committed exposure", began, delivered, e, false)
		if f.exposureSequence("reviewer_b") <= exposureBefore {
			t.Fatal("exposure not committed")
		}
		for _, query := range []struct {
			name, sql string
			args      []any
		}{
			{"owner", `EXPLAIN (ANALYZE,BUFFERS) SELECT id::text FROM feedback_tickets WHERE ($1::boolean OR owner_user_id=$2) AND ($3='' OR status=$3) AND ($4='' OR category=$4) AND ($5::timestamptz IS NULL OR (created_at,id)<($5::timestamptz,$6::uuid)) ORDER BY created_at DESC,id DESC LIMIT $7`, []any{false, f.ids["learner_a"], "", "", nil, nil, 51}},
			{"review", `EXPLAIN (ANALYZE,BUFFERS) SELECT id::text FROM feedback_tickets WHERE ($1::boolean OR owner_user_id=$2) AND ($3='' OR status=$3) AND ($4='' OR category=$4) AND ($5::timestamptz IS NULL OR (created_at,id)<($5::timestamptz,$6::uuid)) ORDER BY created_at DESC,id DESC LIMIT $7`, []any{true, f.ids["reviewer_a"], "new", "math_error", nil, nil, 51}},
			{"events", `EXPLAIN (ANALYZE,BUFFERS) SELECT sequence,kind,CASE WHEN actor_user_id=$2 THEN 'submitter' ELSE 'review_team' END,from_status,to_status,message,resolution,recorded_at FROM feedback_events WHERE ticket_id=$1 AND sequence>$3 ORDER BY sequence LIMIT $4`, []any{longID, f.ids["learner_a"], 50, 51}},
		} {
			rows, e := f.db.Query(query.sql, query.args...)
			if e != nil {
				t.Fatal(e)
			}
			for rows.Next() {
				var line string
				if e = rows.Scan(&line); e != nil {
					t.Fatal(e)
				}
				t.Log(query.name + " plan: " + line)
			}
			e = rows.Err()
			rows.Close()
			if e != nil {
				t.Fatal(e)
			}
		}
		if largest >= 8*time.Second {
			t.Fatal("operation deadline", largest)
		}
		t.Logf("tickets=%d events=%d init=%s largest_operation=%s status_race_successes=%d", tickets, events, initDuration, largest, successes)
	})
	t.Run("missing_feedback_migration_keeps_old_learning", func(t *testing.T) {
		f := newFeedbackFixture(t)
		if e := tryDownSeven(t, f.db); e != nil {
			t.Fatal(e)
		}
		if countFeedbackTables(t, f.db) != 0 {
			t.Fatal("empty Down incomplete")
		}
		if _, e := f.repo.StartLearning(f.ctx, f.Access("learner_a", false), f.knowledge.ID, f.startInput()); e != nil {
			t.Fatal("old learning unavailable without feedback migration", e)
		}
		page, e := f.repo.ListFeedbackTickets(f.ctx, f.Access("learner_a", false), false, feedback.ListQuery{})
		if !errors.Is(e, feedback.ErrNotConfigured) || page.ActorID != "" {
			t.Fatal("missing migration must fail closed", e)
		}
		before := f.count(`SELECT count(*) FROM learning_events`)
		if e = store.Up(f.ctx, f.db, "../../../db/migrations"); e != nil {
			t.Fatal(e)
		}
		if countFeedbackTables(t, f.db) != 4 || f.count(`SELECT count(*) FROM learning_events`) != before {
			t.Fatal("restore changed old learning")
		}
		if _, e = f.repo.ReadFeedbackContext(f.ctx, f.Access("learner_a", false), feedback.ContextQuery{Kind: "site", Area: "home"}); e != nil {
			t.Fatal("feedback restore", e)
		}
	})
	t.Run("nonempty_learning_facts_unchanged", func(t *testing.T) {
		f := newFeedbackFixture(t)
		f.publishLearningGraph()
		if _, e := f.repo.StartLearning(f.ctx, f.Access("learner_a", false), f.knowledge.ID, f.startInput()); e != nil {
			t.Fatal(e)
		}
		if _, e := f.repo.CompleteLearning(f.ctx, f.Access("learner_a", false), f.knowledge.ID, f.completeInput()); e != nil {
			t.Fatal(e)
		}
		fact := f.passedFixture("learner_a", assessment.ModeDiagnostic)
		if _, e := f.repo.LearningApplyForTest(f.ctx, f.Access("learner_a", false), fact); e != nil {
			t.Fatal(e)
		}
		for _, table := range []string{"learning_events", "assessment_answers", "assessment_results", "learning_qualification_events", "learning_unlocks"} {
			if f.count(`SELECT count(*) FROM `+table) == 0 {
				t.Fatal("empty learning evidence", table)
			}
		}
		snapshot := func() string {
			t.Helper()
			var raw string
			e := f.db.QueryRow(`SELECT jsonb_build_array(
   (SELECT jsonb_agg(to_jsonb(x) ORDER BY x.id) FROM learning_events x),
   (SELECT jsonb_agg(to_jsonb(x) ORDER BY x.attempt_id,x.position) FROM assessment_answers x),
   (SELECT jsonb_agg(to_jsonb(x) ORDER BY x.attempt_id) FROM assessment_results x),
   (SELECT jsonb_agg(to_jsonb(x) ORDER BY x.id) FROM learning_qualification_events x),
   (SELECT jsonb_agg(to_jsonb(x) ORDER BY x.owner_user_id,x.knowledge_id) FROM learning_unlocks x))::text`).Scan(&raw)
			if e != nil {
				t.Fatal(e)
			}
			return raw
		}
		before := snapshot()
		m := f.feedbackCreate("learner_a", f.feedbackInput())
		m = f.feedbackTransition("reviewer_a", m, feedback.Processing, nil)
		f.feedbackTransition("reviewer_b", m, feedback.Resolved, &feedback.Resolution{Kind: "clarified"})
		if snapshot() != before {
			t.Fatal("handling altered existing learning facts")
		}
	})
}
