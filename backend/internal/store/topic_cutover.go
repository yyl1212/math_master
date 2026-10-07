package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"github.com/yyl1212/math_master/backend/internal/publication"
	"github.com/yyl1212/math_master/backend/internal/study"
	"github.com/yyl1212/math_master/backend/internal/taxonomy"
	"regexp"
	"time"
)

var trustedCodePattern = regexp.MustCompile(`^[a-f0-9]{40}$`)
var backupRecordPattern = regexp.MustCompile(`^[a-f0-9]{64}$`)

func (s *Store) inspectTopicCutoverTx(ctx context.Context, tx *sql.Tx) (study.CutoverReport, error) {
	out := study.CutoverReport{CodeCompatible: trustedCodePattern.MatchString(s.codeSHA)}
	if e := tx.QueryRowContext(ctx, "SELECT experience_mode FROM topic_learning_state WHERE singleton").Scan(&out.Mode); e != nil {
		return out, studyError(e)
	}
	if e := taxonomyConfigured(ctx, tx); e != nil {
		return out, nil
	}
	if e := studyConfigured(ctx, tx); e != nil {
		return out, nil
	}
	if e := cutoverConfigured(ctx, tx); e != nil {
		return out, nil
	}
	out.SchemaReady = true
	migration, e := inspectStudyMigrationTx(ctx, tx)
	if e != nil {
		return out, e
	}
	out.MigrationDone = migration.MigrationDone
	out.UnmappedLegacyEvents = migration.UnmappedLegacyEvents
	out.Conflicts = migration.Conflicts + migration.InvalidLinks
	out.MigrationBatchID = migration.MigrationBatchID
	scope, e := studyScopeTx(ctx, tx)
	if e == nil {
		out.Pair = &scope.pair
	} else if !errors.Is(e, study.ErrNotConfigured) && !errors.Is(e, taxonomy.ErrNotConfigured) {
		return out, studyError(e)
	}
	on, e := learningConfigured(ctx, tx)
	if e != nil {
		return out, studyError(e)
	}
	if on {
		if e = questionConfigured(ctx, tx); e == nil {
			_, e = correctionConfigured(ctx, tx)
			out.HistoryReady = e == nil
		}
	} else {
		out.HistoryReady = true
	}
	var id string
	var at time.Time
	e = tx.QueryRowContext(ctx, "SELECT id::text,recorded_at FROM topic_cutovers WHERE singleton").Scan(&id, &at)
	if e != nil && !errors.Is(e, sql.ErrNoRows) {
		return out, studyError(e)
	}
	if e == nil {
		out.CutoverID = &id
		at = at.UTC()
		out.RecordedAt = &at
	}
	return out, nil
}
func (s *Store) InspectTopicCutover(ctx context.Context) (study.CutoverReport, error) {
	ctx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()
	tx, e := s.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true, Isolation: sql.LevelRepeatableRead})
	if e != nil {
		return study.CutoverReport{}, studyError(e)
	}
	defer tx.Rollback()
	out, e := s.inspectTopicCutoverTx(ctx, tx)
	if e != nil {
		return out, e
	}
	return out, studyError(tx.Commit())
}
func (s *Store) ActivateTopicExperience(ctx context.Context, in study.CutoverInput) (study.CutoverReport, error) {
	out := study.CutoverReport{}
	if !validTopicPair(in.ExpectedPair) || in.ExpectedPair.TaxonomyHead == nil || !trustedCodePattern.MatchString(in.CodeSHA) || in.CodeSHA != s.codeSHA || !study.ValidID(in.ExpectedMigrationBatchID) || !publication.ValidNote(in.Reason) || !backupRecordPattern.MatchString(in.BackupRecord) {
		return out, study.ErrCutoverNotReady
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
	var mode taxonomy.ExperienceMode
	if e = tx.QueryRowContext(ctx, "SELECT experience_mode FROM topic_learning_state WHERE singleton FOR UPDATE").Scan(&mode); e != nil {
		return out, studyError(e)
	}
	for _, lock := range []int64{adminLockID, 1296127048} {
		if _, e = tx.ExecContext(ctx, "SELECT pg_advisory_xact_lock($1)", lock); e != nil {
			return out, studyError(e)
		}
	}
	out, e = s.inspectTopicCutoverTx(ctx, tx)
	if e != nil {
		return out, e
	}
	if !out.SchemaReady || !out.HistoryReady || !out.CodeCompatible {
		return out, study.ErrCutoverNotReady
	}
	if mode == taxonomy.ModeTopics {
		if out.CutoverID == nil {
			return out, study.ErrCutoverNotReady
		}
		return out, studyError(tx.Commit())
	}
	if mode != taxonomy.ModeLegacy || out.Pair == nil || !taxonomy.SamePair(*out.Pair, in.ExpectedPair) || out.MigrationBatchID == nil || *out.MigrationBatchID != in.ExpectedMigrationBatchID {
		return out, study.ErrCutoverConflict
	}
	if !out.MigrationDone || out.UnmappedLegacyEvents != 0 || out.Conflicts != 0 {
		return out, study.ErrCutoverNotReady
	}
	// Inspect's NOT EXISTS runs after configuration and all old writers are fenced.
	var done bool
	if e = tx.QueryRowContext(ctx, "SELECT done FROM study_migration_batches WHERE id=$1 FOR SHARE", in.ExpectedMigrationBatchID).Scan(&done); e != nil || !done {
		return out, study.ErrCutoverNotReady
	}
	id, e := workflowID()
	if e != nil {
		return out, e
	}
	at, e := dbClock(ctx, tx)
	if e != nil {
		return out, studyError(e)
	}
	raw, e := json.Marshal(in.ExpectedPair)
	if e != nil {
		return out, e
	}
	if _, e = tx.ExecContext(ctx, `INSERT INTO topic_cutovers(singleton,id,expected_pair,code_sha,migration_batch_id,reason,backup_record,recorded_at) VALUES(true,$1,$2,$3,$4,$5,$6,$7)`, id, string(raw), s.codeSHA, in.ExpectedMigrationBatchID, in.Reason, in.BackupRecord, at); e != nil {
		return out, studyError(e)
	}
	if _, e = tx.ExecContext(ctx, "UPDATE topic_learning_state SET experience_mode='topics',retired_at=$1 WHERE singleton", at); e != nil {
		return out, studyError(e)
	}
	out.Mode = taxonomy.ModeTopics
	out.CutoverID = &id
	at = at.UTC()
	out.RecordedAt = &at
	out.Activated = true
	return out, studyError(tx.Commit())
}
