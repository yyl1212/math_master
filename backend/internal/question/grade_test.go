package question

import (
	"errors"
	"testing"
)

func TestExactGrade(t *testing.T) {
	half := Rational{Numerator: "1", Denominator: "2"}
	for _, c := range []struct {
		mode, input string
		want        bool
	}{{"rational", "2/4", true}, {"rational", "0.50", true}, {"rational", "0.499999999999999999", false}, {"percentage", "50%", true}, {"percentage", "49.999999%", false}, {"rational", "-0.5", false}} {
		r, e := GradeNumeric(half, c.mode, c.input)
		if e != nil || r.Correct != c.want {
			t.Fatalf("exact grade %q: %v %v", c.input, r, e)
		}
	}
	for _, input := range []string{"1/0", "NaN", "2*0.25"} {
		_, e := GradeNumeric(half, "rational", input)
		var f *NumericFormatError
		if !errors.As(e, &f) {
			t.Fatal("format silently graded incorrect")
		}
	}
	if _, e := GradeNumeric(Rational{"2", "4"}, "rational", "1/2"); !errors.Is(e, ErrInvalid) {
		t.Fatal("invalid stored answer", e)
	}
	choices := []Choice{{ID: "first", Text: ".5"}, {ID: "second", Text: "1"}}
	for _, c := range []struct {
		input string
		want  bool
	}{{"first", true}, {"second", false}} {
		g, e := GradeChoice(choices, "first", c.input)
		if e != nil || g.Correct != c.want {
			t.Fatal("choice grade", g, e)
		}
	}
	_, e := GradeChoice(choices, "first", "third")
	var f *NumericFormatError
	if !errors.As(e, &f) {
		t.Fatal("unknown choice must be format error")
	}
	if _, e := GradeChoice(choices, "absent", "first"); !errors.Is(e, ErrInvalid) {
		t.Fatal("malformed stored answer")
	}
}
