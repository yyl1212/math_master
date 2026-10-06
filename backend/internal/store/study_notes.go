package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"github.com/yyl1212/math_master/backend/internal/auth"
	"github.com/yyl1212/math_master/backend/internal/study"
	"time"
)

func ownedStudyNoteTx(ctx context.Context, tx *sql.Tx, actor, id string, lock bool) (study.NoteView, bool, error) {
	out := study.NoteView{ActorID: actor, KnowledgeID: id}
	query := "SELECT revision,body,knowledge_ref,updated_at FROM study_notes WHERE owner_user_id=$1 AND knowledge_id=$2"
	if lock {
		query += " FOR UPDATE"
	}
	var raw []byte
	e := tx.QueryRowContext(ctx, query, actor, id).Scan(&out.Revision, &out.Body, &raw, &out.UpdatedAt)
	if errors.Is(e, sql.ErrNoRows) {
		return out, false, nil
	}
	if e != nil {
		return out, false, e
	}
	var ref study.KnowledgeRef
	if json.Unmarshal(raw, &ref) != nil || !study.ValidRef(ref) || ref.ID != id {
		return out, false, study.ErrNotConfigured
	}
	out.Knowledge = &ref
	return out, true, nil
}
func studyNoteOwnerTx(ctx context.Context, tx *sql.Tx, actor, id string) (study.StudyRecord, study.KnowledgeRef, error) {
	var raw, ref []byte
	var r study.StudyRecord
	var k study.KnowledgeRef
	e := tx.QueryRowContext(ctx, "SELECT body,last_known_ref FROM study_records WHERE owner_user_id=$1 AND knowledge_id=$2 FOR UPDATE", actor, id).Scan(&raw, &ref)
	if errors.Is(e, sql.ErrNoRows) {
		return r, k, auth.ErrNotFound
	}
	if e != nil {
		return r, k, e
	}
	if json.Unmarshal(raw, &r) != nil || json.Unmarshal(ref, &k) != nil || r.KnowledgeID != id || k.ID != id || !study.ValidRef(k) {
		return r, k, study.ErrNotConfigured
	}
	return r, k, nil
}
func (s *Store) ReadStudyNote(ctx context.Context, a study.Access, id string) (study.NoteView, error) {
	var out study.NoteView
	if !study.ValidKnowledgeID(id) {
		return out, study.ErrInvalid
	}
	e := s.studyReadTx(ctx, a, func(ctx context.Context, tx *sql.Tx, u auth.User) error {
		var found bool
		var e error
		out, found, e = ownedStudyNoteTx(ctx, tx, u.ID, id, false)
		if e != nil || found {
			return e
		}
		var raw []byte
		if e = tx.QueryRowContext(ctx, "SELECT last_known_ref FROM study_records WHERE owner_user_id=$1 AND knowledge_id=$2", u.ID, id).Scan(&raw); errors.Is(e, sql.ErrNoRows) {
			return auth.ErrNotFound
		} else if e != nil {
			return e
		}
		var ref study.KnowledgeRef
		if json.Unmarshal(raw, &ref) != nil || !study.ValidRef(ref) {
			return study.ErrNotConfigured
		}
		out.Knowledge = &ref
		return nil
	})
	return out, e
}
func (s *Store) SaveStudyNote(ctx context.Context, a study.Access, id string, in study.NoteInput) (study.NoteReceipt, error) {
	var out study.NoteReceipt
	if !study.ValidKnowledgeID(id) || in.Knowledge.ID != id || !study.ValidRef(in.Knowledge) || in.ExpectedRevision < 0 || in.ExpectedRevision >= study.MaxSequence || study.ValidateNote(in.Body) != nil {
		return out, study.ErrInvalid
	}
	e := s.studyTx(ctx, a, study.SaveNote, func(ctx context.Context, tx *sql.Tx, u auth.User, now time.Time) error {
		r, lastRef, e := studyNoteOwnerTx(ctx, tx, u.ID, id)
		if e != nil {
			return e
		}
		replay, found, e := studyReplay[study.NoteReceipt](ctx, tx, u.ID, a, study.SaveNote, id, in)
		if e != nil {
			return e
		}
		if found {
			out = replay
			return nil
		}
		note, exists, e := ownedStudyNoteTx(ctx, tx, u.ID, id, true)
		if e != nil {
			return e
		}
		if note.Revision != in.ExpectedRevision {
			return study.ErrNoteConflict
		}
		scope, e := studyScopeTx(ctx, tx)
		if e != nil {
			return e
		}
		current, available := scope.knowledge[id]
		matches := in.Knowledge == lastRef || exists && note.Knowledge != nil && in.Knowledge == *note.Knowledge || available && in.Knowledge == current.KnowledgeRef
		if !matches {
			return study.ErrVersionStale
		}
		ref, _ := json.Marshal(in.Knowledge)
		revision := note.Revision + 1
		if exists {
			_, e = tx.ExecContext(ctx, "UPDATE study_notes SET revision=$3,body=$4,knowledge_ref=$5,deleted=false,updated_at=$6 WHERE owner_user_id=$1 AND knowledge_id=$2", u.ID, id, revision, in.Body, string(ref), now)
		} else {
			_, e = tx.ExecContext(ctx, "INSERT INTO study_notes(owner_user_id,knowledge_id,revision,body,knowledge_ref,updated_at) VALUES($1,$2,$3,$4,$5,$6)", u.ID, id, revision, in.Body, string(ref), now)
		}
		if e != nil {
			return e
		}
		_ = r
		if e = studyEventTx(ctx, tx, u.ID, a, study.SaveNote, in.Knowledge, scope.pair.TaxonomyVersionID, *scope.pair.TaxonomyHead, "note-saved", &revision, nil, now); e != nil {
			return e
		}
		out = study.NoteReceipt{ActorID: u.ID, KnowledgeID: id, Revision: revision, UpdatedAt: now}
		return studyRemember(ctx, tx, u.ID, a, study.SaveNote, id, in, out)
	})
	return out, e
}
func (s *Store) DeleteStudyNote(ctx context.Context, a study.Access, id string, in study.NoteDeleteInput) (study.NoteReceipt, error) {
	var out study.NoteReceipt
	if !study.ValidKnowledgeID(id) || in.ExpectedRevision < 0 || in.ExpectedRevision >= study.MaxSequence {
		return out, study.ErrInvalid
	}
	e := s.studyTx(ctx, a, study.DeleteNote, func(ctx context.Context, tx *sql.Tx, u auth.User, now time.Time) error {
		if _, _, e := studyNoteOwnerTx(ctx, tx, u.ID, id); e != nil {
			return e
		}
		replay, found, e := studyReplay[study.NoteReceipt](ctx, tx, u.ID, a, study.DeleteNote, id, in)
		if e != nil {
			return e
		}
		if found {
			out = replay
			return nil
		}
		note, exists, e := ownedStudyNoteTx(ctx, tx, u.ID, id, true)
		if e != nil {
			return e
		}
		if !exists {
			return auth.ErrNotFound
		}
		if note.Revision != in.ExpectedRevision {
			return study.ErrNoteConflict
		}
		scope, e := studyScopeTx(ctx, tx)
		if e != nil {
			return e
		}
		revision := note.Revision + 1
		if _, e = tx.ExecContext(ctx, "UPDATE study_notes SET revision=$3,body='',deleted=true,updated_at=$4 WHERE owner_user_id=$1 AND knowledge_id=$2", u.ID, id, revision, now); e != nil {
			return e
		}
		if e = studyEventTx(ctx, tx, u.ID, a, study.DeleteNote, *note.Knowledge, scope.pair.TaxonomyVersionID, *scope.pair.TaxonomyHead, "note-deleted", &revision, nil, now); e != nil {
			return e
		}
		out = study.NoteReceipt{ActorID: u.ID, KnowledgeID: id, Revision: revision, Deleted: true, UpdatedAt: now}
		return studyRemember(ctx, tx, u.ID, a, study.DeleteNote, id, in, out)
	})
	return out, e
}

var _ study.Repository = (*Store)(nil)
