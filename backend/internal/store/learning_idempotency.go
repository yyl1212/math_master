package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"github.com/yyl1212/math_master/backend/internal/auth"
	"github.com/yyl1212/math_master/backend/internal/learning"
	"github.com/yyl1212/math_master/backend/internal/question"
	"strings"
)

func learningKey(actor, action, target, key, digest string) error {
	if !question.ValidID(actor) || !question.ValidID(key) || !question.ValidSHA(digest) || !learning.IsIdempotent(learning.Action(action)) || len(target) == 0 || len(target) > 80 || strings.ContainsRune(target, 0) {
		return auth.ErrInvalidInput
	}
	return nil
}
func learningReceipt(action string, r learning.Receipt) error {
	status, kind := 200, "assessment"
	switch learning.Action(action) {
	case learning.StartKnowledgeAction:
		status, kind = 201, "knowledge"
	case learning.CompleteKnowledgeAction:
		kind = "knowledge"
	case learning.EnrollPathAction:
		status, kind = 201, "enrollment"
	case learning.CreatePracticeAction:
		status, kind = 201, "practice"
	case learning.AnswerPracticeAction, learning.RevealPracticeAction, learning.AbandonPracticeAction:
		kind = "practice"
	case learning.CreateAssessmentAction:
		status = 201
	}
	if r.Status != status || r.ResourceKind != kind {
		return auth.ErrInvalidInput
	}
	if kind == "knowledge" {
		if !question.ValidMathID(r.ResourceID) {
			return auth.ErrInvalidInput
		}
	} else if !question.ValidID(r.ResourceID) {
		return auth.ErrInvalidInput
	}
	return nil
}

// The receipt identifies the business result, never caches an answer response.
func learningReplay(ctx context.Context, tx *sql.Tx, actor, action, target, key, digest string) (learning.Receipt, bool, error) {
	if err := learningKey(actor, action, target, key, digest); err != nil {
		return learning.Receipt{}, false, err
	}
	var savedTarget, savedDigest string
	var raw []byte
	err := tx.QueryRowContext(ctx, `SELECT target,request_sha256,receipt_bytes FROM learning_idempotency WHERE owner_user_id=$1 AND action=$2 AND key=$3`, actor, action, key).Scan(&savedTarget, &savedDigest, &raw)
	if errors.Is(err, sql.ErrNoRows) {
		return learning.Receipt{}, false, nil
	}
	if err != nil {
		return learning.Receipt{}, false, err
	}
	if savedTarget != target || savedDigest != digest {
		return learning.Receipt{}, false, question.ErrIdempotencyConflict
	}
	var r learning.Receipt
	if len(raw) > 4096 || json.Unmarshal(raw, &r) != nil || learningReceipt(action, r) != nil {
		return learning.Receipt{}, false, auth.ErrUnavailable
	}
	return r, true, nil
}
func learningRemember(ctx context.Context, tx *sql.Tx, actor, action, target, key, digest string, r learning.Receipt) error {
	if err := learningKey(actor, action, target, key, digest); err != nil {
		return err
	}
	if err := learningReceipt(action, r); err != nil {
		return err
	}
	raw, err := json.Marshal(r)
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO learning_idempotency(owner_user_id,action,target,key,request_sha256,receipt,receipt_bytes) VALUES($1,$2,$3,$4,$5,$6,$7)`, actor, action, target, key, digest, string(raw), raw)
	return err
}
