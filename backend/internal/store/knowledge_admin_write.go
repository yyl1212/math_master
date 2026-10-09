package store

import (
	"context"
	"database/sql"
	"errors"
	"github.com/yyl1212/math_master/backend/internal/auth"
	"github.com/yyl1212/math_master/backend/internal/knowledgeadmin"
)

func managedTopics(ctx context.Context, tx *sql.Tx, id string) ([]string, error) {
	rows, e := tx.QueryContext(ctx, "SELECT topic_key FROM managed_knowledge_topics WHERE internal_id=$1 AND active ORDER BY topic_key", id)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	out := []string{}
	for rows.Next() {
		var k string
		if e = rows.Scan(&k); e != nil {
			return nil, e
		}
		out = append(out, k)
	}
	return out, rows.Err()
}
func managedEvent(ctx context.Context, tx *sql.Tx, u auth.User, action string, old *knowledgeadmin.Knowledge, next knowledgeadmin.Knowledge, oldTopics, newTopics []string, evidence any) error {
	var oldsha any
	if old != nil {
		oldsha = old.Ref.ContentSHA256
	}
	var id string
	e := tx.QueryRowContext(ctx, `INSERT INTO managed_knowledge_events(actor_user_id,knowledge_id,action,old_sha256,new_sha256,old_topics,new_topics,evidence) VALUES($1,$2,$3,$4,$5,$6,$7,$8) RETURNING event_id::text`, u.ID, next.ID, action, oldsha, next.Ref.ContentSHA256, knowledgeJSON(oldTopics), knowledgeJSON(newTopics), knowledgeJSON(evidence)).Scan(&id)
	if e != nil {
		return e
	}
	if action == "update" || action == "unpublish" || action == "delete" || action == "restore" || action == "publish" || action == "link" {
		_, e = tx.ExecContext(ctx, `INSERT INTO managed_study_content_changes(event_id,knowledge_id,kind,current_ref) VALUES($1,$2,$3,$4)`, id, next.ID, action, knowledgeJSON(next.Ref))
	}
	return e
}
func createManaged(ctx context.Context, tx *sql.Tx, u auth.User, i knowledgeadmin.CurrentInput, publish bool) (knowledgeadmin.Knowledge, error) {
	id := knowledgeadmin.KnowledgeID(i.ExternalID)
	topics := knowledgeadmin.CurrentTopicKeys(i)
	sha, e := knowledgeadmin.CurrentSHA(i, topics)
	if e != nil {
		return knowledgeadmin.Knowledge{}, e
	}
	_, e = tx.ExecContext(ctx, `INSERT INTO managed_knowledge(internal_id,external_id,point,public_sources,content_sha256,published,created_by,updated_by) VALUES($1,$2,$3,$4,$5,$6,$7,$7)`, id, i.ExternalID, knowledgeJSON(i.Point), knowledgeJSON(i.Sources), sha, publish, u.ID)
	if e != nil {
		return knowledgeadmin.Knowledge{}, e
	}
	for _, t := range topics {
		if _, e = tx.ExecContext(ctx, `INSERT INTO managed_knowledge_topics(internal_id,external_id,topic_key) VALUES($1,$2,$3)`, id, i.ExternalID, t); e != nil {
			return knowledgeadmin.Knowledge{}, e
		}
	}
	return readManaged(ctx, tx, id, false)
}
func (s *Store) CreateManagedKnowledge(ctx context.Context, a knowledgeadmin.Access, i knowledgeadmin.CurrentInput) (out knowledgeadmin.Knowledge, e error) {
	if e = knowledgeadmin.ValidateCurrent(i); e != nil {
		return
	}
	digest := knowledgeFingerprint(i)
	e = s.knowledgeTx(ctx, a, true, true, func(ctx context.Context, tx *sql.Tx, u auth.User) error {
		if ok, e := knowledgeReplay(ctx, tx, u, a, "create", i.ExternalID, digest, &out); e != nil || ok {
			return e
		}
		var e error
		out, e = createManaged(ctx, tx, u, i, false)
		if e != nil {
			return e
		}
		if e = managedEvent(ctx, tx, u, "create", nil, out, []string{}, knowledgeadmin.CurrentTopicKeys(i), i); e != nil {
			return e
		}
		return knowledgeReceipt(ctx, tx, u, a, "create", i.ExternalID, digest, out)
	})
	return
}
func (s *Store) UpdateManagedKnowledge(ctx context.Context, a knowledgeadmin.Access, id, token string, i knowledgeadmin.CurrentInput) (out knowledgeadmin.Knowledge, e error) {
	if e = knowledgeadmin.ValidateCurrent(i); e != nil {
		return
	}
	digest := knowledgeFingerprint(struct {
		ID, Token string
		Input     knowledgeadmin.CurrentInput
	}{id, token, i})
	e = s.knowledgeTx(ctx, a, true, true, func(ctx context.Context, tx *sql.Tx, u auth.User) error {
		if ok, e := knowledgeReplay(ctx, tx, u, a, "update", id, digest, &out); e != nil || ok {
			return e
		}
		old, e := readManaged(ctx, tx, id, true)
		if e != nil {
			return e
		}
		if old.EditToken != token {
			return knowledgeadmin.ErrStale
		}
		if old.Deleted {
			return knowledgeadmin.ErrConflict
		}
		if old.ExternalID != i.ExternalID {
			return knowledgeadmin.ErrInvalid
		}
		before, e := managedTopics(ctx, tx, id)
		if e != nil {
			return e
		}
		after := knowledgeadmin.CurrentTopicKeys(i)
		if _, e = tx.ExecContext(ctx, `UPDATE managed_knowledge_topics SET active=false,manual_override=true,updated_at=clock_timestamp() WHERE internal_id=$1 AND active AND NOT(topic_key=ANY($2))`, id, after); e != nil {
			return e
		}
		for _, t := range after {
			if _, e = tx.ExecContext(ctx, `INSERT INTO managed_knowledge_topics(internal_id,external_id,topic_key,manual_override) VALUES($1,$2,$3,true) ON CONFLICT(external_id,topic_key) DO UPDATE SET active=true,manual_override=true,updated_at=clock_timestamp()`, id, i.ExternalID, t); e != nil {
				return e
			}
		}
		sha, e := knowledgeadmin.CurrentSHA(i, after)
		if e != nil {
			return e
		}
		if _, e = tx.ExecContext(ctx, `UPDATE managed_knowledge SET point=$2,public_sources=$3,content_sha256=$4,edit_token=gen_random_uuid(),updated_by=$5,updated_at=clock_timestamp() WHERE internal_id=$1`, id, knowledgeJSON(i.Point), knowledgeJSON(i.Sources), sha, u.ID); e != nil {
			return e
		}
		out, e = readManaged(ctx, tx, id, false)
		if e != nil {
			return e
		}
		if e = managedEvent(ctx, tx, u, "update", &old, out, before, after, struct{ Before, After knowledgeadmin.CurrentInput }{knowledgeadmin.CurrentInput{ExternalID: old.ExternalID, Point: old.Point, Sources: old.Sources}, i}); e != nil {
			return e
		}
		return knowledgeReceipt(ctx, tx, u, a, "update", id, digest, out)
	})
	return
}
func (s *Store) SetManagedKnowledgeState(ctx context.Context, a knowledgeadmin.Access, id, token, action string) (out knowledgeadmin.Knowledge, e error) {
	if action != "publish" && action != "unpublish" && action != "delete" && action != "restore" {
		return out, knowledgeadmin.ErrInvalid
	}
	digest := knowledgeFingerprint([]string{id, token, action})
	e = s.knowledgeTx(ctx, a, true, true, func(ctx context.Context, tx *sql.Tx, u auth.User) error {
		if ok, e := knowledgeReplay(ctx, tx, u, a, action, id, digest, &out); e != nil || ok {
			return e
		}
		old, e := readManaged(ctx, tx, id, true)
		if e != nil {
			return e
		}
		if old.EditToken != token {
			return knowledgeadmin.ErrStale
		}
		if old.Deleted && action != "restore" {
			return knowledgeadmin.ErrConflict
		}
		if !old.Deleted && action == "restore" {
			return knowledgeadmin.ErrConflict
		}
		published := action == "publish"
		if action == "publish" {
			if e = knowledgeadmin.ValidateCurrent(knowledgeadmin.CurrentInput{ExternalID: old.ExternalID, Point: old.Point, Sources: old.Sources}); e != nil {
				return e
			}
		}
		deleted := action == "delete"
		_, e = tx.ExecContext(ctx, `UPDATE managed_knowledge SET published=$2,deleted_at=CASE WHEN $3 THEN clock_timestamp() ELSE NULL END,edit_token=gen_random_uuid(),updated_by=$4,updated_at=clock_timestamp() WHERE internal_id=$1`, id, published, deleted, u.ID)
		if e != nil {
			return e
		}
		out, e = readManaged(ctx, tx, id, false)
		if e != nil {
			return e
		}
		topics, e := managedTopics(ctx, tx, id)
		if e != nil {
			return e
		}
		if e = managedEvent(ctx, tx, u, action, &old, out, topics, topics, map[string]any{"published": published, "deleted": deleted}); e != nil {
			return e
		}
		return knowledgeReceipt(ctx, tx, u, a, action, id, digest, out)
	})
	return
}
func knownSourceCore(ctx context.Context, tx *sql.Tx, k knowledgeadmin.Knowledge, p knowledgeadmin.SourcePoint) (bool, error) {
	core, e := knowledgeadmin.SourceCoreSHA(p)
	if e != nil {
		return false, e
	}
	current, e := knowledgeadmin.SourceCoreSHA(k.Point)
	if e != nil {
		return false, e
	}
	if core == current {
		return true, nil
	}
	var accepted bool
	e = tx.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM managed_knowledge_sources WHERE internal_id=$1 AND source_core_sha=$2)", k.ID, core).Scan(&accepted)
	return accepted, e
}
func existingExternal(ctx context.Context, tx *sql.Tx, id string) (knowledgeadmin.Knowledge, bool, error) {
	k, e := scanManaged(tx.QueryRowContext(ctx, "SELECT "+managedColumns+" FROM managed_knowledge WHERE external_id=$1", id))
	if errors.Is(e, sql.ErrNoRows) {
		return k, false, nil
	}
	return k, e == nil, e
}
