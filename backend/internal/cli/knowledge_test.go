package cli

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

func TestKnowledgeCLIExplicitPrivateInputs(t *testing.T) {
	for _, args := range [][]string{{}, {"activate"}, {"plan", "--database-url=secret"}, {"clean-old", "--input=missing"}, {"unknown"}} {
		var out, err bytes.Buffer
		if code := RunKnowledge(context.Background(), args, &out, &err); code != 2 || out.Len() != 0 || strings.Contains(err.String(), "secret") {
			t.Fatal(code, out.String(), err.String())
		}
	}
}
