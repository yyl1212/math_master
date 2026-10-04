package store

import (
	"context"
	"database/sql"
	"github.com/yyl1212/math_master/backend/internal/auth"
	"github.com/yyl1212/math_master/backend/internal/content"
	"github.com/yyl1212/math_master/backend/internal/publication"
	"time"
)

var _ publication.Repository = (*Store)(nil)

func workflowWithdrawalTarget(ctx context.Context, tx *sql.Tx, target publication.WithdrawalTarget) (string, error) {
	if err := publication.ValidateWithdrawalTarget(target); err != nil {
		return "", err
	}
	var sha string
	var err error
	switch target.Kind {
	case "knowledge":
		err = tx.QueryRowContext(ctx, `SELECT sha256 FROM knowledge_versions WHERE id=$1 AND version=$2`, target.ID, target.Version).Scan(&sha)
	case "unit":
		err = tx.QueryRowContext(ctx, `SELECT sha256 FROM unit_versions WHERE id=$1 AND version=$2`, target.ID, target.Version).Scan(&sha)
	case "path":
		err = tx.QueryRowContext(ctx, `SELECT sha256 FROM path_versions WHERE id=$1 AND version=$2`, target.ID, target.Version).Scan(&sha)
	case "asset":
		err = tx.QueryRowContext(ctx, `SELECT encode(sha256(bytes),'hex') FROM assets WHERE sha256=$1`, target.SHA256).Scan(&sha)
		if err == nil && sha != target.SHA256 {
			return "", publication.ErrContentInvalid
		}
	}
	return sha, workflowRowError(err)
}
func (s *Store) workflowWithdrawalBase(ctx context.Context, tx *sql.Tx, head *string) (publication.Candidate, error) {
	if head != nil {
		return s.loadWorkflowCandidate(ctx, tx, head)
	}
	var out publication.Candidate
	var version int
	if err := tx.QueryRowContext(ctx, `SELECT COALESCE(max(version),0) FROM catalogue_versions`).Scan(&version); err != nil {
		return out, err
	}
	if version == 0 {
		return out, publication.ErrContentNotConfigured
	}
	_, sha, err := workflowCatalogue(ctx, tx, version)
	if err != nil {
		return out, err
	}
	out.Manifest = publication.Manifest{CatalogueVersion: version, CatalogueSHA256: sha, Members: []publication.ManifestMember{}, Bindings: []content.AssetBinding{}}
	out.Snapshot = content.Snapshot{CatalogueVersion: version, Knowledge: []content.Knowledge{}, Units: []content.Unit{}, Paths: []content.Path{}, Assets: []content.Asset{}, Bindings: []content.AssetBinding{}}
	return out, nil
}
func (s *Store) PreviewWithdrawal(ctx context.Context, a publication.Access, input publication.WithdrawalPreviewInput) (publication.WithdrawalPreview, error) {
	var out publication.WithdrawalPreview
	if err := publication.ValidateWithdrawalTarget(input.Target); err != nil {
		return out, err
	}
	err := s.workflowReadTx(ctx, a, publication.PreviewWithdrawalAction, func(ctx context.Context, tx *sql.Tx, u auth.User) error {
		if _, err := workflowWithdrawalTarget(ctx, tx, input.Target); err != nil {
			return err
		}
		head, err := workflowHead(ctx, tx)
		if err != nil {
			return err
		}
		base, err := s.workflowWithdrawalBase(ctx, tx, head)
		if err != nil {
			return err
		}
		candidate, err := publication.WithdrawCandidate(base, input.Target)
		if err != nil {
			return err
		}
		if err = workflowValidateCandidate(ctx, tx, candidate, false); err != nil {
			return err
		}
		out = publication.WithdrawalPreview{CurrentHead: head, Target: input.Target, Diff: candidate.Diff}
		return workflowResponseSize(out)
	})
	return out, err
}
func (s *Store) WithdrawVersion(ctx context.Context, a publication.Access, input publication.WithdrawalInput) (publication.WithdrawalResult, error) {
	var out publication.WithdrawalResult
	if err := publication.ValidateWithdrawalTarget(input.Target); err != nil {
		return out, err
	}
	if !validWorkflowHead(input.ExpectedHead) || !publication.ValidNote(input.Reason) {
		return out, auth.ErrInvalidInput
	}
	err := s.workflowTx(ctx, a, publication.WithdrawVersionAction, nil, func(ctx context.Context, tx *sql.Tx, u auth.User, now time.Time) error {
		prior, found, err := workflowJSONReplay[publication.WithdrawalResult](s, ctx, tx, u, a, publication.WithdrawVersionAction, "", input)
		if err != nil {
			return err
		}
		if found {
			out = prior
			return nil
		}
		head, err := workflowHead(ctx, tx)
		if err != nil {
			return err
		}
		if !sameWorkflowHead(head, input.ExpectedHead) {
			return publication.ErrPublicationStale
		}
		sha, err := workflowWithdrawalTarget(ctx, tx, input.Target)
		if err != nil {
			return err
		}
		var exists bool
		if err = tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM content_withdrawals WHERE (kind='asset' AND $1='asset' AND sha256=$4) OR (kind=$1 AND kind<>'asset' AND target_id=$2 AND target_version=$3))`, input.Target.Kind, input.Target.ID, input.Target.Version, sha).Scan(&exists); err != nil {
			return err
		}
		if exists {
			return publication.ErrImmutableConflict
		}
		base, err := s.workflowWithdrawalBase(ctx, tx, head)
		if err != nil {
			return err
		}
		candidate, err := publication.WithdrawCandidate(base, input.Target)
		if err != nil {
			return err
		}
		eventID, err := workflowID()
		if err != nil {
			return err
		}
		var targetID any
		var targetVersion any
		if input.Target.Kind != "asset" {
			targetID = input.Target.ID
			targetVersion = input.Target.Version
		}
		if _, err = tx.ExecContext(ctx, `INSERT INTO content_withdrawals(id,kind,target_id,target_version,sha256,actor_user_id,reason,request_id,created_at) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9)`, eventID, input.Target.Kind, targetID, targetVersion, sha, u.ID, input.Reason, a.RequestID, now); err != nil {
			return err
		}
		if err = correctionEnqueueWithdrawal(ctx, tx, "content", eventID, now); err != nil {
			return err
		}
		// Withdrawal and its correction outbox share this transaction and content lock.
		if err = workflowValidateCandidate(ctx, tx, candidate, false); err != nil {
			return err
		}
		view, err := s.insertWorkflowPublication(ctx, tx, u, now, candidate, "published")
		if err != nil {
			return err
		}
		if _, err = tx.ExecContext(ctx, `INSERT INTO publication_heads VALUES(true,$1) ON CONFLICT(singleton) DO UPDATE SET snapshot_id=EXCLUDED.snapshot_id`, view.ID); err != nil {
			return err
		}
		out = publication.WithdrawalResult{EventID: eventID, PreviousHead: head, Publication: view}
		if err = workflowResponseSize(out); err != nil {
			return err
		}
		beforeSHA := ""
		beforeState := ""
		if head != nil {
			beforeSHA, _ = publication.ManifestDigest(base.Manifest)
			beforeState = "published:" + *head
		}
		if err = workflowEvent(ctx, tx, u, a, publication.WithdrawVersionAction, "withdrawal", eventID, beforeSHA, view.ManifestSHA, beforeState, "published:"+view.ID, input.Reason, now); err != nil {
			return err
		}
		return workflowJSONRemember(s, ctx, tx, u, a, publication.WithdrawVersionAction, "", input, out)
	})
	return out, err
}
