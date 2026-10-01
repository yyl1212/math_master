package store

import (
	"encoding/json"
	"github.com/yyl1212/math_master/backend/internal/question"
	"strings"
	"testing"
)

func TestQuestionInstancePageBudget(t *testing.T) {
	page := question.Page[string]{Items: []string{strings.Repeat("x", question.MaxResponseBytes/2), strings.Repeat("y", question.MaxResponseBytes/2)}, Total: 2, Limit: 2, Offset: 0}
	if err := questionFitPage(&page); err != nil {
		t.Fatal(err)
	}
	raw, _ := json.Marshal(page)
	if len(raw) > question.MaxResponseBytes || len(page.Items) != 1 || page.Limit != 1 || page.Total != 2 {
		t.Fatal("actual page was not reduced")
	}
	page = question.Page[string]{Items: []string{strings.Repeat("x", question.MaxResponseBytes)}, Total: 1, Limit: 1}
	if err := questionFitPage(&page); err != question.ErrLimitExceeded {
		t.Fatal("oversized single item accepted", err)
	}
}

func TestQuestionSubmissionResponseBudget(t *testing.T) {
	out := question.SubmissionView{ID: "11111111-1111-4111-8111-111111111111", Frozen: question.FrozenBody{Objectives: []question.ResolvedObjective{{Text: strings.Repeat("x", question.MaxResponseBytes-5000)}}}}
	if err := questionSubmissionBudget(out); err != question.ErrLimitExceeded {
		t.Fatal("accepted a review response that would exceed 4MiB", err)
	}
	out.Frozen.Objectives[0].Text = "Small frozen objective."
	if err := questionSubmissionBudget(out); err != nil {
		t.Fatal(err)
	}
	if out.Review != nil || out.Status != "" {
		t.Fatal("budget check rewrote submission")
	}
}

func TestQuestionWrappedPageBudget(t *testing.T) {
	page := question.Page[string]{Items: []string{strings.Repeat("x", question.MaxResponseBytes/2-100), strings.Repeat("y", question.MaxResponseBytes/2-100)}, Total: 2, Limit: 2}
	wrapper := struct {
		question.Page[string]
		Metadata string `json:"metadata"`
	}{Page: page, Metadata: strings.Repeat("z", 500)}
	if err := questionFitPage(&wrapper.Page, func() any { return wrapper }); err != nil {
		t.Fatal(err)
	}
	raw, _ := json.Marshal(wrapper)
	if len(raw) > question.MaxResponseBytes || wrapper.Limit != 1 || len(wrapper.Items) != 1 {
		t.Fatal("ignored complete response envelope")
	}
}
