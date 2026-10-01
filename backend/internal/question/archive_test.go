package question

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"
)

func TestQuestionArchiveTrustBoundaries(t *testing.T) {
	input, refs := sealFixture()
	sealed, _, err := ValidateAndSeal(context.Background(), input, refs)
	if err != nil {
		t.Fatal(err)
	}
	input.QuestionPackage = sealed.Package
	g, v := UsedEngineVersions(sealed.Package, sealed.Instances)
	a := Archive{Envelope: input, PackageSHA: sealed.PackageSHA, Instances: sealed.Instances, GeneratorVersions: g, VerifierVersions: v, SourceResponsibility: SourceResponsibility{AuthorIDs: []string{}, LegacyUnattributed: true}}
	raw, _ := json.Marshal(a)
	if _, err = DecodeArchive(bytes.NewReader(raw)); err != nil {
		t.Fatal(err)
	}
	for _, bad := range [][]byte{bytes.Replace(raw, []byte(`"envelope":`), []byte(`"Envelope":`), 1), bytes.Replace(raw, []byte(`"authorIds":[]`), []byte(`"authorIds":null`), 1), bytes.Replace(raw, []byte(`"envelope":`), []byte(`"approved":true,"envelope":`), 1)} {
		if _, err = DecodeArchive(bytes.NewReader(bad)); err == nil {
			t.Fatal("untrusted archive shape accepted")
		}
	}
	a.Instances[0].Body.Prompt = "Changed prompt with same claimed digest"
	if _, _, err = ValidateArchive(context.Background(), a, refs); err == nil {
		t.Fatal("claimed instance bytes trusted")
	}
	g, v = UsedEngineVersions(QuestionPackage{}, []Instance{})
	if len(g) != 0 || len(v) != 0 {
		t.Fatal("unused engines recorded")
	}
	witness := VerificationWitness{Engine: EngineSpec{GeneratorVersion: 1, VerifierVersion: 1}}
	g, v = UsedEngineVersions(QuestionPackage{}, []Instance{{Origin: "fixed", Body: QuestionBody{Witness: &witness}}})
	if len(g) != 0 || len(v) != 1 || v[0] != 1 {
		t.Fatal("fixed witness assigned a generator")
	}
}
