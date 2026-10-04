package correction

import (
	"github.com/yyl1212/math_master/backend/internal/question"
	"testing"
)

func TestCorrectionEquivalentGenerated(t *testing.T) {
	b := sampleBasis()
	o, r := b.OriginalItems[0], b.EffectiveItems[0]
	if ok, why, e := Equivalent(o, r); e != nil || !ok || why != AnswerCorrected {
		t.Fatalf("answer-only correction refused: %t %s %v", ok, why, e)
	}
	for name, change := range map[string]func(*question.Instance){"prompt": func(i *question.Instance) { i.Body.Prompt = "A different mathematical question." }, "format": func(i *question.Instance) { s := "integer"; i.Body.AnswerFormat = &s }, "parameters": func(i *question.Instance) { i.Parameters = []question.ParameterValue{{Name: "n", Value: "2"}} }, "assets": func(i *question.Instance) {
		i.Body.Assets = []question.AssetRef{{ID: "diagram", SHA256: o.Identity.SHA256}}
	}, "choice-order": func(i *question.Instance) {
		i.Body.Choices = []question.Choice{{ID: "b", Text: "2"}, {ID: "a", Text: "1"}}
	}} {
		t.Run(name, func(t *testing.T) {
			x := r
			change(&x)
			ok, _, e := Equivalent(o, x)
			if e != nil || ok {
				t.Fatalf("changed intent accepted: %t %v", ok, e)
			}
		})
	}
	o.Parameters = []question.ParameterValue{{Name: "n", Value: "1"}}
	r.Parameters = []question.ParameterValue{{Name: "n", Value: "1"}}
	o.Origin = "generated"
	r.Origin = "generated"
	o.Template = &question.Identity{ID: "family", Version: 1, SHA256: o.Identity.SHA256}
	r.Template = &question.Identity{ID: "family", Version: 2, SHA256: r.Identity.SHA256}
	if ok, _, e := Equivalent(o, r); e != nil || !ok {
		t.Fatal("actual generated parameters equivalent", e)
	}
}
