package correction

import "testing"

func TestCorrectionState(t *testing.T) {
	for _, c := range []struct {
		from   PlanStatus
		action Action
		want   PlanStatus
	}{{Draft, SubmitPlanAction, Pending}, {Pending, Action("approve"), Approved}, {Pending, Action("reject"), Rejected}, {Draft, UpdatePlanAction, Draft}} {
		got, e := NextPlanState(c.from, c.action)
		if e != nil || got != c.want {
			t.Fatalf("%s/%s = %s (%v)", c.from, c.action, got, e)
		}
	}
	for _, s := range []PlanStatus{Pending, Approved, Rejected} {
		if _, e := NextPlanState(s, UpdatePlanAction); e == nil {
			t.Fatal("submitted body editable", s)
		}
	}
	if _, e := NextPlanState(Draft, Action("approve")); e == nil {
		t.Fatal("unsubmitted plan approved")
	}
}
