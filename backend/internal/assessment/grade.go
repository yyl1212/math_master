package assessment

import "github.com/yyl1212/math_master/backend/internal/question"

// ValidateAnswer proves the tagged input before callers may reveal correctness.
func ValidateAnswer(i question.Instance, a Answer, allowSkipped bool) error {
	if a.Kind == "skipped" {
		if !allowSkipped || a.ChoiceID != nil || a.Raw != nil {
			return question.ErrInvalid
		}
		return nil
	}
	switch i.Body.Type {
	case "single_choice":
		if a.Kind != "choice" || a.ChoiceID == nil || a.Raw != nil || i.Body.CorrectChoiceID == nil {
			return question.ErrInvalid
		}
		_, err := question.GradeChoice(i.Body.Choices, *i.Body.CorrectChoiceID, *a.ChoiceID)
		return err
	case "numeric":
		if a.Kind != "numeric" || a.Raw == nil || a.ChoiceID != nil || i.Body.AnswerFormat == nil || i.Body.CorrectNumeric == nil {
			return question.ErrInvalid
		}
		_, err := question.ParseNumeric(*a.Raw, *i.Body.AnswerFormat)
		return err
	}
	return question.ErrInvalid
}
func ValidateAnswers(items []question.Instance, in SubmitInput) error {
	if len(items) != 5 || len(in.Answers) != 5 {
		return question.ErrInvalid
	}
	identities := map[question.Identity]bool{}
	for _, i := range items {
		if identities[i.Identity] {
			return question.ErrInvalid
		}
		identities[i.Identity] = true
	}
	positions := map[int]bool{}
	for _, a := range in.Answers {
		if a.Position < 1 || a.Position > 5 || positions[a.Position] || a.Instance != items[a.Position-1].Identity {
			return question.ErrInvalid
		}
		positions[a.Position] = true
		if err := ValidateAnswer(items[a.Position-1], a.Answer, true); err != nil {
			return err
		}
	}
	return nil
}
func GradeAnswer(i question.Instance, a Answer) (bool, error) {
	if err := ValidateAnswer(i, a, true); err != nil {
		return false, err
	}
	if a.Kind == "skipped" {
		return false, nil
	}
	var r question.GradeResult
	var err error
	if a.Kind == "choice" {
		r, err = question.GradeChoice(i.Body.Choices, *i.Body.CorrectChoiceID, *a.ChoiceID)
	} else {
		r, err = question.GradeNumeric(*i.Body.CorrectNumeric, *i.Body.AnswerFormat, *a.Raw)
	}
	return r.Correct, err
}
func GradeFive(items []question.Instance, in SubmitInput) (int, bool, error) {
	if err := ValidateAnswers(items, in); err != nil {
		return 0, false, err
	}
	score := 0
	for _, a := range in.Answers {
		correct, err := GradeAnswer(items[a.Position-1], a.Answer)
		if err != nil {
			return 0, false, err
		}
		if correct {
			score++
		}
	}
	return score, score >= 4, nil
}
