package store

import (
	"context"
	"errors"
	"github.com/yyl1212/math_master/backend/internal/correction"
	"github.com/yyl1212/math_master/backend/internal/question"
	"strings"
	"testing"
)

func CorrectionCandidateSQLForTest() string {
	return learningCorrectionCandidateSQL(correctionWithConfig(context.Background(), true))
}
func TestCorrectionOwnResponseSize(t *testing.T) {
	if e := correctionResponseSize(strings.Repeat("a", correction.MaxResponseBytes-2)); e != nil {
		t.Fatal(e)
	}
	if e := correctionResponseSize(strings.Repeat("a", correction.MaxResponseBytes-1)); !errors.Is(e, question.ErrLimitExceeded) {
		t.Fatal("over response limit", e)
	}
}
