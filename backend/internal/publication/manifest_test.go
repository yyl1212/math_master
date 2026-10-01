package publication

import (
	"strings"
	"testing"
)

func TestManifestFrozenCanonical(t *testing.T) {
	batch := reviewedBatch(t, releasePackage(), "44444444-4444-4444-8444-444444444444")
	candidate, err := BuildCandidate(Candidate{}, []ReviewedBatch{batch})
	if err != nil {
		t.Fatal(err)
	}
	m := candidate.Manifest
	one, err := ManifestDigest(m)
	if err != nil {
		t.Fatal(err)
	}
	m.Members[0], m.Members[2] = m.Members[2], m.Members[0]
	two, err := ManifestDigest(m)
	if err != nil || one != two {
		t.Fatal("array order changed canonical manifest")
	}
	m.Members[0].Identity.SHA256 = strings.Repeat("d", 64)
	two, err = ManifestDigest(m)
	if err != nil || one == two {
		t.Fatal("member change did not change manifest SHA")
	}
}
