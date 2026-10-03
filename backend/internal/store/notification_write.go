package store

import (
	"context"
	"database/sql"
	"github.com/yyl1212/math_master/backend/internal/auth"
	"github.com/yyl1212/math_master/backend/internal/correction"
	"github.com/yyl1212/math_master/backend/internal/notification"
	"github.com/yyl1212/math_master/backend/internal/question"
)

func notificationAppend(ctx context.Context, tx *sql.Tx, owner string, in notification.Source) error {
	if !question.ValidID(owner) || !question.ValidID(in.CaseID) || !correction.ValidEvidence(in.Evidence, true) || in.DedupKey == "" || len(in.DedupKey) > 512 || in.ResultID != nil && !question.ValidID(*in.ResultID) {
		return auth.ErrInvalidInput
	}
	switch in.Type {
	case notification.Checking, notification.Corrected, notification.Retake, notification.ReviewMaterial, notification.PathUnavailable:
	default:
		return auth.ErrInvalidInput
	}
	_, e := tx.ExecContext(ctx, `INSERT INTO notifications(owner_user_id,dedup_key,type,evidence_kind,evidence_id,case_id,result_id) VALUES($1,$2,$3,$4,$5,$6,$7) ON CONFLICT(dedup_key) DO NOTHING`, owner, in.DedupKey, in.Type, in.Evidence.Kind, in.Evidence.ID, in.CaseID, in.ResultID)
	return e
}
