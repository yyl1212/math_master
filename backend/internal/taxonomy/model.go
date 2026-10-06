// Package taxonomy manages versioned classification independently of immutable mathematical bodies.
package taxonomy

import (
	"errors"
	"github.com/yyl1212/math_master/backend/internal/content"
)

type ExperienceMode string

const (
	ModeLegacy ExperienceMode = "legacy"
	ModeTopics ExperienceMode = "topics"
)

var (
	ErrInvalid             = errors.New("invalid taxonomy")
	ErrLimit               = errors.New("taxonomy resource limit")
	ErrNotConfigured       = errors.New("taxonomy not configured")
	ErrConflict            = errors.New("taxonomy conflict")
	ErrHeadStale           = errors.New("taxonomy head changed")
	ErrIdempotencyConflict = errors.New("taxonomy idempotency conflict")
)

type KnowledgeRef struct {
	ID      string `json:"id"`
	Version int    `json:"version"`
	SHA256  string `json:"sha256"`
}
type TopicNode struct {
	ID       string  `json:"id"`
	Code     string  `json:"code"`
	Name     string  `json:"name"`
	NameZh   string  `json:"nameZh"`
	Kind     string  `json:"kind"`
	Level    int     `json:"level"`
	ParentID *string `json:"parentId"`
}
type PairRef struct {
	KnowledgeHead     *string `json:"knowledgeHead"`
	TaxonomyHead      *string `json:"taxonomyHead"`
	TaxonomyVersionID string  `json:"taxonomyVersionId"`
}
type Query struct {
	Q, ParentID, Kind    string
	Level, Limit, Offset int
}
type TopicSummary struct {
	TopicNode
	Ancestors               []TopicNode `json:"ancestors"`
	PublishedKnowledgeCount int         `json:"publishedKnowledgeCount"`
	HasChildren             bool        `json:"hasChildren"`
}
type TopicDetail struct {
	Summary TopicSummary `json:"summary"`
	Pair    PairRef      `json:"pair"`
}
type KnowledgeSummary struct {
	KnowledgeRef
	Title    string   `json:"title"`
	TitleZh  string   `json:"titleZh"`
	TopicIDs []string `json:"topicIds"`
}
type Page[T any] struct {
	Items  []T     `json:"items"`
	Total  int     `json:"total"`
	Limit  int     `json:"limit"`
	Offset int     `json:"offset"`
	Pair   PairRef `json:"pair"`
}
type SourceRecordRef struct {
	SourceID     string `json:"sourceId"`
	WorkFamilyID string `json:"workFamilyId"`
	RecordID     string `json:"recordId"`
	Path         string `json:"path"`
	SHA256       string `json:"sha256"`
}
type AssignmentInput struct {
	Knowledge      content.VersionRef `json:"knowledge"`
	TopicIDs       []string           `json:"topicIds"`
	SourceRefs     []SourceRecordRef  `json:"sourceRefs"`
	SourceBatchSHA string             `json:"sourceBatchSHA"`
}
type SourceFile struct {
	Path      string `json:"path"`
	SizeBytes int64  `json:"sizeBytes"`
	SHA256    string `json:"sha256"`
}
type CaptureIssue struct {
	Code string `json:"code"`
	Path string `json:"path"`
}
type CaptureDiff struct {
	Added   []string `json:"added"`
	Changed []string `json:"changed"`
	Missing []string `json:"missing"`
}
type TopicCaptureManifest struct {
	SchemaVersion int            `json:"schemaVersion"`
	Batch         int            `json:"batch"`
	SourceFiles   []SourceFile   `json:"sourceFiles"`
	SnapshotID    string         `json:"snapshotId"`
	Diff          CaptureDiff    `json:"diff"`
	Issues        []CaptureIssue `json:"issues"`
	Accepted      bool           `json:"accepted"`
}
type CapturedBatch struct {
	Manifest             TopicCaptureManifest `json:"manifest"`
	Nodes                []TopicNode          `json:"nodes"`
	SourceRecordIndex    []SourceRecordRef    `json:"sourceRecordIndex"`
	RawClassificationSHA string               `json:"rawClassificationSHA"`
	Attribution          string               `json:"attribution"`
	License              string               `json:"license"`
}
type Version struct {
	ID          string `json:"id"`
	SnapshotID  string `json:"snapshotId"`
	TaxonomySHA string `json:"taxonomySHA"`
	Batch       int    `json:"batch"`
}
type DraftTopicInput struct {
	ExpectedDraftRevision      int64           `json:"expectedDraftRevision"`
	ExpectedAssignmentRevision int64           `json:"expectedAssignmentRevision"`
	TaxonomyVersionID          string          `json:"taxonomyVersionId"`
	Member                     AssignmentInput `json:"member"`
}
type DraftTopicView struct {
	DraftID            string            `json:"draftId"`
	DraftRevision      int64             `json:"draftRevision"`
	AssignmentRevision int64             `json:"assignmentRevision"`
	TaxonomyVersionID  string            `json:"taxonomyVersionId"`
	Members            []AssignmentInput `json:"members"`
	Digest             string            `json:"digest"`
	ReadyToSubmit      bool              `json:"readyToSubmit"`
}
type PrepareInput struct {
	SubmissionIDs []string `json:"submissionIds"`
	ExpectedPair  PairRef  `json:"expectedPair"`
	Reason        string   `json:"reason"`
}
type ActivateInput struct {
	ExpectedPair PairRef `json:"expectedPair"`
	ManifestSHA  string  `json:"manifestSHA"`
	Reason       string  `json:"reason"`
}
type MembershipChange struct {
	Knowledge   KnowledgeRef `json:"knowledge"`
	OldTopicIDs []string     `json:"oldTopicIds"`
	NewTopicIDs []string     `json:"newTopicIds"`
}
type ReleaseDiff struct {
	Added                   []KnowledgeRef     `json:"added"`
	Removed                 []KnowledgeRef     `json:"removed"`
	ChangedTopicMemberships []MembershipChange `json:"changedTopicMemberships"`
}
type ReleaseView struct {
	ID                     string      `json:"id"`
	Status                 string      `json:"status"`
	Pair                   PairRef     `json:"pair"`
	ManifestSHA            string      `json:"manifestSHA"`
	AssignmentsSHA         string      `json:"assignmentsSHA"`
	KnowledgePublicationID *string     `json:"knowledgePublicationId"`
	Diff                   ReleaseDiff `json:"diff"`
	CreatedAt              string      `json:"createdAt"`
}
type KnowledgeSet map[string]KnowledgeRef
