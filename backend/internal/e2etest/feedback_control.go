package e2etest

import (
	"context"
	"database/sql"
	"github.com/yyl1212/math_master/backend/internal/auth"
	"github.com/yyl1212/math_master/backend/internal/feedback"
	"github.com/yyl1212/math_master/backend/internal/question"
	"github.com/yyl1212/math_master/backend/internal/store"
)

type feedbackDatabaseState struct {
	Tickets      int                   `json:"tickets"`
	Events       int                   `json:"events"`
	Rates        int                   `json:"rates"`
	TicketID     string                `json:"ticketId"`
	WithdrawalID *string               `json:"withdrawalId"`
	Replacement  *feedback.Replacement `json:"replacement"`
}

func feedbackState(ctx context.Context, db *sql.DB) (feedbackDatabaseState, error) {
	v := feedbackDatabaseState{}
	e := db.QueryRowContext(ctx, `SELECT (SELECT count(*) FROM feedback_tickets),(SELECT count(*) FROM feedback_events),(SELECT count(*) FROM feedback_rate_limits)`).Scan(&v.Tickets, &v.Events, &v.Rates)
	if e != nil {
		return v, e
	}
	v.TicketID, _, e = fixtureFeedbackTarget(ctx, db)
	if e != nil {
		return v, e
	}
	var w string
	e = db.QueryRowContext(ctx, `SELECT id FROM question_withdrawals WHERE kind='instance' ORDER BY created_at DESC LIMIT 1`).Scan(&w)
	if e == nil {
		v.WithdrawalID = &w
	} else if e != sql.ErrNoRows {
		return v, e
	}
	var id question.Identity
	var pub string
	e = db.QueryRowContext(ctx, `SELECT m.id,m.version,m.sha256,m.publication_id FROM question_publication_members m JOIN question_instances i ON i.id=m.id AND i.version=m.version WHERE m.publication_id=(SELECT publication_id FROM question_heads) AND m.kind='instance' AND i.template_id='learning-root-addition' AND i.template_version=2 ORDER BY m.id LIMIT 1`).Scan(&id.ID, &id.Version, &id.SHA256, &pub)
	if e == nil {
		v.Replacement = &feedback.Replacement{Kind: "instance", Identity: &id, PublicationID: pub}
	} else if e != sql.ErrNoRows {
		return v, e
	}
	return v, nil
}
func feedbackChange(ctx context.Context, db *sql.DB, s *store.Store, accounts *auth.Service, root, scene string) (bool, error) {
	switch scene {
	case "feedback-withdraw-instance":
		_, target, e := fixtureFeedbackTarget(ctx, db)
		if e != nil {
			return true, e
		}
		var kh, qh string
		if e = db.QueryRowContext(ctx, `SELECT (SELECT snapshot_id::text FROM publication_heads),(SELECT publication_id::text FROM question_heads)`).Scan(&kh, &qh); e != nil {
			return true, e
		}
		a, e := fixtureAccess(ctx, accounts, "content_admin", true)
		if e != nil {
			return true, e
		}
		_, e = s.WithdrawQuestionVersion(ctx, a, question.WithdrawalInput{Target: question.WithdrawalTarget{Kind: "instance", ID: target.Identity.ID, Version: target.Identity.Version}, ExpectedKnowledgeHead: &kh, ExpectedQuestionHead: &qh, Reason: "Permanent exact original fixture withdrawal."})
		return true, e
	case "feedback-new-instance":
		_, e := publishFeedbackReplacement(ctx, db, s, accounts, root)
		return true, e
	case "feedback-history":
		id, _, e := fixtureFeedbackTarget(ctx, db)
		if e != nil {
			return true, e
		}
		a, e := fixtureAccess(ctx, accounts, "content_reviewer", false)
		if e != nil {
			return true, e
		}
		for n := 0; n < 60; n++ {
			a, _ = nextFixtureAccess(a)
			m, e := s.ReadFeedbackTicket(ctx, a, id, true)
			if e != nil {
				return true, e
			}
			a, _ = nextFixtureAccess(a)
			if _, e = s.TransitionFeedback(ctx, a, id, feedback.TransitionInput{ExpectedSequence: m.Data.Sequence, Status: feedback.Processing, Message: "Original history reply. 中"}); e != nil {
				return true, e
			}
		}
		return true, nil
	}
	return false, nil
}
