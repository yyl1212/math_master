package store_test

import (
	"github.com/yyl1212/math_master/backend/internal/study"
	"testing"
	"time"
)

func TestStudyHistoryEqualTimestampCursor(t *testing.T) {
	f := newStudyFixture(t)
	f.begin()
	f.complete(1)
	round, e := f.repo.StartStudyReview(f.ctx, f.Access("author_a"), f.Ref().ID, f.Command(2))
	if e != nil {
		t.Fatal(e)
	}
	if _, e = f.repo.FinishStudyReview(f.ctx, f.Access("author_a"), f.Ref().ID, study.ReviewInput{CommandInput: f.Command(3), ReviewID: *round.Record.ActiveReviewID}); e != nil {
		t.Fatal(e)
	}
	f.exec("ALTER TABLE study_events DISABLE TRIGGER study_event_immutable")
	f.exec("UPDATE study_events SET recorded_at='2026-10-06T10:00:00Z'")
	f.exec("ALTER TABLE study_events ENABLE TRIGGER study_event_immutable")
	seen := map[string]bool{}
	cursor := ""
	for i := 0; i < 5; i++ {
		page, e := f.repo.ListStudyHistory(f.ctx, f.Access("author_a"), study.HistoryQuery{Limit: 1, Cursor: cursor})
		if e != nil {
			t.Fatal(e)
		}
		for _, item := range page.Items {
			if seen[item.ID] {
				t.Fatal("duplicate equal-time event")
			}
			seen[item.ID] = true
		}
		if page.NextCursor == nil {
			break
		}
		cursor = *page.NextCursor
	}
	if len(seen) != 4 {
		t.Fatal("equal-time history lost", len(seen))
	}
	if page, e := f.repo.ListStudyHistory(f.ctx, f.Access("author_b"), study.HistoryQuery{}); e != nil || len(page.Items) != 0 {
		t.Fatal("other history leaked", page, e)
	}
	from := time.Date(2026, 10, 6, 10, 0, 0, 0, time.UTC)
	to := from.Add(time.Second)
	p, e := f.repo.ListStudyHistory(f.ctx, f.Access("author_a"), study.HistoryQuery{From: &from, To: &to, TopicID: "msc-00"})
	if e != nil || len(p.Items) != 4 {
		t.Fatal("UTC/current-topic filter", p, e)
	}
}
