package assessment

import (
	"encoding/json"
	"github.com/yyl1212/math_master/backend/internal/question"
	"strings"
	"testing"
)

func TestLearningProjectionWhitelistAndOwnedSlices(t *testing.T) {
	i := fiveChoiceItems()[0]
	i.Template = &question.Identity{ID: "secret-template", Version: 1, SHA256: strings.Repeat("b", 64)}
	i.Parameters = []question.ParameterValue{{Name: "left", Value: "private-value"}}
	i.Body.Witness = &question.VerificationWitness{}
	out, err := ProjectQuestion(1, i)
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := json.Marshal(out)
	var doc map[string]json.RawMessage
	json.Unmarshal(raw, &doc)
	for k := range doc {
		if !map[string]bool{"position": true, "instance": true, "knowledge": true, "type": true, "prompt": true, "choices": true, "answerFormat": true, "assets": true}[k] {
			t.Fatal("private field", k)
		}
	}
	for _, private := range []string{"correctChoiceId", "correctNumeric", "explanation", "witness", "parameters", "secret-template", "private-value", "authorIds", "sourceMap", "body_bytes"} {
		if strings.Contains(string(raw), private) {
			t.Fatal("private value", string(raw))
		}
	}
	if out.Type != "choice" || out.Position != 1 || out.Knowledge.ID != "fractions" || out.Assets == nil || out.Choices == nil {
		t.Fatal(out)
	}
	out.Choices[0].Text = "changed"
	if i.Body.Choices[0].Text != "one" {
		t.Fatal("projection aliased source")
	}
	if _, e := ProjectQuestion(0, i); e == nil {
		t.Fatal("invalid position")
	}
	i.Body.Type = "unknown"
	if _, e := ProjectQuestion(1, i); e == nil {
		t.Fatal("unknown type")
	}
}
