package knowledgeadmin

import (
	"fmt"
	"testing"
)

func TestPublishInputExactBounds(t *testing.T) {
	valid := PublishCreateInput{Mode: PublishSelected}
	for n := 0; n < 1000; n++ {
		valid.Items = append(valid.Items, PublishSelection{ID: fmt.Sprintf("k-%056x", n), EditToken: "11111111-1111-4111-8111-111111111111"})
	}
	if e := valid.Validate(); e != nil {
		t.Fatal(e)
	}
	for _, v := range []PublishCreateInput{{Mode: PublishSelected}, {Mode: PublishAll, Items: valid.Items[:1]}, {Mode: PublishSelected, Items: append(valid.Items, valid.Items[0])}, {Mode: PublishSelected, Items: []PublishSelection{{ID: valid.Items[0].ID, EditToken: "bad"}}}} {
		if v.Validate() == nil {
			t.Fatal("accepted invalid create")
		}
	}
	input := PublishStepInput{ExpectedSequence: 1}
	if size, e := input.Normalize(); e != nil || size != 20 {
		t.Fatal(size, e)
	}
	for _, size := range []int{0, 21, -1} {
		input.BatchSize = &size
		if _, e := input.Normalize(); e == nil {
			t.Fatal(size)
		}
	}
	input = PublishStepInput{ExpectedSequence: 9007199254740992}
	if _, e := input.Normalize(); e == nil {
		t.Fatal("unsafe sequence")
	}
}
