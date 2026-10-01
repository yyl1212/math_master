package question

import (
	"math/big"
	"regexp"
	"strings"
	"unicode/utf8"
)

var integerSyntax = regexp.MustCompile(`^[+-]?[0-9]+$`)
var decimalSyntax = regexp.MustCompile(`^[+-]?([0-9]+(\.[0-9]*)?|\.[0-9]+)$`)
var fractionSyntax = regexp.MustCompile(`^([+-]?[0-9]+)[ \t]*/[ \t]*([+-]?[0-9]+)$`)
var canonicalNumerator = regexp.MustCompile(`^(0|-?[1-9][0-9]{0,255})$`)
var canonicalDenominator = regexp.MustCompile(`^[1-9][0-9]{0,255}$`)

func numericError(code string) error { return &NumericFormatError{Code: code} }
func ParseNumeric(input, mode string) (Rational, error) {
	if !utf8.ValidString(input) {
		return Rational{}, numericError("INVALID_SYNTAX")
	}
	if utf8.RuneCountInString(input) > 128 {
		return Rational{}, numericError("INPUT_TOO_LONG")
	}
	if mode != "rational" && mode != "percentage" {
		return Rational{}, numericError("INVALID_MODE")
	}
	input = strings.TrimSpace(input)
	percent := mode == "percentage"
	if percent {
		if !strings.HasSuffix(input, "%") {
			return Rational{}, numericError("PERCENT_REQUIRED")
		}
		input = strings.TrimSuffix(input, "%")
		if !decimalSyntax.MatchString(input) {
			return Rational{}, numericError("INVALID_SYNTAX")
		}
	}
	rat, e := parseLiteral(input)
	if e != nil {
		return Rational{}, e
	}
	if percent {
		rat.Quo(rat, big.NewRat(100, 1))
	}
	return rationalFromRat(rat)
}

// parseLiteral has no user-input length policy; its caller bounds bytes first.
// It is shared only for reading exact parameters, never for deriving an answer.
func parseLiteral(input string) (*big.Rat, error) {
	if len(input) > 514 {
		return nil, numericError("INPUT_TOO_LONG")
	}
	if m := fractionSyntax.FindStringSubmatch(input); m != nil {
		n, ok := new(big.Int).SetString(m[1], 10)
		if !ok {
			return nil, numericError("INVALID_SYNTAX")
		}
		d, ok := new(big.Int).SetString(m[2], 10)
		if !ok {
			return nil, numericError("INVALID_SYNTAX")
		}
		if d.Sign() == 0 {
			return nil, numericError("ZERO_DENOMINATOR")
		}
		r := new(big.Rat).SetFrac(n, d)
		if _, e := rationalFromRat(r); e != nil {
			return nil, e
		}
		return r, nil
	}
	if !decimalSyntax.MatchString(input) {
		return nil, numericError("INVALID_SYNTAX")
	}
	negative := strings.HasPrefix(input, "-")
	input = strings.TrimPrefix(strings.TrimPrefix(input, "+"), "-")
	parts := strings.Split(input, ".")
	places := 0
	digits := parts[0]
	if len(parts) == 2 {
		places = len(parts[1])
		digits += parts[1]
	}
	n, ok := new(big.Int).SetString(digits, 10)
	if !ok {
		return nil, numericError("INVALID_SYNTAX")
	}
	if negative {
		n.Neg(n)
	}
	d := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(places)), nil)
	r := new(big.Rat).SetFrac(n, d)
	if _, e := rationalFromRat(r); e != nil {
		return nil, e
	}
	return r, nil
}
func rationalFromRat(r *big.Rat) (Rational, error) {
	if r == nil {
		return Rational{}, ErrInvalid
	}
	n, d := r.Num().String(), r.Denom().String()
	if len(strings.TrimPrefix(n, "-")) > 256 || len(d) > 256 {
		return Rational{}, numericError("RESULT_TOO_LARGE")
	}
	return Rational{Numerator: n, Denominator: d}, nil
}

// rationalRat validates persisted canonical answers without the 128-character user limit.
func rationalRat(value Rational) (*big.Rat, error) {
	if !canonicalNumerator.MatchString(value.Numerator) || !canonicalDenominator.MatchString(value.Denominator) {
		return nil, ErrInvalid
	}
	n, _ := new(big.Int).SetString(value.Numerator, 10)
	d, _ := new(big.Int).SetString(value.Denominator, 10)
	r := new(big.Rat).SetFrac(n, d)
	if r.Num().String() != value.Numerator || r.Denom().String() != value.Denominator {
		return nil, ErrInvalid
	}
	return r, nil
}
func terminatingDecimal(r *big.Rat) (string, bool) {
	d := new(big.Int).Set(r.Denom())
	two, five := 0, 0
	for _, factor := range []int64{2, 5} {
		count := 0
		for {
			q, rem := new(big.Int), new(big.Int)
			q.QuoRem(d, big.NewInt(factor), rem)
			if rem.Sign() != 0 {
				break
			}
			d = q
			count++
		}
		if factor == 2 {
			two = count
		} else {
			five = count
		}
	}
	if d.Cmp(big.NewInt(1)) != 0 {
		return "", false
	}
	places := max(two, five)
	s := r.FloatString(places)
	if strings.Contains(s, ".") {
		s = strings.TrimRight(s, "0")
		s = strings.TrimSuffix(s, ".")
	}
	if strings.HasPrefix(s, "0.") {
		s = s[1:]
	} else if strings.HasPrefix(s, "-0.") {
		s = "-" + s[2:]
	}
	return s, true
}
func CanEnterAnswer(value Rational, mode string) bool {
	r, e := rationalRat(value)
	if e != nil {
		return false
	}
	if mode == "percentage" {
		r.Mul(r, big.NewRat(100, 1))
		s, ok := terminatingDecimal(r)
		return ok && len(s)+1 <= 128
	}
	if mode != "rational" {
		return false
	}
	if r.IsInt() && len(r.Num().String()) <= 128 {
		return true
	}
	if len(r.Num().String())+1+len(r.Denom().String()) <= 128 {
		return true
	}
	s, ok := terminatingDecimal(r)
	return ok && len(s) <= 128
}
