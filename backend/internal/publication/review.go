package publication

import (
	"github.com/yyl1212/math_master/backend/internal/auth"
	"reflect"
)

func ValidateReviewInput(v ReviewInput) error {
	if !validStrings(reflect.ValueOf(v)) || v.Decision != "approve" && v.Decision != "return" {
		return auth.ErrInvalidInput
	}
	if !ValidNote(v.Note) {
		return ErrContentNotReady
	}
	if v.Decision == "approve" {
		c := v.Checks
		if !c.Mathematics || !c.Explanations || !c.Relationships || !c.Sources || !c.Illustrations || !ValidNote(v.IndependenceNote) {
			return ErrContentNotReady
		}
	} else if v.IndependenceNote != "" && !ValidNote(v.IndependenceNote) {
		return auth.ErrInvalidInput
	}
	return nil
}
