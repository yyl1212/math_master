package store_test

import (
	"context"
	"testing"
	"time"
)

// Frozen pre-fix query provides a differential check on small, hand-counted data.
const overviewBeforeAggregation = `SELECT count(*),count(*) FILTER(WHERE r.state='completed'),count(*) FILTER(WHERE r.state='learning'),count(*) FILTER(WHERE r.state='reviewing'),count(*) FILTER(WHERE NOT EXISTS(SELECT 1 FROM managed_knowledge_topics t WHERE t.internal_id=k.internal_id AND t.active)) FROM managed_knowledge k LEFT JOIN managed_study_records r ON r.knowledge_id=k.internal_id AND r.owner_user_id=$1 WHERE k.published AND k.deleted_at IS NULL`

func TestManagedOverviewAggregationEquivalent(t *testing.T) {
	f := newKnowledgeAdminFixture(t)
	k := f.PreparedManagedKnowledge()
	f.exec(`INSERT INTO managed_knowledge(internal_id,external_id,point,public_sources,content_sha256,published,deleted_at,created_by,updated_by)
SELECT 'k-'||lpad(to_hex(i),56,'0'),'overview-'||i,jsonb_set(k.point,'{id}',to_jsonb(('overview-'||i)::text)),k.public_sources,k.content_sha256,i<=6,CASE WHEN i=8 THEN clock_timestamp() END,k.created_by,k.updated_by
FROM generate_series(1,8)i CROSS JOIN managed_knowledge k WHERE k.internal_id=$1`, k.ID)
	f.exec(`INSERT INTO managed_knowledge_topics(internal_id,external_id,topic_key,active)
SELECT 'k-'||lpad(to_hex(i),56,'0'),'overview-'||i,topic,NOT(i=4 OR (i=6 AND topic='97F50'))
FROM generate_series(1,8)i CROSS JOIN unnest(ARRAY['97F40','97F50','97A20'])topic
WHERE (i IN(1,3) OR topic='97F40' OR (i=6 AND topic='97F50')) AND i<>5`)
	f.exec(`INSERT INTO managed_study_records(owner_user_id,knowledge_id,state)
SELECT $1,'k-'||lpad(to_hex(i),56,'0'),CASE i WHEN 1 THEN 'unlearned' WHEN 2 THEN 'learning' WHEN 3 THEN 'completed' WHEN 4 THEN 'reviewing' WHEN 5 THEN 'unlearned' WHEN 6 THEN 'learning' WHEN 7 THEN 'completed' ELSE 'reviewing' END FROM generate_series(1,8)i`, f.ids["learner_a"])
	f.exec(`INSERT INTO managed_study_records(owner_user_id,knowledge_id,state) SELECT $1,'k-'||lpad(to_hex(i),56,'0'),'completed' FROM generate_series(1,8)i`, f.ids["learner_b"])
	for _, test := range []struct {
		actor string
		want  [6]int
	}{
		{"learner_a", [6]int{8, 1, 2, 1, 2, 2}},
		{"learner_b", [6]int{8, 6, 0, 0, 2, 2}},
		{"admin_a", [6]int{8, 0, 0, 0, 2, 0}},
	} {
		t.Run(test.actor, func(t *testing.T) {
			got, err := f.repo.ReadManagedOverview(f.ctx, f.Access(test.actor, "overview-equivalent"))
			if err != nil {
				t.Fatal(err)
			}
			counts := [6]int{got.Total, got.Completed, got.Learning, got.Reviewing, got.Unclassified, got.Unavailable}
			if counts != test.want || got.ActorID != f.ids[test.actor] || got.MaterialChanged != 0 || len(got.Reminders) != 0 {
				t.Fatalf("counts=%v want=%v overview=%+v", counts, test.want, got)
			}
			var before [5]int
			if err := f.db.QueryRowContext(f.ctx, overviewBeforeAggregation, f.ids[test.actor]).Scan(&before[0], &before[1], &before[2], &before[3], &before[4]); err != nil {
				t.Fatal(err)
			}
			if before != [5]int{got.Total, got.Completed, got.Learning, got.Reviewing, got.Unclassified} {
				t.Fatalf("changed original statistics: %v vs %v", before, counts)
			}
		})
	}
	f.exec(`UPDATE managed_knowledge SET published=false`)
	got, err := f.repo.ReadManagedOverview(f.ctx, f.Access("learner_a", "overview-empty"))
	if err != nil || got.Total != 0 || got.Completed != 0 || got.Learning != 0 || got.Reviewing != 0 || got.Unclassified != 0 || got.Unavailable != 8 {
		t.Fatal("empty public catalogue", got, err)
	}
}

func TestManagedOverviewCapacity10000(t *testing.T) {
	f := newKnowledgeAdminFixture(t)
	k := f.PreparedManagedKnowledge()
	// Fixture construction is outside the read budget; each knowledge has several
	// active memberships, so a per-row existence scan cannot hide at small scale.
	f.exec(`INSERT INTO managed_knowledge(internal_id,external_id,point,public_sources,content_sha256,published,created_by,updated_by)
SELECT 'k-'||lpad(to_hex(i),56,'0'),'overview-'||i,jsonb_set(k.point,'{id}',to_jsonb(('overview-'||i)::text)),k.public_sources,k.content_sha256,true,k.created_by,k.updated_by
FROM generate_series(1,9998)i CROSS JOIN managed_knowledge k WHERE k.internal_id=$1`, k.ID)
	f.exec(`INSERT INTO managed_knowledge_topics(internal_id,external_id,topic_key)
SELECT 'k-'||lpad(to_hex(i),56,'0'),'overview-'||i,topic FROM generate_series(1,9998)i CROSS JOIN unnest(ARRAY['97F40','97F50','97A20','97B50','97C30'])topic`)
	f.exec(`ANALYZE managed_knowledge`)
	f.exec(`ANALYZE managed_knowledge_topics`)
	ctx, cancel := context.WithTimeout(f.ctx, 2*time.Second)
	defer cancel()
	began := time.Now()
	got, err := f.repo.ReadManagedOverview(ctx, f.Access("learner_a", "overview-capacity"))
	if err != nil {
		t.Fatalf("10000-point overview failed after %s (deadline=%v): %v", time.Since(began), ctx.Err(), err)
	}
	if got.Total != 10000 || got.Completed != 0 || got.Learning != 0 || got.Reviewing != 0 || got.Unclassified != 0 {
		t.Fatal("multiple memberships changed counts", got)
	}
	t.Logf("10000知识、49990条新增关联：概览读取%s", time.Since(began))
}
