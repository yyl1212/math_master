package question

import (
	"fmt"
	"math/rand"
	"testing"
	"time"
)

// bruteCover enumerates five combinations independently of the production bitmask/DP implementation.
func bruteCover(core []int, pool []CandidateCoverage) bool {
	for a := 0; a < len(pool); a++ {
		for b := a + 1; b < len(pool); b++ {
			for c := b + 1; c < len(pool); c++ {
				for d := c + 1; d < len(pool); d++ {
					for e := d + 1; e < len(pool); e++ {
						seen := map[int]bool{}
						for _, i := range []int{a, b, c, d, e} {
							for _, goal := range pool[i].ObjectiveIndices {
								seen[goal] = true
							}
						}
						ok := true
						for _, goal := range core {
							ok = ok && seen[goal]
						}
						if ok {
							return true
						}
					}
				}
			}
		}
	}
	return false
}
func TestFiveDistinctQuestionsCoverAllCore(t *testing.T) {
	core := []int{0, 1, 2, 3, 4, 5}
	trap := []CandidateCoverage{}
	for i := 0; i < 6; i++ {
		trap = append(trap, CandidateCoverage{InstanceID: fmt.Sprintf("q-%d", i), ObjectiveIndices: []int{i}})
	}
	if _, ok, e := FiveQuestionCover(core, trap); e != nil || ok {
		t.Fatal("union needing six accepted", e)
	}
	enough := append([]CandidateCoverage{}, trap...)
	enough[0].ObjectiveIndices = []int{0, 5}
	ids, ok, e := FiveQuestionCover(core, enough)
	if e != nil || !ok || len(ids) != 5 {
		t.Fatal("exact five witness", ids, ok, e)
	}
	duplicates := []CandidateCoverage{}
	for i := 0; i < 5; i++ {
		duplicates = append(duplicates, CandidateCoverage{InstanceID: "same", ObjectiveIndices: []int{0}})
	}
	if _, ok, _ := FiveQuestionCover([]int{0}, duplicates); ok {
		t.Fatal("duplicates counted twice")
	}
	eight := []int{0, 1, 2, 3, 4, 5, 6, 7}
	large := []CandidateCoverage{}
	for i := 0; i < 1000; i++ {
		large = append(large, CandidateCoverage{InstanceID: fmt.Sprintf("q-%04d", i), ObjectiveIndices: eight})
	}
	start := time.Now()
	if _, ok, e := FiveQuestionCover(eight, large); e != nil || !ok {
		t.Fatal("1000 pool/8 core", e)
	}
	if elapsed := time.Since(start); elapsed >= 8*time.Second {
		t.Fatal("coverage exceeded 8 seconds")
	}
	t.Logf("1000 candidate coverage: %s", time.Since(start))
	if _, _, e := FiveQuestionCover(append(eight, 8), large); e == nil {
		t.Fatal("9 core accepted")
	}
	if _, _, e := FiveQuestionCover(eight, append(large, CandidateCoverage{InstanceID: "overflow", ObjectiveIndices: eight})); e == nil {
		t.Fatal("1001 pool accepted")
	}
	rng := rand.New(rand.NewSource(20261002))
	for sample := 0; sample < 100; sample++ {
		n := 5 + rng.Intn(8)
		goals := 1 + rng.Intn(8)
		core := []int{}
		for j := 0; j < goals; j++ {
			core = append(core, j)
		}
		pool := []CandidateCoverage{}
		for j := 0; j < n; j++ {
			indices := []int{}
			for k := 0; k < goals; k++ {
				if rng.Intn(3) == 0 {
					indices = append(indices, k)
				}
			}
			pool = append(pool, CandidateCoverage{InstanceID: fmt.Sprintf("sample-%d-q-%d", sample, j), ObjectiveIndices: indices})
		}
		ids, ok, e := FiveQuestionCover(core, pool)
		if e != nil || ok != bruteCover(core, pool) {
			t.Fatalf("oracle mismatch sample %d: %v %v", sample, ok, e)
		}
		if ok {
			seenID := map[string]bool{}
			covered := map[int]bool{}
			for _, id := range ids {
				if seenID[id] {
					t.Fatal("reused candidate parent")
				}
				seenID[id] = true
				found := false
				for _, p := range pool {
					if p.InstanceID == id {
						found = true
						for _, o := range p.ObjectiveIndices {
							covered[o] = true
						}
					}
				}
				if !found {
					t.Fatal("unknown witness")
				}
			}
			if len(ids) != 5 {
				t.Fatal("wrong cardinality")
			}
			for _, goal := range core {
				if !covered[goal] {
					t.Fatal("incomplete witness")
				}
			}
		}
	}
}
