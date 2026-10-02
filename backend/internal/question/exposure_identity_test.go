package question

import (
	"context"
	"encoding/json"
	"testing"
)

func TestCanonicalTemplateExposureMatchesGeneratedIdentity(t *testing.T) {
	// Use original package fixtures and change rational spelling without changing mathematics.
	t1 := templateFixture()
	items, _, e := Generate(context.Background(), t1)
	if e != nil || len(items) == 0 {
		t.Fatal(e)
	}
	raw, sha, e := CanonicalTemplate(t1)
	if e != nil || sha != items[0].Template.SHA256 {
		t.Fatal("raw draft did not match approved template", sha, e)
	}
	var env struct {
		Body Template `json:"body"`
	}
	if json.Unmarshal(raw, &env) != nil {
		t.Fatal("canonical envelope")
	}
	_, again, e := CanonicalTemplate(env.Body)
	if e != nil || again != sha {
		t.Fatal("normalization unstable", again, e)
	}
}
