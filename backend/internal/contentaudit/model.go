// Package contentaudit evaluates operational facts without opening a database or publishing content.
package contentaudit

import (
	"context"
	"errors"
	"github.com/yyl1212/math_master/backend/internal/catalogue"
	"github.com/yyl1212/math_master/backend/internal/content"
	"github.com/yyl1212/math_master/backend/internal/publication"
	"github.com/yyl1212/math_master/backend/internal/question"
	"time"
)

const (
	MaxMetadataBytes  = 4 << 20
	MaxSourceMapBytes = 256 << 10
	MaxReportBytes    = 8 << 20
)

var (
	ErrInvalid = errors.New("invalid content audit input")
	ErrLimit   = errors.New("content audit capacity exceeded")
)

type Mode string

const (
	Draft     Mode = "draft"
	Published Mode = "published"
)

type Conclusion string

const (
	NotReady       Conclusion = "not_ready"
	DraftReady     Conclusion = "draft_ready"
	AwaitingReview Conclusion = "awaiting_review"
	Accepted       Conclusion = "accepted"
)

type Request struct {
	Mode        Mode
	Route       content.VersionRef
	FixtureOnly bool
	CodeSHA     string
	At          time.Time
}
type IndexClaim struct {
	Index  string  `json:"index"`
	SHA256 *string `json:"sha256"`
}
type SelectedSource struct {
	Path      string   `json:"path"`
	SHA256    string   `json:"sha256"`
	DatasetID string   `json:"datasetId"`
	RecordIDs []string `json:"recordIds"`
}
type SourceIssue struct {
	Code           string       `json:"code"`
	Path           string       `json:"path"`
	IndexClaims    []IndexClaim `json:"indexClaims"`
	ActualSHA256   *string      `json:"actualSha256"`
	BlocksSelected bool         `json:"blocksSelected"`
}
type SourceReport struct {
	SchemaVersion       int              `json:"schemaVersion"`
	PolicyVersion       int              `json:"policyVersion"`
	SnapshotID          string           `json:"snapshotId"`
	SelectedFiles       []SelectedSource `json:"selectedFiles"`
	Issues              []SourceIssue    `json:"issues"`
	Ready               bool             `json:"ready"`
	PublicationApproved bool             `json:"publicationApproved"`
}
type MappedSource struct {
	ID             string           `json:"id"`
	Path           string           `json:"path"`
	RecordID       string           `json:"recordId"`
	DatasetID      string           `json:"datasetId"`
	FileSHA256     string           `json:"fileSha256"`
	PublicSources  []content.Source `json:"publicSources"`
	Use            string           `json:"use"`
	LegacyIDs      []string         `json:"legacyIds"`
	ReviewStatus   string           `json:"reviewStatus"`
	ConditionsNote string           `json:"conditionsNote"`
}
type SourceObject struct {
	Kind      string   `json:"kind"`
	ID        string   `json:"id"`
	Version   *int     `json:"version"`
	SHA256    string   `json:"sha256"`
	SourceIDs []string `json:"sourceIds"`
	Origin    string   `json:"origin"`
}
type SourceMap struct {
	SchemaVersion      int            `json:"schemaVersion"`
	PolicyVersion      int            `json:"policyVersion"`
	SnapshotID         string         `json:"snapshotId"`
	SourceReportSHA256 string         `json:"sourceReportSha256"`
	Sources            []MappedSource `json:"sources"`
	Objects            []SourceObject `json:"objects"`
}
type SourceBundle struct {
	SnapshotID, ReportSHA string
	Report                SourceReport
	Mapping               SourceMap
}
type DraftInput struct {
	Catalogue  catalogue.Catalogue
	Content    content.Package
	Questions  []question.QuestionPackage
	AssetsRoot string
}
type DraftFacts struct {
	Content         content.Snapshot
	Path            content.Path
	References      question.ReferenceSnapshot
	Sealed          []question.SealedPackage
	QuestionReports []question.ValidationReport
}
type Head struct {
	ID     string `json:"id"`
	SHA256 string `json:"sha256"`
}
type ObjectIdentity struct {
	Kind    string `json:"kind"`
	ID      string `json:"id"`
	Version *int   `json:"version"`
	SHA256  string `json:"sha256"`
}
type ApprovalFact struct {
	Space                         string
	Object                        ObjectIdentity
	Evidence                      publication.MemberEvidence
	AuthorIDs                     []string
	ReviewerID                    string
	ChecksComplete, FrozenMatches bool
}
type Exclusion struct {
	Object ObjectIdentity
	Code   string
}
type PublishedFacts struct {
	CatalogueVersion            int
	CatalogueSHA                string
	KnowledgeHead, QuestionHead *Head
	Path                        content.Path
	PathSHA                     string
	Content                     content.Snapshot
	Bank                        question.Candidate
	Approvals                   []ApprovalFact
	EligibleInstances           []question.Identity
	Excluded                    []Exclusion
	FixtureOnly                 bool
}
type ReviewAttestation struct {
	DecisionID           string `json:"decisionId"`
	IndependenceVerified bool   `json:"independenceVerified"`
}
type LearningCheck struct {
	Name           string `json:"name"`
	Result         string `json:"result"`
	EvidenceSHA256 string `json:"evidenceSha256"`
}
type AcceptanceEvidence struct {
	SchemaVersion      int                 `json:"schemaVersion"`
	CodeSHA            string              `json:"codeSHA"`
	RouteSHA           string              `json:"routeSHA"`
	KnowledgeHead      *Head               `json:"knowledgeHead"`
	QuestionHead       *Head               `json:"questionHead"`
	FixtureOnly        bool                `json:"fixtureOnly"`
	ReviewAttestations []ReviewAttestation `json:"reviewAttestations"`
	LearningChecks     []LearningCheck     `json:"learningChecks"`
}
type Counts struct {
	Knowledge          int `json:"knowledge"`
	Templates          int `json:"templates"`
	FixedInstances     int `json:"fixedInstances"`
	GeneratedInstances int `json:"generatedInstances"`
	EffectiveInstances int `json:"effectiveInstances"`
	Duplicates         int `json:"duplicates"`
	Excluded           int `json:"excluded"`
}
type Reason struct {
	Code string `json:"code"`
	Path string `json:"path"`
}
type NodeReport struct {
	Knowledge            question.Identity  `json:"knowledge"`
	Blueprint            *question.Identity `json:"blueprint"`
	EffectiveInstances   int                `json:"effectiveInstances"`
	AssessmentInstances  int                `json:"assessmentInstances"`
	Core                 []int              `json:"core"`
	FiveWitness          []string           `json:"fiveWitness"`
	AfterPracticeWitness bool               `json:"afterPracticeWitness"`
	Ready                bool               `json:"ready"`
	Reasons              []Reason           `json:"reasons"`
}
type ReportContext struct {
	CatalogueVersion int                `json:"catalogueVersion"`
	CatalogueSHA     string             `json:"catalogueSha"`
	Route            content.VersionRef `json:"route"`
	RouteSHA         string             `json:"routeSha"`
	KnowledgeHead    *Head              `json:"knowledgeHead"`
	QuestionHead     *Head              `json:"questionHead"`
}
type ReportSource struct {
	SnapshotID      string `json:"snapshotId"`
	PolicyVersion   int    `json:"policyVersion"`
	ReportSHA       string `json:"reportSha"`
	Complete        bool   `json:"complete"`
	UnresolvedCount int    `json:"unresolvedCount"`
}
type Quality struct {
	TwoAngles         bool `json:"twoAngles"`
	Examples          bool `json:"examples"`
	Assets            bool `json:"assets"`
	IndependentReview bool `json:"independentReview"`
	LearningComplete  bool `json:"learningComplete"`
}
type Report struct {
	SchemaVersion int           `json:"schemaVersion"`
	Mode          Mode          `json:"mode"`
	FixtureOnly   bool          `json:"fixtureOnly"`
	CreatedAt     string        `json:"createdAt"`
	CodeSHA       string        `json:"codeSHA"`
	Context       ReportContext `json:"context"`
	Source        ReportSource  `json:"source"`
	DraftCounts   Counts        `json:"draftCounts"`
	FormalCounts  Counts        `json:"formalCounts"`
	Nodes         []NodeReport  `json:"nodes"`
	Quality       Quality       `json:"quality"`
	Conclusion    Conclusion    `json:"conclusion"`
	Reasons       []Reason      `json:"reasons"`
}

func checkContext(ctx context.Context) error { return ctx.Err() }
