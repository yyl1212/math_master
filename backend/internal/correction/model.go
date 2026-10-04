// Package correction defines immutable correction evidence and deterministic regrading.
package correction

import (
	"errors"
	"github.com/yyl1212/math_master/backend/internal/assessment"
	"github.com/yyl1212/math_master/backend/internal/question"
	"time"
)

const (
	MaxSequence      int64 = 9007199254740991
	MaxVersion             = 2147483647
	MaxRequestBytes        = 65536
	MaxResponseBytes       = 2097152
	DefaultLimit           = 20
	MaxLimit               = 50
)

type CaseKind string
type EvidenceKind string
type PlanStatus string
type ResultStatus string
type DispositionReason string
type JobState string
type JobType string

const (
	WithdrawalCase        CaseKind          = "withdrawal"
	GradingRuleCase       CaseKind          = "grading_rule"
	LearningEventEvidence EvidenceKind      = "learning-event"
	PracticeEvidence      EvidenceKind      = "practice"
	AssessmentEvidence    EvidenceKind      = "assessment"
	EnrollmentEvidence    EvidenceKind      = "enrollment"
	Draft                 PlanStatus        = "draft"
	Pending               PlanStatus        = "pending"
	Approved              PlanStatus        = "approved"
	Rejected              PlanStatus        = "rejected"
	CorrectedPassed       ResultStatus      = "corrected_passed"
	CorrectedFailed       ResultStatus      = "corrected_failed"
	RetakeRequired        ResultStatus      = "retake_required"
	ReviewMaterial        ResultStatus      = "review_material"
	CheckedUnaffected     ResultStatus      = "checked_unaffected"
	AwaitingReview        ResultStatus      = "awaiting_review"
	AnswerCorrected       DispositionReason = "answer_corrected"
	RuleRegraded          DispositionReason = "rule_regraded"
	SourceInvalid         DispositionReason = "source_invalid"
	IntentChanged         DispositionReason = "intent_changed"
	CoverageChanged       DispositionReason = "coverage_changed"
	KnowledgeChanged      DispositionReason = "knowledge_changed"
	InsufficientItems     DispositionReason = "insufficient_items"
	NoApprovedBasis       DispositionReason = "no_approved_basis"
	ConflictingBasis      DispositionReason = "conflicting_basis"
	PathWithdrawn         DispositionReason = "path_withdrawn"
	Unaffected            DispositionReason = "unaffected"
	Queued                JobState          = "queued"
	Running               JobState          = "running"
	Succeeded             JobState          = "succeeded"
	RetryWait             JobState          = "retry_wait"
	JobFailed             JobState          = "failed"
	WithdrawalImpact      JobType           = "withdrawal_impact"
	RuleImpact            JobType           = "rule_impact"
	ApprovedPlan          JobType           = "approved_plan"
	AttemptTerminal       JobType           = "attempt_terminal"
)

var (
	ErrNotConfigured = errors.New("correction not configured")
	ErrConflict      = errors.New("correction conflict")
	ErrSourceStale   = errors.New("correction source stale")
	ErrAnswerOverlap = errors.New("correction answer overlap")
	ErrLeaseLost     = errors.New("correction lease lost")
)

type RateError struct{ RetryAt time.Time }

func (*RateError) Error() string { return "correction rate limit exceeded" }

type WithdrawalRef struct {
	Space string `json:"space"`
	ID    string `json:"id"`
}
type RuleScope struct {
	RuleVersion int                `json:"ruleVersion"`
	Kind        string             `json:"kind"`
	Knowledge   *question.Identity `json:"knowledge"`
}
type CaseInput struct {
	Kind       CaseKind       `json:"kind"`
	Withdrawal *WithdrawalRef `json:"withdrawal"`
	Rule       *RuleScope     `json:"rule"`
}
type CaseMetadata struct {
	ID              string         `json:"id"`
	Kind            CaseKind       `json:"kind"`
	Withdrawal      *WithdrawalRef `json:"withdrawal"`
	Rule            *RuleScope     `json:"rule"`
	Cutoff          *time.Time     `json:"cutoff"`
	Sequence        int64          `json:"sequence"`
	CreatedAt       time.Time      `json:"createdAt"`
	HasApprovedPlan bool           `json:"hasApprovedPlan"`
}
type EvidenceRef struct {
	Kind EvidenceKind `json:"kind"`
	ID   string       `json:"id"`
}
type Dependency struct {
	Kind    string `json:"kind"`
	ID      string `json:"id"`
	Version *int   `json:"version"`
	SHA256  string `json:"sha256"`
}
type PublishedInstance struct {
	Identity      question.Identity `json:"identity"`
	PublicationID string            `json:"publicationId"`
}
type Mapping struct {
	Original              question.Identity `json:"original"`
	OriginalPublicationID string            `json:"originalPublicationId"`
	Replacement           PublishedInstance `json:"replacement"`
}
type PlanRef struct {
	ID      string `json:"id"`
	Version int    `json:"version"`
}
type PlanInput struct {
	ExpectedSequence *int64    `json:"expectedSequence"`
	Parent           *PlanRef  `json:"parent"`
	AlgorithmVersion int       `json:"algorithmVersion"`
	Mappings         []Mapping `json:"mappings"`
	Reason           string    `json:"reason"`
}
type SubmitInput struct {
	ExpectedSequence int64 `json:"expectedSequence"`
}
type DecisionInput struct {
	ExpectedSequence int64  `json:"expectedSequence"`
	Decision         string `json:"decision"`
	Reason           string `json:"reason"`
}
type RetryInput struct {
	ExpectedSequence int64 `json:"expectedSequence"`
}
type PlanMetadata struct {
	Ref              PlanRef    `json:"ref"`
	CaseID           string     `json:"caseId"`
	Status           PlanStatus `json:"status"`
	Sequence         int64      `json:"sequence"`
	AlgorithmVersion int        `json:"algorithmVersion"`
	MappingCount     int        `json:"mappingCount"`
	CreatedAt        time.Time  `json:"createdAt"`
	UpdatedAt        time.Time  `json:"updatedAt"`
	Digest           *string    `json:"digest"`
}
type PlanDetail struct {
	Plan           PlanMetadata `json:"plan"`
	Parent         *PlanRef     `json:"parent"`
	Mappings       []Mapping    `json:"mappings"`
	Reason         *string      `json:"reason"`
	DecisionReason *string      `json:"decisionReason"`
}
type ResultMetadata struct {
	ID             string              `json:"id"`
	CaseID         string              `json:"caseId"`
	Plan           *PlanRef            `json:"plan"`
	ParentResultID *string             `json:"parentResultId"`
	Evidence       EvidenceRef         `json:"evidence"`
	Knowledge      *question.Identity  `json:"knowledge"`
	Status         ResultStatus        `json:"status"`
	Reason         DispositionReason   `json:"reason"`
	Score          *int                `json:"score"`
	Passed         *bool               `json:"passed"`
	Validity       assessment.Validity `json:"validity"`
	HandledCaseIDs []string            `json:"handledCaseIds"`
	CreatedAt      time.Time           `json:"createdAt"`
}
type CorrectedItem struct {
	Position          int                 `json:"position"`
	Original          question.Identity   `json:"original"`
	Effective         question.Identity   `json:"effective"`
	OriginalTemplate  *question.Identity  `json:"originalTemplate"`
	EffectiveTemplate *question.Identity  `json:"effectiveTemplate"`
	Prompt            string              `json:"prompt"`
	Choices           []question.Choice   `json:"choices"`
	Answer            assessment.Answer   `json:"answer"`
	Correct           bool                `json:"correct"`
	Explanation       string              `json:"explanation"`
	Assets            []question.AssetRef `json:"assets"`
}
type ResultDetail struct {
	Result     ResultMetadata  `json:"result"`
	Items      []CorrectedItem `json:"items"`
	PlanReason *string         `json:"planReason"`
}
type JobMetadata struct {
	ID             string     `json:"id"`
	CaseID         string     `json:"caseId"`
	Plan           *PlanRef   `json:"plan"`
	Type           JobType    `json:"type"`
	State          JobState   `json:"state"`
	Sequence       int64      `json:"sequence"`
	Epoch          int        `json:"epoch"`
	Attempt        int        `json:"attempt"`
	NextRunAt      *time.Time `json:"nextRunAt"`
	ProcessedCount int64      `json:"processedCount"`
	ErrorClass     *string    `json:"errorClass"`
}
type Receipt struct {
	Status int           `json:"status"`
	Case   *CaseMetadata `json:"case"`
	Plan   *PlanMetadata `json:"plan"`
	Job    *JobMetadata  `json:"job"`
}
type Query struct {
	Limit  int    `json:"limit"`
	Cursor string `json:"cursor"`
}
type Basis struct {
	OriginalSeal    assessment.Seal     `json:"originalSeal"`
	OriginalAnswers []assessment.Answer `json:"originalAnswers"`
	OriginalItems   []question.Instance `json:"originalItems"`
	EffectiveItems  []question.Instance `json:"effectiveItems"`
	HandledCaseIDs  []string            `json:"handledCaseIds"`
	ParentResultID  *string             `json:"parentResultId"`
	ParentResultIDs []string            `json:"parentResultIds"`
	PlanRefs        []PlanRef           `json:"planRefs"`
	AuditDeps       []Dependency        `json:"auditDeps"`
	EffectiveDeps   []Dependency        `json:"effectiveDeps"`
}
type Evaluation struct {
	Status  ResultStatus      `json:"status"`
	Reason  DispositionReason `json:"reason"`
	Score   *int              `json:"score"`
	Passed  *bool             `json:"passed"`
	Correct []bool            `json:"correct"`
}
type Lease struct {
	JobID   string    `json:"jobId"`
	Token   int64     `json:"token"`
	Until   time.Time `json:"until"`
	Epoch   int       `json:"epoch"`
	Attempt int       `json:"attempt"`
}
type Batch struct {
	Claimed   bool     `json:"claimed"`
	Processed int      `json:"processed"`
	Remaining bool     `json:"remaining"`
	State     JobState `json:"state"`
}
type ScanKey struct {
	Kind EvidenceKind `json:"kind"`
	ID   string       `json:"id"`
}
type PlanProof struct {
	CaseID              string            `json:"caseId"`
	AlgorithmVersion    int               `json:"algorithmVersion"`
	Parent              *PlanRef          `json:"parent"`
	Mappings            []ResolvedMapping `json:"mappings"`
	Authors             []string          `json:"authors"`
	ContentApprovalIDs  []string          `json:"contentApprovalIds"`
	QuestionApprovalIDs []string          `json:"questionApprovalIds"`
}
type ResolvedMapping struct {
	Original                 question.Instance       `json:"original"`
	Replacement              question.Instance       `json:"replacement"`
	OriginalPublicationID    string                  `json:"originalPublicationId"`
	ReplacementPublicationID string                  `json:"replacementPublicationId"`
	OriginalApproval         question.MemberEvidence `json:"originalApproval"`
	ReplacementApproval      question.MemberEvidence `json:"replacementApproval"`
}
type Page[T any] struct {
	Items      []T     `json:"items"`
	NextCursor *string `json:"nextCursor"`
}
type Envelope[T any] struct {
	ActorID string `json:"actorId"`
	Data    T      `json:"data"`
}
