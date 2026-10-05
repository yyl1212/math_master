package store

import (
	"context"
	"database/sql"
	"github.com/yyl1212/math_master/backend/internal/auth"
	"github.com/yyl1212/math_master/backend/internal/publication"
	"github.com/yyl1212/math_master/backend/internal/question"
	"time"
)

func (s *Store) DecideQuestionReview(ctx context.Context, a question.Access, id string, input question.ReviewInput) (question.SubmissionView, error) {
	var out question.SubmissionView
	if !question.ValidID(id) {
		return out, auth.ErrInvalidInput
	}
	err := s.questionTx(ctx, a, question.DecideReviewAction, nil, func(ctx context.Context, tx *sql.Tx, u auth.User, now time.Time) error {
		sub, _, err := s.readQuestionSubmission(ctx, tx, u, id, false)
		if err != nil {
			return err
		}
		prior, found, err := questionJSONReplay[question.SubmissionView](s, ctx, tx, u, a, question.DecideReviewAction, id, input)
		if err != nil {
			return err
		}
		if found {
			out = prior
			return nil
		}
		if sub.Status != "pending" || sub.Review != nil {
			return question.ErrReviewConflict
		}
		if err = question.ValidateReviewInput(input, len(sub.Frozen.QuestionPackage.Templates) > 0); err != nil {
			return err
		}
		for _, author := range sub.Frozen.AuthorIDs {
			if author == u.ID && !publication.HasRole(u, auth.RoleAdmin) {
				return auth.ErrForbidden
			}
		}
		if input.Decision == "approve" {
			if !sub.Gate.ReadyToSubmit || sub.Gate.StructuralTotal != 0 || sub.Gate.CompletenessTotal != 0 {
				return question.ErrNotReady
			}
		}
		reviewID, err := workflowID()
		if err != nil {
			return err
		}
		review := question.ReviewDecision{ID: reviewID, SubmissionID: id, ReviewerID: u.ID, FrozenDigest: sub.Frozen.FrozenDigest, Decision: input.Decision, Checks: input.Checks, IndependenceNote: input.IndependenceNote, GenerationNote: input.GenerationNote, Note: input.Note, CreatedAt: now.UTC().Format(time.RFC3339)}
		_, err = tx.ExecContext(ctx, `INSERT INTO question_review_decisions(id,submission_id,reviewer_user_id,frozen_digest,decision,checks,independence_note,generation_note,note,created_at) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`, reviewID, id, u.ID, review.FrozenDigest, input.Decision, body(input.Checks), input.IndependenceNote, input.GenerationNote, input.Note, now)
		if err != nil {
			return err
		}
		status := "approved"
		if input.Decision == "return" {
			status = "returned"
		}
		if _, err = tx.ExecContext(ctx, `UPDATE question_submissions SET status=$2 WHERE id=$1 AND status='pending'`, id, status); err != nil {
			return err
		}
		if status == "returned" {
			result, err := tx.ExecContext(ctx, `UPDATE question_workspaces SET status='editing',revision=revision+1,updated_at=$4 WHERE id=$1 AND owner_user_id=$2 AND revision=$3 AND status='submitted'`, sub.WorkspaceID, sub.OwnerID, sub.Revision, now)
			if err != nil {
				return err
			}
			n, err := result.RowsAffected()
			if err != nil {
				return err
			}
			if n != 1 {
				return question.ErrDraftConflict
			}
		}
		out = sub
		out.Status = status
		out.Review = &review
		if err = questionEvent(ctx, tx, u, a, question.DecideReviewAction, "submission", id, review.FrozenDigest, review.FrozenDigest, "pending", status, input.Note, now); err != nil {
			return err
		}
		return questionJSONRemember(s, ctx, tx, u, a, question.DecideReviewAction, id, input, out)
	})
	return out, err
}
