package store

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/yyl1212/math_master/backend/internal/auth"
	"github.com/yyl1212/math_master/backend/internal/knowledgeadmin"
	"github.com/yyl1212/math_master/backend/internal/notesharing"
	"time"
)

func shareID() (string, error) {
	var b [16]byte
	if _, e := rand.Read(b[:]); e != nil {
		return "", auth.ErrUnavailable
	}
	b[6] = (b[6] & 15) | 64
	b[8] = (b[8] & 63) | 128
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[:4], b[4:6], b[6:8], b[8:10], b[10:]), nil
}
func sharingConfigured(ctx context.Context, tx *sql.Tx) error {
	if e := managedStudyMode(ctx, tx); e != nil {
		return e
	}
	return noteSharingSchemaIntegrity(ctx, tx)
}
func noteShareWriteGuard(ctx context.Context, tx *sql.Tx, owner, kid string) error {
	installed, e := noteSharingInstalled(ctx, tx)
	if e != nil {
		return e
	}
	if !installed {
		return nil
	}
	if e = noteSharingSchemaIntegrity(ctx, tx); e != nil {
		return e
	}
	var status string
	e = tx.QueryRowContext(ctx, "SELECT status FROM note_share_heads WHERE owner_user_id=$1 AND knowledge_id=$2", owner, kid).Scan(&status)
	if errors.Is(e, sql.ErrNoRows) {
		return nil
	}
	if e != nil {
		return notesharing.ErrNotReady
	}
	if status == notesharing.Pending || status == notesharing.Shared {
		return notesharing.ErrLocked
	}
	return nil
}
func readShareHead(ctx context.Context, tx *sql.Tx, owner, kid string, lock bool) (h notesharing.Head, e error) {
	h = notesharing.Head{ActorID: owner, KnowledgeID: kid, Status: notesharing.Unshared}
	query := `SELECT status,sequence,current_submission_id::text,approved_at FROM note_share_heads WHERE owner_user_id=$1 AND knowledge_id=$2`
	if lock {
		query += " FOR UPDATE"
	}
	e = tx.QueryRowContext(ctx, query, owner, kid).Scan(&h.Status, &h.Sequence, &h.SubmissionID, &h.ApprovedAt)
	if errors.Is(e, sql.ErrNoRows) {
		return h, nil
	}
	if e != nil {
		return h, notesharing.ErrNotReady
	}
	e = tx.QueryRowContext(ctx, `SELECT coalesce((SELECT reason FROM note_share_events WHERE owner_user_id=$1 AND knowledge_id=$2 AND action IN ('return','hide') ORDER BY sequence DESC LIMIT 1),'')`, owner, kid).Scan(&h.LastDecisionReason)
	return
}
func shareTarget(ctx context.Context, tx *sql.Tx, owner, kid string, lock bool) (k knowledgeadmin.Knowledge, n knowledgeadmin.ManagedNote, e error) {
	k, e = managedStudyCurrent(ctx, tx, kid, lock)
	if e != nil {
		return
	}
	if _, e = readManagedRecord(ctx, tx, owner, kid, lock); e != nil {
		return
	}
	n, e = readManagedNote(ctx, tx, owner, kid, lock)
	if e != nil {
		return
	}
	if n.Revision == 0 {
		e = notesharing.ErrNotFound
	}
	return
}
func shareReplay(ctx context.Context, tx *sql.Tx, actor string, a knowledgeadmin.Access, action, resource string, input any) (notesharing.Receipt, bool, error) {
	var out notesharing.Receipt
	var digest, storedResource string
	var raw []byte
	e := tx.QueryRowContext(ctx, `SELECT input_sha256,resource,receipt FROM note_share_receipts WHERE actor_user_id=$1 AND action=$2 AND key=$3`, actor, action, a.IdempotencyKey).Scan(&digest, &storedResource, &raw)
	if errors.Is(e, sql.ErrNoRows) {
		return out, false, nil
	}
	if e != nil {
		return out, false, notesharing.ErrNotReady
	}
	if digest != knowledgeFingerprint(input) || resource != storedResource {
		return out, false, notesharing.ErrIdempotency
	}
	if e = json.Unmarshal(raw, &out); e != nil {
		return out, false, notesharing.ErrNotReady
	}
	return out, true, nil
}
func writeShareDecision(ctx context.Context, tx *sql.Tx, actor string, a knowledgeadmin.Access, owner, kid, action, resource, reason string, input any, h notesharing.Head, now time.Time) (notesharing.Receipt, error) {
	if h.SubmissionID == nil {
		return notesharing.Receipt{}, notesharing.ErrStale
	}
	_, e := tx.ExecContext(ctx, `UPDATE note_share_heads SET status=$3,sequence=$4,current_submission_id=$5,approved_at=$6 WHERE owner_user_id=$1 AND knowledge_id=$2`, owner, kid, h.Status, h.Sequence, *h.SubmissionID, h.ApprovedAt)
	if e != nil {
		return notesharing.Receipt{}, e
	}
	id, e := shareID()
	if e != nil {
		return notesharing.Receipt{}, e
	}
	_, e = tx.ExecContext(ctx, `INSERT INTO note_share_events(event_id,owner_user_id,knowledge_id,submission_id,actor_user_id,action,reason,sequence,recorded_at) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9)`, id, owner, kid, *h.SubmissionID, actor, action, reason, h.Sequence, now)
	if e != nil {
		return notesharing.Receipt{}, e
	}
	out := notesharing.Receipt{SubmissionID: *h.SubmissionID, KnowledgeID: kid, Action: action, Status: h.Status, Sequence: h.Sequence}
	_, e = tx.ExecContext(ctx, `INSERT INTO note_share_receipts(actor_user_id,action,key,resource,input_sha256,receipt) VALUES($1,$2,$3,$4,$5,$6)`, actor, action, a.IdempotencyKey, resource, knowledgeFingerprint(input), knowledgeJSON(out))
	return out, e
}
