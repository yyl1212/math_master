package assessment

import (
	"errors"
	"fmt"
	"github.com/yyl1212/math_master/backend/internal/question"
	"strings"
	"testing"
)

func str(v string) *string { return &v }
func testIdentity(id string) question.Identity {
	return question.Identity{ID: id, Version: 1, SHA256: strings.Repeat("a", 64)}
}
func fiveChoiceItems() []question.Instance {
	out := make([]question.Instance, 5)
	for j := range out {
		out[j] = question.Instance{Identity: testIdentity(fmt.Sprintf("item-%d", j+1)), Origin: "fixed", Body: question.QuestionBody{Knowledge: question.Ref{ID: "fractions", Version: 1}, Type: "single_choice", Prompt: "Choose one.", Explanation: "Private explanation.", Choices: []question.Choice{{ID: "a", Text: "one"}, {ID: "b", Text: "two"}}, CorrectChoiceID: str("a")}}
	}
	return out
}
func allChoiceAnswers(ids ...string) SubmitInput {
	in := SubmitInput{Answers: make([]PositionAnswer, 5)}
	for j, id := range ids {
		in.Answers[j] = PositionAnswer{Position: j + 1, Instance: testIdentity(fmt.Sprintf("item-%d", j+1)), Answer: Answer{Kind: "choice", ChoiceID: str(id)}}
	}
	return in
}
func TestLearningGradeFourOfFive(t *testing.T) {
	for _, tt := range []struct {
		ids   []string
		score int
		pass  bool
	}{{[]string{"a", "a", "a", "a", "b"}, 4, true}, {[]string{"a", "a", "a", "b", "b"}, 3, false}, {[]string{"a", "a", "a", "a", "a"}, 5, true}} {
		score, pass, err := GradeFive(fiveChoiceItems(), allChoiceAnswers(tt.ids...))
		if err != nil || score != tt.score || pass != tt.pass {
			t.Fatal(score, pass, err)
		}
	}
	in := allChoiceAnswers("a", "a", "a", "a", "a")
	in.Answers[4].Answer = Answer{Kind: "skipped"}
	score, passed, err := GradeFive(fiveChoiceItems(), in)
	if err != nil || score != 4 || !passed {
		t.Fatal(score, passed, err)
	}
}
func TestLearningGradeRejectsEntireMalformedSubmission(t *testing.T) {
	for _, tt := range []struct {
		name   string
		mutate func(*SubmitInput)
	}{
		{"unknown-choice", func(in *SubmitInput) { in.Answers[4].Answer.ChoiceID = str("unknown") }},
		{"four", func(in *SubmitInput) { in.Answers = in.Answers[:4] }},
		{"six", func(in *SubmitInput) { in.Answers = append(in.Answers, in.Answers[0]) }},
		{"repeated-position", func(in *SubmitInput) { in.Answers[4].Position = 1 }},
		{"wrong-sha", func(in *SubmitInput) { in.Answers[4].Instance.SHA256 = strings.Repeat("b", 64) }},
		{"wrong-version", func(in *SubmitInput) { in.Answers[4].Instance.Version = 2 }},
		{"wrong-kind", func(in *SubmitInput) { in.Answers[4].Answer = Answer{Kind: "numeric", Raw: str("1")} }},
		{"extra-value", func(in *SubmitInput) { in.Answers[4].Answer.Raw = str("1") }},
		{"skip-value", func(in *SubmitInput) { in.Answers[4].Answer = Answer{Kind: "skipped", ChoiceID: str("a")} }},
	} {
		t.Run(tt.name, func(t *testing.T) {
			in := allChoiceAnswers("a", "a", "a", "a", "a")
			tt.mutate(&in)
			score, pass, err := GradeFive(fiveChoiceItems(), in)
			if err == nil || score != 0 || pass {
				t.Fatal("partial score or accepted malformed submission", score, pass, err)
			}
		})
	}
}
func TestLearningGradeNumericExactAndUnicodeBound(t *testing.T) {
	for _, tt := range []struct {
		raw, mode      string
		valid, correct bool
	}{
		{".5", "rational", true, true}, {"2/4", "rational", true, true}, {"50%", "percentage", true, true}, {"0.5000000000000000000000000001", "rational", true, false},
		{".5%", "percentage", true, false}, {"50", "percentage", false, false}, {"1e-1", "rational", false, false}, {"1/0", "rational", false, false},
		{".5" + strings.Repeat("　", 126), "rational", true, true}, {".5" + strings.Repeat("　", 127), "rational", false, false}, {string([]byte{0xff}), "rational", false, false},
	} {
		t.Run(fmt.Sprintf("%s-%d", tt.mode, len(tt.raw)), func(t *testing.T) {
			items := fiveChoiceItems()
			items[4].Body = question.QuestionBody{Type: "numeric", Knowledge: question.Ref{ID: "fractions", Version: 1}, AnswerFormat: str(tt.mode), CorrectNumeric: &question.Rational{Numerator: "1", Denominator: "2"}}
			in := allChoiceAnswers("a", "a", "a", "a", "a")
			in.Answers[4].Answer = Answer{Kind: "numeric", Raw: str(tt.raw)}
			score, pass, err := GradeFive(items, in)
			if !tt.valid {
				if err == nil || score != 0 || pass {
					t.Fatal("invalid format leaked score", score, pass, err)
				}
				var format *question.NumericFormatError
				if !errors.As(err, &format) {
					t.Fatal("lost format code", err)
				}
				return
			}
			want := 4
			if tt.correct {
				want = 5
			}
			if err != nil || score != want || !pass {
				t.Fatal(score, pass, err)
			}
		})
	}
}
