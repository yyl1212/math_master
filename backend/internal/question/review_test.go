package question

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
)

func TestQuestionReviewInput(t *testing.T) {
	valid := ReviewInput{Decision: "approve", Checks: ReviewChecks{true, true, true, true, true, true}, IndependenceNote: "I independently checked every frozen question.", Note: "The complete frozen review is technically sound."}
	if err := ValidateReviewInput(valid, true); err != nil {
		t.Fatal(err)
	}
	if err := ValidateReviewInput(valid, false); !errors.Is(err, ErrNotReady) {
		t.Fatal("missing non-applicable note accepted", err)
	}
	valid.GenerationNote = "This bank has fixed questions and no generated templates."
	if err := ValidateReviewInput(valid, false); err != nil {
		t.Fatal(err)
	}
	for _, bad := range []string{strings.Repeat(" ", 10), "short", strings.Repeat("界", 1001), "Invalid NUL\x00 in review", string([]byte{0xff})} {
		input := valid
		input.GenerationNote = bad
		if err := ValidateReviewInput(input, false); err == nil {
			t.Fatal("invalid note accepted")
		}
	}
	for index := 0; index < 6; index++ {
		input := valid
		checks := []*bool{&input.Checks.Mathematics, &input.Checks.Explanations, &input.Checks.Objectives, &input.Checks.Sources, &input.Checks.Illustrations, &input.Checks.Generation}
		*checks[index] = false
		if err := ValidateReviewInput(input, true); !errors.Is(err, ErrNotReady) {
			t.Fatal(index, err)
		}
	}
	if err := ValidateReviewInput(ReviewInput{Decision: "return", Note: "Please correct this frozen explanation."}, false); err != nil {
		t.Fatal(err)
	}
}

func TestQuestionFixedOnlyFrozenBudget(t *testing.T) {
	input, refs := sealFixture()
	input.QuestionPackage.Templates = []Template{}
	input.QuestionPackage.FixedQuestions = []FixedQuestion{}
	input.QuestionPackage.Blueprints[0].Sources = []BlueprintSource{}
	for n := 0; n < 5; n++ {
		f := numericFixed(fmt.Sprintf("fixed-only-%d", n))
		input.QuestionPackage.FixedQuestions = append(input.QuestionPackage.FixedQuestions, f)
		input.QuestionPackage.Blueprints[0].Sources = append(input.QuestionPackage.Blueprints[0].Sources, BlueprintSource{Kind: "instance", Ref: Ref{ID: f.ID, Version: 1}})
	}
	sealed, gate, err := ValidateAndSeal(context.Background(), input, refs)
	if err != nil {
		t.Fatal(err)
	}
	ids := []Identity{}
	for _, i := range sealed.Instances {
		ids = append(ids, i.Identity)
	}
	generators, verifiers := UsedEngineVersions(sealed.Package, sealed.Instances)
	frozen := FrozenBody{CatalogueVersion: input.CatalogueVersion, CatalogueSHA256: refs.CatalogueSHA256, QuestionPackage: sealed.Package, SourceMap: input.SourceMap, AuthorIDs: []string{}, Resolved: sealed.Resolved, Objectives: sealed.Objectives, Generation: sealed.Generation, InstanceIdentities: ids, Coverage: gate.Coverage, GeneratorVersions: generators, VerifierVersions: verifiers}
	raw, _, err := CanonicalFrozen(frozen, sealed.Instances)
	if err != nil || gate.FrozenBytes != len(raw) {
		t.Fatal("invented generation version in frozen budget", gate.FrozenBytes, len(raw), err)
	}
}
