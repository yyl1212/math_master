package study

import "github.com/yyl1212/math_master/backend/internal/taxonomy"

func InputDigest(in any) (string, error) {
	switch v := in.(type) {
	case CommandInput:
		if ValidateCommand(v) != nil {
			return "", ErrInvalid
		}
	case ReviewInput:
		if ValidateCommand(v.CommandInput) != nil || !ValidID(v.ReviewID) {
			return "", ErrInvalid
		}
	case NoteInput:
		if v.ExpectedRevision < 0 || v.ExpectedRevision > MaxSequence || !ValidRef(v.Knowledge) || ValidateNote(v.Body) != nil {
			return "", ErrInvalid
		}
	case NoteDeleteInput:
		if v.ExpectedRevision < 0 || v.ExpectedRevision > MaxSequence {
			return "", ErrInvalid
		}
	default:
		return "", ErrInvalid
	}
	return taxonomy.Digest(in)
}
