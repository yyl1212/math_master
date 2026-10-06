package study_test

import (
	"github.com/yyl1212/math_master/backend/internal/study"
	"strings"
	"testing"
)

func TestStudyCommandBoundaries(t *testing.T) {
	good := study.CommandInput{Knowledge: study.KnowledgeRef{ID: "fractions", Version: 1, SHA256: strings.Repeat("a", 64)}, ExpectedKnowledgeHead: "11111111-1111-4111-8111-111111111111", ExpectedSequence: 0}
	if e := study.ValidateCommand(good); e != nil {
		t.Fatal(e)
	}
	for _, mutate := range []func(*study.CommandInput){func(v *study.CommandInput) { v.ExpectedSequence = -1 }, func(v *study.CommandInput) { v.ExpectedSequence = 9007199254740992 }, func(v *study.CommandInput) { v.Knowledge.Version = 0 }, func(v *study.CommandInput) { v.Knowledge.SHA256 = "bad" }, func(v *study.CommandInput) { v.Knowledge.ID = "../private" }, func(v *study.CommandInput) { v.ExpectedKnowledgeHead = "bad" }} {
		v := good
		mutate(&v)
		if study.ValidateCommand(v) == nil {
			t.Fatal("invalid command accepted", v)
		}
	}
}
func TestStudyNoteUnicodeBoundaries(t *testing.T) {
	for _, s := range []string{"", strings.Repeat("😀", 16000), "A safe formula $x^2$"} {
		if e := study.ValidateNote(s); e != nil {
			t.Fatal(e)
		}
	}
	for _, s := range []string{strings.Repeat("😀", 16001), strings.Repeat("a", 16001), "bad\x00text", string([]byte{0xff})} {
		if study.ValidateNote(s) == nil {
			t.Fatal("invalid note accepted", len(s))
		}
	}
}
