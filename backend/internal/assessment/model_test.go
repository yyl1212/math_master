package assessment

import (
	"github.com/yyl1212/math_master/backend/internal/question"
	"testing"
)

func TestLearningProjectionRejectsInvalidKnowledgeRef(t *testing.T) {
	for _, ref := range []question.Ref{{ID: "fractions", Version: 0}, {ID: "../private", Version: 1}} {
		i := fiveChoiceItems()[0]
		i.Body.Knowledge = ref
		if _, err := ProjectQuestion(1, i); err == nil {
			t.Fatal("unsafe knowledge ref accepted", ref)
		}
	}
}
func TestLearningGradePracticeCannotSkip(t *testing.T) {
	if err := ValidateAnswer(fiveChoiceItems()[0], Answer{Kind: "skipped"}, false); err == nil {
		t.Fatal("practice accepted a skipped response")
	}
}
