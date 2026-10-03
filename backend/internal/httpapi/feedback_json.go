package httpapi

import (
	"bytes"
	"encoding/json"
	"github.com/yyl1212/math_master/backend/internal/auth"
	"github.com/yyl1212/math_master/backend/internal/feedback"
	"io"
	"reflect"
	"strings"
	"unicode/utf8"
)

func feedbackNoNUL(v any) bool {
	switch x := v.(type) {
	case string:
		return !strings.ContainsRune(x, 0)
	case map[string]any:
		for k, v := range x {
			if strings.ContainsRune(k, 0) || !feedbackNoNUL(v) {
				return false
			}
		}
	case []any:
		for _, v := range x {
			if !feedbackNoNUL(v) {
				return false
			}
		}
	}
	return true
}
func DecodeFeedbackInput(raw []byte, action feedback.Action) (any, error) {
	if len(raw) > feedback.MaxRequestBytes || !utf8.Valid(raw) || !json.Valid(raw) || !validPrivateEscapes(raw) {
		return nil, auth.ErrInvalidInput
	}
	var generic any
	if json.Unmarshal(raw, &generic) != nil || !feedbackNoNUL(generic) {
		return nil, auth.ErrInvalidInput
	}
	d := json.NewDecoder(bytes.NewReader(raw))
	d.UseNumber()
	first, e := d.Token()
	if e != nil || first != json.Delim('{') || feedbackWalkJSON(d, first, 1, nil) != nil {
		return nil, auth.ErrInvalidInput
	}
	if _, e = d.Token(); e != io.EOF {
		return nil, auth.ErrInvalidInput
	}
	var out any
	switch action {
	case feedback.CreateAction:
		out = &feedback.CreateInput{}
	case feedback.ReplyAction:
		out = &feedback.ReplyInput{}
	case feedback.TransitionAction:
		out = &feedback.TransitionInput{}
	default:
		return nil, auth.ErrInvalidInput
	}
	d = json.NewDecoder(bytes.NewReader(raw))
	d.DisallowUnknownFields()
	if d.Decode(out) != nil {
		return nil, auth.ErrInvalidInput
	}
	encoded, e := json.Marshal(out)
	if e != nil {
		return nil, auth.ErrInvalidInput
	}
	var canonical any
	if json.Unmarshal(encoded, &canonical) != nil || !reflect.DeepEqual(generic, canonical) {
		return nil, auth.ErrInvalidInput
	}
	switch v := out.(type) {
	case *feedback.CreateInput:
		if feedback.ValidateCreate(*v) != nil {
			return nil, auth.ErrInvalidInput
		}
		return *v, nil
	case *feedback.ReplyInput:
		if feedback.ValidateReply(*v) != nil {
			return nil, auth.ErrInvalidInput
		}
		return *v, nil
	case *feedback.TransitionInput:
		if feedback.ValidateTransitionInput(*v) != nil {
			return nil, auth.ErrInvalidInput
		}
		return *v, nil
	}
	return nil, auth.ErrInvalidInput
}
func readFeedbackInput(r io.Reader, action feedback.Action) (any, error) {
	raw, e := io.ReadAll(io.LimitReader(r, feedback.MaxRequestBytes+1))
	if e != nil {
		return nil, auth.ErrInvalidInput
	}
	return DecodeFeedbackInput(raw, action)
}

func feedbackWalkJSON(dec *json.Decoder, first json.Token, depth int, allowed map[string]bool) error {
	if first == nil {
		return nil
	}
	if depth > 8 {
		return auth.ErrInvalidInput
	}
	delim, ok := first.(json.Delim)
	if !ok {
		return nil
	}
	switch delim {
	case '{':
		seen := make(map[string]bool)
		for dec.More() {
			token, err := dec.Token()
			if err != nil {
				return auth.ErrInvalidInput
			}
			key, ok := token.(string)
			if !ok {
				return auth.ErrInvalidInput
			}
			fold := strings.ToLower(key)
			if seen[fold] || (allowed != nil && !allowed[key]) {
				return auth.ErrInvalidInput
			}
			seen[fold] = true
			value, err := dec.Token()
			if err != nil {
				return auth.ErrInvalidInput
			}
			if err = feedbackWalkJSON(dec, value, depth+1, nil); err != nil {
				return err
			}
		}
		end, err := dec.Token()
		if err != nil || end != json.Delim('}') {
			return auth.ErrInvalidInput
		}
	case '[':
		for dec.More() {
			value, err := dec.Token()
			if err != nil {
				return auth.ErrInvalidInput
			}
			if err = feedbackWalkJSON(dec, value, depth+1, nil); err != nil {
				return err
			}
		}
		end, err := dec.Token()
		if err != nil || end != json.Delim(']') {
			return auth.ErrInvalidInput
		}
	default:
		return auth.ErrInvalidInput
	}
	return nil
}
