// Package studyreview manages private reading/recall sessions and self-rated schedules.
package studyreview

import (
	"errors"
	"github.com/yyl1212/math_master/backend/internal/knowledgeadmin"
	"time"
)

type Access = knowledgeadmin.Access
type Ref = knowledgeadmin.Ref
type Mode string
type Target string
type Status string
type Rating string

const (
	Read                 Mode   = "read"
	Recall               Mode   = "recall"
	Statement            Target = "statement"
	Proof                Target = "proof"
	Working              Status = "working"
	Compared             Status = "compared"
	Completed            Status = "completed"
	Abandoned            Status = "abandoned"
	Forgotten            Rating = "forgotten"
	Uncertain            Rating = "uncertain"
	Familiar             Rating = "familiar"
	MaxSequence          int64  = 9007199254740991
	MaxResponseBytes            = 1048576
	MaxMaterialBytes            = 524288
	MaxMaterialWireBytes        = 589824
	MaxMetadataWireBytes        = 65536
	MaxAnswerWireBytes          = 393216
)

var (
	ErrInvalid           = errors.New("invalid study review input")
	ErrReferenceTooLarge = errors.New("review reference too large")
	ErrConflict          = errors.New("review state conflict")
	ErrIdempotency       = errors.New("review idempotency conflict")
	ErrStale             = errors.New("review material changed")
	ErrUnavailable       = errors.New("review knowledge unavailable")
	ErrNotFound          = errors.New("review not found")
	ErrNotConfigured     = errors.New("review not configured")
)

type Health struct {
	Capability  bool `json:"capability"`
	SchemaReady bool `json:"schemaReady"`
}
type Schedule struct {
	Stage      int       `json:"stage"`
	DueAt      time.Time `json:"dueAt"`
	LastRef    Ref       `json:"lastRef"`
	LastRating Rating    `json:"lastRating"`
	Sequence   int64     `json:"sequence"`
	GradedAt   time.Time `json:"gradedAt"`
}
type Section struct {
	Kind  string   `json:"kind"`
	Items []string `json:"items"`
}
type FrozenMaterial struct {
	Prompt   []Section `json:"prompt"`
	Expected []Section `json:"expected"`
	SHA256   string    `json:"sha256"`
}
type ExpectedReference struct {
	Sections []Section `json:"sections"`
	SHA256   string    `json:"sha256"`
}
type StartInput struct {
	Knowledge Ref    `json:"knowledge"`
	Mode      Mode   `json:"mode"`
	Target    Target `json:"target"`
}
type DraftInput struct {
	ExpectedSequence      int64  `json:"expectedSequence"`
	ExpectedDraftRevision int64  `json:"expectedDraftRevision"`
	Body                  string `json:"body"`
}
type CompareInput struct {
	ExpectedSequence int64  `json:"expectedSequence"`
	Body             string `json:"body"`
	Unanswered       bool   `json:"unanswered"`
}
type GradeInput struct {
	ExpectedSequence int64  `json:"expectedSequence"`
	Rating           Rating `json:"rating"`
}
type AbandonInput struct {
	ExpectedSequence int64 `json:"expectedSequence"`
}
type GradeResult struct {
	Rating   Rating    `json:"rating"`
	GradedAt time.Time `json:"gradedAt"`
	Schedule Schedule  `json:"schedule"`
}
type CommandReceipt struct {
	ActorID       string       `json:"actorId"`
	Action        string       `json:"action"`
	SessionID     string       `json:"sessionId"`
	KnowledgeID   string       `json:"knowledgeId"`
	ReviewID      string       `json:"reviewId"`
	Knowledge     Ref          `json:"knowledge"`
	Sequence      int64        `json:"sequence"`
	DraftRevision int64        `json:"draftRevision"`
	Status        Status       `json:"status"`
	AppliedAt     time.Time    `json:"appliedAt"`
	Grade         *GradeResult `json:"grade"`
}
type Session struct {
	ActorID         string             `json:"actorId"`
	SessionID       string             `json:"sessionId"`
	KnowledgeID     string             `json:"knowledgeId"`
	ReviewID        string             `json:"reviewId"`
	Knowledge       Ref                `json:"knowledge"`
	Mode            Mode               `json:"mode"`
	Target          Target             `json:"target"`
	Status          Status             `json:"status"`
	Sequence        int64              `json:"sequence"`
	DraftRevision   int64              `json:"draftRevision"`
	Title           string             `json:"title"`
	TitleZH         string             `json:"titleZh"`
	KnowledgeType   string             `json:"knowledgeType"`
	CreatedAt       time.Time          `json:"createdAt"`
	ComparedAt      *time.Time         `json:"comparedAt"`
	GradedAt        *time.Time         `json:"gradedAt"`
	AbandonedAt     *time.Time         `json:"abandonedAt"`
	DraftUpdatedAt  *time.Time         `json:"draftUpdatedAt"`
	Available       bool               `json:"available"`
	MaterialChanged bool               `json:"materialChanged"`
	Unanswered      bool               `json:"unanswered"`
	Prompt          []Section          `json:"prompt"`
	Answer          *string            `json:"answer"`
	Reference       *ExpectedReference `json:"reference"`
	Schedule        *Schedule          `json:"schedule"`
	Grade           *GradeResult       `json:"grade"`
}
type QueueQuery struct {
	Q, TopicKey, Bucket string
	Limit, Offset       int
}
type HistoryQuery struct {
	KnowledgeID, TopicKey string
	Mode                  Mode
	Rating                Rating
	Cursor                string
	Limit                 int
}
type Overview struct {
	ActorID    string    `json:"actorId"`
	AsOf       time.Time `json:"asOf"`
	Total      int       `json:"total"`
	Unfinished int       `json:"unfinished"`
	Changed    int       `json:"changed"`
	Due        int       `json:"due"`
	Unplanned  int       `json:"unplanned"`
	Future     int       `json:"future"`
}
type QueueItem struct {
	Knowledge       Ref       `json:"knowledge"`
	Title           string    `json:"title"`
	TitleZH         string    `json:"titleZh"`
	KnowledgeType   string    `json:"knowledgeType"`
	TopicKeys       []string  `json:"topicKeys"`
	Bucket          string    `json:"bucket"`
	MaterialChanged bool      `json:"materialChanged"`
	ProofAvailable  bool      `json:"proofAvailable"`
	ActiveSessionID *string   `json:"activeSessionId"`
	Schedule        *Schedule `json:"schedule"`
}
type QueuePage struct {
	ActorID string      `json:"actorId"`
	AsOf    time.Time   `json:"asOf"`
	Items   []QueueItem `json:"items"`
	Total   int         `json:"total"`
	Limit   int         `json:"limit"`
	Offset  int         `json:"offset"`
}
type HistoryItem struct {
	SessionID       string      `json:"sessionId"`
	Knowledge       Ref         `json:"knowledge"`
	Title           string      `json:"title"`
	TitleZH         string      `json:"titleZh"`
	KnowledgeType   string      `json:"knowledgeType"`
	Mode            Mode        `json:"mode"`
	Target          Target      `json:"target"`
	Grade           GradeResult `json:"grade"`
	Available       bool        `json:"available"`
	MaterialChanged bool        `json:"materialChanged"`
}
type HistoryPage struct {
	ActorID    string        `json:"actorId"`
	AsOf       time.Time     `json:"asOf"`
	Items      []HistoryItem `json:"items"`
	NextCursor *string       `json:"nextCursor"`
}

// ActiveSessionError only names the caller's own session, allowing an explicit continue choice.
type ActiveSessionError struct{ SessionID string }

func (e *ActiveSessionError) Error() string { return "review session already active" }
func (e *ActiveSessionError) Unwrap() error { return ErrConflict }
