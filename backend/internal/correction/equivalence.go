package correction

import (
	"github.com/yyl1212/math_master/backend/internal/auth"
	"github.com/yyl1212/math_master/backend/internal/question"
	"reflect"
)

// IDs and template revisions may change. Actual mathematical input may not.
func Equivalent(original, replacement question.Instance) (bool, DispositionReason, error) {
	if !ValidIdentity(original.Identity, true) || !ValidIdentity(replacement.Identity, true) {
		return false, SourceInvalid, auth.ErrInvalidInput
	}
	a, b := original.Body, replacement.Body
	if a.Knowledge != b.Knowledge {
		return false, KnowledgeChanged, nil
	}
	if !reflect.DeepEqual(a.Coverage, b.Coverage) {
		return false, CoverageChanged, nil
	}
	a.CorrectChoiceID = nil
	b.CorrectChoiceID = nil
	a.CorrectNumeric = nil
	b.CorrectNumeric = nil
	a.Explanation = ""
	b.Explanation = ""
	if !reflect.DeepEqual(a, b) || !reflect.DeepEqual(original.Parameters, replacement.Parameters) {
		return false, IntentChanged, nil
	}
	return true, AnswerCorrected, nil
}
