package study_test

import (
	"github.com/yyl1212/math_master/backend/internal/study"
	"strings"
	"testing"
)

func TestStudyInputDigest(t *testing.T) {
	in := study.CommandInput{Knowledge: study.KnowledgeRef{ID: "fractions", Version: 1, SHA256: strings.Repeat("a", 64)}, ExpectedKnowledgeHead: "11111111-1111-4111-8111-111111111111", ExpectedSequence: 0}
	one, e := study.InputDigest(in)
	if e != nil || len(one) != 64 {
		t.Fatal(one, e)
	}
	two, e := study.InputDigest(in)
	if e != nil || one != two {
		t.Fatal("digest unstable")
	}
	in.ExpectedSequence++
	changed, e := study.InputDigest(in)
	if e != nil || changed == one {
		t.Fatal("input not bound")
	}
	in.ExpectedSequence = -1
	if _, e = study.InputDigest(in); e == nil {
		t.Fatal("invalid input hashed")
	}
	if _, e = study.InputDigest(map[string]string{"body": "untyped"}); e == nil {
		t.Fatal("unknown contract hashed")
	}
}
