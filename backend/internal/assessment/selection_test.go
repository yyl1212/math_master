package assessment

import (
	"context"
	"fmt"
	"github.com/yyl1212/math_master/backend/internal/question"
	"reflect"
	"testing"
	"time"
)

func pool(n int) []Candidate {
	out := make([]Candidate, n)
	for i := range out {
		out[i] = Candidate{Identity: testIdentity(fmt.Sprintf("item-%d", i)), Coverage: []int{0}}
	}
	return out
}
func TestLearningSelectionUnseenCoverageAndSeed(t *testing.T) {
	cs := pool(6)
	cs[0].Seen = true
	cs[0].Coverage = []int{1}
	cs[1].Seen = true
	cs[2].Coverage = []int{0, 1}
	var seed [32]byte
	got, ok, err := SelectFive(context.Background(), []int{0, 1}, cs, seed)
	if err != nil || !ok || len(got) != 5 {
		t.Fatal(got, ok, err)
	}
	found := map[question.Identity]bool{}
	unseen := 0
	for _, id := range got {
		if found[id] {
			t.Fatal("repeated")
		}
		found[id] = true
		for _, c := range cs {
			if c.Identity == id && !c.Seen {
				unseen++
			}
		}
	}
	if unseen != 4 {
		t.Fatal("not maximum unseen", got)
	}
	again, _, _ := SelectFive(context.Background(), []int{0, 1}, cs, seed)
	if !reflect.DeepEqual(got, again) {
		t.Fatal("unstable seed")
	}
	for i, j := 0, len(cs)-1; i < j; i, j = i+1, j-1 {
		cs[i], cs[j] = cs[j], cs[i]
	}
	again, _, _ = SelectFive(context.Background(), []int{0, 1}, cs, seed)
	if !reflect.DeepEqual(got, again) {
		t.Fatal("depends on input order")
	}
}
func TestLearningSelectionLimitsAndCancellation(t *testing.T) {
	cs := pool(1000)
	for i := range cs {
		cs[i].Coverage = []int{0, 1, 2, 3, 4, 5, 6, 7}
	}
	got, ok, e := SelectFive(context.Background(), []int{0, 1, 2, 3, 4, 5, 6, 7}, cs, [32]byte{})
	if e != nil || !ok || len(got) != 5 {
		t.Fatal(ok, e)
	}
	cs = append(cs, Candidate{Identity: testIdentity("extra"), Coverage: []int{0}})
	if _, _, e = SelectFive(context.Background(), []int{0}, cs, [32]byte{}); e == nil {
		t.Fatal("1001 accepted")
	}
	if _, _, e = SelectFive(context.Background(), []int{0, 1, 2, 3, 4, 5, 6, 7, 8}, pool(5), [32]byte{}); e == nil {
		t.Fatal("9 goals accepted")
	}
	dup := pool(5)
	dup[4].Identity = dup[0].Identity
	if _, _, e = SelectFive(context.Background(), []int{0}, dup, [32]byte{}); e == nil {
		t.Fatal("duplicate identity accepted")
	}
	if _, ok, e = SelectFive(context.Background(), []int{1}, pool(5), [32]byte{}); e != nil || ok {
		t.Fatal("infeasible cover", ok, e)
	}
	ctx, c := context.WithCancel(context.Background())
	c()
	if _, _, e = SelectFive(ctx, []int{0}, pool(1000), [32]byte{}); e != context.Canceled {
		t.Fatal("cancellation", e)
	}
}
func TestLearningRetryAtRealCoverageBoundary(t *testing.T) {
	now := time.Date(2026, 10, 2, 1, 0, 0, 0, time.UTC)
	cs := pool(5)
	past := now.Add(-25 * time.Minute)
	cs[4].ExposedAt = &past
	at, e := EarliestReady(context.Background(), []int{0}, cs, now, [32]byte{})
	if e != nil || at == nil || !at.Equal(now.Add(5*time.Minute)) {
		t.Fatal(at, e)
	}
	at, e = EarliestReady(context.Background(), []int{0}, cs, now.Add(5*time.Minute), [32]byte{})
	if e != nil || at != nil {
		t.Fatal("already ready should not advise waiting", at, e)
	}
	cs[0].RecentSubmitted = true
	at, e = EarliestReady(context.Background(), []int{0}, cs, now, [32]byte{})
	if e != nil || at != nil {
		t.Fatal("recent submitted never expires", at, e)
	}
}
