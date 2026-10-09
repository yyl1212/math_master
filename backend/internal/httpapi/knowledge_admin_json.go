package httpapi

import (
	"bytes"
	"encoding/json"
	"errors"
	"github.com/yyl1212/math_master/backend/internal/knowledgeadmin"
	"io"
)

func decodeKnowledgeJSON(r io.Reader, limit int64, out any) error {
	b, e := io.ReadAll(io.LimitReader(r, limit+1))
	if e != nil {
		return knowledgeadmin.ErrInvalid
	}
	if int64(len(b)) > limit {
		return &knowledgeadmin.DecodeError{Code: "INPUT_TOO_LARGE", Path: "/"}
	}
	if _, e = knowledgeadmin.StrictJSON(b); e != nil {
		return e
	}
	d := json.NewDecoder(bytes.NewReader(b))
	d.DisallowUnknownFields()
	if e = d.Decode(out); e != nil {
		return knowledgeadmin.ErrInvalid
	}
	if e = d.Decode(new(any)); !errors.Is(e, io.EOF) {
		return knowledgeadmin.ErrInvalid
	}
	return nil
}
