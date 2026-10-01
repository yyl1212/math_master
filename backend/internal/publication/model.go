package publication

import (
	"errors"
	"github.com/yyl1212/math_master/backend/internal/auth"
	"github.com/yyl1212/math_master/backend/internal/content"
)

type Access struct {
	TokenHash      auth.Digest `json:"-"`
	CSRF           auth.Secret `json:"-"`
	IdempotencyKey string      `json:"-"`
	RequestID      string      `json:"-"`
}
type SourceLink struct {
	Knowledge    content.VersionRef `json:"knowledge"`
	BatchSHA256  string             `json:"batchSha256"`
	RelativePath string             `json:"relativePath"`
	SHA256       string             `json:"sha256"`
	LegacyID     string             `json:"legacyId"`
	Note         string             `json:"note"`
}
type AssetInput struct {
	ID     string `json:"id"`
	Base64 string `json:"base64"`
}
type DraftInput struct {
	CatalogueVersion int             `json:"catalogueVersion"`
	Package          content.Package `json:"package"`
	AssetBytes       []AssetInput    `json:"assetBytes"`
	SourceMap        []SourceLink    `json:"sourceMap"`
}
type SaveDraftInput struct {
	DraftInput
	ExpectedRevision int64 `json:"expectedRevision"`
}
type AdoptInput struct {
	PackageID      string `json:"packageId"`
	PackageVersion int    `json:"packageVersion"`
	Reason         string `json:"reason"`
}
type ValidateInput struct {
	ExpectedRevision int64 `json:"expectedRevision"`
}
type SubmitInput struct {
	ExpectedRevision int64  `json:"expectedRevision"`
	ExpectedDigest   string `json:"expectedDigest"`
}
type GateReport struct {
	StructuralErrors        []content.Issue `json:"structuralErrors"`
	CompletenessErrors      []content.Issue `json:"completenessErrors"`
	HumanReviewRequirements []content.Issue `json:"humanReviewRequirements"`
	StructuralTotal         int             `json:"structuralTotal"`
	CompletenessTotal       int             `json:"completenessTotal"`
	HumanReviewTotal        int             `json:"humanReviewTotal"`
	Truncated               bool            `json:"truncated"`
	ReadyToSubmit           bool            `json:"readyToSubmit"`
	Digest                  string          `json:"digest"`
}
type DraftView struct {
	ID                 string              `json:"id"`
	OwnerID            string              `json:"ownerId"`
	CatalogueSHA256    string              `json:"catalogueSha256"`
	CatalogueVersion   int                 `json:"catalogueVersion"`
	Revision           int64               `json:"revision"`
	Status             string              `json:"status"`
	Package            content.Package     `json:"package"`
	SourceMap          []SourceLink        `json:"sourceMap"`
	AuthorIDs          []string            `json:"authorIds"`
	LegacyUnattributed bool                `json:"legacyUnattributed"`
	Assets             []content.AssetView `json:"assets"`
	Gate               GateReport          `json:"gate"`
	CreatedAt          string              `json:"createdAt"`
	UpdatedAt          string              `json:"updatedAt"`
}
type FrozenBody struct {
	CatalogueVersion   int                 `json:"catalogueVersion"`
	CatalogueSHA256    string              `json:"catalogueSha256"`
	Package            content.Package     `json:"package"`
	SourceMap          []SourceLink        `json:"sourceMap"`
	AuthorIDs          []string            `json:"authorIds"`
	LegacyUnattributed bool                `json:"legacyUnattributed"`
	Assets             []content.AssetView `json:"assets"`
	FrozenDigest       string              `json:"frozenDigest"`
}
type ReviewChecks struct {
	Mathematics   bool `json:"mathematics"`
	Explanations  bool `json:"explanations"`
	Relationships bool `json:"relationships"`
	Sources       bool `json:"sources"`
	Illustrations bool `json:"illustrations"`
}
type ReviewInput struct {
	Decision         string       `json:"decision"`
	Checks           ReviewChecks `json:"checks"`
	IndependenceNote string       `json:"independenceNote"`
	Note             string       `json:"note"`
}
type ReviewDecision struct {
	ID               string       `json:"id"`
	SubmissionID     string       `json:"submissionId"`
	ReviewerID       string       `json:"reviewerId"`
	FrozenDigest     string       `json:"frozenDigest"`
	Decision         string       `json:"decision"`
	Checks           ReviewChecks `json:"checks"`
	IndependenceNote string       `json:"independenceNote"`
	Note             string       `json:"note"`
	CreatedAt        string       `json:"createdAt"`
}
type SubmissionView struct {
	ID          string          `json:"id"`
	WorkspaceID string          `json:"workspaceId"`
	OwnerID     string          `json:"ownerId"`
	Status      string          `json:"status"`
	Revision    int64           `json:"revision"`
	Frozen      FrozenBody      `json:"frozen"`
	Gate        GateReport      `json:"gate"`
	Review      *ReviewDecision `json:"review"`
	CreatedAt   string          `json:"createdAt"`
}
type MemberIdentity struct {
	Kind           string `json:"kind"`
	ID             string `json:"id"`
	PackageID      string `json:"packageId"`
	SHA256         string `json:"sha256"`
	Version        int    `json:"version"`
	PackageVersion int    `json:"packageVersion"`
}
type MemberEvidence struct {
	SubmissionID  string  `json:"submissionId"`
	DecisionID    string  `json:"decisionId"`
	FrozenDigest  string  `json:"frozenDigest"`
	InheritedFrom *string `json:"inheritedFrom"`
}
type ManifestMember struct {
	Identity MemberIdentity `json:"identity"`
	Evidence MemberEvidence `json:"evidence"`
}
type Manifest struct {
	CatalogueVersion int                    `json:"catalogueVersion"`
	CatalogueSHA256  string                 `json:"catalogueSha256"`
	BaseHead         *string                `json:"baseHead"`
	Members          []ManifestMember       `json:"members"`
	Bindings         []content.AssetBinding `json:"bindings"`
}
type Change struct {
	Kind   string          `json:"kind"`
	ID     string          `json:"id"`
	Reason string          `json:"reason"`
	Before *MemberIdentity `json:"before"`
	After  *MemberIdentity `json:"after"`
}
type Diff struct {
	Added    int      `json:"added"`
	Replaced int      `json:"replaced"`
	Removed  int      `json:"removed"`
	Changes  []Change `json:"changes"`
}
type PublicationView struct {
	ID          string   `json:"id"`
	Status      string   `json:"status"`
	ManifestSHA string   `json:"manifestSha"`
	CreatedAt   string   `json:"createdAt"`
	Manifest    Manifest `json:"manifest"`
	Diff        Diff     `json:"diff"`
}
type PrepareInput struct {
	SubmissionIDs []string `json:"submissionIds"`
	ExpectedHead  *string  `json:"expectedHead"`
	Reason        string   `json:"reason"`
}
type ActivateInput struct {
	ExpectedHead        *string `json:"expectedHead"`
	ExpectedManifestSHA string  `json:"expectedManifestSha"`
	Reason              string  `json:"reason"`
}
type WithdrawalTarget struct {
	Kind    string `json:"kind"`
	ID      string `json:"id,omitempty"`
	Version int    `json:"version,omitempty"`
	SHA256  string `json:"sha256,omitempty"`
}
type WithdrawalInput struct {
	Target       WithdrawalTarget `json:"target"`
	ExpectedHead *string          `json:"expectedHead"`
	Reason       string           `json:"reason"`
}
type WithdrawalPreviewInput struct {
	Target WithdrawalTarget `json:"target"`
}
type WithdrawalPreview struct {
	CurrentHead *string          `json:"currentHead"`
	Target      WithdrawalTarget `json:"target"`
	Diff        Diff             `json:"diff"`
}
type WithdrawalResult struct {
	EventID      string          `json:"eventId"`
	PreviousHead *string         `json:"previousHead"`
	Publication  PublicationView `json:"publication"`
}
type ListQuery struct {
	Scope  string `json:"scope"`
	Status string `json:"status"`
	Limit  int    `json:"limit"`
	Offset int    `json:"offset"`
}
type Page[T any] struct {
	Items  []T `json:"items"`
	Total  int `json:"total"`
	Limit  int `json:"limit"`
	Offset int `json:"offset"`
}
type PublicationPage struct {
	Page[PublicationView]
	Head *string `json:"head"`
}
type DraftSummary struct {
	ID                string `json:"id"`
	OwnerID           string `json:"ownerId"`
	PackageID         string `json:"packageId"`
	Status            string `json:"status"`
	CreatedAt         string `json:"createdAt"`
	UpdatedAt         string `json:"updatedAt"`
	PackageVersion    int    `json:"packageVersion"`
	CatalogueVersion  int    `json:"catalogueVersion"`
	StructuralTotal   int    `json:"structuralTotal"`
	CompletenessTotal int    `json:"completenessTotal"`
	Revision          int64  `json:"revision"`
}
type SubmissionSummary struct {
	ID               string `json:"id"`
	WorkspaceID      string `json:"workspaceId"`
	OwnerID          string `json:"ownerId"`
	PackageID        string `json:"packageId"`
	Status           string `json:"status"`
	FrozenDigest     string `json:"frozenDigest"`
	CreatedAt        string `json:"createdAt"`
	PackageVersion   int    `json:"packageVersion"`
	CatalogueVersion int    `json:"catalogueVersion"`
	Revision         int64  `json:"revision"`
}
type Candidate struct {
	PublicationID string           `json:"-"`
	Manifest      Manifest         `json:"manifest"`
	Diff          Diff             `json:"diff"`
	Snapshot      content.Snapshot `json:"snapshot"`
}

var (
	ErrDraftConflict        = errors.New("draft revision conflict")
	ErrReviewConflict       = errors.New("review already finalized")
	ErrImmutableConflict    = errors.New("immutable content conflict")
	ErrIdempotencyConflict  = errors.New("idempotency request conflict")
	ErrVersionConflict      = errors.New("fixed version conflict")
	ErrPublicationStale     = errors.New("publication head changed")
	ErrContentNotReady      = errors.New("content not ready")
	ErrContentInvalid       = errors.New("invalid content")
	ErrReviewRequired       = errors.New("independent review required")
	ErrContentLimitExceeded = errors.New("content capacity exceeded")
	ErrContentNotConfigured = errors.New("content workflow not configured")
)
