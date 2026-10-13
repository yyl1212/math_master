package notesharing

import (
	"context"
	"github.com/yyl1212/math_master/backend/internal/knowledgeadmin"
)

type Repository interface {
	ListOwnNotes(context.Context, knowledgeadmin.Access, Query) (OwnNotePage, error)
	ReadOwnHead(context.Context, knowledgeadmin.Access, string) (Head, error)
	Submit(context.Context, knowledgeadmin.Access, string, SubmitInput) (Receipt, error)
	Withdraw(context.Context, knowledgeadmin.Access, string, WithdrawInput) (Receipt, error)
	Latest(context.Context) (LatestSharedNotePage, error)
	ReadPublic(context.Context, string) (SharedNoteDetail, error)
	ListShared(context.Context, knowledgeadmin.Access, Query) (SharedNotePage, error)
	ReadShared(context.Context, knowledgeadmin.Access, string) (AuthenticatedSharedNoteDetail, error)
	ListAdmin(context.Context, knowledgeadmin.Access, Query) (AdminNoteSharePage, error)
	ReadAdmin(context.Context, knowledgeadmin.Access, string, string) (AdminNoteShareDetail, error)
	Decide(context.Context, knowledgeadmin.Access, string, string, DecisionInput) (Receipt, error)
	ReadSchemaHealth(context.Context) (NoteSharingHealth, error)
}
