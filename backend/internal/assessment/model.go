// Package assessment owns safe question presentation and deterministic assessment rules.
// It may import question, but never learning or store.
package assessment

import (
	"github.com/yyl1212/math_master/backend/internal/question"
	"time"
)

type Mode string
type AttemptState string
type Outcome string
type PracticeState string
type Validity string
type RestrictionReason string

const (
	ModeNode             Mode              = "node"
	ModeDiagnostic       Mode              = "diagnostic"
	ModeReview           Mode              = "review"
	Active               AttemptState      = "active"
	Submitted            AttemptState      = "submitted"
	Abandoned            AttemptState      = "abandoned"
	Expired              AttemptState      = "expired"
	Passed               Outcome           = "passed"
	Failed               Outcome           = "failed"
	Affected             Outcome           = "affected"
	PracticeActive       PracticeState     = "active"
	PracticeAnswered     PracticeState     = "answered"
	PracticeRevealed     PracticeState     = "revealed"
	PracticeAbandoned    PracticeState     = "abandoned"
	PracticeExpired      PracticeState     = "expired"
	Effective            Validity          = "effective"
	Restricted           Validity          = "restricted"
	Stale                Validity          = "stale"
	KnowledgeUpdated     RestrictionReason = "knowledge-updated"
	KnowledgeWithdrawn   RestrictionReason = "knowledge-withdrawn"
	UnitWithdrawn        RestrictionReason = "unit-withdrawn"
	AssetWithdrawn       RestrictionReason = "asset-withdrawn"
	TemplateWithdrawn    RestrictionReason = "template-withdrawn"
	InstanceWithdrawn    RestrictionReason = "instance-withdrawn"
	BlueprintWithdrawn   RestrictionReason = "blueprint-withdrawn"
	ExposedAfterCreation RestrictionReason = "exposed-after-creation"
	GradingIssue         RestrictionReason = "grading-issue"
)

type SafeQuestion struct {
	Position     int                 `json:"position"`
	Instance     question.Identity   `json:"instance"`
	Knowledge    question.Identity   `json:"knowledge"`
	Type         string              `json:"type"`
	Prompt       string              `json:"prompt"`
	Choices      []question.Choice   `json:"choices"`
	AnswerFormat *string             `json:"answerFormat"`
	Assets       []question.AssetRef `json:"assets"`
}
type Answer struct {
	Kind     string  `json:"kind"`
	ChoiceID *string `json:"choiceId,omitempty"`
	Raw      *string `json:"raw,omitempty"`
}
type PracticeAnswerInput = Answer
type PositionAnswer struct {
	Position int               `json:"position"`
	Instance question.Identity `json:"instance"`
	Answer   Answer            `json:"answer"`
}
type SubmitInput struct {
	Answers []PositionAnswer `json:"answers"`
}
type PracticeCreateInput struct {
	Knowledge             question.Identity `json:"knowledge"`
	ExpectedKnowledgeHead string            `json:"expectedKnowledgeHead"`
	ExpectedQuestionHead  string            `json:"expectedQuestionHead"`
}
type CreateInput struct {
	Knowledge             question.Identity `json:"knowledge"`
	Blueprint             question.Identity `json:"blueprint"`
	Mode                  Mode              `json:"mode"`
	ExpectedKnowledgeHead string            `json:"expectedKnowledgeHead"`
	ExpectedQuestionHead  string            `json:"expectedQuestionHead"`
}
type AttemptSummary struct {
	ID          string            `json:"id"`
	Kind        string            `json:"kind"`
	Knowledge   question.Identity `json:"knowledge"`
	Mode        *Mode             `json:"mode"`
	State       string            `json:"state"`
	CreatedAt   time.Time         `json:"createdAt"`
	ExpiresAt   time.Time         `json:"expiresAt"`
	SubmittedAt *time.Time        `json:"submittedAt"`
}
type AttemptView struct {
	Summary   AttemptSummary `json:"summary"`
	Questions []SafeQuestion `json:"questions"`
}
type PracticeView struct {
	Summary  AttemptSummary  `json:"summary"`
	Question SafeQuestion    `json:"question"`
	Result   *PracticeResult `json:"result"`
}
type ResultItem struct {
	Question        SafeQuestion        `json:"question"`
	Answer          *Answer             `json:"answer"`
	Correct         *bool               `json:"correct"`
	CorrectChoiceID *string             `json:"correctChoiceId"`
	CorrectNumeric  *question.Rational  `json:"correctNumeric"`
	Explanation     *string             `json:"explanation"`
	Validity        Validity            `json:"validity"`
	Reasons         []RestrictionReason `json:"reasons"`
}
type PracticeResult struct {
	Outcome PracticeState `json:"outcome"`
	Item    ResultItem    `json:"item"`
}
type ProgressUpdate struct {
	Knowledge            question.Identity   `json:"knowledge"`
	QualificationGranted bool                `json:"qualificationGranted"`
	NewlyUnlocked        []question.Identity `json:"newlyUnlocked"`
}
type ResultView struct {
	Summary     AttemptSummary      `json:"summary"`
	RuleVersion int                 `json:"ruleVersion"`
	Score       *int                `json:"score"`
	Passed      *bool               `json:"passed"`
	Outcome     Outcome             `json:"outcome"`
	Validity    Validity            `json:"validity"`
	Reasons     []RestrictionReason `json:"reasons"`
	Items       []ResultItem        `json:"items"`
	Progress    ProgressUpdate      `json:"progress"`
}

// Facts and source/seal types below are internal and never sent through HTTP.
type AttemptFact struct {
	ID          string
	Knowledge   question.Identity
	Mode        Mode
	Outcome     Outcome
	Score       *int
	Passed      *bool
	SubmittedAt time.Time
	Validity    Validity
}
type Candidate struct {
	Identity        question.Identity
	Template        *question.Identity
	Coverage        []int
	Seen            bool
	LastSeenAt      *time.Time
	ExposedAt       *time.Time
	RecentSubmitted bool
}
type SourcePool struct {
	Knowledge         question.Identity
	KnowledgeHead     string
	QuestionHead      string
	Blueprint         *question.Blueprint
	BlueprintIdentity *question.Identity
	Candidates        []Candidate
}
type ItemBinding struct {
	Position int                     `json:"position"`
	Instance question.Identity       `json:"instance"`
	Template *question.Identity      `json:"template"`
	Coverage []int                   `json:"coverage"`
	Units    []question.Identity     `json:"units"`
	Assets   []question.AssetRef     `json:"assets"`
	Approval question.MemberEvidence `json:"approval"`
}
type Seal struct {
	Kind                   string             `json:"kind"`
	Knowledge              question.Identity  `json:"knowledge"`
	Mode                   *Mode              `json:"mode"`
	Blueprint              *question.Identity `json:"blueprint"`
	KnowledgePublicationID string             `json:"knowledgePublicationId"`
	QuestionPublicationID  string             `json:"questionPublicationId"`
	RuleVersion            int                `json:"ruleVersion"`
	Core                   []int              `json:"core"`
	Seed                   string             `json:"seed"`
	Items                  []ItemBinding      `json:"items"`
}
