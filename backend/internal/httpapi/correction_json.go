package httpapi

import (
	"bytes"
	"encoding/json"
	"github.com/yyl1212/math_master/backend/internal/auth"
	"github.com/yyl1212/math_master/backend/internal/correction"
	"io"
	"reflect"
	"unicode/utf8"
)

func correctionDecode(raw []byte, out any) error {
	if len(raw) > correction.MaxRequestBytes || !utf8.Valid(raw) || !json.Valid(raw) || !validPrivateEscapes(raw) {
		return auth.ErrInvalidInput
	}
	var generic any
	if json.Unmarshal(raw, &generic) != nil || !feedbackNoNUL(generic) {
		return auth.ErrInvalidInput
	}
	d := json.NewDecoder(bytes.NewReader(raw))
	d.UseNumber()
	first, e := d.Token()
	if e != nil || first != json.Delim('{') || feedbackWalkJSON(d, first, 1, nil) != nil {
		return auth.ErrInvalidInput
	}
	if _, e = d.Token(); e != io.EOF {
		return auth.ErrInvalidInput
	}
	d = json.NewDecoder(bytes.NewReader(raw))
	d.DisallowUnknownFields()
	if d.Decode(out) != nil {
		return auth.ErrInvalidInput
	}
	encoded, e := json.Marshal(out)
	var canonical any
	if e != nil || json.Unmarshal(encoded, &canonical) != nil || !reflect.DeepEqual(generic, canonical) {
		return auth.ErrInvalidInput
	}
	return nil
}
func DecodeCorrectionInput(raw []byte, action correction.Action) (any, error) {
	var out any
	switch action {
	case correction.CreateCaseAction:
		out = &correction.CaseInput{}
	case correction.CreatePlanAction, correction.UpdatePlanAction:
		out = &correction.PlanInput{}
	case correction.SubmitPlanAction:
		out = &correction.SubmitInput{}
	case correction.DecidePlanAction:
		out = &correction.DecisionInput{}
	case correction.RetryJobAction:
		out = &correction.RetryInput{}
	default:
		return nil, auth.ErrInvalidInput
	}
	if e := correctionDecode(raw, out); e != nil {
		return nil, e
	}
	switch v := out.(type) {
	case *correction.CaseInput:
		if e := correction.ValidateCase(*v); e != nil {
			return nil, e
		}
		return *v, nil
	case *correction.PlanInput:
		if e := correction.ValidatePlan(*v); e != nil || action == correction.CreatePlanAction && v.ExpectedSequence != nil || action == correction.UpdatePlanAction && v.ExpectedSequence == nil {
			return nil, auth.ErrInvalidInput
		}
		return *v, nil
	case *correction.SubmitInput:
		if e := correction.ValidateSubmit(*v); e != nil {
			return nil, e
		}
		return *v, nil
	case *correction.DecisionInput:
		if e := correction.ValidateDecision(*v); e != nil {
			return nil, e
		}
		return *v, nil
	case *correction.RetryInput:
		if e := correction.ValidateRetry(*v); e != nil {
			return nil, e
		}
		return *v, nil
	}
	return nil, auth.ErrInvalidInput
}
func readCorrectionInput(r io.Reader, a correction.Action) (any, error) {
	raw, e := io.ReadAll(io.LimitReader(r, correction.MaxRequestBytes+1))
	if e != nil {
		return nil, auth.ErrInvalidInput
	}
	return DecodeCorrectionInput(raw, a)
}
