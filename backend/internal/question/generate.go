package question

import (
	"context"
	"crypto/sha256"
	"fmt"
	"math/big"
	"reflect"
	"sort"
)

const MaxParameters = 4
const MaxParameterValues = 32
const MaxCombinations = 1000
const MaxQuestionTextBytes = 8 * 1024

// normalizeParameters enforces the generic finite-space budget before any family allocates instances.
func normalizeParameters(params []Parameter) ([]Parameter, int, error) {
	if len(params) == 0 || len(params) > MaxParameters {
		return nil, 0, ErrLimitExceeded
	}
	out := make([]Parameter, 0, len(params))
	names := map[string]bool{}
	count := 1
	for _, p := range params {
		if !ValidMathID(p.Name) || names[p.Name] {
			return nil, 0, ErrInvalid
		}
		names[p.Name] = true
		if len(p.Values) == 0 || len(p.Values) > MaxParameterValues {
			return nil, 0, ErrLimitExceeded
		}
		values := map[string]*big.Rat{}
		for _, s := range p.Values {
			v, e := parseLiteral(s)
			if e != nil {
				return nil, 0, ErrInvalid
			}
			values[v.Num().String()+"/"+v.Denom().String()] = v
		}
		keys := make([]string, 0, len(values))
		for k := range values {
			keys = append(keys, k)
		}
		sort.Slice(keys, func(i, j int) bool {
			cmp := values[keys[i]].Cmp(values[keys[j]])
			if cmp != 0 {
				return cmp < 0
			}
			return keys[i] < keys[j]
		})
		if count > MaxCombinations/len(keys) {
			return nil, 0, ErrLimitExceeded
		}
		count *= len(keys)
		out = append(out, Parameter{Name: p.Name, Values: keys})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, count, nil
}
func engineParameters(e EngineSpec) ([]string, error) {
	if e.GeneratorVersion != 1 || e.VerifierVersion != 1 {
		return nil, ErrInvalid
	}
	switch e.Family {
	case "rational_arithmetic":
		if e.UnknownSide != nil || !arithmeticOperation(e.Operation) {
			return nil, ErrInvalid
		}
		return []string{"left", "right"}, nil
	case "rational_comparison":
		if e.UnknownSide != nil || e.Operation != "compare" {
			return nil, ErrInvalid
		}
		return []string{"left", "right"}, nil
	case "missing_operand":
		if e.UnknownSide == nil || (*e.UnknownSide != "left" && *e.UnknownSide != "right") || !arithmeticOperation(e.Operation) {
			return nil, ErrInvalid
		}
		return []string{"known", "result"}, nil
	default:
		return nil, ErrInvalid
	}
}
func arithmeticOperation(op string) bool {
	return op == "add" || op == "subtract" || op == "multiply" || op == "divide"
}
func normalizeTemplate(t Template) (Template, int, error) {
	t = canonicalValue(reflect.ValueOf(t)).Interface().(Template)
	if !ValidMathID(t.ID) || t.Version < 1 || t.Version > 2147483647 || !ValidMathID(t.Knowledge.ID) || t.Knowledge.Version < 1 || t.Knowledge.Version > 2147483647 {
		return Template{}, 0, ErrInvalid
	}
	names, e := engineParameters(t.Engine)
	if e != nil {
		return Template{}, 0, e
	}
	normalized, count, e := normalizeParameters(t.Parameters)
	t.Parameters = normalized
	if e != nil {
		return Template{}, 0, e
	}
	if len(t.Parameters) != len(names) {
		return Template{}, 0, ErrInvalid
	}
	for i, p := range t.Parameters {
		if p.Name != names[i] {
			return Template{}, 0, ErrInvalid
		}
	}
	if t.Type == "numeric" {
		if t.AnswerFormat == nil || (*t.AnswerFormat != "rational" && *t.AnswerFormat != "percentage") || len(t.Distractors) != 0 || t.Engine.Family == "rational_comparison" {
			return Template{}, 0, ErrInvalid
		}
	} else if t.Type == "single_choice" {
		if t.AnswerFormat != nil {
			return Template{}, 0, ErrInvalid
		}
		if t.Engine.Family == "rational_comparison" {
			if len(t.Distractors) != 0 {
				return Template{}, 0, ErrInvalid
			}
		} else if len(t.Distractors) == 0 || len(t.Distractors) > 4 {
			return Template{}, 0, ErrInvalid
		}
	} else {
		return Template{}, 0, ErrInvalid
	}
	seen := map[Constraint]bool{}
	for _, c := range t.Constraints {
		if seen[c] {
			return Template{}, 0, ErrInvalid
		}
		seen[c] = true
		switch c {
		case NonzeroDivisor:
			if t.Engine.Operation != "divide" || (t.Engine.Family != "rational_arithmetic" && (t.Engine.Family != "missing_operand" || *t.Engine.UnknownSide != "left")) {
				return Template{}, 0, ErrInvalid
			}
		case NonnegativeResult:
			if t.Engine.Family == "rational_comparison" {
				return Template{}, 0, ErrInvalid
			}
		case DistinctOperands:
			if t.Engine.Family == "missing_operand" {
				return Template{}, 0, ErrInvalid
			}
		default:
			return Template{}, 0, ErrInvalid
		}
	}
	ds := map[Distractor]bool{}
	for _, d := range t.Distractors {
		if ds[d] || (d != Negate && d != PlusOne && d != MinusOne && d != Reciprocal) {
			return Template{}, 0, ErrInvalid
		}
		ds[d] = true
	}
	t.Constraints = append([]Constraint{}, t.Constraints...)
	sort.Slice(t.Constraints, func(i, j int) bool { return t.Constraints[i] < t.Constraints[j] })
	t.Distractors = append([]Distractor{}, t.Distractors...)
	sort.Slice(t.Distractors, func(i, j int) bool { return t.Distractors[i] < t.Distractors[j] })
	body := sortBody(QuestionBody{Coverage: t.Coverage, Units: t.Units, Assets: t.Assets})
	t.Coverage, t.Units, t.Assets = body.Coverage, body.Units, body.Assets
	if len(t.Assets) > 8 || len(t.PromptTemplate)+len(t.ExplanationTemplate) > MaxQuestionTextBytes {
		return Template{}, 0, ErrLimitExceeded
	}
	sample := map[string]string{}
	for _, name := range names {
		sample[name] = "1"
	}
	if _, e := renderQuestion(t.PromptTemplate, sample, "1", false, t.Assets); e != nil {
		return Template{}, 0, e
	}
	if _, e := renderQuestion(t.ExplanationTemplate, sample, "1", true, t.Assets); e != nil {
		return Template{}, 0, e
	}
	return t, count, nil
}

// generatedAnswer is generation-only. The verifier never calls it or Rat operation helpers.
func generatedAnswer(engine EngineSpec, p map[string]*big.Rat) (*big.Rat, string, error) {
	if engine.Family == "rational_comparison" {
		cmp := p["left"].Cmp(p["right"])
		relation := "eq"
		if cmp < 0 {
			relation = "lt"
		} else if cmp > 0 {
			relation = "gt"
		}
		return nil, relation, nil
	}
	if engine.Family == "rational_arithmetic" {
		a, b := p["left"], p["right"]
		out := new(big.Rat)
		switch engine.Operation {
		case "add":
			out.Add(a, b)
		case "subtract":
			out.Sub(a, b)
		case "multiply":
			out.Mul(a, b)
		case "divide":
			if b.Sign() == 0 {
				return nil, "", ErrInvalid
			}
			out.Quo(a, b)
		}
		return out, "", nil
	}
	known, result := p["known"], p["result"]
	out := new(big.Rat)
	switch engine.Operation {
	case "add":
		out.Sub(result, known)
	case "subtract":
		if *engine.UnknownSide == "left" {
			out.Add(result, known)
		} else {
			out.Sub(known, result)
		}
	case "multiply":
		if known.Sign() == 0 {
			return nil, "", ErrInvalid
		}
		out.Quo(result, known)
	case "divide":
		if *engine.UnknownSide == "left" {
			if known.Sign() == 0 {
				return nil, "", ErrInvalid
			}
			out.Mul(result, known)
		} else {
			if result.Sign() == 0 || known.Sign() == 0 {
				return nil, "", ErrInvalid
			}
			out.Quo(known, result)
		}
	}
	return out, "", nil
}
func excludedCombination(t Template, p map[string]*big.Rat, counts map[Constraint]int) (bool, error) {
	excluded := false

	for _, c := range t.Constraints {
		hit := false
		switch c {
		case NonzeroDivisor:
			if t.Engine.Family == "missing_operand" {
				hit = p["known"].Sign() == 0
			} else {
				hit = p["right"].Sign() == 0
			}
		case DistinctOperands:
			hit = p["left"].Cmp(p["right"]) == 0
		}
		if hit {
			counts[c]++
			excluded = true
		}
	}
	for _, c := range t.Constraints {
		if c != NonnegativeResult {
			continue
		}
		negative := false
		if t.Engine.Family == "missing_operand" {
			negative = p["result"].Sign() < 0
		} else if !(excluded && t.Engine.Operation == "divide" && p["right"].Sign() == 0) {
			answer, _, e := generatedAnswer(t.Engine, p)
			if e != nil {
				return false, e
			}
			negative = answer.Sign() < 0
		}
		if negative {
			counts[c]++
			excluded = true
		}
	}
	return excluded, nil
}
func numericChoices(answer *big.Rat, distractors []Distractor) ([]Choice, string, error) {
	values := []*big.Rat{new(big.Rat).Set(answer)}
	for _, rule := range distractors {
		v := new(big.Rat)
		switch rule {
		case Negate:
			v.Neg(answer)
		case PlusOne:
			v.Add(answer, big.NewRat(1, 1))
		case MinusOne:
			v.Sub(answer, big.NewRat(1, 1))
		case Reciprocal:
			if answer.Sign() == 0 {
				return nil, "", ErrInvalid
			}
			v.Inv(answer)
		}
		values = append(values, v)
	}
	choices := []Choice{}
	seen := map[string]bool{}
	correctID := ""
	for index, v := range values {
		r, e := rationalFromRat(v)
		if e != nil {
			return nil, "", e
		}
		literal := r.Numerator + "/" + r.Denominator
		if seen[literal] {
			return nil, "", ErrInvalid
		}
		seen[literal] = true
		h := sha256.Sum256([]byte(literal))
		id := fmt.Sprintf("choice-%x", h[:12])
		if index == 0 {
			correctID = id
		}
		choices = append(choices, Choice{ID: id, Text: displayRational(r)})
	}
	sort.Slice(choices, func(i, j int) bool { return choices[i].ID < choices[j].ID })
	return choices, correctID, nil
}
func Generate(ctx context.Context, t Template) ([]Instance, GenerationReport, error) {
	report := GenerationReport{Template: Identity{ID: t.ID, Version: t.Version}, GeneratorVersion: t.Engine.GeneratorVersion, VerifierVersion: t.Engine.VerifierVersion, ConstraintCounts: []ConstraintCount{}}
	if e := ctx.Err(); e != nil {
		return nil, report, e
	}
	t, rawCount, e := normalizeTemplate(t)
	if e != nil {
		return nil, report, e
	}
	_, templateSHA, e := canonical("question-template-v1", t)
	if e != nil {
		return nil, report, e
	}
	report.Template.SHA256 = templateSHA
	report.RawCombinations = rawCount
	instances := make([]Instance, 0, rawCount)
	constraintCounts := map[Constraint]int{}
	totalBytes := 0
	var failed error
	for combination := 0; combination < rawCount; combination++ {
		if e = ctx.Err(); e != nil {
			return nil, report, e
		}
		params := make([]ParameterValue, len(t.Parameters))
		values := map[string]*big.Rat{}
		display := map[string]string{}
		n := combination
		for j := len(t.Parameters) - 1; j >= 0; j-- {
			p := t.Parameters[j]
			literal := p.Values[n%len(p.Values)]
			n /= len(p.Values)
			params[j] = ParameterValue{Name: p.Name, Value: literal}
			v, _ := parseLiteral(literal)
			values[p.Name] = v
			r, _ := rationalFromRat(v)
			display[p.Name] = displayRational(r)
		}
		exclude, e := excludedCombination(t, values, constraintCounts)
		if e != nil {
			failed = e
			continue
		}
		if exclude {
			report.ExcludedCombinations++
			continue
		}
		answer, relation, e := generatedAnswer(t.Engine, values)
		if e != nil {
			failed = e
			continue
		}
		body := QuestionBody{Type: t.Type, Knowledge: t.Knowledge, Coverage: t.Coverage, Units: t.Units, AnswerFormat: t.AnswerFormat, Choices: []Choice{}, Witness: &VerificationWitness{Engine: t.Engine, Parameters: params}, Assets: t.Assets, Sources: t.Sources}
		answerText := relation
		if answer != nil {
			r, e := rationalFromRat(answer)
			if e != nil {
				failed = e
				continue
			}
			answerText = displayRational(r)
			if t.Type == "numeric" {
				if !CanEnterAnswer(r, *t.AnswerFormat) {
					failed = ErrNotReady
					continue
				}
				body.CorrectNumeric = &r
			} else {
				choices, correct, e := numericChoices(answer, t.Distractors)
				if e != nil {
					failed = e
					continue
				}
				body.Choices = choices
				body.CorrectChoiceID = &correct
			}
		} else {
			body.Choices = []Choice{{ID: "lt", Text: "lt"}, {ID: "eq", Text: "eq"}, {ID: "gt", Text: "gt"}}
			body.CorrectChoiceID = &relation
		}
		body.Prompt, e = renderQuestion(t.PromptTemplate, display, answerText, false, t.Assets)
		if e != nil {
			return nil, report, e
		}
		body.Explanation, e = renderQuestion(t.ExplanationTemplate, display, answerText, true, t.Assets)
		if e != nil {
			return nil, report, e
		}
		if len(body.Prompt)+len(body.Explanation) > MaxQuestionTextBytes {
			return nil, report, ErrLimitExceeded
		}
		_, identitySHA, e := canonical("question-instance-identity-v1", struct {
			Template         Identity         `json:"template"`
			Parameters       []ParameterValue `json:"parameters"`
			GeneratorVersion int              `json:"generatorVersion"`
			VerifierVersion  int              `json:"verifierVersion"`
		}{report.Template, params, 1, 1})
		if e != nil {
			return nil, report, e
		}
		gv, vv := 1, 1
		templateIdentity := report.Template
		instance := Instance{Identity: Identity{ID: "qi-" + identitySHA, Version: 1}, Origin: "template", Template: &templateIdentity, Parameters: params, GeneratorVersion: &gv, VerifierVersion: &vv, Body: body}
		canonicalBytes, bodySHA, e := CanonicalInstance(instance)
		if e != nil {
			return nil, report, e
		}
		instance.Identity.SHA256 = bodySHA
		totalBytes += len(canonicalBytes)
		if totalBytes > MaxEnvelopeBytes {
			return nil, report, ErrLimitExceeded
		}
		if e = VerifyInstance(instance); e != nil {
			failed = e
			continue
		}
		instances = append(instances, instance)
		report.ValidInstances++
	}
	for _, c := range t.Constraints {
		report.ConstraintCounts = append(report.ConstraintCounts, ConstraintCount{Constraint: c, Count: constraintCounts[c]})
	}
	if failed != nil {
		return nil, report, failed
	}
	if len(instances) == 0 {
		return nil, report, ErrNotReady
	}
	sort.Slice(instances, func(i, j int) bool { return instances[i].Identity.ID < instances[j].Identity.ID })
	return instances, report, nil
}
