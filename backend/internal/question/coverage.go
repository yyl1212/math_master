package question

import "sort"

// FiveQuestionCover keeps immutable witnesses per (question count, objective mask).
// Descending counts prevent the current candidate from being reused.
func FiveQuestionCover(core []int, candidates []CandidateCoverage) ([]string, bool, error) {
	if len(core) < 1 || len(core) > 8 || len(candidates) > 1000 {
		return nil, false, ErrLimitExceeded
	}
	bits := map[int]uint{}
	for i, goal := range core {
		if goal < 0 || goal > 2147483647 {
			return nil, false, ErrInvalid
		}
		if _, exists := bits[goal]; exists {
			return nil, false, ErrInvalid
		}
		bits[goal] = uint(i)
	}
	pool := append([]CandidateCoverage{}, candidates...)
	sort.Slice(pool, func(i, j int) bool { return pool[i].InstanceID < pool[j].InstanceID })
	for i, p := range pool {
		if !ValidInstanceID(p.InstanceID) || (i > 0 && p.InstanceID == pool[i-1].InstanceID) {
			return nil, false, ErrInvalid
		}
		for _, goal := range p.ObjectiveIndices {
			if goal < 0 || goal > 2147483647 {
				return nil, false, ErrInvalid
			}
		}
	}
	if len(pool) < 5 {
		return nil, false, nil
	}
	var paths [6][256][]string
	paths[0][0] = []string{}
	target := (1 << len(core)) - 1
	for _, p := range pool {
		mask := 0
		for _, goal := range p.ObjectiveIndices {
			if bit, ok := bits[goal]; ok {
				mask |= 1 << bit
			}
		}
		for count := 5; count >= 1; count-- {
			for previous := 0; previous <= target; previous++ {
				parent := paths[count-1][previous]
				next := previous | mask
				if parent == nil || paths[count][next] != nil {
					continue
				}
				witness := make([]string, len(parent)+1)
				copy(witness, parent)
				witness[len(parent)] = p.InstanceID
				paths[count][next] = witness
			}
		}
		if paths[5][target] != nil {
			return paths[5][target], true, nil
		}
	}
	return nil, false, nil
}
