package e2etest

import (
	"context"
	"database/sql"
	"github.com/yyl1212/math_master/backend/internal/auth"
	"github.com/yyl1212/math_master/backend/internal/store"
)

// 仅用于已校验随机库的原创技术夹具，不进入生产路由。
func resetQueryPagination(ctx context.Context, db *sql.DB, s *store.Store, accounts *auth.Service, admin *auth.AdminService, root string) error {
	if e := resetManagedKnowledge(ctx, db, s, accounts, admin, root); e != nil {
		return e
	}
	_, e := db.ExecContext(ctx, `WITH source AS (SELECT * FROM managed_knowledge ORDER BY internal_id LIMIT 1), fixtures AS (SELECT source.*, 'pagination-fixture-'||n AS external FROM source CROSS JOIN generate_series(1,39) n)
 INSERT INTO managed_knowledge(internal_id,external_id,point,public_sources,content_sha256,published,created_by,updated_by)
 SELECT 'k-'||substr(encode(sha256(convert_to(external,'UTF8')),'hex'),1,56),external,jsonb_set(jsonb_set(point,'{id}',to_jsonb(external)),'{title}',to_jsonb('Pagination original fixture '||external)),public_sources,content_sha256,true,created_by,updated_by FROM fixtures`)
	return e
}
