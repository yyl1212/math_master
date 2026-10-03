package store

import (
	"context"
	"database/sql"
	"github.com/yyl1212/math_master/backend/internal/correction"
	"github.com/yyl1212/math_master/backend/internal/learning"
	"github.com/yyl1212/math_master/backend/internal/question"
	"time"
)

func correctionGrantEvidence(ctx context.Context, tx *sql.Tx, owner string, k question.Identity, v learning.QualificationView, now time.Time) error {
	if !correctionEnabled(ctx) || v.CorrectionID == nil || v.Knowledge != k {
		return correction.ErrSourceStale
	}
	current, e := correctionPassingEvidence(ctx, tx, owner, k)
	if e != nil {
		return e
	}
	if current == nil || current.CorrectionID == nil || *current.CorrectionID != *v.CorrectionID || current.EvidenceAttemptID != v.EvidenceAttemptID {
		return correction.ErrSourceStale
	}
	raw, h, e := correction.Canonical("correction-command-v1", v)
	if e != nil {
		return e
	}
	_, e = tx.ExecContext(ctx, `INSERT INTO correction_events(id,subject_kind,subject_id,case_id,result_id,owner_user_id,kind,sequence,body,body_bytes,body_digest,recorded_at) SELECT gen_random_uuid(),'result',r.id,r.case_id,r.id,$2,'qualification_granted',(SELECT coalesce(max(sequence),0)+1 FROM correction_events WHERE subject_kind='result' AND subject_id=r.id),$3::jsonb,$4,$5,$6 FROM correction_results r WHERE r.id=$1 AND r.owner_user_id=$2 AND r.sealed ON CONFLICT DO NOTHING`, *v.CorrectionID, owner, string(raw), raw, h, now)
	return e
}
