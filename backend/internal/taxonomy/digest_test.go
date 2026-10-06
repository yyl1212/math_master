package taxonomy

import (
	"bytes"
	"github.com/yyl1212/math_master/backend/internal/content"
	"strings"
	"testing"
)

func assignmentFixture() AssignmentInput {
	return AssignmentInput{Knowledge: content.VersionRef{ID: "knowledge-a", Version: 1}, TopicIDs: []string{"msc-13c60", "msc-13c10"}, SourceRefs: []SourceRecordRef{{SourceID: "source-a", WorkFamilyID: "work-a", RecordID: "original.1", Path: "Algebra/source/records.json", SHA256: strings.Repeat("a", 64)}}, SourceBatchSHA: strings.Repeat("b", 64)}
}
func TestTaxonomyCanonicalDigest(t *testing.T) {
	in := assignmentFixture()
	raw, h, e := CanonicalAssignment(in)
	if e != nil {
		t.Fatal(e)
	}
	in.TopicIDs[0], in.TopicIDs[1] = in.TopicIDs[1], in.TopicIDs[0]
	raw2, h2, e := CanonicalAssignment(in)
	if e != nil || h != h2 || !bytes.Equal(raw, raw2) {
		t.Fatal("ordering changed identity", e)
	}
	in.SourceRefs[0].RecordID = "original.2"
	_, h3, e := CanonicalAssignment(in)
	if e != nil || h == h3 {
		t.Fatal("source change not bound", e)
	}
}
func TestTaxonomyAssignmentRejectsUnsafeAndAmbiguous(t *testing.T) {
	for _, mutate := range []func(*AssignmentInput){
		func(v *AssignmentInput) { v.TopicIDs = []string{"msc-13c99"} },
		func(v *AssignmentInput) { v.TopicIDs = []string{"msc-13"} },
		func(v *AssignmentInput) { v.TopicIDs = append(v.TopicIDs, v.TopicIDs[0]) },
		func(v *AssignmentInput) { v.SourceRefs[0].Path = "../secret.json" },
		func(v *AssignmentInput) { v.SourceBatchSHA = "bad" },
		func(v *AssignmentInput) { v.Knowledge.Version = 0 },
	} {
		v := assignmentFixture()
		mutate(&v)
		if _, _, e := CanonicalAssignment(v); e == nil {
			t.Fatal("invalid assignment accepted")
		}
	}
}
