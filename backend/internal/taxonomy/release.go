package taxonomy

import (
	"encoding/json"
	"github.com/yyl1212/math_master/backend/internal/content"
	"reflect"
	"sort"
)

// 发布正文只存于后台，公开 ReleaseView 不包含来源或作者核查依据。
type ReleaseEvidence struct {
	SubmissionID      string `json:"submissionId"`
	DecisionID        string `json:"decisionId"`
	AssignmentDigest  string `json:"assignmentDigest"`
	TaxonomyVersionID string `json:"taxonomyVersionId"`
}
type ReleasedAssignment struct {
	Knowledge      KnowledgeRef      `json:"knowledge"`
	TopicIDs       []string          `json:"topicIds"`
	SourceRefs     []SourceRecordRef `json:"sourceRefs"`
	SourceBatchSHA string            `json:"sourceBatchSHA"`
	Evidence       ReleaseEvidence   `json:"evidence"`
	InheritedFrom  *string           `json:"inheritedFrom"`
}
type ReleaseDocument struct {
	View                 ReleaseView          `json:"view"`
	KnowledgeManifestSHA string               `json:"knowledgeManifestSHA"`
	Assignments          []ReleasedAssignment `json:"assignments"`
}

func SamePair(a, b PairRef) bool {
	return sameHead(a.KnowledgeHead, b.KnowledgeHead) && sameHead(a.TaxonomyHead, b.TaxonomyHead) && a.TaxonomyVersionID == b.TaxonomyVersionID
}
func sameHead(a, b *string) bool { return a == nil && b == nil || a != nil && b != nil && *a == *b }
func CanonicalReleasedAssignments(in []ReleasedAssignment) ([]ReleasedAssignment, string, error) {
	out := make([]ReleasedAssignment, 0, len(in))
	seen := map[string]bool{}
	for _, a := range in {
		if seen[a.Knowledge.ID] || !ValidSHA(a.Knowledge.SHA256) || !ValidSHA(a.Evidence.AssignmentDigest) || !ValidSHA(a.Evidence.TaxonomyVersionID) {
			return nil, "", ErrInvalid
		}
		seen[a.Knowledge.ID] = true
		b, _, e := CanonicalAssignment(AssignmentInput{Knowledge: content.VersionRef{ID: a.Knowledge.ID, Version: a.Knowledge.Version}, TopicIDs: a.TopicIDs, SourceRefs: a.SourceRefs, SourceBatchSHA: a.SourceBatchSHA})
		if e != nil {
			return nil, "", e
		}
		var v AssignmentInput
		if e = json.Unmarshal(b, &v); e != nil {
			return nil, "", e
		}
		a.TopicIDs = v.TopicIDs
		a.SourceRefs = v.SourceRefs
		out = append(out, a)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Knowledge.ID < out[j].Knowledge.ID })
	sha, e := Digest(out)
	return out, sha, e
}
func ReleaseDocumentDigest(d ReleaseDocument) (string, error) {
	d.View.Status = "draft"
	d.View.ManifestSHA = ""
	return Digest(d)
}
func AssignmentDiff(old, next []ReleasedAssignment) ReleaseDiff {
	d := ReleaseDiff{Added: []KnowledgeRef{}, Removed: []KnowledgeRef{}, ChangedTopicMemberships: []MembershipChange{}}
	before := map[string]ReleasedAssignment{}
	after := map[string]ReleasedAssignment{}
	for _, a := range old {
		before[a.Knowledge.ID] = a
	}
	for _, a := range next {
		after[a.Knowledge.ID] = a
	}
	for id, a := range before {
		b, ok := after[id]
		if !ok || a.Knowledge != b.Knowledge {
			d.Removed = append(d.Removed, a.Knowledge)
		}
	}
	for id, a := range after {
		b, ok := before[id]
		if !ok || a.Knowledge != b.Knowledge {
			d.Added = append(d.Added, a.Knowledge)
		} else if !reflect.DeepEqual(a.TopicIDs, b.TopicIDs) {
			d.ChangedTopicMemberships = append(d.ChangedTopicMemberships, MembershipChange{Knowledge: a.Knowledge, OldTopicIDs: b.TopicIDs, NewTopicIDs: a.TopicIDs})
		}
	}
	sort.Slice(d.Added, func(i, j int) bool { return d.Added[i].ID < d.Added[j].ID })
	sort.Slice(d.Removed, func(i, j int) bool { return d.Removed[i].ID < d.Removed[j].ID })
	sort.Slice(d.ChangedTopicMemberships, func(i, j int) bool {
		return d.ChangedTopicMemberships[i].Knowledge.ID < d.ChangedTopicMemberships[j].Knowledge.ID
	})
	return d
}
