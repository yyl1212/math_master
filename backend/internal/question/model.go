// Package question defines the independently versioned trusted question-bank contract.
package question

import (
	"errors"
	"github.com/yyl1212/math_master/backend/internal/content"
	"github.com/yyl1212/math_master/backend/internal/publication"
)

type Ref = content.VersionRef
type Access = publication.Access
type SourceLink = publication.SourceLink
type MemberEvidence = publication.MemberEvidence
type Issue = content.Issue
type Constraint string
type Distractor string

const (
	NonzeroDivisor    Constraint = "nonzero_divisor"
	NonnegativeResult Constraint = "nonnegative_result"
	DistinctOperands  Constraint = "distinct_operands"
	Negate            Distractor = "negate"
	PlusOne           Distractor = "plus_one"
	MinusOne          Distractor = "minus_one"
	Reciprocal        Distractor = "reciprocal"
)

type Identity struct {
	ID      string `json:"id"`
	Version int    `json:"version"`
	SHA256  string `json:"sha256"`
}
type KnownObject struct {
	Ref
	SHA256 string `json:"sha256"`
	Kind   string `json:"kind"`
}
type ObjectiveCoverage struct {
	Knowledge        Ref   `json:"knowledge"`
	ObjectiveIndices []int `json:"objectiveIndices"`
}
type Rational struct {
	Numerator   string `json:"numerator"`
	Denominator string `json:"denominator"`
}
type Parameter struct {
	Name   string   `json:"name"`
	Values []string `json:"values"`
}
type ParameterValue struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}
type EngineSpec struct {
	Family           string  `json:"family"`
	Operation        string  `json:"operation"`
	UnknownSide      *string `json:"unknownSide"`
	GeneratorVersion int     `json:"generatorVersion"`
	VerifierVersion  int     `json:"verifierVersion"`
}
type Choice struct {
	ID   string `json:"id"`
	Text string `json:"text"`
}
type AssetRef struct {
	ID     string `json:"id"`
	SHA256 string `json:"sha256"`
}
type VerificationWitness struct {
	Engine     EngineSpec       `json:"engine"`
	Parameters []ParameterValue `json:"parameters"`
}
type QuestionBody struct {
	Type            string               `json:"type"`
	Knowledge       Ref                  `json:"knowledge"`
	Coverage        []ObjectiveCoverage  `json:"coverage"`
	Units           []Ref                `json:"units"`
	Prompt          string               `json:"prompt"`
	Explanation     string               `json:"explanation"`
	AnswerFormat    *string              `json:"answerFormat"`
	Choices         []Choice             `json:"choices"`
	CorrectChoiceID *string              `json:"correctChoiceId"`
	CorrectNumeric  *Rational            `json:"correctNumeric"`
	Witness         *VerificationWitness `json:"witness"`
	Assets          []AssetRef           `json:"assets"`
	Sources         []content.Source     `json:"sources"`
}
type Template struct {
	ID                  string              `json:"id"`
	Version             int                 `json:"version"`
	Knowledge           Ref                 `json:"knowledge"`
	Coverage            []ObjectiveCoverage `json:"coverage"`
	Units               []Ref               `json:"units"`
	Type                string              `json:"type"`
	AnswerFormat        *string             `json:"answerFormat"`
	PromptTemplate      string              `json:"promptTemplate"`
	ExplanationTemplate string              `json:"explanationTemplate"`
	Engine              EngineSpec          `json:"engine"`
	Parameters          []Parameter         `json:"parameters"`
	Constraints         []Constraint        `json:"constraints"`
	Distractors         []Distractor        `json:"distractors"`
	Assets              []AssetRef          `json:"assets"`
	Sources             []content.Source    `json:"sources"`
}
type FixedQuestion struct {
	ID      string       `json:"id"`
	Version int          `json:"version"`
	Body    QuestionBody `json:"body"`
}
type BlueprintSource struct {
	Kind string `json:"kind"`
	Ref  Ref    `json:"ref"`
}
type Blueprint struct {
	ID                   string            `json:"id"`
	Version              int               `json:"version"`
	Knowledge            Ref               `json:"knowledge"`
	CoreObjectiveIndices []int             `json:"coreObjectiveIndices"`
	Sources              []BlueprintSource `json:"sources"`
	CoverageNote         string            `json:"coverageNote"`
	RuleVersion          int               `json:"ruleVersion"`
	QuestionCount        int               `json:"questionCount"`
	PassCount            int               `json:"passCount"`
}
type QuestionPackage struct {
	Kind           string          `json:"kind"`
	SchemaVersion  int             `json:"schemaVersion"`
	ID             string          `json:"id"`
	Version        int             `json:"version"`
	Templates      []Template      `json:"templates"`
	FixedQuestions []FixedQuestion `json:"fixedQuestions"`
	Blueprints     []Blueprint     `json:"blueprints"`
}
type Instance struct {
	Identity         Identity         `json:"identity"`
	Origin           string           `json:"origin"`
	Template         *Identity        `json:"template"`
	Parameters       []ParameterValue `json:"parameters"`
	GeneratorVersion *int             `json:"generatorVersion"`
	VerifierVersion  *int             `json:"verifierVersion"`
	Body             QuestionBody     `json:"body"`
}
type ResolvedObjective struct {
	Knowledge      Identity `json:"knowledge"`
	ObjectiveIndex int      `json:"objectiveIndex"`
	Text           string   `json:"text"`
}
type FixedKnowledge struct {
	Identity   Identity `json:"identity"`
	Title      string   `json:"title"`
	TitleZh    string   `json:"titleZh"`
	Objectives []string `json:"objectives"`
}
type FixedUnit struct {
	Identity  Identity `json:"identity"`
	Knowledge Ref      `json:"knowledge"`
	AssetIDs  []string `json:"assetIds"`
}
type ReferenceSnapshot struct {
	KnowledgeHead    *string             `json:"knowledgeHead"`
	CatalogueVersion int                 `json:"catalogueVersion"`
	CatalogueSHA256  string              `json:"catalogueSha256"`
	Knowledge        []FixedKnowledge    `json:"knowledge"`
	Units            []FixedUnit         `json:"units"`
	Assets           []content.AssetView `json:"assets"`
}
type ConstraintCount struct {
	Constraint Constraint `json:"constraint"`
	Count      int        `json:"count"`
}
type GenerationReport struct {
	Template             Identity          `json:"template"`
	RawCombinations      int               `json:"rawCombinations"`
	ExcludedCombinations int               `json:"excludedCombinations"`
	ValidInstances       int               `json:"validInstances"`
	GeneratorVersion     int               `json:"generatorVersion"`
	VerifierVersion      int               `json:"verifierVersion"`
	ConstraintCounts     []ConstraintCount `json:"constraintCounts"`
}
type CoverageNode struct {
	Knowledge                     Identity  `json:"knowledge"`
	Blueprint                     *Identity `json:"blueprint"`
	EffectiveInstances            int       `json:"effectiveInstances"`
	FixedInstances                int       `json:"fixedInstances"`
	GeneratedInstances            int       `json:"generatedInstances"`
	AssessmentInstances           int       `json:"assessmentInstances"`
	DuplicateInstances            int       `json:"duplicateInstances"`
	CoreObjectiveIndices          []int     `json:"coreObjectiveIndices"`
	CoveredObjectiveIndices       []int     `json:"coveredObjectiveIndices"`
	FiveQuestionFeasible          bool      `json:"fiveQuestionFeasible"`
	Ready                         bool      `json:"ready"`
	Reasons                       []Issue   `json:"reasons"`
	SupplementaryObjectiveIndices []int     `json:"supplementaryObjectiveIndices"`
}
type CoverageReport struct {
	KnowledgeHead      *string            `json:"knowledgeHead"`
	QuestionHead       *string            `json:"questionHead"`
	PublishedKnowledge int                `json:"publishedKnowledge"`
	ApprovedTemplates  int                `json:"approvedTemplates"`
	FixedQuestions     int                `json:"fixedQuestions"`
	EffectiveInstances int                `json:"effectiveInstances"`
	DuplicateInstances int                `json:"duplicateInstances"`
	Nodes              Page[CoverageNode] `json:"nodes"`
}
type ValidationReport struct {
	StructuralErrors        []Issue            `json:"structuralErrors"`
	CompletenessErrors      []Issue            `json:"completenessErrors"`
	HumanReviewRequirements []Issue            `json:"humanReviewRequirements"`
	StructuralTotal         int                `json:"structuralTotal"`
	CompletenessTotal       int                `json:"completenessTotal"`
	HumanReviewTotal        int                `json:"humanReviewTotal"`
	Truncated               bool               `json:"truncated"`
	ReadyToSubmit           bool               `json:"readyToSubmit"`
	Digest                  string             `json:"digest"`
	Generation              []GenerationReport `json:"generation"`
	Coverage                []CoverageNode     `json:"coverage"`
	PackageBytes            int                `json:"packageBytes"`
	FrozenBytes             int                `json:"frozenBytes"`
}
type SealedPackage struct {
	Package           QuestionPackage     `json:"package"`
	PackageSHA        string              `json:"packageSha"`
	Instances         []Instance          `json:"instances"`
	Generation        []GenerationReport  `json:"generation"`
	Resolved          []KnownObject       `json:"resolved"`
	Objectives        []ResolvedObjective `json:"objectives"`
	ReferenceSnapshot ReferenceSnapshot   `json:"referenceSnapshot"`
}
type DraftInput struct {
	CatalogueVersion int             `json:"catalogueVersion"`
	QuestionPackage  QuestionPackage `json:"questionPackage"`
	SourceMap        []SourceLink    `json:"sourceMap"`
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
type DraftView struct {
	ID                 string           `json:"id"`
	OwnerID            string           `json:"ownerId"`
	CatalogueSHA256    string           `json:"catalogueSha256"`
	Status             string           `json:"status"`
	CreatedAt          string           `json:"createdAt"`
	UpdatedAt          string           `json:"updatedAt"`
	CatalogueVersion   int              `json:"catalogueVersion"`
	Revision           int64            `json:"revision"`
	QuestionPackage    QuestionPackage  `json:"questionPackage"`
	SourceMap          []SourceLink     `json:"sourceMap"`
	AuthorIDs          []string         `json:"authorIds"`
	LegacyUnattributed bool             `json:"legacyUnattributed"`
	Gate               ValidationReport `json:"gate"`
}
type FrozenBody struct {
	CatalogueVersion   int                 `json:"catalogueVersion"`
	CatalogueSHA256    string              `json:"catalogueSha256"`
	QuestionPackage    QuestionPackage     `json:"questionPackage"`
	SourceMap          []SourceLink        `json:"sourceMap"`
	AuthorIDs          []string            `json:"authorIds"`
	LegacyUnattributed bool                `json:"legacyUnattributed"`
	Resolved           []KnownObject       `json:"resolved"`
	Objectives         []ResolvedObjective `json:"objectives"`
	Generation         []GenerationReport  `json:"generation"`
	InstanceIdentities []Identity          `json:"instanceIdentities"`
	Coverage           []CoverageNode      `json:"coverage"`
	GeneratorVersions  []int               `json:"generatorVersions"`
	VerifierVersions   []int               `json:"verifierVersions"`
	FrozenDigest       string              `json:"frozenDigest"`
}
type FrozenPayload struct {
	Body      FrozenBody `json:"body"`
	Instances []Instance `json:"instances"`
}
type SubmissionSummary struct {
	ID               string `json:"id"`
	WorkspaceID      string `json:"workspaceId"`
	OwnerID          string `json:"ownerId"`
	PackageID        string `json:"packageId"`
	Status           string `json:"status"`
	FrozenDigest     string `json:"frozenDigest"`
	CreatedAt        string `json:"createdAt"`
	Revision         int64  `json:"revision"`
	PackageVersion   int    `json:"packageVersion"`
	CatalogueVersion int    `json:"catalogueVersion"`
}
type SubmissionView struct {
	ID          string           `json:"id"`
	WorkspaceID string           `json:"workspaceId"`
	OwnerID     string           `json:"ownerId"`
	Status      string           `json:"status"`
	CreatedAt   string           `json:"createdAt"`
	Revision    int64            `json:"revision"`
	Frozen      FrozenBody       `json:"frozen"`
	Gate        ValidationReport `json:"gate"`
	Review      *ReviewDecision  `json:"review"`
}
type ReviewChecks struct {
	Mathematics   bool `json:"mathematics"`
	Explanations  bool `json:"explanations"`
	Objectives    bool `json:"objectives"`
	Sources       bool `json:"sources"`
	Illustrations bool `json:"illustrations"`
	Generation    bool `json:"generation"`
}
type ReviewInput struct {
	Decision         string       `json:"decision"`
	Checks           ReviewChecks `json:"checks"`
	IndependenceNote string       `json:"independenceNote"`
	GenerationNote   string       `json:"generationNote"`
	Note             string       `json:"note"`
}
type ReviewDecision struct {
	ID               string       `json:"id"`
	SubmissionID     string       `json:"submissionId"`
	ReviewerID       string       `json:"reviewerId"`
	FrozenDigest     string       `json:"frozenDigest"`
	CreatedAt        string       `json:"createdAt"`
	Decision         string       `json:"decision"`
	Checks           ReviewChecks `json:"checks"`
	IndependenceNote string       `json:"independenceNote"`
	GenerationNote   string       `json:"generationNote"`
	Note             string       `json:"note"`
}
type MemberIdentity struct {
	Kind           string `json:"kind"`
	ID             string `json:"id"`
	PackageID      string `json:"packageId"`
	SHA256         string `json:"sha256"`
	Version        int    `json:"version"`
	PackageVersion int    `json:"packageVersion"`
}
type ManifestMember struct {
	Identity MemberIdentity `json:"identity"`
	Evidence MemberEvidence `json:"evidence"`
}
type Manifest struct {
	CatalogueVersion  int              `json:"catalogueVersion"`
	CatalogueSHA256   string           `json:"catalogueSha256"`
	BaseKnowledgeHead *string          `json:"baseKnowledgeHead"`
	BaseQuestionHead  *string          `json:"baseQuestionHead"`
	Members           []ManifestMember `json:"members"`
	Resolved          []KnownObject    `json:"resolved"`
}
type Change struct {
	Kind   string          `json:"kind"`
	ID     string          `json:"id"`
	Reason string          `json:"reason"`
	Before *MemberIdentity `json:"before"`
	After  *MemberIdentity `json:"after"`
}
type DiffSummary struct {
	Added    int `json:"added"`
	Replaced int `json:"replaced"`
	Removed  int `json:"removed"`
}
type PublicationSummary struct {
	ID                string      `json:"id"`
	Status            string      `json:"status"`
	ManifestSHA       string      `json:"manifestSha"`
	CreatedAt         string      `json:"createdAt"`
	BaseKnowledgeHead *string     `json:"baseKnowledgeHead"`
	BaseQuestionHead  *string     `json:"baseQuestionHead"`
	CatalogueVersion  int         `json:"catalogueVersion"`
	CatalogueSHA256   string      `json:"catalogueSha256"`
	TemplateCount     int         `json:"templateCount"`
	InstanceCount     int         `json:"instanceCount"`
	BlueprintCount    int         `json:"blueprintCount"`
	Diff              DiffSummary `json:"diff"`
}
type PrepareInput struct {
	SubmissionIDs         []string `json:"submissionIds"`
	ExpectedKnowledgeHead *string  `json:"expectedKnowledgeHead"`
	ExpectedQuestionHead  *string  `json:"expectedQuestionHead"`
	Reason                string   `json:"reason"`
}
type ActivateInput struct {
	ExpectedKnowledgeHead *string `json:"expectedKnowledgeHead"`
	ExpectedQuestionHead  *string `json:"expectedQuestionHead"`
	ExpectedManifestSHA   string  `json:"expectedManifestSha"`
	Reason                string  `json:"reason"`
}
type WithdrawalTarget struct {
	Kind    string `json:"kind"`
	ID      string `json:"id"`
	Version int    `json:"version"`
}
type WithdrawalPreviewInput struct {
	Target WithdrawalTarget `json:"target"`
}
type WithdrawalInput struct {
	Target                WithdrawalTarget `json:"target"`
	ExpectedKnowledgeHead *string          `json:"expectedKnowledgeHead"`
	ExpectedQuestionHead  *string          `json:"expectedQuestionHead"`
	Reason                string           `json:"reason"`
}
type WithdrawalPreview struct {
	CurrentKnowledgeHead *string          `json:"currentKnowledgeHead"`
	CurrentQuestionHead  *string          `json:"currentQuestionHead"`
	Target               WithdrawalTarget `json:"target"`
	AffectedTemplates    int              `json:"affectedTemplates"`
	AffectedInstances    int              `json:"affectedInstances"`
	AffectedBlueprints   int              `json:"affectedBlueprints"`
	Diff                 DiffSummary      `json:"diff"`
	Changes              Page[Change]     `json:"changes"`
	ImpactDigest         string           `json:"impactDigest"`
}
type WithdrawalResult struct {
	EventID               string             `json:"eventId"`
	PreviousKnowledgeHead *string            `json:"previousKnowledgeHead"`
	PreviousQuestionHead  *string            `json:"previousQuestionHead"`
	Publication           PublicationSummary `json:"publication"`
}
type ListQuery struct {
	Scope  string `json:"scope"`
	Status string `json:"status"`
	Limit  int    `json:"limit"`
	Offset int    `json:"offset"`
}
type CoverageQuery struct {
	Limit       int    `json:"limit"`
	Offset      int    `json:"offset"`
	KnowledgeID string `json:"knowledgeId"`
}
type BaseManifest struct {
	Templates  []Template  `json:"templates"`
	Blueprints []Blueprint `json:"blueprints"`
	Head       *string     `json:"head"`
	Manifest   *Manifest   `json:"manifest"`
	Instances  []Instance  `json:"instances"`
}
type ApprovedSubmission struct {
	SubmissionID string         `json:"submissionId"`
	Frozen       FrozenBody     `json:"frozen"`
	Instances    []Instance     `json:"instances"`
	Decision     ReviewDecision `json:"decision"`
}
type Candidate struct {
	Templates  []Template  `json:"templates"`
	Blueprints []Blueprint `json:"blueprints"`
	Manifest   Manifest    `json:"manifest"`
	Diff       DiffSummary `json:"diff"`
	Changes    []Change    `json:"changes"`
	Instances  []Instance  `json:"instances"`
}
type ReplacementFact struct {
	From          Identity  `json:"from"`
	To            *Identity `json:"to"`
	PublicationID string    `json:"publicationId"`
	CreatedAt     string    `json:"createdAt"`
}
type WithdrawalFact struct {
	EventID   string           `json:"eventId"`
	Target    WithdrawalTarget `json:"target"`
	Reason    string           `json:"reason"`
	CreatedAt string           `json:"createdAt"`
}
type HistoricalFacts struct {
	Identity     Identity          `json:"identity"`
	Approval     *MemberEvidence   `json:"approval"`
	Replacements []ReplacementFact `json:"replacements"`
	Withdrawals  []WithdrawalFact  `json:"withdrawals"`
}
type CandidateCoverage struct {
	InstanceID       string `json:"instanceId"`
	ObjectiveIndices []int  `json:"objectiveIndices"`
}
type GradeResult struct {
	Correct bool `json:"correct"`
}
type SourceResponsibility struct {
	AuthorIDs          []string `json:"authorIds"`
	LegacyUnattributed bool     `json:"legacyUnattributed"`
}
type Archive struct {
	Envelope             DraftInput           `json:"envelope"`
	PackageSHA           string               `json:"packageSha"`
	Instances            []Instance           `json:"instances"`
	GeneratorVersions    []int                `json:"generatorVersions"`
	VerifierVersions     []int                `json:"verifierVersions"`
	SourceResponsibility SourceResponsibility `json:"sourceResponsibility"`
}
type QuestionImportResult struct {
	PackageID          string `json:"packageId"`
	PackageSHA         string `json:"packageSha"`
	Status             string `json:"status"`
	PackageVersion     int    `json:"packageVersion"`
	ImportedInstances  int    `json:"importedInstances"`
	DuplicateInstances int    `json:"duplicateInstances"`
}
type SaveDraftInput struct {
	DraftInput
	ExpectedRevision int64 `json:"expectedRevision"`
}
type Page[T any] struct {
	Items  []T `json:"items"`
	Total  int `json:"total"`
	Limit  int `json:"limit"`
	Offset int `json:"offset"`
}
type PublicationPage struct {
	Page[PublicationSummary]
	Head *string `json:"head"`
}
type MemberPage struct {
	Page[ManifestMember]
	PublicationID string `json:"publicationId"`
	ManifestSHA   string `json:"manifestSha"`
}
type ChangePage struct {
	Page[Change]
	PublicationID string `json:"publicationId"`
	ManifestSHA   string `json:"manifestSha"`
}
type NumericFormatError struct {
	Code string `json:"code"`
}

func (e *NumericFormatError) Error() string { return "invalid numeric format: " + e.Code }

var (
	ErrDraftConflict       = errors.New("question draft revision conflict")
	ErrPublicationStale    = errors.New("question publication heads changed")
	ErrReviewConflict      = errors.New("question review finalized")
	ErrIdempotencyConflict = errors.New("question idempotency conflict")
	ErrImmutableConflict   = errors.New("immutable question conflict")
	ErrVersionConflict     = errors.New("fixed question version conflict")
	ErrInvalid             = errors.New("invalid question")
	ErrNotReady            = errors.New("question not ready")
	ErrLimitExceeded       = errors.New("question capacity exceeded")
	ErrReviewRequired      = errors.New("independent question review required")
	ErrNotConfigured       = errors.New("question bank not configured")
)
