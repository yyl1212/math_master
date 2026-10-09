package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"github.com/yyl1212/math_master/backend/internal/study"
	"github.com/yyl1212/math_master/backend/internal/taxonomy"
	"time"
)

var cutoverTables = []string{"topic_cutovers", "study_migration_batches", "study_legacy_event_links"}

func cutoverConfigured(ctx context.Context, tx *sql.Tx) error {
	var n int
	var intact bool
	if e := tx.QueryRowContext(ctx, `SELECT count(*) FROM unnest($1::text[]) name WHERE to_regclass('public.'||name) IS NOT NULL`, cutoverTables).Scan(&n); e != nil {
		return e
	}
	if n != 3 {
		return study.ErrNotConfigured
	}
	if e := tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM goose_db_version WHERE version_id=0 AND topic_cutover_enabled) AND EXISTS(SELECT 1 FROM goose_db_version WHERE version_id=12 AND is_applied) AND EXISTS(SELECT 1 FROM topic_learning_state WHERE singleton AND retirement_enabled)`).Scan(&intact); e != nil {
		return e
	}
	if !intact {
		return study.ErrNotConfigured
	}
	if e := tx.QueryRowContext(ctx, `SELECT count(*) FROM (VALUES ('study_migration_batch_immutable','study_migration_batches','reject_content_update()',27),('topic_cutover_immutable','topic_cutovers','reject_content_update()',27),('study_legacy_link_immutable','study_legacy_event_links','reject_content_update()',27),('topic_cutover_marker_guard','goose_db_version','guard_topic_cutover_marker()',27),('study_legacy_link_guard','study_legacy_event_links','guard_study_legacy_link()',5)) expected(name,table_name,function_name,bits) JOIN pg_trigger t ON t.tgname=expected.name AND t.tgrelid=to_regclass('public.'||expected.table_name) AND t.tgfoid=to_regprocedure('public.'||expected.function_name) AND t.tgtype=expected.bits AND t.tgenabled IN ('O','A') AND NOT t.tgisinternal`).Scan(&n); e != nil {
		return e
	}
	if n != 5 {
		return study.ErrNotConfigured
	}
	if e := tx.QueryRowContext(ctx, `SELECT NOT EXISTS(SELECT 1 FROM jsonb_each_text($1::jsonb) g WHERE NOT EXISTS(SELECT 1 FROM pg_constraint c WHERE c.conrelid=to_regclass('public.'||split_part(g.key,'/',1)) AND c.conname=split_part(g.key,'/',2) AND c.convalidated AND c.contype::text||':'||md5(pg_get_constraintdef(c.oid))=g.value OFFSET 0))`, string(body(topicSchemaConstraintGuards))).Scan(&intact); e != nil {
		return e
	}
	if !intact {
		return study.ErrNotConfigured
	}
	return nil
}
func optionalCutoverConfigured(ctx context.Context, tx *sql.Tx) (bool, error) {
	var n int
	var marker bool
	if e := tx.QueryRowContext(ctx, `SELECT (SELECT count(*) FROM unnest($1::text[]) name WHERE to_regclass('public.'||name) IS NOT NULL),EXISTS(SELECT 1 FROM pg_attribute WHERE attrelid=to_regclass('public.goose_db_version') AND attname='topic_cutover_enabled' AND NOT attisdropped)`, cutoverTables).Scan(&n, &marker); e != nil {
		return false, e
	}
	if n == 0 && !marker {
		return false, nil
	}
	if e := cutoverConfigured(ctx, tx); e != nil {
		return false, studyError(e)
	}
	return true, nil
}

type legacyStudyFact struct {
	id, actor, kind, pub, sourceSHA string
	ref                             study.KnowledgeRef
	at                              time.Time
}

func (s *Store) MigrateLegacyStudyBatch(ctx context.Context, limit int, cursor *study.LegacyCursor) (study.MigrationReport, error) {
	out := study.MigrationReport{Cursor: cursor}
	if limit < 1 || limit > 50 || cursor != nil && (!study.ValidID(cursor.EventID) || cursor.RecordedAt.IsZero()) {
		return out, study.ErrInvalid
	}
	ctx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()
	tx, e := s.db.BeginTx(ctx, nil)
	if e != nil {
		return out, studyError(e)
	}
	defer tx.Rollback()
	if _, e = tx.ExecContext(ctx, "SET LOCAL lock_timeout='1s'"); e != nil {
		return out, studyError(e)
	}
	var managedMigration bool
	if e = tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM goose_db_version WHERE version_id=13 AND is_applied)`).Scan(&managedMigration); e != nil {
		return out, studyError(e)
	}
	if managedMigration && knowledgeConfigured(ctx, tx) != nil {
		return out, study.ErrNotConfigured
	}
	var managedMarker bool
	if e = tx.QueryRowContext(ctx, `SELECT coalesce((to_jsonb(g)->>'managed_knowledge_enabled')::boolean,false) FROM goose_db_version g WHERE version_id=0`).Scan(&managedMarker); e != nil {
		return out, studyError(e)
	}
	if managedMarker {
		return out, study.ErrModuleRetired
	}
	// Maintenance requires A/B/C, but does not need to materialize the public
	// knowledge scope for every finite batch. Lock the same mode fence first.
	var mode taxonomy.ExperienceMode
	if e = tx.QueryRowContext(ctx, "SELECT experience_mode FROM topic_learning_state WHERE singleton FOR SHARE").Scan(&mode); e != nil {
		return out, studyError(e)
	}
	if mode != taxonomy.ModeLegacy && mode != taxonomy.ModeTopics {
		return out, study.ErrNotConfigured
	}
	if e = taxonomyConfigured(ctx, tx); e != nil {
		return out, studyError(e)
	}
	if e = cutoverConfigured(ctx, tx); e != nil {
		return out, studyError(e)
	}
	if e = studyConfigured(ctx, tx); e != nil {
		return out, studyError(e)
	}
	for _, lock := range []int64{adminLockID, 1296127048} {
		if _, e = tx.ExecContext(ctx, "SELECT pg_advisory_xact_lock_shared($1)", lock); e != nil {
			return out, studyError(e)
		}
	}
	if e = tx.QueryRowContext(ctx, `SELECT coalesce((to_jsonb(g)->>'managed_knowledge_enabled')::boolean,false) FROM goose_db_version g WHERE version_id=0`).Scan(&managedMarker); e != nil {
		return out, studyError(e)
	}
	if managedMarker {
		return out, study.ErrModuleRetired
	}

	if _, e = tx.ExecContext(ctx, "SELECT pg_advisory_xact_lock(1296127050)"); e != nil {
		return out, studyError(e)
	}
	ok, err := learningConfigured(ctx, tx)
	if err != nil || !ok {
		return out, study.ErrNotConfigured
	}
	var protections int
	if e = tx.QueryRowContext(ctx, `SELECT count(*) FROM (VALUES('learning_event_immutable','reject_content_update()',27),('learning_event_complete','learning_event_complete()',5)) expected(name,fn,bits) JOIN pg_trigger t ON t.tgrelid=to_regclass('public.learning_events') AND t.tgname=expected.name AND t.tgfoid=to_regprocedure('public.'||expected.fn) AND t.tgtype=expected.bits AND t.tgenabled IN ('O','A') AND NOT t.tgisinternal`).Scan(&protections); e != nil {
		return out, studyError(e)
	}
	if protections != 2 {
		return out, study.ErrNotConfigured
	}
	started, e := dbClock(ctx, tx)
	if e != nil {
		return out, studyError(e)
	}
	out.BatchID, e = workflowID()
	if e != nil {
		return out, e
	}
	var boundaryAt, boundaryID any
	if cursor != nil {
		boundaryAt = cursor.RecordedAt
		boundaryID = cursor.EventID
	}
	rows, e := tx.QueryContext(ctx, `SELECT id::text,owner_user_id::text,knowledge_id,knowledge_version,knowledge_sha256,knowledge_publication_id,seal_sha256,kind,recorded_at FROM learning_events WHERE kind IN ('started','completed') AND ($1::timestamptz IS NULL OR (recorded_at,id)>($1,$2::uuid)) ORDER BY recorded_at,id LIMIT $3`, boundaryAt, boundaryID, limit)
	if e != nil {
		return out, studyError(e)
	}
	facts := []legacyStudyFact{}
	owners := map[string]bool{}
	for rows.Next() {
		var f legacyStudyFact
		if e = rows.Scan(&f.id, &f.actor, &f.ref.ID, &f.ref.Version, &f.ref.SHA256, &f.pub, &f.sourceSHA, &f.kind, &f.at); e != nil {
			rows.Close()
			return out, studyError(e)
		}
		f.at = f.at.UTC()
		facts = append(facts, f)
		owners[f.actor] = true
	}
	e = rows.Err()
	rows.Close()
	if e != nil {
		return out, studyError(e)
	}
	ids := []string{}
	for id := range owners {
		ids = append(ids, id)
	}
	rows, e = tx.QueryContext(ctx, "SELECT id::text FROM auth_users WHERE id=ANY($1::uuid[]) ORDER BY id FOR UPDATE", ids)
	if e != nil {
		return out, studyError(e)
	}
	locked := 0
	for rows.Next() {
		locked++
	}
	e = rows.Err()
	rows.Close()
	if e != nil || locked != len(ids) {
		return out, study.ErrNotConfigured
	}

	selected := []map[string]any{}
	for _, f := range facts {
		selected = append(selected, map[string]any{"id": f.id, "owner": f.actor, "kid": f.ref.ID})
	}
	selectedJSON := string(body(selected))
	const selection = `jsonb_to_recordset($1::jsonb) x(id uuid,owner uuid,kid text)`
	links := map[string]string{}
	rows, e = tx.QueryContext(ctx, `SELECT l.legacy_event_id::text,l.source_sha FROM `+selection+` JOIN study_legacy_event_links l ON l.owner_user_id=x.owner AND l.legacy_event_id=x.id`, selectedJSON)
	if e != nil {
		return out, studyError(e)
	}
	for rows.Next() {
		var id, sha string
		if e = rows.Scan(&id, &sha); e != nil {
			rows.Close()
			return out, studyError(e)
		}
		links[id] = sha
	}
	e = rows.Err()
	rows.Close()
	if e != nil {
		return out, studyError(e)
	}
	type projected struct {
		record  study.StudyRecord
		managed bool
		ref     study.KnowledgeRef
		actor   string
		dirty   bool
	}
	records := map[string]*projected{}
	key := func(actor, id string) string { return actor + "/" + id }
	rows, e = tx.QueryContext(ctx, `SELECT r.owner_user_id::text,r.knowledge_id,r.body,
 EXISTS(SELECT 1 FROM study_legacy_event_links l WHERE l.owner_user_id=r.owner_user_id AND l.knowledge_id=r.knowledge_id AND l.projected_record)
 AND NOT EXISTS(SELECT 1 FROM study_events e WHERE e.owner_user_id=r.owner_user_id AND e.knowledge_id=r.knowledge_id AND e.source_kind='native')
 AND NOT EXISTS(SELECT 1 FROM study_idempotency i WHERE i.owner_user_id=r.owner_user_id AND i.knowledge_id=r.knowledge_id)
 AND NOT EXISTS(SELECT 1 FROM study_notes n WHERE n.owner_user_id=r.owner_user_id AND n.knowledge_id=r.knowledge_id)
 FROM study_records r JOIN (SELECT DISTINCT owner,kid FROM `+selection+`) x ON x.owner=r.owner_user_id AND x.kid=r.knowledge_id ORDER BY r.owner_user_id,r.knowledge_id FOR UPDATE OF r`, selectedJSON)
	if e != nil {
		return out, studyError(e)
	}
	for rows.Next() {
		var actor, id string
		var raw []byte
		var managed bool
		var record study.StudyRecord
		if e = rows.Scan(&actor, &id, &raw, &managed); e != nil {
			rows.Close()
			return out, studyError(e)
		}
		if json.Unmarshal(raw, &record) != nil || record.KnowledgeID != id || !study.ValidState(record.State) {
			rows.Close()
			return out, study.ErrNotConfigured
		}
		records[key(actor, id)] = &projected{record: record, managed: managed && record.Sequence == 0 && record.LastReadAt == nil, actor: actor}
	}
	e = rows.Err()
	rows.Close()
	if e != nil {
		return out, studyError(e)
	}
	newEvents := []map[string]any{}
	for _, f := range facts {
		if sha, exists := links[f.id]; exists {
			if sha != f.sourceSHA {
				return out, study.ErrNotConfigured
			}
			out.LinkedEvents++
		} else {
			r := records[key(f.actor, f.ref.ID)]
			if r == nil {
				r = &projected{record: study.StudyRecord{KnowledgeID: f.ref.ID, State: study.Unlearned}, managed: true, actor: f.actor}
				records[key(f.actor, f.ref.ID)] = r
			}
			if r.managed {
				at := f.at
				if f.kind == "started" {
					if r.record.FirstStartedAt == nil || at.Before(*r.record.FirstStartedAt) {
						r.record.FirstStartedAt = &at
					}
					if r.record.FirstCompletedAt == nil {
						r.record.State = study.Learning
					}
				} else {
					if r.record.FirstCompletedAt == nil || at.Before(*r.record.FirstCompletedAt) {
						r.record.FirstCompletedAt = &at
					}
					if r.record.LastCompletedAt == nil || !at.Before(*r.record.LastCompletedAt) {
						r.record.LastCompletedAt = &at
						ref := f.ref
						r.record.CompletedRef = &ref
					}
					r.record.State = study.Completed
				}
				r.ref = f.ref
				if r.record.CompletedRef != nil {
					r.ref = *r.record.CompletedRef
				}
				r.dirty = true
			}
			eventID, err := workflowID()
			if err != nil {
				return out, err
			}
			newEvents = append(newEvents, map[string]any{"id": eventID, "owner": f.actor, "kid": f.ref.ID, "version": f.ref.Version, "sha": f.ref.SHA256, "kind": f.kind, "at": f.at, "origin": f.id, "sourceSHA": f.sourceSHA, "publication": f.pub, "projected": r.managed})
			out.CreatedEvents++
		}
		out.Processed++
		out.Cursor = &study.LegacyCursor{RecordedAt: f.at, EventID: f.id}
	}
	saves := []map[string]any{}
	for _, r := range records {
		if r.dirty {
			saves = append(saves, map[string]any{"owner": r.actor, "kid": r.record.KnowledgeID, "state": r.record.State, "sequence": r.record.Sequence, "body": r.record, "ref": r.ref})
		}
	}
	if len(saves) > 0 {
		if _, e = tx.ExecContext(ctx, `INSERT INTO study_records(owner_user_id,knowledge_id,state,sequence,body,last_known_ref,updated_at) SELECT owner,kid,state,sequence,body,ref,$2 FROM jsonb_to_recordset($1::jsonb) x(owner uuid,kid text,state text,sequence bigint,body jsonb,ref jsonb) ON CONFLICT(owner_user_id,knowledge_id) DO UPDATE SET state=EXCLUDED.state,sequence=EXCLUDED.sequence,body=EXCLUDED.body,last_known_ref=EXCLUDED.last_known_ref,updated_at=EXCLUDED.updated_at`, string(body(saves)), started); e != nil {
			return out, studyError(e)
		}
	}
	if len(newEvents) > 0 {
		raw := string(body(newEvents))
		const events = `jsonb_to_recordset($1::jsonb) x(id uuid,owner uuid,kid text,version integer,sha text,kind text,at timestamptz,origin uuid,"sourceSHA" text,publication text,projected boolean)`
		if _, e = tx.ExecContext(ctx, `INSERT INTO study_events(id,owner_user_id,knowledge_id,knowledge_version,knowledge_sha256,kind,recorded_at,source_kind,origin_event_id,action,idempotency_key) SELECT id,owner,kid,version,sha,kind,at,'legacy',origin,'legacy-migration',origin FROM `+events, raw); e != nil {
			return out, studyError(e)
		}
		if _, e = tx.ExecContext(ctx, `INSERT INTO study_legacy_event_links(owner_user_id,legacy_event_id,study_event_id,knowledge_id,source_sha,knowledge_publication_id,projected_record,batch_id,recorded_at,linked_at) SELECT owner,origin,id,kid,"sourceSHA",publication,projected,$2,at,$3 FROM `+events, raw, out.BatchID, started); e != nil {
			return out, studyError(e)
		}
	}
	// An indexed look ahead avoids rescanning the already imported prefix on
	// every batch. At the end we still check the entire source, including late
	// facts before the caller's cursor; Done can never hide them.
	var ahead bool
	if out.Cursor != nil {
		if e = tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM learning_events WHERE kind IN ('started','completed') AND (recorded_at,id)>($1,$2::uuid) AND NOT EXISTS(SELECT 1 FROM study_legacy_event_links l WHERE l.owner_user_id=learning_events.owner_user_id AND l.legacy_event_id=learning_events.id))`, out.Cursor.RecordedAt, out.Cursor.EventID).Scan(&ahead); e != nil {
			return out, studyError(e)
		}
	}
	if !ahead {
		if e = tx.QueryRowContext(ctx, `SELECT NOT EXISTS(SELECT 1 FROM learning_events e WHERE e.kind IN ('started','completed') AND NOT EXISTS(SELECT 1 FROM study_legacy_event_links l WHERE l.owner_user_id=e.owner_user_id AND l.legacy_event_id=e.id))`).Scan(&out.Done); e != nil {
			return out, studyError(e)
		}
	}
	completed, e := dbClock(ctx, tx)
	if e != nil {
		return out, studyError(e)
	}
	var before, after any
	if cursor != nil {
		before = string(body(cursor))
	}
	if out.Cursor != nil {
		after = string(body(out.Cursor))
	}
	if _, e = tx.ExecContext(ctx, `INSERT INTO study_migration_batches(id,started_at,completed_at,input_cursor,output_cursor,processed,created_events,linked_events,conflicts,done) VALUES($1,$2,$3,$4,$5,$6,$7,$8,0,$9)`, out.BatchID, started, completed, before, after, out.Processed, out.CreatedEvents, out.LinkedEvents, out.Done); e != nil {
		return out, studyError(e)
	}
	return out, studyError(tx.Commit())
}
func inspectStudyMigrationTx(ctx context.Context, tx *sql.Tx) (study.MigrationInspection, error) {
	out := study.MigrationInspection{}
	if e := cutoverConfigured(ctx, tx); e != nil {
		return out, studyError(e)
	}
	if e := studyConfigured(ctx, tx); e != nil {
		return out, studyError(e)
	}
	out.SchemaReady = true
	if e := tx.QueryRowContext(ctx, `SELECT count(*) FROM learning_events e WHERE e.kind IN ('started','completed') AND NOT EXISTS(SELECT 1 FROM study_legacy_event_links l WHERE l.owner_user_id=e.owner_user_id AND l.legacy_event_id=e.id)`).Scan(&out.UnmappedLegacyEvents); e != nil {
		return out, studyError(e)
	}
	if e := tx.QueryRowContext(ctx, `SELECT count(*) FROM study_legacy_event_links l JOIN learning_events e ON e.id=l.legacy_event_id AND e.owner_user_id=l.owner_user_id JOIN study_events s ON s.id=l.study_event_id WHERE l.source_sha<>e.seal_sha256 OR l.recorded_at<>e.recorded_at OR l.knowledge_publication_id<>e.knowledge_publication_id OR s.source_kind<>'legacy' OR s.origin_event_id<>e.id OR s.owner_user_id<>e.owner_user_id OR s.knowledge_id<>e.knowledge_id OR s.knowledge_version<>e.knowledge_version OR s.knowledge_sha256<>e.knowledge_sha256 OR s.kind<>e.kind OR s.recorded_at<>e.recorded_at OR s.taxonomy_version_id IS NOT NULL OR s.taxonomy_head IS NOT NULL`).Scan(&out.InvalidLinks); e != nil {
		return out, studyError(e)
	}
	var id string
	var done bool
	e := tx.QueryRowContext(ctx, `SELECT id::text,done,conflicts FROM study_migration_batches ORDER BY sequence DESC LIMIT 1`).Scan(&id, &done, &out.Conflicts)
	if e != nil && !errors.Is(e, sql.ErrNoRows) {
		return out, studyError(e)
	}
	if e == nil {
		out.MigrationBatchID = &id
		out.MigrationDone = done && out.UnmappedLegacyEvents == 0 && out.InvalidLinks == 0 && out.Conflicts == 0
	}
	return out, nil
}
func (s *Store) InspectStudyMigration(ctx context.Context) (study.MigrationInspection, error) {
	ctx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()
	tx, e := s.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true, Isolation: sql.LevelRepeatableRead})
	if e != nil {
		return study.MigrationInspection{}, studyError(e)
	}
	defer tx.Rollback()
	out, e := inspectStudyMigrationTx(ctx, tx)
	if e != nil {
		return out, e
	}
	return out, studyError(tx.Commit())
}
