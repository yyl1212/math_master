package notification

import (
	"github.com/yyl1212/math_master/backend/internal/auth"
	"github.com/yyl1212/math_master/backend/internal/correction"
	"github.com/yyl1212/math_master/backend/internal/question"
)

func ValidateQuery(q Query) error {
	return correction.ValidateQuery(correction.Query{Limit: q.Limit, Cursor: q.Cursor})
}
func ValidateSource(s Source) error {
	if len(s.DedupKey) > 512 || !correction.ValidText(s.DedupKey, 1, 512) || !correction.ValidEvidence(s.Evidence, true) || !question.ValidID(s.CaseID) || s.ResultID != nil && !question.ValidID(*s.ResultID) {
		return auth.ErrInvalidInput
	}
	switch s.Type {
	case Checking, Corrected, Retake, ReviewMaterial, PathUnavailable:
		return nil
	}
	return auth.ErrInvalidInput
}
