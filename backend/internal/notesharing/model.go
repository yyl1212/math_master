package notesharing

import (
	"errors"
	"github.com/yyl1212/math_master/backend/internal/knowledgeadmin"
	"time"
)

var (
	ErrLocked      = errors.New("note sharing locks editing")
	ErrStale       = errors.New("note share changed")
	ErrIdempotency = errors.New("note share request key conflict")
	ErrNotPublic   = errors.New("note share is not publicly available")
	ErrNotFound    = errors.New("note share not found")
	ErrInvalid     = errors.New("invalid note share input")
	ErrNotReady    = errors.New("note sharing schema is not ready")
)

const (
	MaxRequestBytes = 131072
	MaxDetailBytes  = 1048576
	MaxPageBytes    = 4 << 20
	DefaultLimit    = 20
	MaxLimit        = 50
	Unshared        = "unshared"
	Pending         = "pending"
	Shared          = "shared"
	Returned        = "returned"
	Withdrawn       = "withdrawn"
	Hidden          = "hidden"
)

type Query struct {
	Q, TopicKey, Status, Cursor string
	Limit                       int
}
type SubmitInput struct {
	ExpectedRevision int64  `json:"expectedRevision"`
	ExpectedSequence int64  `json:"expectedSequence"`
	Title            string `json:"title"`
}
type WithdrawInput struct {
	SubmissionID     string `json:"submissionId"`
	ExpectedSequence int64  `json:"expectedSequence"`
}
type DecisionInput struct {
	ExpectedSequence int64  `json:"expectedSequence"`
	Reason           string `json:"reason,omitempty"`
}
type Head struct {
	ActorID            string     `json:"actorId"`
	KnowledgeID        string     `json:"knowledgeId"`
	Status             string     `json:"status"`
	Sequence           int64      `json:"sequence"`
	SubmissionID       *string    `json:"submissionId"`
	ApprovedAt         *time.Time `json:"approvedAt"`
	LastDecisionReason string     `json:"lastDecisionReason"`
}
type Receipt struct {
	SubmissionID string `json:"submissionId"`
	KnowledgeID  string `json:"knowledgeId"`
	Action       string `json:"action"`
	Status       string `json:"status"`
	Sequence     int64  `json:"sequence"`
}
type OwnNoteSummary struct {
	KnowledgeID      string    `json:"knowledgeId"`
	KnowledgeTitle   string    `json:"knowledgeTitle"`
	KnowledgeTitleZH string    `json:"knowledgeTitleZh"`
	TopicKeys        []string  `json:"topicKeys"`
	Excerpt          string    `json:"excerpt"`
	UpdatedAt        time.Time `json:"updatedAt"`
	Available        bool      `json:"available"`
	NoteRevision     int64     `json:"noteRevision"`
	Share            Head      `json:"share"`
}
type OwnNotePage struct {
	ActorID    string           `json:"actorId"`
	Items      []OwnNoteSummary `json:"items"`
	NextCursor *string          `json:"nextCursor"`
}
type SharedNoteSummary struct {
	SubmissionID     string    `json:"submissionId"`
	Title            string    `json:"title"`
	Excerpt          string    `json:"excerpt"`
	AuthorName       string    `json:"authorName"`
	ApprovedAt       time.Time `json:"approvedAt"`
	KnowledgeID      string    `json:"knowledgeId"`
	KnowledgeTitle   string    `json:"knowledgeTitle"`
	KnowledgeTitleZH string    `json:"knowledgeTitleZh"`
	TopicKeys        []string  `json:"topicKeys"`
	MaterialChanged  bool      `json:"materialChanged"`
}
type SharedNoteDetail struct {
	SharedNoteSummary
	Body string `json:"body"`
}
type AuthenticatedSharedNoteDetail struct {
	ActorID string           `json:"actorId"`
	Note    SharedNoteDetail `json:"note"`
}
type LatestSharedNotePage struct {
	Items []SharedNoteSummary `json:"items"`
}
type SharedNotePage struct {
	ActorID    string              `json:"actorId"`
	Items      []SharedNoteSummary `json:"items"`
	NextCursor *string             `json:"nextCursor"`
}
type AdminSummary struct {
	SubmissionID string    `json:"submissionId"`
	AuthorName   string    `json:"authorName"`
	KnowledgeID  string    `json:"knowledgeId"`
	Title        string    `json:"title"`
	SubmittedAt  time.Time `json:"submittedAt"`
	Status       string    `json:"status"`
	Sequence     int64     `json:"sequence"`
}
type AdminNoteSharePage struct {
	ActorID    string         `json:"actorId"`
	Items      []AdminSummary `json:"items"`
	NextCursor *string        `json:"nextCursor"`
}
type ShareEventSummary struct {
	Action     string    `json:"action"`
	ActorName  string    `json:"actorName"`
	Reason     string    `json:"reason"`
	Sequence   int64     `json:"sequence"`
	RecordedAt time.Time `json:"recordedAt"`
}
type AdminNoteShareDetail struct {
	ActorID string `json:"actorId"`
	AdminSummary
	Body             string              `json:"body"`
	NoteRevision     int64               `json:"noteRevision"`
	NoteKnowledgeRef knowledgeadmin.Ref  `json:"noteKnowledgeRef"`
	KnowledgeTitle   string              `json:"knowledgeTitle"`
	KnowledgeTitleZH string              `json:"knowledgeTitleZh"`
	TopicKeys        []string            `json:"topicKeys"`
	ApprovedAt       *time.Time          `json:"approvedAt"`
	Events           []ShareEventSummary `json:"events"`
	NextEventsCursor *string             `json:"nextEventsCursor"`
}
type NoteSharingHealth struct {
	Capability  bool `json:"capability"`
	SchemaReady bool `json:"schemaReady"`
}
