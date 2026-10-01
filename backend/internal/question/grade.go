package question

func GradeNumeric(correct Rational, mode, input string) (GradeResult, error) {
	want, e := rationalRat(correct)
	if e != nil {
		return GradeResult{}, ErrInvalid
	}
	answer, e := ParseNumeric(input, mode)
	if e != nil {
		return GradeResult{}, e
	}
	got, e := rationalRat(answer)
	if e != nil {
		return GradeResult{}, ErrInvalid
	}
	return GradeResult{Correct: got.Cmp(want) == 0}, nil
}
func GradeChoice(choices []Choice, correctID, inputID string) (GradeResult, error) {
	seen := map[string]bool{}
	for _, c := range choices {
		if !ValidMathID(c.ID) || seen[c.ID] {
			return GradeResult{}, ErrInvalid
		}
		seen[c.ID] = true
	}
	if !seen[correctID] {
		return GradeResult{}, ErrInvalid
	}
	if !seen[inputID] {
		return GradeResult{}, numericError("UNKNOWN_CHOICE")
	}
	return GradeResult{Correct: inputID == correctID}, nil
}
