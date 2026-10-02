package store

import (
	"context"
	"database/sql"
	"errors"
	"github.com/yyl1212/math_master/backend/internal/auth"
	"github.com/yyl1212/math_master/backend/internal/learning"
	"github.com/yyl1212/math_master/backend/internal/question"
	"time"
)

func (s *Store) StartLearning(ctx context.Context, a question.Access, id string, in learning.StartInput) (learning.KnowledgeState, error) {
	return s.learningAction(ctx, a, id, in, learning.StartKnowledgeAction)
}
func (s *Store) CompleteLearning(ctx context.Context, a question.Access, id string, in learning.CompleteInput) (learning.KnowledgeState, error) {
	return s.learningAction(ctx, a, id, in, learning.CompleteKnowledgeAction)
}
func (s *Store) learningAction(ctx context.Context, a question.Access, id string, in learning.StartInput, action learning.Action) (learning.KnowledgeState, error) {
	var out learning.KnowledgeState
	if !question.ValidMathID(id) || in.Knowledge.ID != id || in.Knowledge.Version < 1 || !question.ValidSHA(in.Knowledge.SHA256) || in.ExpectedKnowledgeHead == "" {
		return out, auth.ErrInvalidInput
	}
	e := s.learningTx(ctx, a, action, func(ctx context.Context, tx *sql.Tx, u auth.User, now time.Time) error {
		digest := workflowRequestSHA(id, in)
		_, found, e := learningReplay(ctx, tx, u.ID, string(action), id, a.IdempotencyKey, digest)
		if e != nil {
			return e
		}
		k, head, e := learningKnowledgeIdentity(ctx, tx, id)
		if e != nil {
			return e
		}
		if !found && (k != in.Knowledge || head != in.ExpectedKnowledgeHead) {
			return learning.ErrVersionStale
		}
		if found {
			out, e = learningKnowledgeState(ctx, tx, u.ID, k)
			return e
		}
		basis, e := learningLectureBasis(ctx, tx, k)
		if e != nil {
			return e
		}
		var startedID string
		e = tx.QueryRowContext(ctx, `SELECT started_event_id::text FROM learning_records WHERE owner_user_id=$1 AND knowledge_id=$2 AND knowledge_version=$3 AND knowledge_sha256=$4 FOR UPDATE`, u.ID, k.ID, k.Version, k.SHA256).Scan(&startedID)
		exists := e == nil
		if e != nil && !errors.Is(e, sql.ErrNoRows) {
			return e
		}
		if action == learning.StartKnowledgeAction {
			state, e := learningKnowledgeState(ctx, tx, u.ID, k)
			if e != nil {
				return e
			}
			if !state.CanEnter {
				return learning.ErrPrerequisitesUnmet
			}
			if !exists {
				startedID, e = learningAddEvent(ctx, tx, u.ID, basis, "started", now)
				if e != nil {
					return e
				}
				if _, e = tx.ExecContext(ctx, `INSERT INTO learning_records(owner_user_id,knowledge_id,knowledge_version,knowledge_sha256,started_event_id,started_at) VALUES($1,$2,$3,$4,$5,$6)`, u.ID, k.ID, k.Version, k.SHA256, startedID, now); e != nil {
					return e
				}
			}
			if _, e = learningUnlock(ctx, tx, u.ID, k, "learning-event", startedID, now); e != nil {
				return e
			}
		} else {
			if !exists {
				return learning.ErrStateConflict
			}
			eventID, e := learningAddEvent(ctx, tx, u.ID, basis, "completed", now)
			if e != nil {
				return e
			}
			if _, e = tx.ExecContext(ctx, `UPDATE learning_records SET completed_event_id=$5,completed_at=$6 WHERE owner_user_id=$1 AND knowledge_id=$2 AND knowledge_version=$3 AND knowledge_sha256=$4 AND completed_event_id IS NULL`, u.ID, k.ID, k.Version, k.SHA256, eventID, now); e != nil {
				return e
			}
			if _, e = learningGrantEvidence(ctx, tx, u.ID, k, now); e != nil {
				return e
			}
			if _, e = learningUnlockSuccessors(ctx, tx, u.ID, k, "learning-event", eventID, now); e != nil {
				return e
			}
		}
		status := 200
		if action == learning.StartKnowledgeAction {
			status = 201
		}
		if e = learningRemember(ctx, tx, u.ID, string(action), id, a.IdempotencyKey, digest, learning.Receipt{ResourceKind: "knowledge", ResourceID: id, Status: status}); e != nil {
			return e
		}
		out, e = learningKnowledgeState(ctx, tx, u.ID, k)
		return e
	})
	return out, e
}
func learningAddEvent(ctx context.Context, tx *sql.Tx, actor string, basis learning.EventSeal, kind string, now time.Time) (string, error) {
	var existing string
	e := tx.QueryRowContext(ctx, `SELECT id::text FROM learning_events WHERE owner_user_id=$1 AND knowledge_id=$2 AND knowledge_version=$3 AND knowledge_sha256=$4 AND kind=$5 AND material_sha256=learning_material_hash(jsonb_build_object('body',$6::jsonb))`, actor, basis.Knowledge.ID, basis.Knowledge.Version, basis.Knowledge.SHA256, kind, body(basis)).Scan(&existing)
	if e == nil {
		return existing, nil
	}
	if !errors.Is(e, sql.ErrNoRows) {
		return "", e
	}
	basis.ID, e = workflowID()
	if e != nil {
		return "", e
	}
	basis.ActorID = actor
	basis.Kind = kind
	basis.RecordedAt = now.UTC()
	raw, sha, e := learning.CanonicalLearningEvent(basis)
	if e != nil {
		return "", e
	}
	_, e = tx.ExecContext(ctx, `INSERT INTO learning_events(id,owner_user_id,knowledge_id,knowledge_version,knowledge_sha256,knowledge_publication_id,kind,seal,seal_bytes,seal_sha256,recorded_at) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)`, basis.ID, actor, basis.Knowledge.ID, basis.Knowledge.Version, basis.Knowledge.SHA256, basis.KnowledgePublicationID, kind, string(raw), raw, sha, now)
	if e != nil {
		return "", e
	}
	return basis.ID, learningSaveDependencies(ctx, tx, actor, "learning-event", basis.ID, raw, true)
}
