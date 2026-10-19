package notesharing

import (
	"context"
	"github.com/yyl1212/math_master/backend/internal/knowledgeadmin"
	"github.com/yyl1212/math_master/backend/internal/pagination"
)

type NumberedRepository interface {
	ListOwnNotePage(context.Context, knowledgeadmin.Access, Query, pagination.Request) (pagination.ActorPage[OwnNoteSummary], error)
	ListSharedNotePage(context.Context, knowledgeadmin.Access, Query, pagination.Request) (pagination.ActorPage[SharedNoteSummary], error)
	ListAdminNoteSharePage(context.Context, knowledgeadmin.Access, Query, pagination.Request) (pagination.ActorPage[AdminSummary], error)
}
