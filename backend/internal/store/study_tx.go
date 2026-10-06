package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/yyl1212/math_master/backend/internal/auth"
	"github.com/yyl1212/math_master/backend/internal/content"
	"github.com/yyl1212/math_master/backend/internal/publication"
	"github.com/yyl1212/math_master/backend/internal/study"
	"github.com/yyl1212/math_master/backend/internal/taxonomy"
	"time"
)

const studyTimeout = 8 * time.Second

var studyTables = []string{"study_records", "study_events", "study_notes", "study_idempotency", "study_content_changes"}

func studyError(e error) error {
	for _, known := range []error{study.ErrInvalid, study.ErrNotConfigured, study.ErrStateConflict, study.ErrVersionStale, study.ErrIdempotencyConflict, study.ErrNoteConflict} {
		if errors.Is(e, known) {
			return e
		}
	}
	if errors.Is(e, taxonomy.ErrNotConfigured) {
		return study.ErrNotConfigured
	}
	if errors.Is(e, taxonomy.ErrHeadStale) {
		return study.ErrVersionStale
	}
	var pg *pgconn.PgError
	if errors.As(e, &pg) && (pg.Code == "42P01" || pg.Code == "42703") {
		return study.ErrNotConfigured
	}
	return authError(e)
}
func studyConfigured(ctx context.Context, tx *sql.Tx) error {
	var n int
	if e := tx.QueryRowContext(ctx, "SELECT count(*) FROM unnest($1::text[]) name WHERE to_regclass('public.'||name) IS NOT NULL", studyTables).Scan(&n); e != nil {
		return e
	}
	if n != 5 {
		return study.ErrNotConfigured
	}
	var marker bool
	if e := tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM information_schema.columns WHERE table_schema='public' AND table_name='goose_db_version' AND column_name='topic_study_enabled')`).Scan(&marker); e != nil {
		return e
	}
	if !marker {
		return study.ErrNotConfigured
	}
	if e := tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM goose_db_version WHERE version_id=0 AND topic_study_enabled) AND EXISTS(SELECT 1 FROM topic_learning_state WHERE singleton AND study_enabled)`).Scan(&marker); e != nil {
		return e
	}
	if !marker {
		return study.ErrNotConfigured
	}
	if e := tx.QueryRowContext(ctx, `SELECT count(*) FROM (VALUES
 ('study_event_immutable','study_events','reject_content_update()'),('study_receipt_immutable','study_idempotency','reject_content_update()'),('study_change_immutable','study_content_changes','reject_content_update()'),('study_note_guard','study_notes','guard_study_note()'),('topic_study_marker_guard','goose_db_version','guard_topic_study_marker()')
 ) expected(name,table_name,function_name) JOIN pg_trigger t ON t.tgname=expected.name AND t.tgrelid=to_regclass('public.'||expected.table_name) AND t.tgfoid=to_regprocedure('public.'||expected.function_name) AND t.tgtype=27 AND NOT t.tgisinternal AND t.tgenabled IN ('O','A')`).Scan(&n); e != nil {
		return e
	}
	if n != 5 {
		return study.ErrNotConfigured
	}
	return nil
}
func studyIdentity(ctx context.Context, tx *sql.Tx, a study.Access, action study.Action, lock bool) (auth.User, time.Time, error) {
	proof := publication.Access{TokenHash: a.TokenHash, CSRF: a.CSRF, IdempotencyKey: a.IdempotencyKey, RequestID: a.RequestID}
	u, _, now, e := managedIdentity(ctx, tx, proof, action != study.Read, lock, nil)
	if e != nil {
		return u, now, e
	}
	if e = study.Authorize(u, action); e != nil {
		return u, now, e
	}
	if action != study.Read && !study.ValidID(a.IdempotencyKey) {
		return u, now, auth.ErrInvalidInput
	}
	return u, now, nil
}
func (s *Store) studyTx(ctx context.Context, a study.Access, action study.Action, fn func(context.Context, *sql.Tx, auth.User, time.Time) error) error {
	ctx, cancel := context.WithTimeout(ctx, studyTimeout)
	defer cancel()
	tx, e := s.db.BeginTx(ctx, nil)
	if e != nil {
		return studyError(e)
	}
	defer tx.Rollback()
	if _, e = tx.ExecContext(ctx, "SET LOCAL lock_timeout='1s'"); e != nil {
		return studyError(e)
	}
	// Preserve the established global lock order; row ownership is then locked
	// before the current pair and the personal record.
	for _, id := range []int64{adminLockID, 1296127048} {
		if _, e = tx.ExecContext(ctx, "SELECT pg_advisory_xact_lock_shared($1)", id); e != nil {
			return studyError(e)
		}
	}
	u, now, e := studyIdentity(ctx, tx, a, action, true)
	if e != nil {
		return studyError(e)
	}
	if e = studyConfigured(ctx, tx); e != nil {
		return studyError(e)
	}
	if e = fn(ctx, tx, u, now); e != nil {
		return studyError(e)
	}
	if _, _, e = studyIdentity(ctx, tx, a, action, false); e != nil {
		return studyError(e)
	}
	return studyError(tx.Commit())
}
func (s *Store) studyReadTx(ctx context.Context, a study.Access, fn func(context.Context, *sql.Tx, auth.User) error) error {
	ctx, cancel := context.WithTimeout(ctx, studyTimeout)
	defer cancel()
	tx, e := s.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true, Isolation: sql.LevelRepeatableRead})
	if e != nil {
		return studyError(e)
	}
	defer tx.Rollback()
	u, _, e := studyIdentity(ctx, tx, a, study.Read, false)
	if e != nil {
		return studyError(e)
	}
	if e = studyConfigured(ctx, tx); e != nil {
		return studyError(e)
	}
	if e = fn(ctx, tx, u); e != nil {
		return studyError(e)
	}
	return studyError(tx.Commit())
}
func (s *Store) StudyPreflight(ctx context.Context, a study.Access, action study.Action) (auth.User, error) {
	var out auth.User
	e := s.studyReadTx(ctx, a, func(_ context.Context, _ *sql.Tx, u auth.User) error { out = u; return study.Authorize(u, action) })
	return out, e
}

type studyScope struct {
	pair      taxonomy.PairRef
	knowledge map[string]taxonomy.KnowledgeSummary
	relations map[string][]content.Relation
}

func studyScopeTx(ctx context.Context, tx *sql.Tx) (studyScope, error) {
	out := studyScope{knowledge: map[string]taxonomy.KnowledgeSummary{}, relations: map[string][]content.Relation{}}
	if e := taxonomyConfigured(ctx, tx); e != nil {
		return out, e
	}
	var version string
	if e := tx.QueryRowContext(ctx, "SELECT r.taxonomy_version_id FROM taxonomy_heads h JOIN taxonomy_releases r ON r.id=h.release_id AND r.status='published' WHERE h.singleton").Scan(&version); e != nil {
		if errors.Is(e, sql.ErrNoRows) {
			return out, study.ErrNotConfigured
		}
		return out, e
	}
	var e error
	out.pair, e = topicCurrentPairTx(ctx, tx, version)
	if e != nil {
		return out, e
	}
	if out.pair.KnowledgeHead == nil {
		return out, nil
	}
	rows, e := tx.QueryContext(ctx, `SELECT k.id,k.version,k.sha256,k.body->>'title',COALESCE(k.body->>'titleZh',''),k.body->'relations' FROM publication_members m JOIN knowledge_versions k ON k.id=m.id AND k.version=m.version JOIN publication_snapshots p ON p.id=m.snapshot_id AND p.status='published' WHERE m.snapshot_id=$1 AND m.kind='knowledge' AND m.availability='active' AND NOT EXISTS(SELECT 1 FROM content_withdrawals w WHERE w.kind='knowledge' AND w.target_id=k.id AND w.target_version=k.version AND w.sha256=k.sha256)`, *out.pair.KnowledgeHead)
	if e != nil {
		return out, e
	}
	for rows.Next() {
		var k taxonomy.KnowledgeSummary
		var raw []byte
		if e = rows.Scan(&k.ID, &k.Version, &k.SHA256, &k.Title, &k.TitleZh, &raw); e != nil {
			rows.Close()
			return out, e
		}
		var relations []content.Relation
		if json.Unmarshal(raw, &relations) != nil {
			rows.Close()
			return out, study.ErrNotConfigured
		}
		k.TopicIDs = []string{}
		out.knowledge[k.ID] = k
		out.relations[k.ID] = relations
		if len(out.knowledge) > 1000 {
			rows.Close()
			return out, study.ErrNotConfigured
		}
	}
	e = rows.Err()
	rows.Close()
	if e != nil {
		return out, e
	}
	colors := map[string]int{}
	eligible := map[string]bool{}
	var valid func(string) bool
	valid = func(id string) bool {
		if colors[id] == 1 {
			return false
		}
		if colors[id] == 2 {
			return eligible[id]
		}
		_, ok := out.knowledge[id]
		if !ok {
			return false
		}
		colors[id] = 1
		for _, r := range out.relations[id] {
			if r.Kind == "prerequisite" {
				target, exists := out.knowledge[r.Target.ID]
				if !exists || target.Version != r.Target.Version || !valid(r.Target.ID) {
					ok = false
					break
				}
			}
		}
		colors[id] = 2
		eligible[id] = ok
		return ok
	}
	for id := range out.knowledge {
		valid(id)
	}
	for id := range out.knowledge {
		if !eligible[id] {
			delete(out.knowledge, id)
		}
	}
	rows, e = tx.QueryContext(ctx, `SELECT knowledge_id,knowledge_version,knowledge_sha,topic_id FROM taxonomy_release_assignments WHERE release_id=$1 ORDER BY knowledge_id,topic_id`, *out.pair.TaxonomyHead)
	if e != nil {
		return out, e
	}
	defer rows.Close()
	for rows.Next() {
		var id, sha, topic string
		var version int
		if e = rows.Scan(&id, &version, &sha, &topic); e != nil {
			return out, e
		}
		k, ok := out.knowledge[id]
		if ok && k.Version == version && k.SHA256 == sha {
			k.TopicIDs = append(k.TopicIDs, topic)
			out.knowledge[id] = k
		}
	}
	return out, rows.Err()
}
func studyReadRecordTx(ctx context.Context, tx *sql.Tx, actor, id string, lock bool) (study.StudyRecord, error) {
	out := study.StudyRecord{KnowledgeID: id, State: study.Unlearned}
	query := "SELECT body FROM study_records WHERE owner_user_id=$1 AND knowledge_id=$2"
	if lock {
		query += " FOR UPDATE"
	}
	var raw []byte
	e := tx.QueryRowContext(ctx, query, actor, id).Scan(&raw)
	if errors.Is(e, sql.ErrNoRows) {
		return out, nil
	}
	if e != nil {
		return out, e
	}
	if json.Unmarshal(raw, &out) != nil || out.KnowledgeID != id || !study.ValidState(out.State) {
		return out, study.ErrNotConfigured
	}
	return out, nil
}
func studySaveRecordTx(ctx context.Context, tx *sql.Tx, actor string, r study.StudyRecord, ref study.KnowledgeRef, now time.Time) error {
	raw, e := json.Marshal(r)
	if e != nil {
		return e
	}
	kr, e := json.Marshal(ref)
	if e != nil {
		return e
	}
	_, e = tx.ExecContext(ctx, `INSERT INTO study_records(owner_user_id,knowledge_id,state,sequence,body,last_known_ref,updated_at) VALUES($1,$2,$3,$4,$5,$6,$7) ON CONFLICT(owner_user_id,knowledge_id) DO UPDATE SET state=EXCLUDED.state,sequence=EXCLUDED.sequence,body=EXCLUDED.body,last_known_ref=EXCLUDED.last_known_ref,updated_at=EXCLUDED.updated_at`, actor, r.KnowledgeID, r.State, r.Sequence, string(raw), string(kr), now)
	return e
}
func studyDetail(actor string, r study.StudyRecord, scope studyScope) study.StudyDetail {
	out := study.StudyDetail{ActorID: actor, Record: r, Pair: scope.pair}
	if k, ok := scope.knowledge[r.KnowledgeID]; ok {
		out.CurrentKnowledge = &k
		out.Available = true
		ack := r.CompletedRef
		if r.LastReviewRef != nil && r.LastReviewedAt != nil && (r.LastCompletedAt == nil || r.LastReviewedAt.After(*r.LastCompletedAt)) {
			ack = r.LastReviewRef
		}
		out.MaterialChanged = study.MaterialChanged(ack, &k.KnowledgeRef)
	}
	return out
}
func studyEventTx(ctx context.Context, tx *sql.Tx, actor string, a study.Access, action study.Action, k study.KnowledgeRef, taxVersion, taxHead, kind string, noteRevision *int64, reviewID *string, now time.Time) error {
	_, e := tx.ExecContext(ctx, `INSERT INTO study_events(id,owner_user_id,knowledge_id,knowledge_version,knowledge_sha256,taxonomy_version_id,taxonomy_head,kind,recorded_at,note_revision,review_id,source_kind,action,idempotency_key) VALUES(gen_random_uuid(),$1,$2,$3,$4,$5,$6,$7,$8,$9,$10,'native',$11,$12)`, actor, k.ID, k.Version, k.SHA256, taxVersion, taxHead, kind, now, noteRevision, reviewID, action, a.IdempotencyKey)
	return e
}
