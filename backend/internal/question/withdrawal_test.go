package question

import (
	"context"
	"errors"
	"testing"
)

func TestQuestionWithdrawCandidate(t *testing.T) {
	sub, refs := candidateFixture(t)
	c, err := BuildCandidate(context.Background(), BaseManifest{}, []ApprovedSubmission{sub}, refs)
	if err != nil {
		t.Fatal(err)
	}
	for _, target := range []WithdrawalTarget{{Kind: "template", ID: c.Templates[0].ID, Version: 1}, {Kind: "instance", ID: c.Instances[0].Identity.ID, Version: 1}, {Kind: "blueprint", ID: c.Blueprints[0].ID, Version: 1}, {Kind: "template", ID: "retired-known-template", Version: 1}} {
		next, err := WithdrawCandidate(context.Background(), c, target)
		if err != nil {
			t.Fatal(err)
		}
		switch target.Kind {
		case "template":
			if target.ID == c.Templates[0].ID {
				if len(next.Templates) != 0 || len(next.Instances) != 0 || len(next.Blueprints) != 0 || next.Diff.Removed != 7 {
					t.Fatal("template closure incomplete")
				}
			} else if next.Diff != (DiffSummary{}) {
				t.Fatal("non-current version affected bank")
			}
		case "instance":
			if len(next.Instances) != 4 || len(next.Templates) != 1 || len(next.Blueprints) != 1 {
				t.Fatal("single generated instance removed its parent")
			}
		case "blueprint":
			if len(next.Instances) != 5 || len(next.Templates) != 1 || len(next.Blueprints) != 0 {
				t.Fatal("blueprint removed correct questions")
			}
		}
	}
	if len(c.Instances) != 5 || len(c.Blueprints) != 1 || len(c.Templates) != 1 {
		t.Fatal("withdrawal rewrote historical candidate")
	}
	if _, err = WithdrawCandidate(context.Background(), c, WithdrawalTarget{Kind: "knowledge", ID: "fractions", Version: 1}); !errors.Is(err, ErrInvalid) {
		t.Fatal("old content kind accepted", err)
	}
}
