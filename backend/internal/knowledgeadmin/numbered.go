package knowledgeadmin

import (
	"context"
	"github.com/yyl1212/math_master/backend/internal/pagination"
	"github.com/yyl1212/math_master/backend/internal/study"
)

type NumberedRepository interface {
	ListManagedHistoryPage(context.Context, Access, study.HistoryQuery, pagination.Request) (pagination.ActorPage[ManagedHistoryEntry], error)
}
