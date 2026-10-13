package studyreview

import (
	"strings"
	"testing"
)

func TestAnswerUnicodeAndEmptyRecall(t *testing.T) {
	for _, s := range []string{"", " \n", strings.Repeat("字", 16000), strings.Repeat("😀", 16000), "证明：$x^2>0$"} {
		if e := ValidateAnswer(s); e != nil {
			t.Fatalf("legal %d: %v", len(s), e)
		}
	}
	for _, s := range []string{strings.Repeat("a", 16001), strings.Repeat("😀", 16384), strings.Repeat("a", 65537), string([]byte{255}), "<script>x</script>", "![x](https://example.com/a)", `$\href{data:x}{x}$`, "\x00"} {
		if e := ValidateAnswer(s); e == nil {
			t.Fatalf("unsafe/oversized accepted bytes %d", len(s))
		}
	}
	for _, in := range []CompareInput{{ExpectedSequence: 1, Body: " \n", Unanswered: true}, {ExpectedSequence: 1, Body: "$x$"}} {
		if e := ValidateCompare(Recall, in); e != nil {
			t.Fatal(e)
		}
	}
	for _, in := range []CompareInput{{ExpectedSequence: 1, Body: " \n"}, {ExpectedSequence: 1, Body: "x", Unanswered: true}, {ExpectedSequence: 0}, {ExpectedSequence: MaxSequence}} {
		if e := ValidateCompare(Recall, in); e == nil {
			t.Fatal("invalid compare accepted")
		}
	}
	if ValidateCompare(Read, CompareInput{ExpectedSequence: 1}) != nil || ValidateCompare(Read, CompareInput{ExpectedSequence: 1, Body: "x"}) == nil {
		t.Fatal("read answer contract")
	}
	if ValidateGrade(true, GradeInput{ExpectedSequence: 2, Rating: Forgotten}) != nil || ValidateGrade(true, GradeInput{ExpectedSequence: 2, Rating: Uncertain}) == nil || ValidateGrade(true, GradeInput{ExpectedSequence: 2, Rating: Familiar}) == nil {
		t.Fatal("unanswered grade")
	}
}
func TestCommandAndQueryValidation(t *testing.T) {
	if ValidateStart(StartInput{Knowledge: testRef("b"), Mode: Read, Target: Statement}) != nil {
		t.Fatal("valid start")
	}
	if ValidateStart(StartInput{Knowledge: testRef("b"), Mode: Read, Target: Proof}) == nil {
		t.Fatal("read proof target")
	}
	if ValidateQueue(QueueQuery{Limit: 20, Bucket: "due"}) != nil || ValidateQueue(QueueQuery{Limit: 51}) == nil || ValidateQueue(QueueQuery{Limit: 20, Q: strings.Repeat("字", 129)}) == nil || ValidateQueue(QueueQuery{Limit: 20, Offset: 100001}) == nil {
		t.Fatal("queue bounds")
	}
	if ValidateHistory(HistoryQuery{Limit: 20}) != nil || ValidateHistory(HistoryQuery{Limit: 50, Rating: Rating("correct")}) == nil {
		t.Fatal("history bounds")
	}
}
