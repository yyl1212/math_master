package question

import (
	"github.com/yyl1212/math_master/backend/internal/auth"
	"github.com/yyl1212/math_master/backend/internal/publication"
	"strings"
	"unicode/utf8"
)

func ValidateReviewInput(v ReviewInput, hasTemplates bool) error {
	for _, s := range []string{v.Decision, v.IndependenceNote, v.GenerationNote, v.Note} {
		if !utf8.ValidString(s) || strings.ContainsRune(s, 0) {
			return auth.ErrInvalidInput
		}
	}
	if v.Decision != "approve" && v.Decision != "return" {
		return auth.ErrInvalidInput
	}
	if !publication.ValidNote(v.Note) {
		return ErrNotReady
	}
	if v.GenerationNote != "" && !publication.ValidNote(v.GenerationNote) {
		return ErrNotReady
	}
	if v.Decision == "approve" {
		c := v.Checks
		if !c.Mathematics || !c.Explanations || !c.Objectives || !c.Sources || !c.Illustrations || !c.Generation || !publication.ValidNote(v.IndependenceNote) || (!hasTemplates && !publication.ValidNote(v.GenerationNote)) {
			return ErrNotReady
		}
	} else if v.IndependenceNote != "" && !publication.ValidNote(v.IndependenceNote) {
		return auth.ErrInvalidInput
	}
	return nil
}
