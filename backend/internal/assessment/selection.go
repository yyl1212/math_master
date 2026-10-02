package assessment

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"github.com/yyl1212/math_master/backend/internal/question"
	"sort"
	"time"
)

// EligibleCandidates uses the caller's database time. The cooldown has the
// half-open interval [exposedAt, exposedAt+30m); the last submission never ages out.
func EligibleCandidates(candidates []Candidate, now time.Time) []Candidate {
	out := make([]Candidate, 0, len(candidates))
	for _, c := range candidates {
		if c.RecentSubmitted || c.ExposedAt != nil && now.Before(c.ExposedAt.Add(30*time.Minute)) {
			continue
		}
		out = append(out, c)
	}
	return out
}
func validatePool(core []int, cs []Candidate) error {
	if len(core) > 8 || len(cs) > 1000 {
		return question.ErrLimitExceeded
	}
	if len(core) == 0 {
		return question.ErrInvalid
	}
	seenCore := map[int]bool{}
	for _, v := range core {
		if v < 0 || seenCore[v] {
			return question.ErrInvalid
		}
		seenCore[v] = true
	}
	seen := map[question.Identity]bool{}
	for _, c := range cs {
		if !question.ValidInstanceID(c.Identity.ID) || c.Identity.Version < 1 || !question.ValidSHA(c.Identity.SHA256) || seen[c.Identity] {
			return question.ErrInvalid
		}
		seen[c.Identity] = true
		for _, v := range c.Coverage {
			if v < 0 {
				return question.ErrInvalid
			}
		}
	}
	return nil
}

type rankedCandidate struct {
	Candidate
	rank [32]byte
}

func candidateRank(seed [32]byte, id question.Identity) [32]byte {
	h := sha256.New()
	h.Write(seed[:])
	h.Write([]byte(id.ID))
	h.Write([]byte{0})
	var v [8]byte
	binary.BigEndian.PutUint64(v[:], uint64(id.Version))
	h.Write(v[:])
	h.Write([]byte(id.SHA256))
	var out [32]byte
	copy(out[:], h.Sum(nil))
	return out
}

type coverState struct {
	valid   bool
	unseen  int
	indices [5]int
}

func betterCover(a, b coverState, n int) bool {
	if !b.valid {
		return true
	}
	if a.unseen != b.unseen {
		return a.unseen > b.unseen
	}
	for j := 0; j < n; j++ {
		if a.indices[j] != b.indices[j] {
			return a.indices[j] < b.indices[j]
		}
	}
	return false
}

// SelectFive maximizes unseen questions within feasible full coverage, then uses
// oldest views and a seeded identity rank. The bounded DP has at most 6*256 states.
// Callers must filter time-based cooldowns with EligibleCandidates first.
func SelectFive(ctx context.Context, core []int, candidates []Candidate, seed [32]byte) ([]question.Identity, bool, error) {
	if err := ctx.Err(); err != nil {
		return nil, false, err
	}
	if err := validatePool(core, candidates); err != nil {
		return nil, false, err
	}
	cs := make([]rankedCandidate, 0, len(candidates))
	for _, c := range candidates {
		if !c.RecentSubmitted {
			cs = append(cs, rankedCandidate{c, candidateRank(seed, c.Identity)})
		}
	}
	sort.Slice(cs, func(i, j int) bool {
		a, b := cs[i], cs[j]
		if a.Seen != b.Seen {
			return !a.Seen
		}
		if a.LastSeenAt == nil && b.LastSeenAt != nil {
			return true
		}
		if a.LastSeenAt != nil && b.LastSeenAt == nil {
			return false
		}
		if a.LastSeenAt != nil && !a.LastSeenAt.Equal(*b.LastSeenAt) {
			return a.LastSeenAt.Before(*b.LastSeenAt)
		}
		for k := range a.rank {
			if a.rank[k] != b.rank[k] {
				return a.rank[k] < b.rank[k]
			}
		}
		if a.Identity.ID != b.Identity.ID {
			return a.Identity.ID < b.Identity.ID
		}
		if a.Identity.Version != b.Identity.Version {
			return a.Identity.Version < b.Identity.Version
		}
		return a.Identity.SHA256 < b.Identity.SHA256
	})
	bits := map[int]int{}
	for j, v := range core {
		bits[v] = 1 << j
	}
	size := 1 << len(core)
	dp := make([][]coverState, 6)
	for j := range dp {
		dp[j] = make([]coverState, size)
	}
	dp[0][0].valid = true
	for i, c := range cs {
		if err := ctx.Err(); err != nil {
			return nil, false, err
		}
		mask := 0
		for _, v := range c.Coverage {
			mask |= bits[v]
		}
		for n := 4; n >= 0; n-- {
			for prev := 0; prev < size; prev++ {
				s := dp[n][prev]
				if !s.valid {
					continue
				}
				s.indices[n] = i
				if !c.Seen {
					s.unseen++
				}
				next := prev | mask
				if betterCover(s, dp[n+1][next], n+1) {
					dp[n+1][next] = s
				}
			}
		}
	}
	best := dp[5][size-1]
	if !best.valid {
		return []question.Identity{}, false, nil
	}
	out := make([]question.Identity, 5)
	for j, i := range best.indices {
		out[j] = cs[i].Identity
	}
	return out, true, nil
}
func EarliestReady(ctx context.Context, core []int, candidates []Candidate, now time.Time, seed [32]byte) (*time.Time, error) {
	if err := validatePool(core, candidates); err != nil {
		return nil, err
	}
	feasible := func(at time.Time) (bool, error) {
		_, ok, err := SelectFive(ctx, core, EligibleCandidates(candidates, at), seed)
		return ok, err
	}
	if ok, err := feasible(now); err != nil || ok {
		return nil, err
	}
	times := make([]time.Time, 0, len(candidates))
	for _, c := range candidates {
		if c.RecentSubmitted || c.ExposedAt == nil {
			continue
		}
		at := c.ExposedAt.Add(30 * time.Minute)
		if at.After(now) {
			times = append(times, at)
		}
	}
	if len(times) == 0 {
		return nil, nil
	}
	sort.Slice(times, func(i, j int) bool { return times[i].Before(times[j]) })
	if ok, err := feasible(times[len(times)-1]); err != nil || !ok {
		return nil, err
	}
	lo, hi := 0, len(times)-1
	for lo < hi {
		mid := lo + (hi-lo)/2
		ok, err := feasible(times[mid])
		if err != nil {
			return nil, err
		}
		if ok {
			hi = mid
		} else {
			lo = mid + 1
		}
	}
	at := times[lo]
	return &at, nil
}
