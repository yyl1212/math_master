package studyreview

import (
	"strings"
	"testing"
	"time"
)

func testRef(ch string) Ref {
	return Ref{ID: "k-" + strings.Repeat("a", 56), ContentSHA256: strings.Repeat(ch, 64), SourceKind: "managed"}
}

// 捕获错误间隔、错误阶段和把本地午夜当排期时间的回归。
func TestNextScheduleExactIntervals(t *testing.T) {
	now := time.Date(2026, 10, 14, 13, 14, 15, 123000000, time.FixedZone("UTC+8", 28800))
	for stage := 0; stage <= 5; stage++ {
		for _, rating := range []Rating{Forgotten, Uncertain, Familiar} {
			wantStage, wantSeconds := 0, int64(600)
			if rating == Uncertain {
				wantStage = stage - 1
				if wantStage < 0 {
					wantStage = 0
				}
				wantSeconds = 86400
			}
			if rating == Familiar {
				wantStage = stage + 1
				if wantStage > 5 {
					wantStage = 5
				}
				wantSeconds = []int64{0, 86400, 259200, 604800, 1209600, 2592000}[wantStage]
			}
			got, err := NextSchedule(&Schedule{Stage: stage, DueAt: now, LastRef: testRef("b"), Sequence: 9}, testRef("b"), rating, now)
			if err != nil || got.Stage != wantStage || got.DueAt.Sub(now) != time.Duration(wantSeconds)*time.Second || got.Sequence != 10 || got.LastRating != rating || got.LastRef != testRef("b") || !got.GradedAt.Equal(now) {
				t.Fatalf("stage %d rating %s: %+v %v", stage, rating, got, err)
			}
		}
	}
	first, e := NextSchedule(nil, testRef("b"), Familiar, now)
	if e != nil || first.Stage != 1 || first.Sequence != 1 || first.DueAt.Sub(now) != 24*time.Hour {
		t.Fatalf("first: %+v %v", first, e)
	}
}
func TestEarlyFamiliarDoesNotAdvance(t *testing.T) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	due := now.Add(3 * time.Hour)
	s, e := NextSchedule(&Schedule{Stage: 4, DueAt: due, LastRef: testRef("b"), Sequence: 1}, testRef("b"), Familiar, now)
	if e != nil || s.Stage != 4 || !s.DueAt.Equal(due) || s.Sequence != 2 {
		t.Fatalf("early: %+v %v", s, e)
	}
	for _, r := range []Rating{Forgotten, Uncertain} {
		s, e = NextSchedule(&Schedule{Stage: 4, DueAt: due, LastRef: testRef("b"), Sequence: 1}, testRef("b"), r, now)
		if e != nil || s.DueAt.Equal(due) {
			t.Fatalf("early %s failed", r)
		}
	}
}
func TestChangedRefResetsEffectiveStage(t *testing.T) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	for _, r := range []Rating{Forgotten, Uncertain, Familiar} {
		got, e := NextSchedule(&Schedule{Stage: 5, DueAt: now.Add(30 * 24 * time.Hour), LastRef: testRef("a"), Sequence: 4}, testRef("b"), r, now)
		want := 0
		seconds := 86400
		if r == Familiar {
			want = 1
		}
		if r == Forgotten {
			seconds = 600
		}
		if e != nil || got.Stage != want || got.DueAt.Sub(now) != time.Duration(seconds)*time.Second {
			t.Fatalf("changed %s: %+v %v", r, got, e)
		}
	}
	b, e := NextSchedule(&Schedule{Stage: 5, DueAt: now, LastRef: testRef("a"), Sequence: 4}, testRef("b"), Uncertain, now)
	if e != nil {
		t.Fatal(e)
	}
	c, e := NextSchedule(&b, testRef("b"), Familiar, b.DueAt)
	if e != nil || c.Stage != 1 || c.DueAt.Sub(b.DueAt) != 24*time.Hour {
		t.Fatalf("A5 B0 B1: %+v %v", c, e)
	}
}
func TestClockOverflow(t *testing.T) {
	for _, n := range []time.Time{time.Time{}, time.Date(9999, 12, 31, 23, 59, 59, 0, time.UTC), time.Date(10000, 1, 1, 0, 0, 0, 0, time.UTC)} {
		if _, e := NextSchedule(nil, testRef("b"), Familiar, n); e == nil {
			t.Fatalf("accepted %s", n)
		}
	}
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	for _, s := range []Schedule{{Stage: -1}, {Stage: 6}, {Stage: 1, Sequence: MaxSequence}} {
		if _, e := NextSchedule(&s, testRef("b"), Familiar, now); e == nil {
			t.Fatal("invalid previous accepted")
		}
	}
	if _, e := NextSchedule(nil, testRef("b"), Rating("correct"), now); e == nil {
		t.Fatal("automatic score accepted")
	}
}
