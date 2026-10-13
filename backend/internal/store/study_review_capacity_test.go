package store_test

import (
	"encoding/json"
	"fmt"
	"github.com/yyl1212/math_master/backend/internal/studyreview"
	"strings"
	"testing"
	"time"
)

// The volume setup is separate from timed repository reads/writes. Real actors
// and a normally imported source are reused; bulk data is isolated test history.
func reviewCapacity(t *testing.T) (*knowledgeAdminFixture, string) {
	t.Helper()
	f := newKnowledgeAdminFixture(t)
	k := f.PreparedManagedKnowledge()
	owner := f.ids["learner_a"]
	tx, e := f.db.BeginTx(f.ctx, nil)
	if e != nil {
		t.Fatal(e)
	}
	defer tx.Rollback()
	queries := []string{
		`CREATE TEMP TABLE review_capacity_ids ON COMMIT DROP AS SELECT i,'k-'||substr(md5('review-capacity-'||i)||md5('review-capacity-tail-'||i),1,56) kid,gen_random_uuid() sid,gen_random_uuid() rid FROM generate_series(1,10000)i`,
		`INSERT INTO managed_knowledge(internal_id,external_id,point,public_sources,content_sha256,published,created_by,updated_by) SELECT ids.kid,'review-capacity-'||i,jsonb_set(k.point,'{id}',to_jsonb(('review-capacity-'||i)::text)),k.public_sources,k.content_sha256,true,k.created_by,k.updated_by FROM review_capacity_ids ids CROSS JOIN managed_knowledge k WHERE k.internal_id=$1`,
		`INSERT INTO managed_knowledge_topics(external_id,topic_key,internal_id,active) SELECT 'review-capacity-'||i,t,kid,true FROM review_capacity_ids CROSS JOIN unnest(ARRAY['97F40','97F50','97A20','97B50','97C30']) t`,
		`INSERT INTO managed_study_records(owner_user_id,knowledge_id,state,sequence,first_started_at,first_completed_at,last_completed_at,completed_ref) SELECT $1,kid,'completed',2,clock_timestamp()-interval '2 days',clock_timestamp()-interval '1 day',clock_timestamp()-interval '1 day',jsonb_build_object('id',kid,'contentSha256',k.content_sha256,'sourceKind','managed') FROM review_capacity_ids ids JOIN managed_knowledge k ON k.internal_id=ids.kid`,
		`INSERT INTO study_review_sessions(session_id,owner_user_id,knowledge_id,review_id,knowledge_ref,mode,target,status,sequence,draft_revision,created_at,compared_at,graded_at,grade) SELECT sid,$1,kid,rid,jsonb_build_object('id',kid,'contentSha256',k.content_sha256,'sourceKind','managed'),'read','statement','completed',3,1,clock_timestamp()-interval '2 hours',clock_timestamp()-interval '1 hour',date_trunc('second',statement_timestamp()),jsonb_build_object('rating','familiar','gradedAt',date_trunc('second',statement_timestamp()),'schedule',jsonb_build_object('stage',1,'dueAt',date_trunc('second',statement_timestamp())+interval '1 day','lastRef',jsonb_build_object('id',kid,'contentSha256',k.content_sha256,'sourceKind','managed'),'lastRating','familiar','sequence',1,'gradedAt',date_trunc('second',statement_timestamp()))) FROM review_capacity_ids ids JOIN managed_knowledge k ON k.internal_id=ids.kid`,
		`INSERT INTO study_review_snapshots(session_id,owner_user_id,knowledge_id,title,title_zh,knowledge_type,prompt,expected,sha256) SELECT sid,$1,kid,k.point->>'title',k.point->>'title_zh',k.point->>'type','[]',jsonb_build_array(jsonb_build_object('kind','statement','items',jsonb_build_array(k.point->>'statement'))),repeat('b',64) FROM review_capacity_ids ids JOIN managed_knowledge k ON k.internal_id=ids.kid`,
		`INSERT INTO study_review_schedules(owner_user_id,knowledge_id,stage,due_at,last_ref,last_rating,sequence,graded_at,source_session_id) SELECT owner_user_id,knowledge_id,1,(grade->'schedule'->>'dueAt')::timestamptz,knowledge_ref,'familiar',1,graded_at,session_id FROM study_review_sessions WHERE owner_user_id=$1`,
		`INSERT INTO study_review_events(session_id,owner_user_id,knowledge_id,review_id,sequence,action,knowledge_ref,mode,target,grade,recorded_at) SELECT session_id,owner_user_id,knowledge_id,review_id,3,'graded',knowledge_ref,mode,target,grade,graded_at FROM study_review_sessions WHERE owner_user_id=$1`,
	}
	for i, q := range queries {
		var e error
		if i == 0 {
			_, e = tx.ExecContext(f.ctx, q)
		} else if i == 1 {
			_, e = tx.ExecContext(f.ctx, q, k.ID)
		} else if i == 2 {
			_, e = tx.ExecContext(f.ctx, q)
		} else {
			_, e = tx.ExecContext(f.ctx, q, owner)
		}
		if e != nil {
			t.Fatalf("capacity setup step %d: %v", i, e)
		}
	}
	var id string
	if e = tx.QueryRowContext(f.ctx, `SELECT kid FROM review_capacity_ids WHERE i=1`).Scan(&id); e != nil {
		t.Fatal(e)
	}
	if e = tx.Commit(); e != nil {
		t.Fatal(e)
	}
	return f, id
}
func TestReviewCapacityRead10000(t *testing.T) {
	f, _ := reviewCapacity(t)
	a := f.Access("learner_a", "capacity-read")
	began := time.Now()
	o, e := f.repo.ReadReviewOverview(f.ctx, a)
	elapsed := time.Since(began)
	if e != nil || o.Total != 10000 || o.Future != 10000 || elapsed > 8*time.Second {
		t.Fatal(o, e, elapsed)
	}
	t.Log("overview10000", elapsed)
	for _, offset := range []int{0, 50, 9950, 10000} {
		began = time.Now()
		p, e := f.repo.ListReviewQueue(f.ctx, a, studyreview.QueueQuery{Bucket: "all", TopicKey: "97-XX", Limit: 50, Offset: offset})
		elapsed = time.Since(began)
		want := 50
		if offset == 10000 {
			want = 0
		}
		if e != nil || p.Total != 10000 || len(p.Items) != want || elapsed > 8*time.Second {
			t.Fatal("queue", offset, e, elapsed)
		}
		raw, _ := json.Marshal(p)
		if len(raw) > 1048576 {
			t.Fatal("oversize page")
		}
		t.Log("queue", offset, elapsed)
	}
	h, e := f.repo.ListReviewHistory(f.ctx, a, studyreview.HistoryQuery{Limit: 50})
	if e != nil || len(h.Items) != 50 || h.NextCursor == nil {
		t.Fatal(e)
	}
}
func TestReviewCapacityWrite10000(t *testing.T) {
	f, id := reviewCapacity(t)
	k, e := f.repo.ReadCurrentKnowledge(f.ctx, id)
	if e != nil {
		t.Fatal(e)
	}
	began := time.Now()
	r, e := f.repo.StartReviewSession(f.ctx, f.Access("learner_a", "start-volume"), studyreview.StartInput{Knowledge: k.Ref, Mode: studyreview.Recall, Target: studyreview.Statement})
	if e != nil || time.Since(began) > 8*time.Second {
		t.Fatal(e)
	}
	body := strings.Repeat("😀", 16000)
	began = time.Now()
	c, e := f.repo.CompareReviewSession(f.ctx, f.Access("learner_a", "compare-volume"), r.SessionID, studyreview.CompareInput{ExpectedSequence: r.Sequence, Body: body})
	if e != nil || time.Since(began) > 8*time.Second {
		t.Fatal(e)
	}
	g, e := f.repo.GradeReviewSession(f.ctx, f.Access("learner_a", "grade-volume"), r.SessionID, studyreview.GradeInput{ExpectedSequence: c.Sequence, Rating: studyreview.Familiar})
	if e != nil || g.Grade.Schedule.Stage != 1 || g.Grade.Schedule.Sequence != 2 {
		t.Fatal(g, e)
	}
	s, e := f.repo.ReadReviewSession(f.ctx, f.Access("learner_a", "read-volume"), r.SessionID)
	if e != nil || s.Answer == nil || len(*s.Answer) != 64000 {
		t.Fatal("max unicode", e)
	}
	raw, _ := json.Marshal(s)
	if len(raw) > 1048576 {
		t.Fatal("max response", len(raw))
	}
	t.Log(fmt.Sprintf("10000记录最大合法答案响应%d字节", len(raw)))
}
