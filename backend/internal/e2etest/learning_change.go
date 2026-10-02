package e2etest

import (
	"context"
	"database/sql"
	"github.com/yyl1212/math_master/backend/internal/auth"
	"github.com/yyl1212/math_master/backend/internal/content"
	"github.com/yyl1212/math_master/backend/internal/publication"
	"github.com/yyl1212/math_master/backend/internal/question"
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
	case "learning-withdraw-first-assessed":
		a, e := fixtureAccess(ctx, accounts, "content_admin", true)
		if e != nil {
			return true, e
		}
		var id, kh, qh string
		var version int
		if e = db.QueryRowContext(ctx, `SELECT i.instance_id,i.instance_version FROM assessment_items i JOIN assessment_attempts a ON a.id=i.attempt_id WHERE a.state='submitted' ORDER BY a.terminal_at DESC,i.position LIMIT 1`).Scan(&id, &version); e != nil {
			return true, e
		}
		if e = db.QueryRowContext(ctx, `SELECT (SELECT snapshot_id::text FROM publication_heads),(SELECT publication_id::text FROM question_heads)`).Scan(&kh, &qh); e != nil {
			return true, e
		}
		_, e = s.WithdrawQuestionVersion(ctx, a, question.WithdrawalInput{Target: question.WithdrawalTarget{Kind: "instance", ID: id, Version: version}, ExpectedKnowledgeHead: &kh, ExpectedQuestionHead: &qh, Reason: "Permanently withdraw the exact original browser fixture instance."})
		return true, e
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
