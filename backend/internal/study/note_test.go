package study_test

import (
	"github.com/yyl1212/math_master/backend/internal/study"
	"strings"
	"testing"
)

func TestStudyNoteSafeMarkup(t *testing.T) {
	for _, body := range []string{"A **personal** derivation with $x^2$.", `\(\frac{1}{2}+\frac{1}{2}=1\)`, "A [safe reference](https://example.com)."} {
		if e := study.ValidateNote(body); e != nil {
			t.Fatal("safe note rejected", e)
		}
	}
	for _, body := range []string{"<script>alert(1)</script>", "<img src=x onerror=alert(1)>", "![remote](https://example.com/p.png)", `$\htmlClass{evil}{x}$`, `$\href{javascript:alert(1)}{x}$`, `$\def\a{\a}\a$`, "[bad](javascript:alert(1))"} {
		if study.ValidateNote(body) == nil {
			t.Fatal("unsafe note accepted")
		}
	}
}
func TestStudyNoteUnicodeLimit(t *testing.T) {
	for _, body := range []string{strings.Repeat("😀", 16000), strings.Repeat("中", 16000), strings.Repeat("a", 16000)} {
		if e := study.ValidateNote(body); e != nil {
			t.Fatal(e)
		}
	}
	for _, body := range []string{strings.Repeat("😀", 16001), strings.Repeat("a", 16001), strings.Repeat("😀", 16384), strings.Repeat("😀", 16384) + "a"} {
		if study.ValidateNote(body) == nil {
			t.Fatal("note boundary accepted", len(body))
		}
	}
}
