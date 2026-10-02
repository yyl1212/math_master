package httpapi

import (
	"bytes"
	"encoding/json"
	"github.com/yyl1212/math_master/backend/internal/auth"
	"github.com/yyl1212/math_master/backend/internal/question"
	"io"
)

func decodeQuestionJSON(reader io.Reader, limit int64, out any) error {
	raw, e := io.ReadAll(io.LimitReader(reader, limit+1))
	if int64(len(raw)) > limit {
		return errQuestionPayloadTooLarge
	}
	if e != nil {
		return auth.ErrInvalidInput
	}
	if e = question.DecodeStrictJSON(bytes.NewReader(raw), int(limit), out); e != nil {
		return auth.ErrInvalidInput
	}
	switch out.(type) {
	case *question.DraftInput, *question.SaveDraftInput:
		var fields map[string]json.RawMessage
		if json.Unmarshal(raw, &fields) != nil {
			return auth.ErrInvalidInput
		}
		if len(fields["questionPackage"]) > question.MaxPackageBytes {
			return question.ErrLimitExceeded
		}
		if _, e = question.DecodePackage(bytes.NewReader(fields["questionPackage"])); e != nil {
			return auth.ErrInvalidInput
		}
	}
	return nil
}
