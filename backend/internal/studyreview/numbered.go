package studyreview

import (
	"context"
	"github.com/yyl1212/math_master/backend/internal/pagination"
	"time"
)

type NumberedHistoryPage struct {
	ActorID string    `json:"actorId"`
	AsOf    time.Time `json:"asOf"`
	pagination.Page[HistoryItem]
}
type NumberedRepository interface {
	ListReviewHistoryPage(context.Context, Access, HistoryQuery, pagination.Request) (NumberedHistoryPage, error)
}
