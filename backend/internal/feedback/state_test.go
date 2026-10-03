package feedback

import "testing"

func TestFeedbackStateOwnerReply(t *testing.T) {
	for _, c := range []struct{ from, want Status }{{"new", "new"}, {"processing", "processing"}, {"waiting_details", "processing"}, {"resolved", "processing"}, {"closed", "processing"}} {
		got, e := OwnerReplyState(c.from)
		if e != nil || got != c.want {
			t.Fatal(c, got, e)
		}
	}
	if _, e := OwnerReplyState("unknown"); e == nil {
		t.Fatal("unknown state")
	}
}
func TestFeedbackStateTransitions(t *testing.T) {
	allowed := map[Status][]Status{"new": {"new", "processing", "waiting_details", "closed"}, "processing": {"processing", "waiting_details", "resolved", "closed"}, "waiting_details": {"waiting_details", "processing", "resolved", "closed"}, "resolved": {"resolved", "processing"}, "closed": {"closed", "processing"}}
	for _, from := range []Status{"new", "processing", "waiting_details", "resolved", "closed"} {
		for _, to := range []Status{"new", "processing", "waiting_details", "resolved", "closed"} {
			in := TransitionInput{ExpectedSequence: 1, Status: to, Message: "Reviewed this report."}
			if to != from && to == "resolved" {
				in.Resolution = &Resolution{Kind: "clarified"}
			}
			if to != from && to == "closed" {
				in.Resolution = &Resolution{Kind: "not_reproducible"}
			}
			want := false
			for _, v := range allowed[from] {
				want = want || v == to
			}
			if e := ValidateTransition(from, in); (e == nil) != want {
				t.Fatal(from, to, want, e)
			}
		}
	}
}
func TestFeedbackStateResolutionBranches(t *testing.T) {
	for _, c := range []struct {
		kind   ResolutionKind
		status Status
	}{{"clarified", "resolved"}, {"withdrawn", "resolved"}, {"revision_published", "resolved"}, {"service_fixed", "resolved"}, {"duplicate", "closed"}, {"not_reproducible", "closed"}, {"out_of_scope", "closed"}, {"suggestion_recorded", "closed"}} {
		r := Resolution{Kind: c.kind}
		if c.kind == "withdrawn" || c.kind == "revision_published" {
			r.Withdrawal = &WithdrawalRef{Space: "question", ID: testUUID}
		}
		if c.kind == "revision_published" {
			r.Replacement = &Replacement{Kind: "instance", Identity: testIdentity("qi-" + testIdentity("x").SHA256), PublicationID: otherUUID}
		}
		if c.kind == "duplicate" {
			r.DuplicateOf = testPtr(otherUUID)
		}
		in := TransitionInput{ExpectedSequence: 1, Status: c.status, Message: "Evidence checked.", Resolution: &r}
		if e := ValidateTransition("processing", in); e != nil {
			t.Fatal(c, e)
		}
		opposite := Status("closed")
		if c.status == "closed" {
			opposite = "resolved"
		}
		in.Status = opposite
		if ValidateTransition("processing", in) == nil {
			t.Fatal("resolution/status mismatch", c)
		}
	}
	for _, in := range []TransitionInput{
		{ExpectedSequence: 0, Status: "processing", Message: "Review"},
		{ExpectedSequence: 1, Status: "processing", Message: "  "},
		{ExpectedSequence: 1, Status: "processing", Message: "Review", Resolution: &Resolution{Kind: "clarified"}},
		{ExpectedSequence: 1, Status: "resolved", Message: "Review"},
		{ExpectedSequence: 1, Status: "closed", Message: "Review", Resolution: &Resolution{Kind: "duplicate", DuplicateOf: testPtr("invalid")}},
		{ExpectedSequence: 1, Status: "resolved", Message: "Review", Resolution: &Resolution{Kind: "clarified", Withdrawal: &WithdrawalRef{Space: "content", ID: testUUID}}},
	} {
		if ValidateTransition("processing", in) == nil {
			t.Fatal("invalid transition accepted", in)
		}
	}
}
