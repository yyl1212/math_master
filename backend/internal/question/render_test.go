package question

import (
	"context"
	"strings"
	"testing"
)

func TestControlledQuestionRender(t *testing.T) {
	for _, c := range []struct{ prompt, explanation string }{
		{"{{unknown}}", "{{answer}}"}, {"{{answer}}", "{{answer}}"}, {"<script>{{left}}</script>", "{{answer}}"},
		{"[Unsafe](https://example.com/{{left}})", "{{answer}}"}, {"`{{left}}`", "{{answer}}"}, {"{{left", "{{answer}}"},
		{"{{left}}", "<img src=x>"}, {"{{left}}", "{{shell}}"}, {"{{left}}", "\\input{secret}"},
	} {
		temp := templateFixture()
		temp.PromptTemplate = c.prompt
		temp.ExplanationTemplate = c.explanation
		if _, _, e := Generate(context.Background(), temp); e == nil {
			t.Fatal("unsafe template accepted")
		}
	}
	temp := templateFixture()
	instances, _, e := Generate(context.Background(), temp)
	if e != nil {
		t.Fatal(e)
	}
	if !strings.Contains(instances[0].Body.Prompt, "1/2") || strings.Contains(instances[0].Body.Prompt, "{{") || strings.Contains(instances[0].Body.Explanation, "{{") {
		t.Fatal("controlled interpolation missing")
	}
}
