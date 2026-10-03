package store

import (
	"context"
	"database/sql"
	"github.com/yyl1212/math_master/backend/internal/auth"
	"github.com/yyl1212/math_master/backend/internal/publication"
	"github.com/yyl1212/math_master/backend/internal/question"
	"time"
)

var _ question.Repository = (*Store)(nil)

func questionWithdrawalTarget(ctx context.Context, tx *sql.Tx, target question.WithdrawalTarget) (string, error) {
	if err := question.ValidateWithdrawalTarget(target); err != nil {
		return "", err
	}
	table := "question_templates"
	where := ""
	switch target.Kind {
	case "instance":
		table = "question_instances"
		where = " AND sealed"
	case "blueprint":
		table = "question_blueprints"
		where = " AND sealed"
	}
	var sha string
	err := tx.QueryRowContext(ctx, `SELECT sha256 FROM `+table+` WHERE id=$1 AND version=$2`+where, target.ID, target.Version).Scan(&sha)
	return sha, workflowRowError(err)
}
func questionWithdrawalBase(ctx context.Context, tx *sql.Tx, khead, qhead *string) (question.Candidate, error) {
	var c question.Candidate
	var err error
	if qhead == nil {
		c, err = questionEmptyCandidate(ctx, tx)
	} else {
		c, err = questionLoadCandidate(ctx, tx, *qhead)
	}
	if err != nil {
		return c, err
	}
	c.Manifest.BaseKnowledgeHead = khead
	c.Manifest.BaseQuestionHead = qhead
	for n := range c.Manifest.Members {
		c.Manifest.Members[n].Evidence.InheritedFrom = qhead
	}
	return c, nil
}
func (s *Store) PreviewQuestionWithdrawal(ctx context.Context, a question.Access, input question.WithdrawalPreviewInput, q question.ListQuery) (question.WithdrawalPreview, error) {
	out := question.WithdrawalPreview{Changes: question.Page[question.Change]{Items: []question.Change{}}}
	if err := question.ValidateWithdrawalTarget(input.Target); err != nil {
		return out, err
	}
	if q.Scope != "" || q.Status != "" {
		return out, auth.ErrInvalidInput
	}
	pq, err := publication.ValidateList(publication.ListQuery{Limit: q.Limit, Offset: q.Offset})
	if err != nil {
		return out, err
	}
	err = s.questionReadTx(ctx, a, question.PreviewWithdrawalAction, func(ctx context.Context, tx *sql.Tx, _ auth.User) error {
		sha, err := questionWithdrawalTarget(ctx, tx, input.Target)
		if err != nil {
			return err
		}
		khead, err := workflowHead(ctx, tx)
		if err != nil {
			return err
		}
		qhead, err := questionHead(ctx, tx)
		if err != nil {
			return err
		}
		base, err := questionWithdrawalBase(ctx, tx, khead, qhead)
		if err != nil {
			return err
		}
		c, err := question.WithdrawCandidate(ctx, base, input.Target)
		if err != nil {
			return err
		}
		if err = questionValidateCandidate(ctx, tx, c, false); err != nil {
			return err
		}
		_, manifestSHA, err := question.CanonicalManifest(c.Manifest)
		if err != nil {
			return err
		}
		_, digest, err := questionCanonical("question-withdrawal-impact-v1", struct {
			Target      question.WithdrawalTarget `json:"target"`
			SHA         string                    `json:"targetSha"`
			ManifestSHA string                    `json:"manifestSha"`
			Changes     []question.Change         `json:"changes"`
		}{input.Target, sha, manifestSHA, c.Changes})
		if err != nil {
			return err
		}
		out.CurrentKnowledgeHead = khead
		out.CurrentQuestionHead = qhead
		out.Target = input.Target
		out.Diff = c.Diff
		out.ImpactDigest = digest
		for _, change := range c.Changes {
			switch change.Kind {
			case "template":
				out.AffectedTemplates++
			case "instance":
				out.AffectedInstances++
			case "blueprint":
				out.AffectedBlueprints++
			}
		}
		out.Changes.Total = len(c.Changes)
		out.Changes.Offset = pq.Offset
		out.Changes.Limit = pq.Limit
		start := pq.Offset
		if start > len(c.Changes) {
			start = len(c.Changes)
		}
		end := start + pq.Limit
		if end > len(c.Changes) {
			end = len(c.Changes)
		}
		out.Changes.Items = c.Changes[start:end]
		return questionFitPage(&out.Changes, func() any { return out })
	})
	return out, err
}
func (s *Store) WithdrawQuestionVersion(ctx context.Context, a question.Access, input question.WithdrawalInput) (question.WithdrawalResult, error) {
	var out question.WithdrawalResult
	if err := question.ValidateWithdrawalTarget(input.Target); err != nil {
		return out, err
	}
	if !validWorkflowHead(input.ExpectedKnowledgeHead) || !validWorkflowHead(input.ExpectedQuestionHead) || !publication.ValidNote(input.Reason) {
		return out, auth.ErrInvalidInput
	}
	err := s.questionTx(ctx, a, question.WithdrawVersionAction, nil, func(ctx context.Context, tx *sql.Tx, u auth.User, now time.Time) error {
		prior, found, err := questionJSONReplay[question.WithdrawalResult](s, ctx, tx, u, a, question.WithdrawVersionAction, "", input)
		if err != nil {
			return err
		}
		if found {
			out = prior
			return nil
		}
		khead, err := workflowHead(ctx, tx)
		if err != nil {
			return err
		}
		qhead, err := questionHead(ctx, tx)
		if err != nil {
			return err
		}
		if !sameWorkflowHead(khead, input.ExpectedKnowledgeHead) || !sameWorkflowHead(qhead, input.ExpectedQuestionHead) {
			return question.ErrPublicationStale
		}
		sha, err := questionWithdrawalTarget(ctx, tx, input.Target)
		if err != nil {
			return err
		}
		var exists bool
		if err = tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM question_withdrawals WHERE kind=$1 AND target_id=$2 AND target_version=$3)`, input.Target.Kind, input.Target.ID, input.Target.Version).Scan(&exists); err != nil {
			return err
		}
		if exists {
			return question.ErrImmutableConflict
		}
		base, err := questionWithdrawalBase(ctx, tx, khead, qhead)
		if err != nil {
			return err
		}
		c, err := question.WithdrawCandidate(ctx, base, input.Target)
		if err != nil {
			return err
		}
		if err = questionValidateCandidate(ctx, tx, c, false); err != nil {
			return err
		}
		withdrawalID, err := workflowID()
		if err != nil {
			return err
		}
		if _, err = tx.ExecContext(ctx, `INSERT INTO question_withdrawals(id,kind,target_id,target_version,sha256,actor_user_id,reason,request_id,created_at) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9)`, withdrawalID, input.Target.Kind, input.Target.ID, input.Target.Version, sha, u.ID, input.Reason, a.RequestID, now); err != nil {
			return err
		}
		if err = correctionEnqueueWithdrawal(ctx, tx, "question", withdrawalID, now); err != nil {
			return err
		}
		published, err := questionInsertPublication(ctx, tx, u, now, c, "published")
		if err != nil {
			return err
		}
		if _, err = tx.ExecContext(ctx, `INSERT INTO question_heads VALUES(true,$1) ON CONFLICT(singleton) DO UPDATE SET publication_id=EXCLUDED.publication_id`, published.ID); err != nil {
			return err
		}
		out = question.WithdrawalResult{EventID: withdrawalID, PreviousKnowledgeHead: khead, PreviousQuestionHead: qhead, Publication: published}
		if err = questionEvent(ctx, tx, u, a, question.WithdrawVersionAction, input.Target.Kind, input.Target.ID, "", published.ManifestSHA, "available", "withdrawn", input.Reason, now); err != nil {
			return err
		}
		return questionJSONRemember(s, ctx, tx, u, a, question.WithdrawVersionAction, "", input, out)
	})
	return out, err
}
