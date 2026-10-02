package store

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"github.com/yyl1212/math_master/backend/internal/auth"
	"github.com/yyl1212/math_master/backend/internal/content"
	"github.com/yyl1212/math_master/backend/internal/learning"
	"github.com/yyl1212/math_master/backend/internal/question"
	"time"
)

func (s *Store) ReadLearningAsset(ctx context.Context, a question.Access, attemptID, sha string) ([]byte, error) {
	if !question.ValidID(attemptID) || !question.ValidSHA(sha) {
		return nil, auth.ErrInvalidInput
	}
	var out []byte
	e := s.learningTx(ctx, a, learning.ReadAssetAction, func(ctx context.Context, tx *sql.Tx, u auth.User, now time.Time) error {
		var raw []byte
		e := tx.QueryRowContext(ctx, `SELECT seal_bytes FROM practice_attempts WHERE id=$1 AND owner_user_id=$2 UNION ALL SELECT seal_bytes FROM assessment_attempts WHERE id=$1 AND owner_user_id=$2 AND sealed`, attemptID, u.ID).Scan(&raw)
		if e != nil {
			return workflowRowError(e)
		}
		seal, e := learningDecodeSeal(raw)
		if e != nil {
			return auth.ErrUnavailable
		}
		allowed := false
		for _, item := range seal.Items {
			referenced := false
			var ref question.AssetRef
			for _, a := range item.Assets {
				if a.SHA256 == sha {
					referenced = true
					ref = a
					break
				}
			}
			if !referenced {
				continue
			}
			rs, e := learningEvidenceRestrictions(ctx, tx, learningItemDependencies(seal, item))
			if e != nil {
				return e
			}
			if len(rs) > 0 {
				continue
			}
			var approved bool
			if e = tx.QueryRowContext(ctx, `SELECT learning_content_approved($1,'asset',$2,1,$3) AND EXISTS(SELECT 1 FROM question_publications WHERE id=$4 AND sealed AND status='published')`, seal.KnowledgePublicationID, ref.ID, sha, seal.QuestionPublicationID).Scan(&approved); e != nil {
				return e
			}
			if approved {
				allowed = true
				break
			}
		}
		if !allowed {
			return auth.ErrNotFound
		}
		e = tx.QueryRowContext(ctx, `SELECT bytes FROM assets WHERE sha256=$1 AND octet_length(bytes)<=1048576`, sha).Scan(&out)
		if errors.Is(e, sql.ErrNoRows) {
			return auth.ErrNotFound
		}
		if e != nil {
			return e
		}
		sum := sha256.Sum256(out)
		if hex.EncodeToString(sum[:]) != sha || content.ValidateSVG(out) != nil {
			return auth.ErrUnavailable
		}
		return nil
	})
	if e != nil {
		return nil, e
	}
	return out, nil
}
