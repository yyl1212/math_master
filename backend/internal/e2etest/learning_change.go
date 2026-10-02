package e2etest

import (
	"context"
	"database/sql"
	"github.com/yyl1212/math_master/backend/internal/auth"
	"github.com/yyl1212/math_master/backend/internal/content"
	"github.com/yyl1212/math_master/backend/internal/publication"
	"github.com/yyl1212/math_master/backend/internal/store"
)

func learningChange(ctx context.Context, db *sql.DB, s *store.Store, accounts *auth.Service, root, scene string) (bool, error) {
	switch scene {
	case "learning-new-route":
		in, e := learningFixtureInput(root)
		if e != nil {
			return true, e
		}
		in.Package.Version = 2
		in.Package.Paths[0].Version = 2
		in.Package.Paths[0].Nodes = append(in.Package.Paths[0].Nodes, content.VersionRef{ID: "learning-practice-only", Version: 1})
		return true, learningPublishContent(ctx, db, s, accounts, in)
	case "learning-withdraw-root":
		a, e := fixtureAccess(ctx, accounts, "content_admin", true)
		if e != nil {
			return true, e
		}
		var head string
		if e = db.QueryRowContext(ctx, "SELECT snapshot_id::text FROM publication_heads").Scan(&head); e != nil {
			return true, e
		}
		_, e = s.WithdrawVersion(ctx, a, publication.WithdrawalInput{Target: publication.WithdrawalTarget{Kind: "knowledge", ID: "learning-root", Version: 1}, ExpectedHead: &head, Reason: "Permanently withdraw original isolated learning fixture root."})
		return true, e
	}
	return false, nil
}
