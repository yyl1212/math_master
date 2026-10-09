package e2etest

import (
	"bytes"
	"context"
	"database/sql"
	"github.com/yyl1212/math_master/backend/internal/auth"
	"github.com/yyl1212/math_master/backend/internal/knowledgeadmin"
	"github.com/yyl1212/math_master/backend/internal/store"
	"github.com/yyl1212/math_master/backend/internal/testutil"
	"os"
	"path/filepath"
	"strings"
)

func resetManagedKnowledge(ctx context.Context, db *sql.DB, s *store.Store, accounts *auth.Service, admin *auth.AdminService, root string) error {
	if e := resetWorkflow(ctx, db, accounts, admin); e != nil {
		return e
	}
	// Only fixture accounts in this disposable database receive these roles.
	if _, e := db.ExecContext(ctx, `INSERT INTO auth_user_roles(user_id,role) SELECT id,'admin' FROM auth_users WHERE username='auth_admin' ON CONFLICT DO NOTHING`); e != nil {
		return e
	}
	batch := testutil.TaxonomyBatch()
	for n := range batch.Nodes {
		if batch.Nodes[n].Code == "97F00" {
			batch.Nodes[n].Code = "97F40"
			batch.Nodes[n].ID = "msc-97f40"
		}
	}
	v, e := s.InstallTaxonomyBatch(ctx, batch)
	if e != nil {
		return e
	}
	var actor string
	if e = db.QueryRowContext(ctx, "SELECT id::text FROM auth_users WHERE username='auth_admin'").Scan(&actor); e != nil {
		return e
	}
	var rid string
	if e = db.QueryRowContext(ctx, "SELECT gen_random_uuid()::text").Scan(&rid); e != nil {
		return e
	}
	if _, e = db.ExecContext(ctx, `INSERT INTO taxonomy_releases(id,taxonomy_version_id,status,manifest_sha,assignments_sha,body,creator_user_id) VALUES($1,$2,'draft',repeat('c',64),repeat('d',64),'{}',$3)`, rid, v.ID, actor); e != nil {
		return e
	}
	if _, e = db.ExecContext(ctx, "UPDATE taxonomy_releases SET status='published' WHERE id=$1", rid); e != nil {
		return e
	}
	if _, e = db.ExecContext(ctx, "INSERT INTO taxonomy_heads(singleton,release_id) VALUES(true,$1)", rid); e != nil {
		return e
	}
	if _, e = db.ExecContext(ctx, `UPDATE knowledge_admin_state SET content_mode='managed',enabled_once=true,activated_at=coalesce(activated_at,clock_timestamp()) WHERE singleton`); e != nil {
		return e
	}
	if _, e = db.ExecContext(ctx, "UPDATE goose_db_version SET managed_knowledge_enabled=true WHERE version_id=0"); e != nil {
		return e
	}
	b, e := os.ReadFile(filepath.Join(root, "backend/internal/knowledgeadmin/testdata/valid-source.json"))
	if e != nil {
		return e
	}
	doc, e := knowledgeadmin.DecodeSource(bytes.NewReader(b))
	if e != nil {
		return e
	}
	proof, e := fixtureAccess(ctx, accounts, "auth_admin", false)
	if e != nil {
		return e
	}
	a := knowledgeadmin.Access{TokenHash: proof.TokenHash, CSRF: proof.CSRF, IdempotencyKey: proof.IdempotencyKey, RequestID: "managed-fixture"}
	p, e := s.PreviewManagedImport(ctx, a, doc, knowledgeadmin.DigestBytes(b))
	if e != nil {
		return e
	}
	a.IdempotencyKey = strings.Replace(proof.IdempotencyKey, "-", "", -1) + "-apply"
	_, e = s.ApplyManagedImport(ctx, a, p.ImportID, knowledgeadmin.ApplyInput{SelectedIndexes: []int{0, 1}, Publish: true, PreviewToken: p.PreviewToken})
	return e
}
