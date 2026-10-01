package question

import (
	"encoding/json"
	"math/big"
)

// fraction operands are only parsed here; no generator arithmetic is used.
type integerFraction struct{ n, d *big.Int }

func readFraction(s string) (integerFraction, error) {
	r, e := parseLiteral(s)
	if e != nil {
		return integerFraction{}, ErrInvalid
	}
	return integerFraction{new(big.Int).Set(r.Num()), new(big.Int).Set(r.Denom())}, nil
}
func multiplyInts(v ...*big.Int) *big.Int {
	out := big.NewInt(1)
	for _, n := range v {
		out.Mul(out, n)
	}
	return out
}

// operationIdentity checks A op B = C using integer identities, never a derived Rat answer.
func operationIdentity(op string, a, b, c integerFraction) bool {
	switch op {
	case "add":
		left := new(big.Int).Add(multiplyInts(a.n, b.d), multiplyInts(b.n, a.d))
		return multiplyInts(left, c.d).Cmp(multiplyInts(c.n, a.d, b.d)) == 0
	case "subtract":
		left := new(big.Int).Sub(multiplyInts(a.n, b.d), multiplyInts(b.n, a.d))
		return multiplyInts(left, c.d).Cmp(multiplyInts(c.n, a.d, b.d)) == 0
	case "multiply":
		return multiplyInts(a.n, b.n, c.d).Cmp(multiplyInts(c.n, a.d, b.d)) == 0
	case "divide":
		return b.n.Sign() != 0 && multiplyInts(a.n, b.d, c.d).Cmp(multiplyInts(c.n, a.d, b.n)) == 0
	default:
		return false
	}
}
func VerifyInstance(i Instance) error {
	body := i.Body
	if body.Type != "single_choice" && body.Type != "numeric" {
		return ErrInvalid
	}
	if len(body.Choices) > 6 || len(body.Assets) > 8 || len(body.Prompt)+len(body.Explanation) > MaxQuestionTextBytes {
		return ErrLimitExceeded
	}
	var answer integerFraction
	if body.Type == "numeric" {
		if body.CorrectNumeric == nil || body.CorrectChoiceID != nil || len(body.Choices) != 0 || body.AnswerFormat == nil {
			return ErrInvalid
		}
		r, e := rationalRat(*body.CorrectNumeric)
		if e != nil || !CanEnterAnswer(*body.CorrectNumeric, *body.AnswerFormat) {
			return ErrInvalid
		}
		answer = integerFraction{new(big.Int).Set(r.Num()), new(big.Int).Set(r.Denom())}
	} else {
		if body.CorrectNumeric != nil || body.AnswerFormat != nil || body.CorrectChoiceID == nil || len(body.Choices) < 2 {
			return ErrInvalid
		}
		ids := map[string]bool{}
		selected := ""
		numbers := map[string]bool{}
		for _, choice := range body.Choices {
			if !ValidMathID(choice.ID) || ids[choice.ID] {
				return ErrInvalid
			}
			ids[choice.ID] = true
			if choice.ID == *body.CorrectChoiceID {
				selected = choice.Text
			}
			if body.Witness != nil && body.Witness.Engine.Family != "rational_comparison" {
				f, e := readFraction(choice.Text)
				if e != nil {
					return ErrInvalid
				}
				key := f.n.String() + "/" + f.d.String()
				if numbers[key] {
					return ErrInvalid
				}
				numbers[key] = true
				if choice.ID == *body.CorrectChoiceID {
					answer = f
				}
			}
		}
		if !ids[*body.CorrectChoiceID] {
			return ErrInvalid
		}
		if body.Witness == nil {
			return verifyIdentity(i)
		} // Conceptual single-choice still requires immutable identity checks.
		if body.Witness.Engine.Family == "rational_comparison" && selected != "lt" && selected != "eq" && selected != "gt" {
			return ErrInvalid
		}
	}
	witness := body.Witness
	if witness == nil {
		return ErrInvalid
	}
	names, e := engineParameters(witness.Engine)
	if e != nil || len(witness.Parameters) != len(names) {
		return ErrInvalid
	}
	params := map[string]integerFraction{}
	for _, p := range witness.Parameters {
		if _, exists := params[p.Name]; exists {
			return ErrInvalid
		}
		value, e := readFraction(p.Value)
		if e != nil {
			return e
		}
		params[p.Name] = value
	}
	for _, name := range names {
		if _, ok := params[name]; !ok {
			return ErrInvalid
		}
	}
	engine := witness.Engine
	if engine.Family == "rational_comparison" {
		if body.Type != "single_choice" || len(body.Choices) != 3 {
			return ErrInvalid
		}
		relations := map[string]bool{}
		for _, c := range body.Choices {
			if (c.Text != "lt" && c.Text != "eq" && c.Text != "gt") || relations[c.Text] {
				return ErrInvalid
			}
			relations[c.Text] = true
		}
		left, right := params["left"], params["right"]
		sign := multiplyInts(left.n, right.d).Cmp(multiplyInts(right.n, left.d))
		want := "eq"
		if sign < 0 {
			want = "lt"
		} else if sign > 0 {
			want = "gt"
		}
		for _, c := range body.Choices {
			if c.ID == *body.CorrectChoiceID && c.Text != want {
				return ErrInvalid
			}
		}
	} else if engine.Family == "rational_arithmetic" {
		if !operationIdentity(engine.Operation, params["left"], params["right"], answer) {
			return ErrInvalid
		}
	} else {
		known, result := params["known"], params["result"]
		// Uniqueness/domain checks are independent of the candidate and precede substitution.
		if engine.Operation == "multiply" && known.n.Sign() == 0 {
			return ErrInvalid
		}
		if engine.Operation == "divide" {
			if *engine.UnknownSide == "left" && known.n.Sign() == 0 {
				return ErrInvalid
			}
			if *engine.UnknownSide == "right" && (known.n.Sign() == 0 || result.n.Sign() == 0 || answer.n.Sign() == 0) {
				return ErrInvalid
			}
		}
		left, right := answer, known
		if *engine.UnknownSide == "right" {
			left, right = known, answer
		}
		if !operationIdentity(engine.Operation, left, right, result) {
			return ErrInvalid
		}
	}
	return verifyIdentity(i)
}

func verifyIdentity(i Instance) error {
	if i.Identity.Version < 1 || i.Identity.Version > 2147483647 {
		return ErrInvalid
	}
	if i.Origin == "template" {
		if i.Body.Witness == nil || i.Template == nil || !ValidMathID(i.Template.ID) || !ValidSHA(i.Template.SHA256) || i.Template.Version < 1 || i.GeneratorVersion == nil || i.VerifierVersion == nil || *i.GeneratorVersion != 1 || *i.VerifierVersion != 1 || i.Identity.Version != 1 {
			return ErrInvalid
		}
		for _, p := range i.Parameters {
			f, e := readFraction(p.Value)
			if e != nil || p.Value != f.n.String()+"/"+f.d.String() {
				return ErrInvalid
			}
		}
		a, _ := json.Marshal(i.Parameters)
		b, _ := json.Marshal(i.Body.Witness.Parameters)
		if string(a) != string(b) {
			return ErrInvalid
		}
		_, id, e := canonical("question-instance-identity-v1", struct {
			Template         Identity         `json:"template"`
			Parameters       []ParameterValue `json:"parameters"`
			GeneratorVersion int              `json:"generatorVersion"`
			VerifierVersion  int              `json:"verifierVersion"`
		}{*i.Template, i.Parameters, *i.GeneratorVersion, *i.VerifierVersion})
		if e != nil || i.Identity.ID != "qi-"+id {
			return ErrInvalid
		}
	} else if i.Origin != "fixed" || !ValidMathID(i.Identity.ID) || i.Template != nil || i.GeneratorVersion != nil || i.VerifierVersion != nil || len(i.Parameters) != 0 {
		return ErrInvalid
	}
	_, sha, e := CanonicalInstance(i)
	if e != nil || i.Identity.SHA256 != sha {
		return ErrInvalid
	}
	return nil
}
