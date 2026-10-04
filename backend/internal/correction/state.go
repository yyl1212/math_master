package correction

// approve/reject are internal decision actions; Authorize rejects both.
func NextPlanState(s PlanStatus, a Action) (PlanStatus, error) {
	switch {
	case s == Draft && a == UpdatePlanAction:
		return Draft, nil
	case s == Draft && a == SubmitPlanAction:
		return Pending, nil
	case s == Pending && a == Action("approve"):
		return Approved, nil
	case s == Pending && a == Action("reject"):
		return Rejected, nil
	}
	return "", ErrConflict
}
