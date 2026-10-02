// Package learning defines explicit learning facts and the unified personal-learning contract.
package learning

import (
	"errors"
	"github.com/yyl1212/math_master/backend/internal/assessment"
	"github.com/yyl1212/math_master/backend/internal/question"
	"time"
)

type State string
type ReadinessReason string

const (
	Unlearned                 State           = "unlearned"
	Learning                  State           = "learning"
	Learned                   State           = "learned"
	NeedsReview               State           = "needs-review"
	Mastered                  State           = "mastered"
	NoBlueprint               ReadinessReason = "no-blueprint"
	NoInstances               ReadinessReason = "no-instances"
	InsufficientCoverage      ReadinessReason = "insufficient-coverage"
	ExposureCooldown          ReadinessReason = "exposure-cooldown"
	RecentAssessmentExclusion ReadinessReason = "recent-assessment-exclusion"
)

var (
	ErrNotConfigured       = errors.New("learning not configured")
	ErrVersionStale        = errors.New("learning version stale")
	ErrPrerequisitesUnmet  = errors.New("learning prerequisites unmet")
	ErrAssessmentNotReady  = errors.New("assessment not ready")
	ErrAssessmentActive    = errors.New("an active attempt exists")
	ErrAssessmentExpired   = errors.New("assessment expired")
	ErrStateConflict       = errors.New("assessment state conflict")
	ErrAnswerFormatInvalid = errors.New("answer format invalid")
)

type QualificationView struct {
	Knowledge         question.Identity   `json:"knowledge"`
	Kind              string              `json:"kind"`
	EvidenceAttemptID string              `json:"evidenceAttemptId"`
	CompletedEventID  *string             `json:"completedEventId"`
	Validity          assessment.Validity `json:"validity"`
}
type PrerequisiteState struct {
	Knowledge question.Identity              `json:"knowledge"`
	Qualified bool                           `json:"qualified"`
	Reasons   []assessment.RestrictionReason `json:"reasons"`
}
type KnowledgeState struct {
	Knowledge       question.Identity   `json:"knowledge"`
	Title           string              `json:"title"`
	TitleZh         string              `json:"titleZh"`
	State           State               `json:"state"`
	CanEnter        bool                `json:"canEnter"`
	EverUnlocked    bool                `json:"everUnlocked"`
	CompletionValid bool                `json:"completionValid"`
	StartedAt       *time.Time          `json:"startedAt"`
	CompletedAt     *time.Time          `json:"completedAt"`
	Qualification   *QualificationView  `json:"qualification"`
	Prerequisites   []PrerequisiteState `json:"prerequisites"`
}
type BlueprintOption struct {
	Blueprint            question.Identity `json:"blueprint"`
	CoreObjectiveIndices []int             `json:"coreObjectiveIndices"`
	Ready                bool              `json:"ready"`
	Reasons              []ReadinessReason `json:"reasons"`
	RetryAt              *time.Time        `json:"retryAt"`
}
type KnowledgeDetail struct {
	KnowledgeHead    string                     `json:"knowledgeHead"`
	QuestionHead     *string                    `json:"questionHead"`
	State            KnowledgeState             `json:"state"`
	Objectives       []string                   `json:"objectives"`
	Blueprints       []BlueprintOption          `json:"blueprints"`
	ActiveAssessment *assessment.AttemptSummary `json:"activeAssessment"`
}
type StartInput struct {
	Knowledge             question.Identity `json:"knowledge"`
	ExpectedKnowledgeHead string            `json:"expectedKnowledgeHead"`
}
type CompleteInput = StartInput
type EnrollInput struct {
	Path                  question.Identity `json:"path"`
	ExpectedKnowledgeHead string            `json:"expectedKnowledgeHead"`
}
type PathSummary struct {
	ID                     string            `json:"id"`
	Path                   question.Identity `json:"path"`
	Title                  string            `json:"title"`
	TitleZh                string            `json:"titleZh"`
	KnowledgePublicationID string            `json:"knowledgePublicationId"`
	TotalNodes             int               `json:"totalNodes"`
	CompletedNodes         int               `json:"completedNodes"`
	PassedNodes            int               `json:"passedNodes"`
	UnlockedNodes          int               `json:"unlockedNodes"`
	NewVersionAvailable    bool              `json:"newVersionAvailable"`
	CreatedAt              time.Time         `json:"createdAt"`
}
type PathNode struct {
	Position  int                            `json:"position"`
	Title     string                         `json:"title"`
	TitleZh   string                         `json:"titleZh"`
	State     KnowledgeState                 `json:"state"`
	Available bool                           `json:"available"`
	Reasons   []assessment.RestrictionReason `json:"reasons"`
}
type PathView struct {
	Summary PathSummary `json:"summary"`
}
type PublishedPathSummary struct {
	Path       question.Identity `json:"path"`
	Title      string            `json:"title"`
	TitleZh    string            `json:"titleZh"`
	TotalNodes int               `json:"totalNodes"`
}
type HistoryEntry struct {
	ID         string              `json:"id"`
	Kind       string              `json:"kind"`
	Path       *question.Identity  `json:"path"`
	Knowledge  question.Identity   `json:"knowledge"`
	OccurredAt time.Time           `json:"occurredAt"`
	State      string              `json:"state"`
	Validity   assessment.Validity `json:"validity"`
	AttemptID  *string             `json:"attemptId"`
}
type Overview struct {
	KnowledgeHead           *string                    `json:"knowledgeHead"`
	QuestionHead            *string                    `json:"questionHead"`
	AvailablePaths          []PublishedPathSummary     `json:"availablePaths"`
	StartedCount            int                        `json:"startedCount"`
	CompletedCount          int                        `json:"completedCount"`
	EffectivePassedCount    int                        `json:"effectivePassedCount"`
	HistoricalUnlockedCount int                        `json:"historicalUnlockedCount"`
	ActivePractice          *assessment.AttemptSummary `json:"activePractice"`
	ActiveAssessment        *assessment.AttemptSummary `json:"activeAssessment"`
	Recent                  []HistoryEntry             `json:"recent"`
}
type ListQuery struct {
	Limit  int
	Offset int
}
type StateFacts struct{ Started, Completed, HadInvalidatedPass, EffectivePass, LatestReviewFailed bool }
type EvidenceFacts struct {
	Knowledge         question.Identity
	Current           bool
	CompletionEventID *string
	Passes            []assessment.AttemptFact
}
type EvidenceView struct {
	Qualified     bool
	Qualification *QualificationView
}
type EventSeal struct {
	ID                     string              `json:"id"`
	ActorID                string              `json:"actorId"`
	Knowledge              question.Identity   `json:"knowledge"`
	KnowledgePublicationID string              `json:"knowledgePublicationId"`
	Units                  []question.Identity `json:"units"`
	Assets                 []question.AssetRef `json:"assets"`
	Kind                   string              `json:"kind"`
	RecordedAt             time.Time           `json:"recordedAt"`
}
type EvidenceDependency struct {
	Kind    string
	ID      string
	Version *int
	SHA256  string
}
type Receipt struct {
	ResourceKind string `json:"resourceKind"`
	ResourceID   string `json:"resourceId"`
	Status       int    `json:"status"`
}
type ExposureRef struct {
	Kind     string
	Identity question.Identity
}

// ErrorCode is closed by the HTTP and OpenAPI boundaries; no raw storage error is public.
type ErrorCode string

const (
	CodeNotConfigured            ErrorCode = "LEARNING_NOT_CONFIGURED"
	CodeVersionStale             ErrorCode = "LEARNING_VERSION_STALE"
	CodePrerequisitesUnmet       ErrorCode = "LEARNING_PREREQUISITES_UNMET"
	CodeAssessmentNotReady       ErrorCode = "ASSESSMENT_NOT_READY"
	CodeAssessmentActive         ErrorCode = "ASSESSMENT_ACTIVE"
	CodeAssessmentExpired        ErrorCode = "ASSESSMENT_EXPIRED"
	CodeStateConflict            ErrorCode = "ASSESSMENT_STATE_CONFLICT"
	CodeAnswerFormatInvalid      ErrorCode = "ANSWER_FORMAT_INVALID"
	CodeInvalidRequest           ErrorCode = "INVALID_REQUEST"
	CodeInvalidCookie            ErrorCode = "INVALID_COOKIE"
	CodeInvalidCredentials       ErrorCode = "INVALID_CREDENTIALS"
	CodeAuthenticationRequired   ErrorCode = "AUTHENTICATION_REQUIRED"
	CodeCSRFFailed               ErrorCode = "CSRF_FAILED"
	CodeForbidden                ErrorCode = "FORBIDDEN"
	CodePasswordChangeRequired   ErrorCode = "PASSWORD_CHANGE_REQUIRED"
	CodeNotFound                 ErrorCode = "NOT_FOUND"
	CodeMethodNotAllowed         ErrorCode = "METHOD_NOT_ALLOWED"
	CodeIdempotencyConflict      ErrorCode = "IDEMPOTENCY_CONFLICT"
	CodePayloadTooLarge          ErrorCode = "PAYLOAD_TOO_LARGE"
	CodeRateLimited              ErrorCode = "RATE_LIMITED"
	CodeServiceUnavailable       ErrorCode = "SERVICE_UNAVAILABLE"
	CodeAuthNotConfigured        ErrorCode = "AUTH_NOT_CONFIGURED"
	CodeUsernameUnavailable      ErrorCode = "USERNAME_UNAVAILABLE"
	CodeAlreadyAuthenticated     ErrorCode = "ALREADY_AUTHENTICATED"
	CodeLastAdminRequired        ErrorCode = "LAST_ADMIN_REQUIRED"
	CodeReauthenticationRequired ErrorCode = "REAUTHENTICATION_REQUIRED"
)
