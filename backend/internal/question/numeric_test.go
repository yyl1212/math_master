package question

import (
	"errors"
	"strings"
	"testing"
)

// Floating point parsing, permissive expressions or counting after trim would violate these hand-derived cases.
func TestNumericModesAndBoundary(t *testing.T) {
	for _, c := range []struct{ input, mode, n, d string }{
		{"1/2", "rational", "1", "2"}, {"2/4", "rational", "1", "2"}, {"0.50", "rational", "1", "2"},
		{"1/-2", "rational", "-1", "2"}, {"-0.5", "rational", "-1", "2"}, {".5", "rational", "1", "2"}, {"1.", "rational", "1", "1"},
		{"+002", "rational", "2", "1"}, {"-0", "rational", "0", "1"}, {"-0.0", "rational", "0", "1"}, {"2 \t/\t 4", "rational", "1", "2"},
		{"50%", "percentage", "1", "2"}, {".5%", "percentage", "1", "200"}, {"-25.%", "percentage", "-1", "4"},
		{strings.Repeat(" ", 127) + "1", "rational", "1", "1"},
	} {
		got, e := ParseNumeric(c.input, c.mode)
		if e != nil || got != (Rational{Numerator: c.n, Denominator: c.d}) {
			t.Fatalf("%q %s: %v %v", c.input, c.mode, got, e)
		}
	}
	for _, c := range []struct{ input, mode string }{
		{"50", "percentage"}, {"1/2%", "percentage"}, {"50 %", "percentage"}, {"50%", "rational"},
		{"1/0", "rational"}, {"1/-0", "rational"}, {"1e2", "rational"}, {"NaN", "rational"}, {"Infinity", "rational"}, {"1+2", "rational"},
		{"１", "rational"}, {"١", "rational"}, {"1 2", "rational"}, {"1. 2", "rational"}, {"1/ - 2", "rational"}, {"1,000", "rational"}, {"1 kg", "rational"},
		{".", "rational"}, {"", "rational"}, {"1", "unknown"}, {strings.Repeat(" ", 128) + "1", "rational"}, {strings.Repeat("界", 128) + "1", "rational"},
	} {
		_, e := ParseNumeric(c.input, c.mode)
		var f *NumericFormatError
		if !errors.As(e, &f) {
			t.Fatalf("accepted invalid numeric input or wrong error: %q %v", c.input, e)
		}
	}
	if _, e := rationalRat(Rational{Numerator: strings.Repeat("9", 256), Denominator: "1"}); e != nil {
		t.Fatal("256-digit internal result", e)
	}
	if _, e := rationalRat(Rational{Numerator: strings.Repeat("9", 257), Denominator: "1"}); e == nil {
		t.Fatal("257-digit internal result")
	}
	if _, e := rationalRat(Rational{Numerator: "1", Denominator: strings.Repeat("9", 256)}); e != nil {
		t.Fatal(e)
	}
	for _, r := range []Rational{{Numerator: "1", Denominator: strings.Repeat("9", 257)}, {Numerator: "2", Denominator: "4"}, {Numerator: "-0", Denominator: "1"}, {Numerator: "01", Denominator: "2"}, {Numerator: "1", Denominator: "-2"}} {
		if _, e := rationalRat(r); e == nil {
			t.Fatal("noncanonical answer accepted")
		}
	}
}
func TestNumericAnswerRepresentable(t *testing.T) {
	for _, c := range []struct {
		r    Rational
		mode string
		want bool
	}{
		{Rational{"1", "3"}, "rational", true}, {Rational{"1", "3"}, "percentage", false}, {Rational{"1", "2"}, "percentage", true},
		{Rational{strings.Repeat("9", 128), "1"}, "rational", true}, {Rational{strings.Repeat("9", 129), "1"}, "rational", false},
		{Rational{"1", "1" + strings.Repeat("0", 127)}, "rational", true}, {Rational{"1", "1" + strings.Repeat("0", 128)}, "rational", false},
		{Rational{"-1", "1" + strings.Repeat("0", 126)}, "rational", true}, {Rational{"-1", "1" + strings.Repeat("0", 127)}, "rational", false},
		{Rational{"1", "1" + strings.Repeat("0", 128)}, "percentage", true}, {Rational{"1", "1" + strings.Repeat("0", 129)}, "percentage", false},
		{Rational{"2", "4"}, "rational", false}, {Rational{"0", "1"}, "percentage", true}, {Rational{"1", "1"}, "invalid", false},
	} {
		if got := CanEnterAnswer(c.r, c.mode); got != c.want {
			t.Fatalf("representable %v %s: %v want %v", c.r, c.mode, got, c.want)
		}
	}
}
