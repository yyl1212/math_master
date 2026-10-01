package publication

import (
	"errors"
	"github.com/yyl1212/math_master/backend/internal/auth"
	"testing"
)

func TestReviewChecksAndInput(t *testing.T) {
	good := ReviewInput{Decision: "approve", Checks: ReviewChecks{true, true, true, true, true}, IndependenceNote: "Independent human accounts verified for this fixture.", Note: "This is a technical fixture approval only."}
	if err := ValidateReviewInput(good); err != nil {
		t.Fatal(err)
	}
	bad := good
	bad.Checks.Sources = false
	if !errors.Is(ValidateReviewInput(bad), ErrContentNotReady) {
		t.Fatal("missing check accepted")
	}
	bad = good
	bad.IndependenceNote = ""
	if !errors.Is(ValidateReviewInput(bad), ErrContentNotReady) {
		t.Fatal("blank independence accepted")
	}
	returned := ReviewInput{Decision: "return", Note: "Please clarify the exact domain assumptions."}
	if err := ValidateReviewInput(returned); err != nil {
		t.Fatal(err)
	}
	returned.Decision = "override"
	if !errors.Is(ValidateReviewInput(returned), auth.ErrInvalidInput) {
		t.Fatal("unknown decision accepted")
	}
}
