package knowledgeadmin

import (
	"context"
	"errors"
	"github.com/yyl1212/math_master/backend/internal/pagination"
	"regexp"
	"time"
)

type PublishMode string

const (
	PublishSelected PublishMode = "selected"
	PublishAll      PublishMode = "all"
)

type PublishItemState string

const (
	PublishPending   PublishItemState = "pending"
	PublishPublished PublishItemState = "published"
	PublishSkipped   PublishItemState = "skipped"
	PublishFailed    PublishItemState = "failed"
)
const MaxPublishInteger int64 = 9007199254740991

var ErrPublishInterrupted = errors.New("publish commit outcome unknown")
var ErrPublishRolledBack = errors.New("publish batch rolled back")
var publishID = regexp.MustCompile(`^k-[0-9a-f]{56}$`)
var publishUUID = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)

func ValidPublishJobID(id string) bool { return publishUUID.MatchString(id) }

type PublishSelection struct {
	ID        string `json:"id"`
	EditToken string `json:"editToken"`
}
type PublishCreateInput struct {
	Mode  PublishMode        `json:"mode"`
	Items []PublishSelection `json:"items,omitempty"`
}

func (i PublishCreateInput) Validate() error {
	if i.Mode == PublishAll {
		if i.Items != nil {
			return ErrInvalid
		}
		return nil
	}
	if i.Mode != PublishSelected || len(i.Items) < 1 || len(i.Items) > 1000 {
		return ErrInvalid
	}
	seen := map[string]bool{}
	for _, v := range i.Items {
		if !publishID.MatchString(v.ID) || !publishUUID.MatchString(v.EditToken) || seen[v.ID] {
			return ErrInvalid
		}
		seen[v.ID] = true
	}
	return nil
}

type PublishJob struct {
	ActorID   string      `json:"actorId"`
	ID        string      `json:"id"`
	Mode      PublishMode `json:"mode"`
	Sequence  int64       `json:"sequence"`
	Total     int64       `json:"total"`
	Pending   int64       `json:"pending"`
	Published int64       `json:"published"`
	Skipped   int64       `json:"skipped"`
	Failed    int64       `json:"failed"`
	CreatedAt time.Time   `json:"createdAt"`
}

func (j PublishJob) Status() string {
	if j.Pending > 0 {
		return "pending"
	}
	if j.Failed > 0 {
		return "partial"
	}
	return "completed"
}

type PublishItem struct {
	ID         string           `json:"id"`
	ExternalID string           `json:"externalId"`
	Title      string           `json:"title"`
	TitleZH    string           `json:"titleZh"`
	State      PublishItemState `json:"state"`
	ErrorCode  string           `json:"errorCode"`
	ErrorPath  string           `json:"errorPath"`
}
type PublishStepInput struct {
	ExpectedSequence int64 `json:"expectedSequence"`
	BatchSize        *int  `json:"batchSize,omitempty"`
}

func (i PublishStepInput) Normalize() (int, error) {
	size := 20
	if i.BatchSize != nil {
		size = *i.BatchSize
	}
	if i.ExpectedSequence < 1 || i.ExpectedSequence >= MaxPublishInteger || size < 1 || size > 20 {
		return 0, ErrInvalid
	}
	return size, nil
}

type PublishStepReceipt struct {
	ActorID   string        `json:"actorId"`
	JobID     string        `json:"jobId"`
	Sequence  int64         `json:"sequence"`
	Items     []PublishItem `json:"items"`
	Pending   int64         `json:"pending"`
	Published int64         `json:"published"`
	Skipped   int64         `json:"skipped"`
	Failed    int64         `json:"failed"`
	Total     int64         `json:"total"`
}
type PublishJobQuery struct {
	CreateKey string
	Limit     int
	Offset    int64
}
type PublishItemQuery struct {
	State  PublishItemState
	Limit  int
	Offset int64
}
type PublishJobDetail struct {
	Job PublishJob `json:"job"`
	pagination.Page[PublishItem]
}
type PublishReceiptLookup struct {
	ActorID string              `json:"actorId"`
	JobID   string              `json:"jobId"`
	Key     string              `json:"key"`
	Receipt *PublishStepReceipt `json:"receipt"`
}
type PublishRepository interface {
	CreatePublishJob(context.Context, Access, PublishCreateInput) (PublishJob, error)
	ListPublishJobs(context.Context, Access, PublishJobQuery) (pagination.ActorPage[PublishJob], error)
	ReadPublishJob(context.Context, Access, string, PublishItemQuery) (PublishJobDetail, error)
	StepPublishJob(context.Context, Access, string, PublishStepInput) (PublishStepReceipt, error)
	ReadPublishJobReceipt(context.Context, Access, string, string) (*PublishStepReceipt, error)
}
