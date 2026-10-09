package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"github.com/yyl1212/math_master/backend/internal/auth"
	"github.com/yyl1212/math_master/backend/internal/knowledgeadmin"
	"github.com/yyl1212/math_master/backend/internal/study"
)

func readManagedNote(ctx context.Context, tx *sql.Tx, owner, id string, lock bool) (out knowledgeadmin.ManagedNote, e error) {
	out = knowledgeadmin.ManagedNote{ActorID: owner, KnowledgeID: id}
	q := "SELECT body,revision,knowledge_ref,deleted,updated_at FROM managed_study_notes WHERE owner_user_id=$1 AND knowledge_id=$2"
	if lock {
		q += " FOR UPDATE"
	}
	var ref []byte
	e = tx.QueryRowContext(ctx, q, owner, id).Scan(&out.Body, &out.Revision, &ref, &out.Deleted, &out.UpdatedAt)
	if errors.Is(e, sql.ErrNoRows) {
		e = nil
		return
	}
	if e != nil {
		return
	}
	out.Knowledge = &knowledgeadmin.Ref{}
	e = json.Unmarshal(ref, out.Knowledge)
	return
}
func (s *Store) ReadManagedNote(ctx context.Context, a knowledgeadmin.Access, id string) (out knowledgeadmin.ManagedNote, e error) {
	e = s.knowledgeTx(ctx, a, false, false, func(ctx context.Context, tx *sql.Tx, u auth.User) error {
		if e := managedStudyMode(ctx, tx); e != nil {
			return e
		}
		k, e := managedStudyCurrent(ctx, tx, id, false)
		if e != nil {
			return e
		}
		out, e = readManagedNote(ctx, tx, u.ID, id, false)
		if e != nil {
			return e
		}
		out.Available = k.Published && !k.Deleted
		if !out.Available && out.Revision == 0 {
			return knowledgeadmin.ErrNotFound
		}
		return nil
	})
	return
}
func (s *Store) SaveManagedNote(ctx context.Context, a knowledgeadmin.Access, id string, in knowledgeadmin.ManagedNoteInput) (knowledgeadmin.ManagedNote, error) {
	return s.writeManagedNote(ctx, a, id, in, false)
}
func (s *Store) DeleteManagedNote(ctx context.Context, a knowledgeadmin.Access, id string, in knowledgeadmin.ManagedNoteInput) (knowledgeadmin.ManagedNote, error) {
	return s.writeManagedNote(ctx, a, id, in, true)
}
func (s *Store) writeManagedNote(ctx context.Context, a knowledgeadmin.Access, id string, in knowledgeadmin.ManagedNoteInput, deleting bool) (out knowledgeadmin.ManagedNote, e error) {
	if !knowledgeadmin.ValidManagedRef(in.Knowledge) || in.Knowledge.ID != id || in.ExpectedRevision < 0 || in.ExpectedRevision >= study.MaxSequence || !deleting && study.ValidateNote(in.Body) != nil || deleting && in.Body != "" {
		return out, knowledgeadmin.ErrInvalid
	}
	action, kind := "save-note", "note-saved"
	if deleting {
		action, kind = "delete-note", "note-deleted"
	}
	digest := knowledgeFingerprint(in)
	e = s.knowledgeTx(ctx, a, true, false, func(ctx context.Context, tx *sql.Tx, u auth.User) error {
		if e := managedStudyMode(ctx, tx); e != nil {
			return e
		}
		k, e := managedStudyCurrent(ctx, tx, id, true)
		if e != nil {
			return e
		}
		if !k.Published || k.Deleted {
			return knowledgeadmin.ErrNotFound
		}
		if k.Ref != in.Knowledge {
			return knowledgeadmin.ErrStale
		}
		if _, e = tx.ExecContext(ctx, `INSERT INTO managed_study_records(owner_user_id,knowledge_id,state) VALUES($1,$2,'unlearned') ON CONFLICT DO NOTHING`, u.ID, id); e != nil {
			return e
		}
		if _, e = readManagedRecord(ctx, tx, u.ID, id, true); e != nil {
			return e
		}
		prior, e := readManagedNote(ctx, tx, u.ID, id, true)
		if e != nil {
			return e
		}
		replay, e := managedStudyReplay(ctx, tx, u.ID, a, id, action, digest)
		if e != nil {
			return e
		}
		if replay {
			out = prior
			out.Available = true
			return nil
		}
		if prior.Revision != in.ExpectedRevision {
			return knowledgeadmin.ErrStale
		}
		now, e := dbClock(ctx, tx)
		if e != nil {
			return e
		}
		revision := prior.Revision + 1
		_, e = tx.ExecContext(ctx, `INSERT INTO managed_study_notes(owner_user_id,knowledge_id,body,revision,knowledge_ref,deleted,updated_at) VALUES($1,$2,$3,$4,$5,$6,$7) ON CONFLICT(owner_user_id,knowledge_id) DO UPDATE SET body=excluded.body,revision=excluded.revision,knowledge_ref=excluded.knowledge_ref,deleted=excluded.deleted,updated_at=excluded.updated_at`, u.ID, id, in.Body, revision, knowledgeJSON(k.Ref), deleting, now)
		if e != nil {
			return e
		}
		_, e = tx.ExecContext(ctx, `INSERT INTO managed_study_events(owner_user_id,knowledge_id,knowledge_ref,topic_keys,kind,note_revision,recorded_at) VALUES($1,$2,$3,$4,$5,$6,$7)`, u.ID, id, knowledgeJSON(k.Ref), knowledgeJSON(k.TopicKeys), kind, revision, now)
		if e != nil {
			return e
		}
		out = knowledgeadmin.ManagedNote{ActorID: u.ID, KnowledgeID: id, Body: in.Body, Revision: revision, Knowledge: &k.Ref, Deleted: deleting, Available: true, UpdatedAt: &now}
		return rememberManagedStudy(ctx, tx, u.ID, a, id, action, digest, map[string]any{"revision": revision, "deleted": deleting})
	})
	return
}
