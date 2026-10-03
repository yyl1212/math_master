package store

import (
	"context"
	"database/sql"
	"errors"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/yyl1212/math_master/backend/internal/auth"
	"github.com/yyl1212/math_master/backend/internal/correction"
	"github.com/yyl1212/math_master/backend/internal/question"
	"time"
)

var correctionTables = []string{"correction_cases", "correction_plans", "correction_jobs", "correction_results", "correction_dependencies", "correction_events", "correction_idempotency", "correction_rate_limits", "notifications", "notification_reads"}

// A permanent marker separates never-enabled legacy databases from damaged or
// rolled-back correction schemas. A version row alone is insufficient evidence.
func correctionConfigured(ctx context.Context, tx *sql.Tx) (bool, error) {
	var n int
	var goose, markerColumn, ever, version bool
	if e := tx.QueryRowContext(ctx, `SELECT count(*) FROM unnest($1::text[]) name WHERE to_regclass('public.'||name) IS NOT NULL`, correctionTables).Scan(&n); e != nil {
		return false, e
	}
	if e := tx.QueryRowContext(ctx, `SELECT to_regclass('public.goose_db_version') IS NOT NULL,EXISTS(SELECT 1 FROM information_schema.columns WHERE table_schema='public' AND table_name='goose_db_version' AND column_name='correction_enabled')`).Scan(&goose, &markerColumn); e != nil {
		return false, e
	}
	if goose {
		if e := tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM goose_db_version WHERE version_id=8 AND is_applied)`).Scan(&version); e != nil {
			return false, e
		}
	}
	if markerColumn {
		if e := tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM goose_db_version WHERE version_id=0 AND correction_enabled)`).Scan(&ever); e != nil {
			return false, e
		}
	}
	if n == 0 && !ever && !version && !markerColumn {
		return false, nil
	}
	if n != len(correctionTables) || !ever || !version {
		return false, correction.ErrNotConfigured
	}
	var intact bool
	if e := tx.QueryRowContext(ctx, `SELECT NOT EXISTS(SELECT 1 FROM jsonb_each_text($1::jsonb) r CROSS JOIN LATERAL unnest(string_to_array(r.value,',')) c WHERE NOT EXISTS(SELECT 1 FROM information_schema.columns i WHERE i.table_schema='public' AND i.table_name=r.key AND i.column_name=c))`, body(correctionColumns)).Scan(&intact); e != nil {
		return false, e
	}
	if !intact {
		return false, correction.ErrNotConfigured
	}
	var triggers int
	if e := tx.QueryRowContext(ctx, `SELECT count(*) FROM pg_trigger WHERE NOT tgisinternal AND tgenabled='O' AND tgname=ANY($1::text[])`, correctionTriggers).Scan(&triggers); e != nil {
		return false, e
	}
	if triggers != len(correctionTriggers) {
		return false, correction.ErrNotConfigured
	}
	return true, nil
}

var correctionColumns = map[string]string{
	"correction_cases":        "id,kind,withdrawal_space,withdrawal_id,content_withdrawal_id,question_withdrawal_id,rule_version,scope_kind,knowledge_id,knowledge_version,knowledge_sha256,cutoff,creator_user_id,sequence,sealed,created_at,last_backfill_at",
	"correction_plans":        "id,version,case_id,parent_id,parent_version,creator_user_id,status,sequence,algorithm_version,body,frozen_body,frozen_bytes,frozen_digest,sealed,created_at,updated_at",
	"correction_jobs":         "id,source_key,case_id,plan_id,plan_version,type,evidence_kind,evidence_id,owner_user_id,state,sequence,epoch,attempt,lease_token,lease_until,next_run_at,cursor_kind,cursor_id,processed_count,error_class,created_at",
	"correction_results":      "id,owner_user_id,case_id,plan_id,plan_version,parent_result_id,evidence_kind,evidence_id,knowledge_id,knowledge_version,knowledge_sha256,status,reason,score,passed,correctness,handled_case_ids,basis,basis_bytes,basis_digest,source_key,sealed,created_at",
	"correction_dependencies": "result_id,role,kind,id,version,sha256,positions,knowledge_id,unit_id,template_id,instance_id,blueprint_id,asset_sha256",
	"correction_events":       "id,subject_kind,subject_id,subject_version,case_id,plan_id,plan_version,job_id,result_id,owner_user_id,kind,sequence,actor_user_id,body,body_bytes,body_digest,recorded_at",
	"correction_idempotency":  "owner_user_id,action,resource,key,digest,status,receipt,created_at",
	"correction_rate_limits":  "id,owner_user_id,scope,command_key,consumed_at",
	"notifications":           "id,owner_user_id,dedup_key,type,evidence_kind,evidence_id,case_id,result_id,created_at",
	"notification_reads":      "owner_user_id,notification_id,read_at",
}
var correctionTriggers = []string{"correction_marker_immutable", "correction_case_source", "correction_case_registered", "correction_plan_freeze", "correction_plan_decision", "correction_job_advance", "correction_job_audit", "correction_result_freeze", "correction_result_basis", "correction_dependency_freeze", "correction_events_immutable", "correction_event_subject", "correction_receipts_immutable", "correction_rates_immutable", "notification_source", "notifications_immutable", "notification_reads_immutable"}

type correctionConfigKey struct{}

func correctionWithConfig(ctx context.Context, enabled bool) context.Context {
	return context.WithValue(ctx, correctionConfigKey{}, enabled)
}
func correctionEnabled(ctx context.Context) bool {
	v, _ := ctx.Value(correctionConfigKey{}).(bool)
	return v
}

// Registration serializes against qualification transactions, so a learner
// cannot commit a grant using a snapshot taken before the newly registered case.
const correctionRegistrationLock int64 = 1296127047

func correctionRegistrationFence(ctx context.Context, tx *sql.Tx, exclusive bool) error {
	if !correctionEnabled(ctx) {
		return nil
	}
	query := `SELECT pg_advisory_xact_lock_shared($1)`
	if exclusive {
		query = `SELECT pg_advisory_xact_lock($1)`
	}
	_, e := tx.ExecContext(ctx, query, correctionRegistrationLock)
	return e
}
func correctionError(e error) error {
	if e == nil {
		return nil
	}
	for _, known := range []error{correction.ErrNotConfigured, correction.ErrConflict, correction.ErrSourceStale, correction.ErrAnswerOverlap, correction.ErrLeaseLost, question.ErrIdempotencyConflict, auth.ErrInvalidInput, auth.ErrNotFound} {
		if errors.Is(e, known) {
			return e
		}
	}
	var rate *correction.RateError
	if errors.As(e, &rate) {
		return e
	}
	var pg *pgconn.PgError
	if errors.As(e, &pg) {
		switch pg.Code {
		case "42P01", "42703", "42883":
			return correction.ErrNotConfigured
		case "23505":
			return correction.ErrConflict
		}
	}
	return authError(e)
}
func correctionIdentity(ctx context.Context, tx *sql.Tx, a question.Access, action correction.Action, lock bool, related []string) (auth.User, time.Time, error) {
	u, _, now, e := managedIdentity(ctx, tx, a, correction.IsWrite(action), lock, related)
	if e != nil {
		return u, now, e
	}
	if e = correction.Authorize(u, action); e != nil {
		return u, now, e
	}
	if correction.IsWrite(action) && !question.ValidID(a.IdempotencyKey) {
		return u, now, auth.ErrInvalidInput
	}
	return u, now, nil
}
func (s *Store) correctionTx(ctx context.Context, a question.Access, action correction.Action, related []string, fn func(context.Context, *sql.Tx, auth.User, time.Time) error) error {
	ctx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()
	tx, e := s.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if e != nil {
		return correctionError(e)
	}
	defer tx.Rollback()
	enabled, e := correctionConfigured(ctx, tx)
	if e != nil {
		return correctionError(e)
	}
	if !enabled {
		return correction.ErrNotConfigured
	}
	ctx = correctionWithConfig(ctx, true)
	ok, e := learningConfigured(ctx, tx)
	if e != nil || !ok {
		return correction.ErrNotConfigured
	}
	if e = questionConfigured(ctx, tx); e != nil {
		return correction.ErrNotConfigured
	}
	if e = learningLocks(ctx, tx); e != nil {
		return correctionError(e)
	}
	if e = correctionRegistrationFence(ctx, tx, action == correction.CreateCaseAction); e != nil {
		return correctionError(e)
	}
	u, now, e := correctionIdentity(ctx, tx, a, action, true, related)
	if e != nil {
		return correctionError(e)
	}
	if e = fn(ctx, tx, u, now); e != nil {
		return correctionError(e)
	}
	if _, _, e = correctionIdentity(ctx, tx, a, action, false, nil); e != nil {
		return correctionError(e)
	}
	return correctionError(tx.Commit())
}
func (s *Store) CorrectionPreflight(ctx context.Context, a question.Access, action correction.Action) (auth.User, error) {
	var user auth.User
	e := s.correctionTx(ctx, a, action, nil, func(_ context.Context, _ *sql.Tx, u auth.User, _ time.Time) error { user = u; return nil })
	return user, e
}

var _ correction.Repository = (*Store)(nil)
