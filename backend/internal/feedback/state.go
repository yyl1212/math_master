package feedback

import "github.com/yyl1212/math_master/backend/internal/auth"

func ValidStatus(s Status) bool {
	switch s {
	case New, Processing, WaitingDetails, Resolved, Closed:
		return true
	}
	return false
}
func OwnerReplyState(s Status) (Status, error) {
	switch s {
	case New, Processing:
		return s, nil
	case WaitingDetails, Resolved, Closed:
		return Processing, nil
	}
	return "", auth.ErrInvalidInput
}
func ValidateTransition(from Status, in TransitionInput) error {
	if !ValidStatus(from) || ValidateTransitionInput(in) != nil {
		return auth.ErrInvalidInput
	}
	if from == in.Status {
		if in.Resolution != nil {
			return auth.ErrInvalidInput
		}
		return nil
	}
	allowed := false
	switch from {
	case New:
		allowed = in.Status == Processing || in.Status == WaitingDetails || in.Status == Closed
	case Processing:
		allowed = in.Status == WaitingDetails || in.Status == Resolved || in.Status == Closed
	case WaitingDetails:
		allowed = in.Status == Processing || in.Status == Resolved || in.Status == Closed
	case Resolved, Closed:
		allowed = in.Status == Processing
	}
	if !allowed {
		return auth.ErrInvalidInput
	}
	if in.Status == Resolved || in.Status == Closed {
		if in.Resolution == nil {
			return auth.ErrInvalidInput
		}
	}
	return nil
}
func ValidateTransitionInput(in TransitionInput) error {
	if ValidateReply(ReplyInput{ExpectedSequence: in.ExpectedSequence, Message: in.Message}) != nil || !ValidStatus(in.Status) {
		return auth.ErrInvalidInput
	}
	if in.Resolution == nil {
		return nil
	}
	if in.Status != Resolved && in.Status != Closed {
		return auth.ErrInvalidInput
	}
	return validateResolution(in.Status, *in.Resolution)
}
func validateResolution(status Status, r Resolution) error {
	if r.Withdrawal != nil && ((r.Withdrawal.Space != "content" && r.Withdrawal.Space != "question") || !validUUID(r.Withdrawal.ID)) {
		return auth.ErrInvalidInput
	}
	if r.Replacement != nil && !validReplacement(*r.Replacement) {
		return auth.ErrInvalidInput
	}
	switch r.Kind {
	case "clarified", "service_fixed":
		if status != Resolved || r.Withdrawal != nil || r.Replacement != nil || r.DuplicateOf != nil {
			return auth.ErrInvalidInput
		}
	case "withdrawn":
		if status != Resolved || r.Withdrawal == nil || r.Replacement != nil || r.DuplicateOf != nil {
			return auth.ErrInvalidInput
		}
	case "revision_published":
		if status != Resolved || r.Withdrawal == nil || r.Replacement == nil || r.DuplicateOf != nil {
			return auth.ErrInvalidInput
		}
	case "duplicate":
		if status != Closed || r.Withdrawal != nil || r.Replacement != nil || r.DuplicateOf == nil || !validUUID(*r.DuplicateOf) {
			return auth.ErrInvalidInput
		}
	case "not_reproducible", "out_of_scope", "suggestion_recorded":
		if status != Closed || r.Withdrawal != nil || r.Replacement != nil || r.DuplicateOf != nil {
			return auth.ErrInvalidInput
		}
	default:
		return auth.ErrInvalidInput
	}
	return nil
}
func validReplacement(r Replacement) bool {
	if !validUUID(r.PublicationID) {
		return false
	}
	if r.Kind == "asset" {
		return r.Identity == nil && validAsset(r.Asset)
	}
	if r.Asset != nil {
		return false
	}
	switch r.Kind {
	case "knowledge", "path", "unit":
		return validIdentity(r.Identity, false)
	case "instance":
		return validIdentity(r.Identity, true)
	}
	return false
}
