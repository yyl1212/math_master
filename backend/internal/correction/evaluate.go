package correction

import (
	"github.com/yyl1212/math_master/backend/internal/assessment"
	"github.com/yyl1212/math_master/backend/internal/question"
	"reflect"
	"sort"
)

func Evaluate(b Basis) (Evaluation, error) {
	out := Evaluation{Status: RetakeRequired, Reason: InsufficientItems, Correct: []bool{}}
	n := 5
	if b.OriginalSeal.Kind == "practice" {
		n = 1
	}
	if (b.OriginalSeal.Kind != "assessment" && b.OriginalSeal.Kind != "practice") || len(b.OriginalItems) != n || len(b.EffectiveItems) != n || len(b.OriginalAnswers) != n || len(b.OriginalSeal.Items) != n {
		return out, nil
	}
	if !RegisteredAlgorithm(b.OriginalSeal.RuleVersion) {
		out.Reason = NoApprovedBasis
		return out, nil
	}
	if len(b.PlanRefs) == 0 {
		if len(b.HandledCaseIDs) > 0 {
			out.Status = AwaitingReview
			out.Reason = NoApprovedBasis
		} else {
			out.Status = CheckedUnaffected
			out.Reason = Unaffected
		}
		return out, nil
	}
	originalIDs := map[question.Identity]bool{}
	effectiveIDs := map[question.Identity]bool{}
	covered := map[int]bool{}
	for pos := 0; pos < n; pos++ {
		o, e := b.OriginalItems[pos], b.EffectiveItems[pos]
		binding := b.OriginalSeal.Items[pos]
		if binding.Position != pos+1 || binding.Instance != o.Identity || originalIDs[o.Identity] || effectiveIDs[e.Identity] {
			return out, nil
		}
		originalIDs[o.Identity] = true
		effectiveIDs[e.Identity] = true
		if o.Body.Knowledge.ID != b.OriginalSeal.Knowledge.ID || o.Body.Knowledge.Version != b.OriginalSeal.Knowledge.Version {
			out.Reason = KnowledgeChanged
			return out, nil
		}
		ok, why, err := Equivalent(o, e)
		if err != nil {
			return out, err
		}
		if !ok {
			out.Reason = why
			return out, nil
		}
		indices := []int{}
		for _, c := range e.Body.Coverage {
			if c.Knowledge.ID == b.OriginalSeal.Knowledge.ID && c.Knowledge.Version == b.OriginalSeal.Knowledge.Version {
				indices = append(indices, c.ObjectiveIndices...)
			}
		}
		if !sameIndices(indices, binding.Coverage) {
			out.Reason = CoverageChanged
			return out, nil
		}
		for _, i := range indices {
			covered[i] = true
		}
	}
	if n == 5 {
		core := map[int]bool{}
		for _, i := range b.OriginalSeal.Core {
			if i < 0 || core[i] {
				out.Reason = CoverageChanged
				return out, nil
			}
			core[i] = true
		}
		for i := range core {
			if !covered[i] {
				out.Reason = CoverageChanged
				return out, nil
			}
		}
		if len(core) == 0 {
			out.Reason = CoverageChanged
			return out, nil
		}
	}
	score := 0
	for pos, i := range b.EffectiveItems {
		correct, e := assessment.GradeAnswer(i, b.OriginalAnswers[pos])
		if e != nil {
			out.Reason = IntentChanged
			out.Correct = []bool{}
			return out, nil
		}
		out.Correct = append(out.Correct, correct)
		if correct {
			score++
		}
	}
	out.Status = CorrectedFailed
	out.Reason = AnswerCorrected
	if score >= 4 && n == 5 || score == 1 && n == 1 {
		out.Status = CorrectedPassed
	}
	if n == 5 {
		passed := score >= 4
		out.Score = &score
		out.Passed = &passed
	}
	return out, nil
}
func sameIndices(a, b []int) bool {
	aa := append([]int{}, a...)
	bb := append([]int{}, b...)
	sort.Ints(aa)
	sort.Ints(bb)
	return reflect.DeepEqual(aa, bb)
}
