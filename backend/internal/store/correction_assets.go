package store

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"github.com/yyl1212/math_master/backend/internal/assessment"
	"github.com/yyl1212/math_master/backend/internal/auth"
	"github.com/yyl1212/math_master/backend/internal/content"
	"github.com/yyl1212/math_master/backend/internal/correction"
	"github.com/yyl1212/math_master/backend/internal/question"
	"time"
)

// The immutable original-attempt asset authorization deliberately remains
// separate. A corrected image is authorized by this exact owned effective basis.
func (s *Store) ReadOwnCorrectionAsset(ctx context.Context, a question.Access, id, sha string) ([]byte, error) {
	if !question.ValidID(id) || !question.ValidSHA(sha) {
		return nil, auth.ErrInvalidInput
	}
	var out []byte
	e := s.correctionTx(ctx, a, correction.ReadOwnDetailAction, nil, func(ctx context.Context, tx *sql.Tx, u auth.User, _ time.Time) error {
		meta, e := correctionOwnResultMetadata(ctx, tx, u.ID, id)
		if e != nil {
			return e
		}
		if meta.Validity != assessment.Effective || meta.Plan == nil || meta.Status != correction.CorrectedPassed && meta.Status != correction.CorrectedFailed {
			return auth.ErrNotFound
		}
		var size int
		var bound bool
		e = tx.QueryRowContext(ctx, `SELECT octet_length(r.basis_bytes),EXISTS(SELECT 1 FROM jsonb_array_elements(r.basis#>'{body,effectiveItems}') i CROSS JOIN LATERAL jsonb_array_elements(i#>'{body,assets}') a WHERE a->>'sha256'=$3) AND EXISTS(SELECT 1 FROM correction_dependencies d WHERE d.result_id=r.id AND d.role='effective' AND d.kind='asset' AND d.sha256=$3) FROM correction_results r WHERE r.id=$1 AND r.owner_user_id=$2 AND r.sealed`, id, u.ID, sha).Scan(&size, &bound)
		if e != nil {
			return workflowRowError(e)
		}
		if size > correction.MaxResponseBytes {
			return question.ErrLimitExceeded
		}
		if !bound {
			return auth.ErrNotFound
		}
		// Only identities are transferred: no prompts, answers or entire basis blob.
		rows, e := tx.QueryContext(ctx, `SELECT i->'identity',i->'template' FROM correction_results r CROSS JOIN LATERAL jsonb_array_elements((r.basis#>'{body,originalItems}')||(r.basis#>'{body,effectiveItems}')) i WHERE r.id=$1 AND r.owner_user_id=$2 AND r.sealed LIMIT 11`, id, u.ID)
		if e != nil {
			return e
		}
		refs := []correctionExposureRef{}
		n := 0
		for rows.Next() {
			var identity, template []byte
			var instance question.Identity
			var source *question.Identity
			if e = rows.Scan(&identity, &template); e != nil {
				rows.Close()
				return e
			}
			if json.Unmarshal(identity, &instance) != nil || json.Unmarshal(template, &source) != nil {
				rows.Close()
				return auth.ErrUnavailable
			}
			n++
			refs = append(refs, correctionExposureRef{Kind: "instance", Identity: instance})
			if source != nil {
				refs = append(refs, correctionExposureRef{Kind: "template", Identity: *source})
			}
		}
		e = rows.Err()
		rows.Close()
		if e != nil {
			return e
		}
		if n > 10 {
			return auth.ErrUnavailable
		}
		e = tx.QueryRowContext(ctx, `SELECT bytes FROM assets WHERE sha256=$1 AND octet_length(bytes)<=1048576`, sha).Scan(&out)
		if e != nil {
			return workflowRowError(e)
		}
		sum := sha256.Sum256(out)
		if hex.EncodeToString(sum[:]) != sha || content.ValidateSVG(out) != nil {
			return auth.ErrUnavailable
		}
		return correctionDetailExposure(ctx, tx, u.ID, meta.CaseID, refs)
	})
	if e != nil {
		return nil, e
	}
	return out, nil
}
