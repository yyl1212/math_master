package knowledgeadmin

import (
	"context"
	"github.com/yyl1212/math_master/backend/internal/study"
	"regexp"
	"time"
)

var managedRefID = regexp.MustCompile(`^k-[0-9a-f]{56}$`)
var contentSHA = regexp.MustCompile(`^[0-9a-f]{64}$`)

func ValidManagedRef(r Ref) bool {
	return managedRefID.MatchString(r.ID) && contentSHA.MatchString(r.ContentSHA256) && r.SourceKind == "managed"
}

type ManagedStudyInput struct {
	Knowledge        Ref     `json:"knowledge"`
	ExpectedSequence int64   `json:"expectedSequence"`
	ReviewID         *string `json:"reviewId,omitempty"`
}
type ManagedNoteInput struct {
	Knowledge        Ref    `json:"knowledge"`
	ExpectedRevision int64  `json:"expectedRevision"`
	Body             string `json:"body"`
}
type ManagedRecord struct {
	KnowledgeID      string      `json:"knowledgeId"`
	State            study.State `json:"state"`
	Sequence         int64       `json:"sequence"`
	FirstStartedAt   *time.Time  `json:"firstStartedAt"`
	FirstCompletedAt *time.Time  `json:"firstCompletedAt"`
	LastCompletedAt  *time.Time  `json:"lastCompletedAt"`
	LastReadAt       *time.Time  `json:"lastReadAt"`
	LastReviewedAt   *time.Time  `json:"lastReviewedAt"`
	CompletedRef     *Ref        `json:"completedRef"`
	LastReviewRef    *Ref        `json:"lastReviewRef"`
	LastReviewID     *string     `json:"lastReviewId"`
	ActiveReviewID   *string     `json:"activeReviewId"`
}
type ManagedDetail struct {
	ActorID          string           `json:"actorId"`
	Record           ManagedRecord    `json:"record"`
	CurrentKnowledge *PublicKnowledge `json:"currentKnowledge"`
	Available        bool             `json:"available"`
	MaterialChanged  bool             `json:"materialChanged"`
}
type ManagedNote struct {
	ActorID     string     `json:"actorId"`
	KnowledgeID string     `json:"knowledgeId"`
	Body        string     `json:"body"`
	Revision    int64      `json:"revision"`
	Knowledge   *Ref       `json:"knowledge"`
	Deleted     bool       `json:"deleted"`
	Available   bool       `json:"available"`
	UpdatedAt   *time.Time `json:"updatedAt"`
}
type ManagedProgress struct {
	TopicKey       string   `json:"topicKey"`
	Total          int      `json:"total"`
	Completed      int      `json:"completed"`
	Learning       int      `json:"learning"`
	Reviewing      int      `json:"reviewing"`
	CompletedRatio *float64 `json:"completedRatio"`
}
type ManagedReminder struct {
	KnowledgeID string    `json:"knowledgeId"`
	Kind        string    `json:"kind"`
	RecordedAt  time.Time `json:"recordedAt"`
}
type ManagedOverview struct {
	ActorID         string            `json:"actorId"`
	Total           int               `json:"total"`
	Completed       int               `json:"completed"`
	Learning        int               `json:"learning"`
	Reviewing       int               `json:"reviewing"`
	Unavailable     int               `json:"unavailable"`
	Unclassified    int               `json:"unclassified"`
	MaterialChanged int               `json:"materialChanged"`
	Reminders       []ManagedReminder `json:"reminders"`
}
type ManagedHistoryEntry struct {
	ID           string    `json:"id"`
	Knowledge    Ref       `json:"knowledge"`
	TopicKeys    []string  `json:"topicKeys"`
	Kind         string    `json:"kind"`
	NoteRevision *int64    `json:"noteRevision"`
	ReviewID     *string   `json:"reviewId"`
	RecordedAt   time.Time `json:"recordedAt"`
	SourceKind   string    `json:"sourceKind"`
}
type ManagedHistoryPage struct {
	ActorID    string                `json:"actorId"`
	Items      []ManagedHistoryEntry `json:"items"`
	NextCursor *string               `json:"nextCursor"`
}
type StudyRepository interface {
	ReadManagedOverview(context.Context, Access) (ManagedOverview, error)
	ListManagedStudyTopics(context.Context, Access, Query) (Page[ManagedProgress], error)
	ListManagedStudyKnowledge(context.Context, Access, Query) (Page[ManagedDetail], error)
	ReadManagedStudy(context.Context, Access, string) (ManagedDetail, error)
	ApplyManagedStudy(context.Context, Access, string, string, ManagedStudyInput) (ManagedDetail, error)
	ReadManagedNote(context.Context, Access, string) (ManagedNote, error)
	SaveManagedNote(context.Context, Access, string, ManagedNoteInput) (ManagedNote, error)
	DeleteManagedNote(context.Context, Access, string, ManagedNoteInput) (ManagedNote, error)
	ListManagedHistory(context.Context, Access, study.HistoryQuery) (ManagedHistoryPage, error)
}
