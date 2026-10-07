package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"github.com/yyl1212/math_master/backend/internal/study"
	"sort"
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
	if _, e = readExperienceModeTx(ctx, tx, true); e != nil {
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
	sort.Strings(ids)
	for _, id := range ids {
		var locked string
		if e = tx.QueryRowContext(ctx, "SELECT id::text FROM auth_users WHERE id=$1 FOR UPDATE", id).Scan(&locked); e != nil {
			return out, studyError(e)
		}
	}
	for _, f := range facts {
		var eventID, sha string
		e = tx.QueryRowContext(ctx, "SELECT study_event_id::text,source_sha FROM study_legacy_event_links WHERE owner_user_id=$1 AND legacy_event_id=$2", f.actor, f.id).Scan(&eventID, &sha)
		if e == nil {
			if sha != f.sourceSHA {
				return out, study.ErrNotConfigured
			}
			out.LinkedEvents++
		} else if !errors.Is(e, sql.ErrNoRows) {
			return out, studyError(e)
		} else {
			var raw []byte
			e = tx.QueryRowContext(ctx, "SELECT body FROM study_records WHERE owner_user_id=$1 AND knowledge_id=$2 FOR UPDATE", f.actor, f.ref.ID).Scan(&raw)
			fresh := errors.Is(e, sql.ErrNoRows)
			if e != nil && !fresh {
				return out, studyError(e)
			}
			record := study.StudyRecord{KnowledgeID: f.ref.ID, State: study.Unlearned}
			if !fresh && json.Unmarshal(raw, &record) != nil {
				return out, study.ErrNotConfigured
			}
			managed := fresh
			if !fresh && record.Sequence == 0 && record.LastReadAt == nil {
				if e = tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM study_legacy_event_links WHERE owner_user_id=$1 AND knowledge_id=$2 AND projected_record) AND NOT EXISTS(SELECT 1 FROM study_events WHERE owner_user_id=$1 AND knowledge_id=$2 AND source_kind='native') AND NOT EXISTS(SELECT 1 FROM study_idempotency WHERE owner_user_id=$1 AND knowledge_id=$2) AND NOT EXISTS(SELECT 1 FROM study_notes WHERE owner_user_id=$1 AND knowledge_id=$2)`, f.actor, f.ref.ID).Scan(&managed); e != nil {
					return out, studyError(e)
				}
			}
			if managed {
				at := f.at
				if f.kind == "started" {
					if record.FirstStartedAt == nil || at.Before(*record.FirstStartedAt) {
						record.FirstStartedAt = &at
					}
					if record.FirstCompletedAt == nil {
						record.State = study.Learning
					}
				}
				if f.kind == "completed" {
					if record.FirstCompletedAt == nil || at.Before(*record.FirstCompletedAt) {
						record.FirstCompletedAt = &at
					}
					if record.LastCompletedAt == nil || !at.Before(*record.LastCompletedAt) {
						record.LastCompletedAt = &at
						ref := f.ref
						record.CompletedRef = &ref
					}
					record.State = study.Completed
				}
				lastRef := f.ref
				if record.CompletedRef != nil {
					lastRef = *record.CompletedRef
				}
				if e = studySaveRecordTx(ctx, tx, f.actor, record, lastRef, started); e != nil {
					return out, studyError(e)
				}
			}
			if e = tx.QueryRowContext(ctx, `INSERT INTO study_events(id,owner_user_id,knowledge_id,knowledge_version,knowledge_sha256,kind,recorded_at,source_kind,origin_event_id,action,idempotency_key) VALUES(gen_random_uuid(),$1,$2,$3,$4,$5,$6,'legacy',$7,'legacy-migration',$7) RETURNING id::text`, f.actor, f.ref.ID, f.ref.Version, f.ref.SHA256, f.kind, f.at, f.id).Scan(&eventID); e != nil {
				return out, studyError(e)
			}
			if _, e = tx.ExecContext(ctx, `INSERT INTO study_legacy_event_links(owner_user_id,legacy_event_id,study_event_id,knowledge_id,source_sha,knowledge_publication_id,projected_record,batch_id,recorded_at,linked_at) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`, f.actor, f.id, eventID, f.ref.ID, f.sourceSHA, f.pub, managed, out.BatchID, f.at, started); e != nil {
				return out, studyError(e)
			}
			out.CreatedEvents++
		}
		out.Processed++
		out.Cursor = &study.LegacyCursor{RecordedAt: f.at, EventID: f.id}
	}
	if e = tx.QueryRowContext(ctx, `SELECT NOT EXISTS(SELECT 1 FROM learning_events e WHERE e.kind IN ('started','completed') AND NOT EXISTS(SELECT 1 FROM study_legacy_event_links l WHERE l.owner_user_id=e.owner_user_id AND l.legacy_event_id=e.id))`).Scan(&out.Done); e != nil {
		return out, studyError(e)
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
