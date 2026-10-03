// Package notification carries only owned static notification metadata.
package notification

import (
	"github.com/yyl1212/math_master/backend/internal/correction"
	"time"
)

type EvidenceRef = correction.EvidenceRef
type Type string

const (
	Checking        Type = "checking"
	Corrected       Type = "corrected"
	Retake          Type = "retake"
	ReviewMaterial  Type = "review_material"
	PathUnavailable Type = "path_unavailable"
)

type Metadata struct {
	ID        string      `json:"id"`
	Type      Type        `json:"type"`
	Evidence  EvidenceRef `json:"evidence"`
	CaseID    string      `json:"caseId"`
	ResultID  *string     `json:"resultId"`
	CreatedAt time.Time   `json:"createdAt"`
	ReadAt    *time.Time  `json:"readAt"`
}
type UnreadCount struct {
	Count int64 `json:"count"`
}
type ReadReceipt struct {
	Status         int       `json:"status"`
	NotificationID string    `json:"notificationId"`
	ReadAt         time.Time `json:"readAt"`
}
type Source struct {
	DedupKey string
	Type     Type
	Evidence EvidenceRef
	CaseID   string
	ResultID *string
}
type Query struct {
	Limit  int
	Cursor string
}
type Page[T any] struct {
	Items      []T     `json:"items"`
	NextCursor *string `json:"nextCursor"`
}
type Envelope[T any] struct {
	ActorID string `json:"actorId"`
	Data    T      `json:"data"`
}
