package store

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/yyl1212/math_master/backend/internal/auth"
	"github.com/yyl1212/math_master/backend/internal/publication"
	"time"
)

func (s *Store) DecideReview(ctx context.Context, a publication.Access, id string, input publication.ReviewInput) (publication.SubmissionView, error) {
	var out publication.SubmissionView
	if !publication.ValidID(id) {
		return out, auth.ErrInvalidInput
	}
	err := s.workflowTx(ctx, a, publication.DecideReviewAction, nil, func(ctx context.Context, tx *sql.Tx, u auth.User, now time.Time) error {
		sub, err := s.readWorkflowSubmission(ctx, tx, u, id, false)
		if err != nil {
			return err
		}
		prior, found, err := workflowJSONReplay[publication.SubmissionView](s, ctx, tx, u, a, publication.DecideReviewAction, id, input)
		if err != nil {
			return err
		}
		if found {
			out = prior
			return nil
		}
		if sub.Status != "pending" || sub.Review != nil {
			return publication.ErrReviewConflict
		}
		if err = publication.ValidateReviewInput(input); err != nil {
			return err
		}
		for _, author := range sub.Frozen.AuthorIDs {
			if author == u.ID && !publication.HasRole(u, auth.RoleAdmin) {
				return auth.ErrForbidden
			}
		}
		if input.Decision == "approve" {
			if !sub.Gate.ReadyToSubmit || sub.Gate.StructuralTotal != 0 || sub.Gate.CompletenessTotal != 0 {
				return publication.ErrContentNotReady
			}
		}
		reviewID, err := workflowID()
		if err != nil {
			return err
		}
		review := publication.ReviewDecision{ID: reviewID, SubmissionID: id, ReviewerID: u.ID, FrozenDigest: sub.Frozen.FrozenDigest, Decision: input.Decision, Checks: input.Checks, IndependenceNote: input.IndependenceNote, Note: input.Note, CreatedAt: now.UTC().Format(time.RFC3339)}
		_, err = tx.ExecContext(ctx, `INSERT INTO content_review_decisions(id,submission_id,reviewer_user_id,frozen_digest,decision,checks,independence_note,note,created_at) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9)`, reviewID, id, u.ID, review.FrozenDigest, input.Decision, body(input.Checks), input.IndependenceNote, input.Note, now)
		if err != nil {
			return err
		}
		if err = bindTopicReviewTx(ctx, tx, id, reviewID); err != nil {
			return err
		}
		status := "approved"
		if input.Decision == "return" {
			status = "returned"
		}
		result, err := tx.ExecContext(ctx, `UPDATE content_submissions SET status=$2 WHERE id=$1 AND sealed AND status='pending'`, id, status)
		if err != nil {
			return err
		}
		n, err := result.RowsAffected()
		if err != nil {
			return err
		}
		if n != 1 {
			return publication.ErrReviewConflict
		}
		afterState := status
		if input.Decision == "return" {
			result, err = tx.ExecContext(ctx, `UPDATE content_workspaces SET status='editing',revision=revision+1,updated_at=$4 WHERE id=$1 AND owner_user_id=$2 AND revision=$3 AND status='submitted'`, sub.WorkspaceID, sub.OwnerID, sub.Revision, now)
			if err != nil {
				return err
			}
			n, err = result.RowsAffected()
			if err != nil {
				return err
			}
			if n != 1 {
				return publication.ErrDraftConflict
			}
			afterState = fmt.Sprintf("returned;editing:%d", sub.Revision+1)
		}
		out = sub
		out.Status = status
		out.Review = &review
		if err = workflowEvent(ctx, tx, u, a, publication.DecideReviewAction, "submission", id, review.FrozenDigest, review.FrozenDigest, "pending", afterState, input.Note, now); err != nil {
			return err
		}
		return workflowJSONRemember(s, ctx, tx, u, a, publication.DecideReviewAction, id, input, out)
	})
	return out, err
}
