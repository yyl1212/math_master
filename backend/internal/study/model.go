// Package study records explicit personal knowledge study without assessment prerequisites.
package study

import (
	"errors"
	"github.com/yyl1212/math_master/backend/internal/auth"
	"github.com/yyl1212/math_master/backend/internal/taxonomy"
	"time"
)

type State string

const (
	Unlearned State = "unlearned"
	Learning  State = "learning"
	Completed State = "completed"
	Reviewing State = "reviewing"
)

type Action string

const (
	Begin        Action = "begin"
	Complete     Action = "complete"
	StartReview  Action = "start-review"
	FinishReview Action = "finish-review"
	SaveNote     Action = "save-note"
	DeleteNote   Action = "delete-note"
)

var (
	ErrInvalid             = errors.New("invalid study input")
	ErrNotConfigured       = errors.New("study not configured")
	ErrStateConflict       = errors.New("study state conflict")
	ErrVersionStale        = errors.New("study material changed")
	ErrIdempotencyConflict = errors.New("study idempotency conflict")
	ErrNoteConflict        = errors.New("study note revision conflict")
)

type KnowledgeRef = taxonomy.KnowledgeRef
type Access struct {
	TokenHash                 auth.Digest
	CSRF                      auth.Secret
	IdempotencyKey, RequestID string
}
type StudyRecord struct {
	KnowledgeID      string        `json:"knowledgeId"`
	State            State         `json:"state"`
	Sequence         int64         `json:"sequence"`
	FirstStartedAt   *time.Time    `json:"firstStartedAt"`
	FirstCompletedAt *time.Time    `json:"firstCompletedAt"`
	LastCompletedAt  *time.Time    `json:"lastCompletedAt"`
	LastReadAt       *time.Time    `json:"lastReadAt"`
	LastReviewedAt   *time.Time    `json:"lastReviewedAt"`
	CompletedRef     *KnowledgeRef `json:"completedRef"`
	LastReviewRef    *KnowledgeRef `json:"lastReviewRef"`
	LastReviewID     *string       `json:"lastReviewId"`
	ActiveReviewID   *string       `json:"activeReviewId"`
}
type StudyDetail struct {
	ActorID          string                     `json:"actorId"`
	Record           StudyRecord                `json:"record"`
	CurrentKnowledge *taxonomy.KnowledgeSummary `json:"currentKnowledge"`
	Pair             taxonomy.PairRef           `json:"pair"`
	Available        bool                       `json:"available"`
	MaterialChanged  bool                       `json:"materialChanged"`
}
type CommandInput struct {
	Knowledge             KnowledgeRef `json:"knowledge"`
	ExpectedKnowledgeHead string       `json:"expectedKnowledgeHead"`
	ExpectedSequence      int64        `json:"expectedSequence"`
}
type ReviewInput struct {
	CommandInput
	ReviewID string `json:"reviewId"`
}
type ListQuery struct {
	TopicID, State, Q string
	ReviewOnly        bool
	Limit, Offset     int
}
type Page[T any] struct {
	ActorID string `json:"actorId"`
	Items   []T    `json:"items"`
	Total   int    `json:"total"`
	Limit   int    `json:"limit"`
	Offset  int    `json:"offset"`
}
type TopicProgress struct {
	TopicID        string   `json:"topicId"`
	Total          int      `json:"total"`
	Completed      int      `json:"completed"`
	Learning       int      `json:"learning"`
	Reviewing      int      `json:"reviewing"`
	Added          int      `json:"added"`
	Removed        int      `json:"removed"`
	CompletedRatio *float64 `json:"completedRatio"`
}
type ContentReminder struct {
	ChangeID    string        `json:"changeId"`
	KnowledgeID string        `json:"knowledgeId"`
	Kind        string        `json:"kind"`
	RecordedAt  time.Time     `json:"recordedAt"`
	CurrentRef  *KnowledgeRef `json:"currentRef"`
	Reviewed    bool          `json:"reviewed"`
}
type Overview struct {
	ActorID         string                  `json:"actorId"`
	Pair            taxonomy.PairRef        `json:"pair"`
	Mode            taxonomy.ExperienceMode `json:"mode"`
	Total           int                     `json:"total"`
	Completed       int                     `json:"completed"`
	Learning        int                     `json:"learning"`
	Reviewing       int                     `json:"reviewing"`
	Unavailable     int                     `json:"unavailable"`
	Unclassified    int                     `json:"unclassified"`
	MaterialChanged int                     `json:"materialChanged"`
	Reminders       []ContentReminder       `json:"reminders"`
}
type HistoryQuery struct {
	KnowledgeID, TopicID, Kind, Cursor string
	From, To                           *time.Time
	Limit                              int
}
type HistoryEntry struct {
	ID                string       `json:"id"`
	Knowledge         KnowledgeRef `json:"knowledge"`
	TaxonomyVersionID *string      `json:"taxonomyVersionId"`
	TaxonomyHead      *string      `json:"taxonomyHead"`
	Kind              string       `json:"kind"`
	OccurredAt        time.Time    `json:"occurredAt"`
	NoteRevision      *int64       `json:"noteRevision"`
	ReviewID          *string      `json:"reviewId"`
	SourceKind        string       `json:"sourceKind"`
	OriginEventID     *string      `json:"originEventId"`
}
type HistoryPage struct {
	ActorID    string         `json:"actorId"`
	Items      []HistoryEntry `json:"items"`
	Limit      int            `json:"limit"`
	NextCursor *string        `json:"nextCursor"`
}
type NoteInput struct {
	ExpectedRevision int64        `json:"expectedRevision"`
	Knowledge        KnowledgeRef `json:"knowledge"`
	Body             string       `json:"body"`
}
type NoteDeleteInput struct {
	ExpectedRevision int64 `json:"expectedRevision"`
}
type NoteView struct {
	ActorID     string        `json:"actorId"`
	KnowledgeID string        `json:"knowledgeId"`
	Revision    int64         `json:"revision"`
	Body        string        `json:"body"`
	Knowledge   *KnowledgeRef `json:"knowledge"`
	UpdatedAt   *time.Time    `json:"updatedAt"`
}
type NoteReceipt struct {
	ActorID     string    `json:"actorId"`
	KnowledgeID string    `json:"knowledgeId"`
	Revision    int64     `json:"revision"`
	Deleted     bool      `json:"deleted"`
	UpdatedAt   time.Time `json:"updatedAt"`
}
