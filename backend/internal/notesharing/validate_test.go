package notesharing

import (
	"github.com/yyl1212/math_master/backend/internal/study"
	"strings"
	"testing"
)

func TestSharingValidateBoundaries(t *testing.T) {
	for _, title := range []string{"笔记", strings.Repeat("中", 120)} {
		if e := ValidateSubmit(SubmitInput{ExpectedRevision: 1, Title: title}); e != nil {
			t.Fatal("valid title", e)
		}
	}
	for _, in := range []SubmitInput{{ExpectedRevision: 0, Title: "笔记"}, {ExpectedRevision: 1, Title: "  "}, {ExpectedRevision: 1, Title: strings.Repeat("中", 121)}, {ExpectedRevision: 1, ExpectedSequence: study.MaxSequence + 1, Title: "笔记"}} {
		if ValidateSubmit(in) == nil {
			t.Fatal("invalid submit accepted")
		}
	}
	for _, n := range []int{10, 1000} {
		if ValidateDecision("return", DecisionInput{Reason: strings.Repeat("中", n)}) != nil {
			t.Fatal("valid reason")
		}
	}
	for _, n := range []int{9, 1001} {
		if ValidateDecision("hide", DecisionInput{Reason: strings.Repeat("中", n)}) == nil {
			t.Fatal("invalid reason")
		}
	}
	if ValidateDecision("approve", DecisionInput{}) != nil || ValidateDecision("approve", DecisionInput{Reason: "unexpected"}) == nil {
		t.Fatal("approval input")
	}
	if ValidateWithdraw(WithdrawInput{SubmissionID: "not-an-id"}) == nil {
		t.Fatal("invalid ID")
	}
	if ValidateBody(strings.Repeat("😀", 16000)) != nil || ValidateBody(strings.Repeat("a", 16001)) == nil || ValidateBody("  \n") == nil || ValidateBody("<script>danger</script>") == nil {
		t.Fatal("body boundary")
	}
	var in SubmitInput
	if Decode(strings.NewReader(`{"expectedRevision":1,"expectedSequence":0,"title":"笔记","unknown":true}`), &in) == nil {
		t.Fatal("unknown field")
	}
	if Decode(strings.NewReader(`{"expectedRevision":1,"expectedRevision":2,"expectedSequence":0,"title":"笔记"}`), &in) == nil {
		t.Fatal("duplicate field")
	}
}
