package e2etest

import (
	"context"
	"database/sql"
	"github.com/yyl1212/math_master/backend/internal/auth"
	"github.com/yyl1212/math_master/backend/internal/store"
)

// 原创合成知识，限定在 harness 已校验的随机数据库。
func resetKnowledgePublish(ctx context.Context, db *sql.DB, s *store.Store, accounts *auth.Service, admin *auth.AdminService, root string) error {
	if e := resetQueryPagination(ctx, db, s, accounts, admin, root); e != nil {
		return e
	}
	_, e := db.ExecContext(ctx, `UPDATE managed_knowledge SET published=false,edit_token=gen_random_uuid()`)
	return e
}
