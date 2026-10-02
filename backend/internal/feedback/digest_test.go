package feedback

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"testing"
)

func TestFeedbackDigestPurposeAndCommandBinding(t *testing.T) {
	in := ReplyInput{ExpectedSequence: 1, Message: "Please check the explanation."}
	raw, h, e := CanonicalCommand("reply", testUUID, in)
	if e != nil {
		t.Fatal(e)
	}
	sum := sha256.Sum256(raw)
	if h != hex.EncodeToString(sum[:]) || !strings.Contains(string(raw), "feedback-command-v1") {
		t.Fatal("purpose/digest")
	}
	_, same, e := CanonicalCommand("reply", testUUID, in)
	if e != nil || same != h {
		t.Fatal("stable")
	}
	changed := in
	changed.Message += "\nMore details."
	seq := in
	seq.ExpectedSequence = 2
	for _, c := range []struct {
		a        Action
		resource string
		in       any
	}{{"reply", otherUUID, in}, {"reply", testUUID, changed}, {"reply", testUUID, seq}, {"transition", testUUID, TransitionInput{ExpectedSequence: 1, Status: "processing", Message: in.Message}}} {
		_, got, e := CanonicalCommand(c.a, c.resource, c.in)
		if e != nil || got == h {
			t.Fatal("command binding", e)
		}
	}
}
