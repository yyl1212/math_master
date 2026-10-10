// Package knowledgeadmin manages current knowledge without publication review versions.
package knowledgeadmin

import (
	"errors"
	"github.com/yyl1212/math_master/backend/internal/auth"
	"time"
)

const MaxSourceBytes = 64 * 1024 * 1024
const MaxSourcePoints = 100

var (
	ErrInvalid       = errors.New("invalid knowledge input")
	ErrConflict      = errors.New("knowledge content conflict")
	ErrStale         = errors.New("current knowledge changed")
	ErrIdempotency   = errors.New("knowledge idempotency conflict")
	ErrNotConfigured = errors.New("current knowledge not configured")
	ErrNotFound      = errors.New("knowledge not found")
	ErrBusy          = errors.New("knowledge upload busy")
	ErrRetired       = errors.New("knowledge workflow retired")
)

type Access struct {
	TokenHash                 auth.Digest
	CSRF                      auth.Secret
	IdempotencyKey, RequestID string
}
type Ref struct {
	ID            string `json:"id"`
	ContentSHA256 string `json:"contentSha256"`
	SourceKind    string `json:"sourceKind"`
}
type SourceProfile struct {
	SourceID     string   `json:"source_id"`
	WorkFamilyID string   `json:"work_family_id"`
	Title        string   `json:"title"`
	Authors      []string `json:"authors"`
	URL          string   `json:"url,omitempty"`
	License      string   `json:"license"`
	RightsStatus string   `json:"rights_status"`
	Attribution  string   `json:"attribution"`
}
type SourceDocument struct {
	Format          string         `json:"format"`
	SchemaVersion   string         `json:"schema_version"`
	DatasetID       string         `json:"dataset_id"`
	DatasetVersion  int            `json:"dataset_version"`
	Source          SourceProfile  `json:"source"`
	KnowledgePoints []SourcePoint  `json:"knowledge_points"`
	Extensions      map[string]any `json:"extensions"`
}
type OriginalBinding struct {
	RecordID                string `json:"record_id"`
	LocalRelativePath       string `json:"local_relative_path"`
	RecordJSONPointer       string `json:"record_json_pointer"`
	SourceFileSHA256        string `json:"source_file_sha256"`
	RecordBodySHA256        string `json:"record_body_sha256"`
	RecordBodyHashAlgorithm string `json:"record_body_hash_algorithm"`
}
type ClassificationEvidence struct {
	MSCCode        string   `json:"msc_code"`
	Kind           string   `json:"kind"`
	Reason         string   `json:"reason"`
	EvidenceFields []string `json:"evidence_fields"`
}
type ProjectOther struct {
	GroupID        string   `json:"group_id"`
	Reason         string   `json:"reason"`
	EvidenceFields []string `json:"evidence_fields"`
}
type Difficulty struct {
	Level          int      `json:"difficulty_level"`
	Rationale      string   `json:"rationale"`
	Prerequisites  []string `json:"prerequisites"`
	ReviewStatus   string   `json:"review_status"`
	RubricVersion  string   `json:"rubric_version"`
	ProofScopeNote string   `json:"proof_scope_note"`
}
type Explanation struct {
	Kind string `json:"kind"`
	Body string `json:"body"`
}
type Provenance struct {
	SourceID     string   `json:"source_id"`
	Title        string   `json:"title"`
	Authors      []string `json:"authors"`
	URL          string   `json:"url,omitempty"`
	License      string   `json:"license"`
	Chapter      string   `json:"chapter,omitempty"`
	Section      string   `json:"section,omitempty"`
	PDFPages     []int    `json:"pdf_pages"`
	PrintedPages []string `json:"printed_pages"`
	LocatorNote  string   `json:"locator_note"`
}
type Relation struct {
	Kind           string `json:"kind"`
	TargetSourceID string `json:"target_source_id"`
	TargetID       string `json:"target_id"`
	TargetVersion  int    `json:"target_version"`
	Status         string `json:"status"`
	Reason         string `json:"reason"`
}
type SourcePoint struct {
	ID                     string                   `json:"id"`
	Version                int                      `json:"version"`
	Title                  string                   `json:"title"`
	TitleZH                string                   `json:"title_zh"`
	Type                   string                   `json:"type"`
	TypeStatus             string                   `json:"type_status"`
	TypeReason             string                   `json:"type_reason"`
	TypeOtherReason        string                   `json:"type_other_reason,omitempty"`
	OriginalType           string                   `json:"original_type,omitempty"`
	Subtype                string                   `json:"subtype,omitempty"`
	ContentOrigin          string                   `json:"content_origin"`
	OriginalBinding        *OriginalBinding         `json:"original_binding,omitempty"`
	Statement              string                   `json:"statement"`
	Conditions             []string                 `json:"conditions"`
	Scope                  string                   `json:"scope"`
	System                 string                   `json:"system"`
	Objectives             []string                 `json:"objectives"`
	ClassificationMode     string                   `json:"classification_mode"`
	MSCCodes               []string                 `json:"msc_codes"`
	ClassificationStatus   string                   `json:"classification_status"`
	ClassificationEvidence []ClassificationEvidence `json:"classification_evidence"`
	ProjectOther           *ProjectOther            `json:"project_other,omitempty"`
	LearningDifficulty     Difficulty               `json:"learning_difficulty"`
	Proof                  string                   `json:"proof"`
	ProofScope             string                   `json:"proof_scope"`
	Equations              []string                 `json:"equations"`
	Explanations           []Explanation            `json:"explanations"`
	Examples               []string                 `json:"examples"`
	Counterexamples        []string                 `json:"counterexamples"`
	CommonMisconceptions   []string                 `json:"common_misconceptions"`
	Provenance             []Provenance             `json:"provenance"`
	Relations              []Relation               `json:"relations"`
	Tags                   []string                 `json:"tags"`
	Extensions             map[string]any           `json:"extensions"`
}
type PublicSource struct {
	SourceID string  `json:"sourceId"`
	Title    string  `json:"title"`
	Citation string  `json:"citation"`
	URL      *string `json:"url,omitempty"`
}
type CurrentInput struct {
	TopicKeys  []string       `json:"topicKeys,omitempty"`
	ExternalID string         `json:"externalId"`
	Point      SourcePoint    `json:"point"`
	Sources    []PublicSource `json:"sources"`
}
type Knowledge struct {
	TopicKeys  []string       `json:"topicKeys"`
	ID         string         `json:"id"`
	ExternalID string         `json:"externalId"`
	Point      SourcePoint    `json:"point"`
	Sources    []PublicSource `json:"sources"`
	Ref        Ref            `json:"ref"`
	Published  bool           `json:"published"`
	Deleted    bool           `json:"deleted"`
	EditToken  string         `json:"editToken"`
	UpdatedAt  time.Time      `json:"updatedAt"`
}
type PublicPoint map[string]any
type PublicKnowledge struct {
	TopicKeys  []string       `json:"topicKeys"`
	ID         string         `json:"id"`
	ExternalID string         `json:"externalId"`
	Point      PublicPoint    `json:"point"`
	Sources    []PublicSource `json:"sources"`
	Ref        Ref            `json:"ref"`
	UpdatedAt  time.Time      `json:"updatedAt"`
}
type Query struct {
	State                     string
	ReviewOnly                bool
	TopicKey, Q, Type, Status string
	Difficulty, Limit, Offset int
}
type Page[T any] struct {
	Items  []T `json:"items"`
	Total  int `json:"total"`
	Limit  int `json:"limit"`
	Offset int `json:"offset"`
}
type CurrentTopic struct {
	TopicKey       string                `json:"topicKey"`
	Title          string                `json:"title"`
	TitleEn        string                `json:"titleEn"`
	Kind           string                `json:"kind"`
	KnowledgeCount int                   `json:"knowledgeCount"`
	Items          Page[PublicKnowledge] `json:"items"`
}
type ContentMode struct {
	Mode       string `json:"mode"`
	Capability bool   `json:"capability"`
}
type PreviewItem struct {
	Index      int      `json:"index"`
	ExternalID string   `json:"externalId"`
	Action     string   `json:"action"`
	TopicKeys  []string `json:"topicKeys"`
	ErrorCode  string   `json:"errorCode,omitempty"`
	ErrorPath  string   `json:"errorPath,omitempty"`
}
type ImportCounts struct {
	CreatedKnowledge int `json:"createdKnowledge"`
	LinkedTopics     int `json:"linkedTopics"`
	SkippedItems     int `json:"skippedItems"`
	Conflicts        int `json:"conflicts"`
	InvalidItems     int `json:"invalidItems"`
}
type Preview struct {
	ImportID     string        `json:"importId"`
	InputSHA256  string        `json:"inputSha256"`
	PreviewToken string        `json:"previewToken"`
	Items        []PreviewItem `json:"items"`
	Counts       ImportCounts  `json:"counts"`
}
type ApplyInput struct {
	SelectedIndexes []int  `json:"selectedIndexes"`
	Publish         bool   `json:"publish"`
	PreviewToken    string `json:"previewToken"`
}
type ItemReceipt struct {
	Index       int      `json:"index"`
	ExternalID  string   `json:"externalId"`
	Action      string   `json:"action"`
	KnowledgeID string   `json:"knowledgeId"`
	TopicKeys   []string `json:"topicKeys"`
	ErrorCode   string   `json:"errorCode,omitempty"`
}
type Receipt struct {
	OperationID string        `json:"operationId"`
	Counts      ImportCounts  `json:"counts"`
	Items       []ItemReceipt `json:"items"`
}
type ImportStatus struct {
	Preview Preview  `json:"preview"`
	Receipt *Receipt `json:"receipt"`
}

type KnowledgeSummary struct {
	ID         string    `json:"id"`
	ExternalID string    `json:"externalId"`
	TopicKeys  []string  `json:"topicKeys"`
	Title      string    `json:"title"`
	TitleZH    string    `json:"titleZh"`
	Type       string    `json:"type"`
	Difficulty int       `json:"difficulty"`
	Ref        Ref       `json:"ref"`
	Published  bool      `json:"published"`
	Deleted    bool      `json:"deleted"`
	EditToken  string    `json:"editToken"`
	UpdatedAt  time.Time `json:"updatedAt"`
}

// CutoverInput is accepted only by the local, protected maintenance command.
type CutoverInput struct {
	ActorID         string    `json:"actorId"`
	OldIDs          []string  `json:"oldIds"`
	BackupRecord    string    `json:"backupRecord"`
	BackupCreatedAt time.Time `json:"backupCreatedAt"`
	RestoreVerified bool      `json:"restoreVerified"`
	OffsiteVerified bool      `json:"offsiteVerified"`
	CodeSHA         string    `json:"codeSha"`
}
type CutoverPlan struct {
	Input                CutoverInput   `json:"input"`
	Counts               map[string]int `json:"counts"`
	OldFingerprint       string         `json:"oldFingerprint"`
	ProtectedFingerprint string         `json:"protectedFingerprint"`
	PlanSHA              string         `json:"planSha"`
	PlannedAt            time.Time      `json:"plannedAt"`
}
type CutoverReceipt struct {
	OperationID            string         `json:"operationId"`
	PlanSHA                string         `json:"planSha"`
	RemovedPublicKnowledge int            `json:"removedPublicKnowledge"`
	Counts                 map[string]int `json:"counts"`
	ProtectedFingerprint   string         `json:"protectedFingerprint"`
	RecordedAt             time.Time      `json:"recordedAt"`
}
