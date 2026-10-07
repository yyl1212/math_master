package buildmeta

import (
	"strings"
	"testing"
)

func TestTopicCutoverBuildRevisionIsEmbeddedAndStrict(t *testing.T) {
	before := Revision
	defer func() { Revision = before }()
	Revision = strings.Repeat("a", 40)
	if CodeSHA() != Revision {
		t.Fatal("release revision unavailable")
	}
	Revision = "not-a-code-sha"
	if CodeSHA() == Revision {
		t.Fatal("invalid metadata trusted")
	}
}
