package study

func ValidState(s State) bool {
	return s == Unlearned || s == Learning || s == Completed || s == Reviewing
}
func ApplyState(s State, a Action, completedRef, currentRef KnowledgeRef) (State, error) {
	if a == Complete && s == Reviewing && completedRef != currentRef {
		return s, ErrStateConflict
	}
	return TransitionState(s, a)
}
func TransitionState(s State, a Action) (State, error) {
	if !ValidState(s) {
		return s, ErrInvalid
	}
	switch a {
	case Begin:
		if s == Unlearned {
			return Learning, nil
		}
		return s, nil
	case Complete:
		if s == Unlearned {
			return s, ErrStateConflict
		}
		if s == Reviewing {
			return s, nil
		}
		return Completed, nil
	case StartReview:
		if s == Completed || s == Reviewing {
			return Reviewing, nil
		}
		return s, ErrStateConflict
	case FinishReview:
		if s == Reviewing || s == Completed {
			return Completed, nil
		}
		return s, ErrStateConflict
	case SaveNote, DeleteNote:
		return s, nil
	default:
		return s, ErrInvalid
	}
}
func MaterialChanged(completed, current *KnowledgeRef) bool {
	return completed != nil && current != nil && *completed != *current
}
