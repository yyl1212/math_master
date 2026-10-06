package taxonomy

import (
	"bytes"
	"encoding/json"
	"github.com/santhosh-tekuri/jsonschema/v6"
	"github.com/yyl1212/math_master/schemas"
	"testing"
)

func TestTaxonomySchemaAssignmentBoundary(t *testing.T) {
	b, e := schemas.Files.ReadFile("topic-assignment.schema.json")
	if e != nil {
		t.Fatal(e)
	}
	doc, e := jsonschema.UnmarshalJSON(bytes.NewReader(b))
	if e != nil {
		t.Fatal(e)
	}
	c := jsonschema.NewCompiler()
	url := "https://math-master.local/schemas/topic-assignment.schema.json"
	if e = c.AddResource(url, doc); e != nil {
		t.Fatal(e)
	}
	contract, e := c.Compile(url)
	if e != nil {
		t.Fatal(e)
	}
	for _, mutate := range []func(*AssignmentInput){nil, func(v *AssignmentInput) { v.Knowledge.Version = 0 }, func(v *AssignmentInput) { v.SourceRefs[0].Path = "../outside.json" }, func(v *AssignmentInput) { v.TopicIDs = append(v.TopicIDs, v.TopicIDs[0]) }} {
		v := assignmentFixture()
		if mutate != nil {
			mutate(&v)
		}
		raw, e := json.Marshal(v)
		if e != nil {
			t.Fatal(e)
		}
		var decoded any
		if e = json.Unmarshal(raw, &decoded); e != nil {
			t.Fatal(e)
		}
		schemaErr := contract.Validate(decoded)
		goErr := ValidateAssignment(v)
		if (schemaErr == nil) != (goErr == nil) {
			t.Fatal("schema and Go differ", schemaErr, goErr)
		}
		if mutate == nil && schemaErr != nil {
			t.Fatal(schemaErr)
		}
	}
}
