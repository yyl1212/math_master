package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"github.com/yyl1212/math_master/backend/internal/studyreview"
	"time"
)

func reviewReplay(ctx context.Context, tx *sql.Tx, owner string, a studyreview.Access, action, resource string, input any) (r studyreview.CommandReceipt, found bool, e error) {
	var digest, stored string
	var raw []byte
	e = tx.QueryRowContext(ctx, `SELECT input_sha256,resource,receipt FROM study_review_receipts WHERE actor_user_id=$1 AND action=$2 AND key=$3`, owner, action, a.IdempotencyKey).Scan(&digest, &stored, &raw)
	if errors.Is(e, sql.ErrNoRows) {
		return r, false, nil
	}
	if e != nil {
		return r, false, e
	}
	if digest != knowledgeFingerprint(input) || resource != stored {
		return r, false, studyreview.ErrIdempotency
	}
	if e = json.Unmarshal(raw, &r); e != nil || r.ActorID != owner || r.Action != action {
		return r, false, studyreview.ErrNotConfigured
	}
	return r, true, nil
}
func reviewEventReceipt(ctx context.Context, tx *sql.Tx, a studyreview.Access, s studyreview.Session, action, resource string, input any, now time.Time) (r studyreview.CommandReceipt, e error) {
	eventAction := map[string]string{"start": "opened", "draft": "draft-saved", "compare": "compared", "grade": "graded", "abandon": "abandoned"}[action]
	var grade any
	if action == "grade" {
		grade = knowledgeJSON(s.Grade)
	}
	if _, e = tx.ExecContext(ctx, `INSERT INTO study_review_events(session_id,owner_user_id,knowledge_id,review_id,sequence,action,knowledge_ref,mode,target,grade,recorded_at) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)`, s.SessionID, s.ActorID, s.KnowledgeID, s.ReviewID, s.Sequence, eventAction, knowledgeJSON(s.Knowledge), s.Mode, s.Target, grade, now); e != nil {
		return r, e
	}
	r = studyreview.CommandReceipt{ActorID: s.ActorID, Action: action, SessionID: s.SessionID, KnowledgeID: s.KnowledgeID, ReviewID: s.ReviewID, Knowledge: s.Knowledge, Sequence: s.Sequence, DraftRevision: s.DraftRevision, Status: s.Status, AppliedAt: now}
	if action == "grade" {
		r.Grade = s.Grade
	}
	_, e = tx.ExecContext(ctx, `INSERT INTO study_review_receipts(actor_user_id,action,key,resource,input_sha256,session_id,knowledge_id,receipt,created_at) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9)`, s.ActorID, action, a.IdempotencyKey, resource, knowledgeFingerprint(input), s.SessionID, s.KnowledgeID, knowledgeJSON(r), now)
	return r, e
}
