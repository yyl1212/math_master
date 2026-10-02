package httpapi

import (
	"bytes"
	"encoding/json"
	"github.com/yyl1212/math_master/backend/internal/assessment"
	"github.com/yyl1212/math_master/backend/internal/auth"
	"github.com/yyl1212/math_master/backend/internal/learning"
	"github.com/yyl1212/math_master/backend/internal/question"
	"io"
	"strings"
	"unicode/utf8"
)

const learningBodyLimit = 8192

func learningNoNull(v any) bool {
	switch x := v.(type) {
	case nil:
		return false
	case string:
		return !strings.ContainsRune(x, 0)
	case map[string]any:
		for k, w := range x {
			if strings.ContainsRune(k, 0) || !learningNoNull(w) {
				return false
			}
		}
	case []any:
		for _, w := range x {
			if !learningNoNull(w) {
				return false
			}
		}
	}
	return true
}
func learningShape(raw []byte, names ...string) (map[string]json.RawMessage, error) {
	var fields map[string]json.RawMessage
	if json.Unmarshal(raw, &fields) != nil || fields == nil || len(fields) != len(names) {
		return nil, auth.ErrInvalidInput
	}
	for _, n := range names {
		if _, ok := fields[n]; !ok {
			return nil, auth.ErrInvalidInput
		}
	}
	return fields, nil
}
func learningDecode(raw []byte, out any) error {
	if question.DecodeStrictJSON(bytes.NewReader(raw), learningBodyLimit, out) != nil {
		return auth.ErrInvalidInput
	}
	return nil
}
func learningIdentity(i question.Identity) bool {
	return question.ValidMathID(i.ID) && i.Version > 0 && i.Version <= 2147483647 && question.ValidSHA(i.SHA256)
}
func learningAnswer(raw []byte, skipped bool) (assessment.Answer, error) {
	var a assessment.Answer
	var kind struct{ Kind string }
	if json.Unmarshal(raw, &kind) != nil {
		return a, auth.ErrInvalidInput
	}
	switch kind.Kind {
	case "choice":
		var v struct {
			Kind     string `json:"kind"`
			ChoiceID string `json:"choiceId"`
		}
		if e := learningDecode(raw, &v); e != nil || !question.ValidMathID(v.ChoiceID) {
			return a, auth.ErrInvalidInput
		}
		a = assessment.Answer{Kind: v.Kind, ChoiceID: &v.ChoiceID}
	case "numeric":
		var v struct {
			Kind string `json:"kind"`
			Raw  string `json:"raw"`
		}
		if e := learningDecode(raw, &v); e != nil {
			return a, e
		}
		if utf8.RuneCountInString(v.Raw) > 128 {
			return a, &question.NumericFormatError{Code: "INPUT_TOO_LONG"}
		}
		a = assessment.Answer{Kind: v.Kind, Raw: &v.Raw}
	case "skipped":
		var v struct {
			Kind string `json:"kind"`
		}
		if !skipped || learningDecode(raw, &v) != nil {
			return a, auth.ErrInvalidInput
		}
		a.Kind = v.Kind
	default:
		return a, auth.ErrInvalidInput
	}
	return a, nil
}

// DecodeLearningInput keeps original numeric spelling and enforces the tagged branches.
func DecodeLearningInput(raw []byte, action learning.Action) (any, error) {
	if len(raw) > learningBodyLimit {
		return nil, errQuestionPayloadTooLarge
	}
	if !utf8.Valid(raw) || !json.Valid(raw) || !validPrivateEscapes(raw) {
		return nil, auth.ErrInvalidInput
	}
	var generic any
	if json.Unmarshal(raw, &generic) != nil || !learningNoNull(generic) {
		return nil, auth.ErrInvalidInput
	}
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	first, e := dec.Token()
	if e != nil || first != json.Delim('{') {
		return nil, auth.ErrInvalidInput
	}
	if e = walkPrivateJSON(dec, first, 1, nil); e != nil {
		return nil, e
	}
	if _, e = dec.Token(); e != io.EOF {
		return nil, auth.ErrInvalidInput
	}
	switch action {
	case learning.StartKnowledgeAction, learning.CompleteKnowledgeAction:
		var v learning.StartInput
		if learningDecode(raw, &v) != nil || !learningIdentity(v.Knowledge) || !question.ValidID(v.ExpectedKnowledgeHead) {
			return nil, auth.ErrInvalidInput
		}
		return v, nil
	case learning.EnrollPathAction:
		var v learning.EnrollInput
		if learningDecode(raw, &v) != nil || !learningIdentity(v.Path) || !question.ValidID(v.ExpectedKnowledgeHead) {
			return nil, auth.ErrInvalidInput
		}
		return v, nil
	case learning.CreatePracticeAction:
		var v assessment.PracticeCreateInput
		if learningDecode(raw, &v) != nil || !learningIdentity(v.Knowledge) || !question.ValidID(v.ExpectedKnowledgeHead) || !question.ValidID(v.ExpectedQuestionHead) {
			return nil, auth.ErrInvalidInput
		}
		return v, nil
	case learning.CreateAssessmentAction:
		var v assessment.CreateInput
		if learningDecode(raw, &v) != nil || !learningIdentity(v.Knowledge) || !learningIdentity(v.Blueprint) || !question.ValidID(v.ExpectedKnowledgeHead) || !question.ValidID(v.ExpectedQuestionHead) || (v.Mode != assessment.ModeNode && v.Mode != assessment.ModeDiagnostic && v.Mode != assessment.ModeReview) {
			return nil, auth.ErrInvalidInput
		}
		return v, nil
	case learning.AnswerPracticeAction:
		return learningAnswer(raw, false)
	case learning.RevealPracticeAction, learning.AbandonPracticeAction, learning.AbandonAssessmentAction:
		var v struct{}
		return v, learningDecode(raw, &v)
	case learning.SubmitAssessmentAction:
		fields, e := learningShape(raw, "answers")
		if e != nil {
			return nil, e
		}
		var entries []json.RawMessage
		if json.Unmarshal(fields["answers"], &entries) != nil || len(entries) != 5 {
			return nil, auth.ErrInvalidInput
		}
		v := assessment.SubmitInput{Answers: []assessment.PositionAnswer{}}
		positions := map[int]bool{}
		ids := map[question.Identity]bool{}
		for _, entry := range entries {
			f, e := learningShape(entry, "position", "instance", "answer")
			if e != nil {
				return nil, e
			}
			var p assessment.PositionAnswer
			if learningDecode(f["position"], &p.Position) != nil || p.Position < 1 || p.Position > 5 || positions[p.Position] || learningDecode(f["instance"], &p.Instance) != nil || !(question.ValidInstanceID(p.Instance.ID) && p.Instance.Version > 0 && p.Instance.Version <= 2147483647 && question.ValidSHA(p.Instance.SHA256)) || ids[p.Instance] {
				return nil, auth.ErrInvalidInput
			}
			p.Answer, e = learningAnswer(f["answer"], true)
			if e != nil {
				return nil, e
			}
			positions[p.Position] = true
			ids[p.Instance] = true
			v.Answers = append(v.Answers, p)
		}
		return v, nil
	}
	return nil, auth.ErrInvalidInput
}
