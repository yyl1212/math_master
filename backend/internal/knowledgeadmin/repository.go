package knowledgeadmin

import (
	"context"
	"github.com/yyl1212/math_master/backend/internal/auth"
)

type Repository interface {
	KnowledgePreflight(context.Context, Access) (auth.User, error)
	ListManagedKnowledge(context.Context, Access, Query) (Page[Knowledge], error)
	ReadManagedKnowledge(context.Context, Access, string) (Knowledge, error)
	CreateManagedKnowledge(context.Context, Access, CurrentInput) (Knowledge, error)
	UpdateManagedKnowledge(context.Context, Access, string, string, CurrentInput) (Knowledge, error)
	SetManagedKnowledgeState(context.Context, Access, string, string, string) (Knowledge, error)
	PreviewManagedImport(context.Context, Access, SourceDocument, string) (Preview, error)
	ApplyManagedImport(context.Context, Access, string, ApplyInput) (Receipt, error)
	ReadManagedImport(context.Context, Access, string) (Preview, *Receipt, error)
}

type CurrentRepository interface {
	ReadContentMode(context.Context) (ContentMode, error)
	ListCurrentKnowledge(context.Context, Query) (Page[PublicKnowledge], error)
	ReadCurrentKnowledge(context.Context, string) (PublicKnowledge, error)
	ListCurrentTopics(context.Context, Query) (Page[CurrentTopic], error)
	ReadCurrentTopic(context.Context, string, Query) (CurrentTopic, error)
}
