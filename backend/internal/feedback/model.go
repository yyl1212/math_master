// Package feedback defines versioned reports without granting publication or learning authority.
package feedback

import (
	"errors"
	"github.com/yyl1212/math_master/backend/internal/knowledgeadmin"
	"github.com/yyl1212/math_master/backend/internal/question"
	"time"
)

type Status string
type Category string
type Area string
type ResolutionKind string
type Action string
type AssetRef = question.AssetRef

const (
	New                 Status = "new"
	Processing          Status = "processing"
	WaitingDetails      Status = "waiting_details"
	Resolved            Status = "resolved"
	Closed              Status = "closed"
	ReadContextAction   Action = "readContext"
	CreateAction        Action = "create"
	ReplyAction         Action = "reply"
	TransitionAction    Action = "transition"
	ListOwnAction       Action = "listOwn"
	ReadOwnAction       Action = "readOwn"
	DiscussOwnAction    Action = "discussOwn"
	ListReviewAction    Action = "listReview"
	ReadReviewAction    Action = "readReview"
	DiscussReviewAction Action = "discussReview"
	MaxSequence         int64  = 9007199254740991
	MaxRequestBytes            = 65536
	MaxResponseBytes           = 2097152
)

var (
	ErrNotConfigured = errors.New("feedback is not configured")
	ErrConflict      = errors.New("feedback sequence conflict")
	ErrTargetStale   = errors.New("feedback target changed")
	ErrAnswerOverlap = errors.New("feedback discussion overlaps an active assessment")
)

type RateError struct{ RetryAt time.Time }

func (e *RateError) Error() string { return "feedback rate limit reached" }

type Part struct {
	Kind  string             `json:"kind"`
	Unit  *question.Identity `json:"unit"`
	Asset *AssetRef          `json:"asset"`
}
type Target struct {
	ManagedRef *knowledgeadmin.Ref `json:"managedRef,omitempty"`
	Kind       string              `json:"kind"`
	Identity   *question.Identity  `json:"identity"`
	Area       *Area               `json:"area"`
	Part       *Part               `json:"part"`
}
type Source struct {
	Kind          string  `json:"kind"`
	PublicationID *string `json:"publicationId"`
	AttemptID     *string `json:"attemptId"`
	Position      *int    `json:"position"`
}
type ContextQuery struct {
	Kind, ID         string
	Area             Area
	PartKind, PartID string
	Position         int
}
type Context struct {
	Target Target `json:"target"`
	Source Source `json:"source"`
	Label  string `json:"label"`
}
type CreateInput struct {
	Target   Target   `json:"target"`
	Source   Source   `json:"source"`
	Category Category `json:"category"`
	Title    string   `json:"title"`
	Message  string   `json:"message"`
	Location string   `json:"location"`
}
type ReplyInput struct {
	ExpectedSequence int64  `json:"expectedSequence"`
	Message          string `json:"message"`
}
type WithdrawalRef struct {
	Space string `json:"space"`
	ID    string `json:"id"`
}
type Replacement struct {
	Kind          string             `json:"kind"`
	Identity      *question.Identity `json:"identity"`
	Asset         *AssetRef          `json:"asset"`
	PublicationID string             `json:"publicationId"`
}
type Resolution struct {
	Kind        ResolutionKind `json:"kind"`
	Withdrawal  *WithdrawalRef `json:"withdrawal"`
	Replacement *Replacement   `json:"replacement"`
	DuplicateOf *string        `json:"duplicateOf"`
}
type TransitionInput struct {
	ExpectedSequence int64       `json:"expectedSequence"`
	Status           Status      `json:"status"`
	Message          string      `json:"message"`
	Resolution       *Resolution `json:"resolution"`
}
type Metadata struct {
	ID             string          `json:"id"`
	Target         Target          `json:"target"`
	Label          string          `json:"label"`
	Category       Category        `json:"category"`
	Status         Status          `json:"status"`
	Sequence       int64           `json:"sequence"`
	CreatedAt      time.Time       `json:"createdAt"`
	UpdatedAt      time.Time       `json:"updatedAt"`
	ResolutionKind *ResolutionKind `json:"resolutionKind"`
	TargetValidity string          `json:"targetValidity"`
	CanHandle      bool            `json:"canHandle"`
}
type EventView struct {
	Sequence   int64       `json:"sequence"`
	Kind       string      `json:"kind"`
	Actor      string      `json:"actor"`
	From       *Status     `json:"from"`
	To         Status      `json:"to"`
	Message    string      `json:"message"`
	Resolution *Resolution `json:"resolution"`
	RecordedAt time.Time   `json:"recordedAt"`
}
type DiscussionPage struct {
	Title      string      `json:"title"`
	Location   string      `json:"location"`
	Items      []EventView `json:"items"`
	NextCursor *string     `json:"nextCursor"`
}
type Page[T any] struct {
	Items      []T     `json:"items"`
	NextCursor *string `json:"nextCursor"`
}
type Receipt struct {
	Status int      `json:"status"`
	Ticket Metadata `json:"ticket"`
}
type Envelope[T any] struct {
	ActorID string `json:"actorId"`
	Data    T      `json:"data"`
}
type ListQuery struct {
	Limit    int
	Cursor   string
	Status   *Status
	Category *Category
}

// Binding is repository-only proof derived from immutable database facts.
type Binding struct {
	Target                                        Target
	Source                                        Source
	OwnerID                                       *string
	Knowledge, Instance, Template                 *question.Identity
	Units                                         []question.Identity
	Assets                                        []AssetRef
	KnowledgePublicationID, QuestionPublicationID *string
	ContentApprovalIDs, QuestionApprovalIDs       []string
}
