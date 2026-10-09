package knowledgeadmin

import (
	"strings"
	"testing"
)

func TestManagedStudyRefValidation(t *testing.T) {
	r := Ref{ID: KnowledgeID("raw"), ContentSHA256: strings.Repeat("a", 64), SourceKind: "managed"}
	if !ValidManagedRef(r) {
		t.Fatal("valid ref rejected")
	}
	r.SourceKind = "legacy"
	if ValidManagedRef(r) {
		t.Fatal("source kind forged")
	}
}
