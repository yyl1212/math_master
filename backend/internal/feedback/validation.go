package feedback

import (
	"github.com/yyl1212/math_master/backend/internal/auth"
	"github.com/yyl1212/math_master/backend/internal/question"
	"strings"
	"unicode/utf8"
)

func ValidCategory(c Category) bool {
	switch c {
	case "math_error", "unclear_explanation", "typo", "accessibility", "technical_issue", "suggestion":
		return true
	}
	return false
}
func ValidArea(a Area) bool {
	switch a {
	case "home", "knowledge_map", "learning_center", "account", "review", "other":
		return true
	}
	return false
}
func validUUID(s string) bool { return question.ValidID(s) }
func validIdentity(i *question.Identity, instance bool) bool {
	if i == nil || i.Version < 1 || i.Version > 2147483647 || !question.ValidSHA(i.SHA256) {
		return false
	}
	if instance {
		return question.ValidInstanceID(i.ID)
	}
	return question.ValidMathID(i.ID)
}
func validAsset(a *AssetRef) bool {
	return a != nil && question.ValidMathID(a.ID) && question.ValidSHA(a.SHA256)
}
func validText(s string, min, max int) bool {
	if !utf8.ValidString(s) || strings.ContainsRune(s, 0) {
		return false
	}
	n := utf8.RuneCountInString(s)
	return n >= min && n <= max && (n == 0 && min == 0 || strings.TrimSpace(s) != "")
}
func ValidateCreate(in CreateInput) error {
	if !ValidCategory(in.Category) || !validText(in.Title, 1, 120) || !validText(in.Message, 1, 4000) || !validText(in.Location, 0, 400) {
		return auth.ErrInvalidInput
	}
	if ValidateTargetSource(in.Target, in.Source) != nil {
		return auth.ErrInvalidInput
	}
	if in.Target.Kind == "site" && (in.Category == "math_error" || in.Category == "unclear_explanation") {
		return auth.ErrInvalidInput
	}
	return nil
}

// ValidateTargetSource validates identity shapes; the repository proves actual ownership and publication.
func ValidateTargetSource(t Target, s Source) error {
	if t.Kind == "site" {
		if t.Identity != nil || t.Part != nil || t.Area == nil || !ValidArea(*t.Area) || s.Kind != "site" || s.PublicationID != nil || s.AttemptID != nil || s.Position != nil {
			return auth.ErrInvalidInput
		}
		return nil
	}
	if t.Area != nil {
		return auth.ErrInvalidInput
	}
	switch t.Kind {
	case "knowledge", "path":
		if !validIdentity(t.Identity, false) || s.Kind != "publication" || s.PublicationID == nil || !validUUID(*s.PublicationID) || s.AttemptID != nil || s.Position != nil {
			return auth.ErrInvalidInput
		}
	case "instance":
		if !validIdentity(t.Identity, true) || s.PublicationID != nil || s.AttemptID == nil || !validUUID(*s.AttemptID) || s.Position == nil {
			return auth.ErrInvalidInput
		}
		if s.Kind == "practice" {
			if *s.Position != 1 {
				return auth.ErrInvalidInput
			}
		} else if s.Kind == "assessment" {
			if *s.Position < 1 || *s.Position > 5 {
				return auth.ErrInvalidInput
			}
		} else {
			return auth.ErrInvalidInput
		}
	default:
		return auth.ErrInvalidInput
	}
	if t.Part != nil {
		switch t.Part.Kind {
		case "unit":
			if t.Kind != "knowledge" || t.Part.Asset != nil || !validIdentity(t.Part.Unit, false) {
				return auth.ErrInvalidInput
			}
		case "asset":
			if (t.Kind != "knowledge" && t.Kind != "instance") || t.Part.Unit != nil || !validAsset(t.Part.Asset) {
				return auth.ErrInvalidInput
			}
		default:
			return auth.ErrInvalidInput
		}
	}
	return nil
}
func ValidateReply(in ReplyInput) error {
	if in.ExpectedSequence < 1 || in.ExpectedSequence > MaxSequence || !validText(in.Message, 1, 4000) {
		return auth.ErrInvalidInput
	}
	return nil
}
