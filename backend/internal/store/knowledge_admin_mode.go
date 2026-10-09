package store

import (
	"context"
	"github.com/yyl1212/math_master/backend/internal/knowledgeadmin"
)

func (s *Store) ReadContentMode(ctx context.Context) (out knowledgeadmin.ContentMode, e error) {
	var markerColumn bool
	e = s.db.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM information_schema.columns WHERE table_schema='public' AND table_name='goose_db_version' AND column_name='managed_knowledge_enabled')`).Scan(&markerColumn)
	if e != nil {
		return out, knowledgeError(e)
	}
	if !markerColumn {
		return knowledgeadmin.ContentMode{Mode: "legacy", Capability: false}, nil
	}
	var marker, once bool
	e = s.db.QueryRowContext(ctx, `SELECT st.content_mode,st.enabled_once,coalesce((SELECT managed_knowledge_enabled FROM goose_db_version WHERE version_id=0),false) FROM knowledge_admin_state st WHERE st.singleton`).Scan(&out.Mode, &once, &marker)
	if e != nil {
		return out, knowledgeadmin.ErrNotConfigured
	}
	if marker != once || once != (out.Mode == "managed") {
		return out, knowledgeadmin.ErrNotConfigured
	}
	out.Capability = once
	return out, nil
}
