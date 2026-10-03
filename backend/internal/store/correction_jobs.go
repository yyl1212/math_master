package store

import (
	"context"
	"database/sql"
	"errors"
	"github.com/yyl1212/math_master/backend/internal/auth"
	"github.com/yyl1212/math_master/backend/internal/correction"
	"github.com/yyl1212/math_master/backend/internal/question"
	"time"
)

func (s *Store) correctionSystemTx(ctx context.Context, locks bool, fn func(context.Context, *sql.Tx, time.Time) error) error {
	ctx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()
	tx, e := s.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if e != nil {
		return correctionError(e)
	}
	defer tx.Rollback()
	on, e := correctionConfigured(ctx, tx)
	if e != nil {
		return correctionError(e)
	}
	if !on {
		return correction.ErrNeverEnabled
	}
	ctx = correctionWithConfig(ctx, true)
	learningOn, e := learningConfigured(ctx, tx)
	if e != nil || !learningOn {
		return correction.ErrNotConfigured
	}
	if e = questionConfigured(ctx, tx); e != nil {
		return correction.ErrNotConfigured
	}
	if _, e = tx.ExecContext(ctx, `SET LOCAL lock_timeout='1s'`); e != nil {
		return correctionError(e)
	}
	if locks {
		if e = learningLocks(ctx, tx); e != nil {
			return correctionError(e)
		}
		if e = correctionRegistrationFence(ctx, tx, false); e != nil {
			return correctionError(e)
		}
	}
	now, e := dbClock(ctx, tx)
	if e != nil {
		return correctionError(e)
	}
	if e = fn(ctx, tx, now); e != nil {
		return correctionError(e)
	}
	return correctionError(tx.Commit())
}
func correctionOutboxContext(ctx context.Context, tx *sql.Tx) (context.Context, bool, error) {
	if value := ctx.Value(correctionConfigKey{}); value != nil {
		return ctx, correctionEnabled(ctx), nil
	}
	on, e := correctionConfigured(ctx, tx)
	return correctionWithConfig(ctx, on), on, e
}
func correctionEnqueueWithdrawal(ctx context.Context, tx *sql.Tx, space, id string, now time.Time) error {
	ctx, on, e := correctionOutboxContext(ctx, tx)
	if e != nil {
		return e
	}
	if !on {
		return nil
	}
	ref := correction.WithdrawalRef{Space: space, ID: id}
	if (space != "content" && space != "question") || !question.ValidID(id) {
		return auth.ErrInvalidInput
	}
	if e = correctionWithdrawalSource(ctx, tx, ref); e != nil {
		return e
	}
	var caseID string
	e = tx.QueryRowContext(ctx, `SELECT id::text FROM correction_cases WHERE withdrawal_space=$1 AND withdrawal_id=$2 AND sealed`, space, id).Scan(&caseID)
	if errors.Is(e, sql.ErrNoRows) {
		caseID, e = workflowID()
		if e != nil {
			return e
		}
		var contentID, questionID any
		if space == "content" {
			contentID = id
		} else {
			questionID = id
		}
		e = tx.QueryRowContext(ctx, `INSERT INTO correction_cases(id,kind,withdrawal_space,withdrawal_id,content_withdrawal_id,question_withdrawal_id,created_at) VALUES($1,'withdrawal',$2,$3,$4,$5,$6) ON CONFLICT(withdrawal_space,withdrawal_id) DO NOTHING RETURNING id::text`, caseID, space, id, contentID, questionID, now).Scan(&caseID)
		if errors.Is(e, sql.ErrNoRows) {
			e = tx.QueryRowContext(ctx, `SELECT id::text FROM correction_cases WHERE withdrawal_space=$1 AND withdrawal_id=$2 AND sealed`, space, id).Scan(&caseID)
		} else if e == nil {
			if e = correctionEvent(ctx, tx, "case", caseID, "case_registered", "", caseID, nil, 1, correction.CaseInput{Kind: correction.WithdrawalCase, Withdrawal: &ref}); e != nil {
				return e
			}
			_, e = tx.ExecContext(ctx, `UPDATE correction_cases SET sealed=true WHERE id=$1`, caseID)
		}
	}
	if e != nil {
		return e
	}
	return correctionEnqueueCase(ctx, tx, caseID, nil, now)
}
func correctionEnqueueTerminal(ctx context.Context, tx *sql.Tx, ref correction.EvidenceRef, owner string, now time.Time) error {
	ctx, on, e := correctionOutboxContext(ctx, tx)
	if e != nil {
		return e
	}
	if !on {
		return nil
	}
	if !correction.ValidEvidence(ref, false) || ref.Kind != correction.PracticeEvidence && ref.Kind != correction.AssessmentEvidence || !question.ValidID(owner) {
		return auth.ErrInvalidInput
	}
	raw, h, e := correction.Canonical("correction-command-v1", map[string]any{"evidence": ref})
	if e != nil {
		return e
	}
	// All matching cases are registered with one INSERT...SELECT, without loading
	// unbounded owners, mathematical bodies or original answers into Go memory.
	_, e = tx.ExecContext(ctx, `WITH evidence AS (`+correctionEvidenceRowsSQL+`), matched AS (SELECT c.id case_id,e.id,e.kind,e.owner FROM evidence e CROSS JOIN correction_cases c WHERE e.id=$1 AND e.kind=$2 AND e.owner=$3 AND e.terminal AND `+correctionCaseAffectsSQL+`), inserted AS (INSERT INTO correction_jobs(id,source_key,case_id,type,evidence_kind,evidence_id,owner_user_id,created_at,next_run_at) SELECT gen_random_uuid(),'terminal:'||case_id::text||':'||kind||':'||id::text,case_id,'attempt_terminal',kind,id,owner,$4,$4 FROM matched ON CONFLICT(source_key) DO NOTHING RETURNING id,case_id) INSERT INTO correction_events(id,subject_kind,subject_id,case_id,kind,sequence,body,body_bytes,body_digest) SELECT gen_random_uuid(),'job',id,case_id,'job_created',1,$5::jsonb,$6,$7 FROM inserted`, ref.ID, ref.Kind, owner, now, string(raw), raw, h)
	return e
}

type correctionJobRecord struct {
	Meta     correction.JobMetadata
	Source   string
	Owner    *string
	Evidence *correction.EvidenceRef
	Cursor   *correction.ScanKey
	Token    int64
	Until    *time.Time
}

const correctionJobColumns = `id::text,case_id::text,plan_id::text,plan_version,type,state,sequence,epoch,attempt,next_run_at,processed_count,error_class,source_key,owner_user_id::text,evidence_kind,evidence_id::text,cursor_kind,cursor_id::text,lease_token,lease_until`

func correctionScanJob(row interface{ Scan(...any) error }) (correctionJobRecord, error) {
	var j correctionJobRecord
	var pid, kind, id, ck, ci sql.NullString
	var pv sql.NullInt64
	e := row.Scan(&j.Meta.ID, &j.Meta.CaseID, &pid, &pv, &j.Meta.Type, &j.Meta.State, &j.Meta.Sequence, &j.Meta.Epoch, &j.Meta.Attempt, &j.Meta.NextRunAt, &j.Meta.ProcessedCount, &j.Meta.ErrorClass, &j.Source, &j.Owner, &kind, &id, &ck, &ci, &j.Token, &j.Until)
	if e != nil {
		return j, e
	}
	if pid.Valid {
		j.Meta.Plan = &correction.PlanRef{ID: pid.String, Version: int(pv.Int64)}
	}
	if id.Valid {
		j.Evidence = &correction.EvidenceRef{Kind: correction.EvidenceKind(kind.String), ID: id.String}
	}
	if ci.Valid {
		j.Cursor = &correction.ScanKey{Kind: correction.EvidenceKind(ck.String), ID: ci.String}
	}
	if j.Meta.NextRunAt != nil {
		v := j.Meta.NextRunAt.UTC()
		j.Meta.NextRunAt = &v
	}
	return j, nil
}
func correctionReadJob(ctx context.Context, tx *sql.Tx, id string, lock bool) (correctionJobRecord, error) {
	suffix := ""
	if lock {
		suffix = " FOR UPDATE"
	}
	j, e := correctionScanJob(tx.QueryRowContext(ctx, `SELECT `+correctionJobColumns+` FROM correction_jobs WHERE id=$1`+suffix, id))
	return j, workflowRowError(e)
}
func correctionLeaseMatches(j correctionJobRecord, l correction.Lease, now time.Time) bool {
	return j.Meta.State == correction.Running && j.Until != nil && now.Before(*j.Until) && j.Token == l.Token && j.Meta.Epoch == l.Epoch && j.Meta.Attempt == l.Attempt
}
func correctionValidLease(l correction.Lease) bool {
	return question.ValidID(l.JobID) && l.Token > 0 && l.Epoch > 0 && l.Attempt >= 1 && l.Attempt <= 8
}

var correctionBackoff = []time.Duration{5 * time.Second, 10 * time.Second, 20 * time.Second, 40 * time.Second, 80 * time.Second, 160 * time.Second, 300 * time.Second}

func correctionValidErrorClass(c string) bool {
	switch c {
	case "database", "deadline", "source", "configuration", "lease", "internal":
		return true
	}
	return false
}
func correctionJobFailure(ctx context.Context, tx *sql.Tx, j correctionJobRecord, class string, now time.Time, expired bool) error {
	state := correction.RetryWait
	var next any
	if j.Meta.Attempt >= 8 {
		state = correction.JobFailed
	} else {
		if j.Meta.Attempt < 1 {
			return correction.ErrConflict
		}
		next = now.Add(correctionBackoff[j.Meta.Attempt-1])
	}
	token := j.Token
	if expired {
		token++
	}
	seq := j.Meta.Sequence + 1
	if seq > correction.MaxSequence {
		return correction.ErrConflict
	}
	if _, e := tx.ExecContext(ctx, `UPDATE correction_jobs SET state=$2,sequence=$3,lease_token=$4,lease_until=NULL,next_run_at=$5,error_class=$6 WHERE id=$1`, j.Meta.ID, state, seq, token, next, class); e != nil {
		return e
	}
	return correctionEvent(ctx, tx, "job", j.Meta.ID, "job_attempt_failed", "", j.Meta.CaseID, nil, seq, map[string]any{"epoch": j.Meta.Epoch, "attempt": j.Meta.Attempt, "errorClass": class})
}
func (s *Store) ClaimCorrectionJob(ctx context.Context) (*correction.Lease, error) {
	var out *correction.Lease
	e := s.correctionSystemTx(ctx, false, func(ctx context.Context, tx *sql.Tx, now time.Time) error {
		j, e := correctionScanJob(tx.QueryRowContext(ctx, `SELECT `+correctionJobColumns+` FROM correction_jobs WHERE (state IN ('queued','retry_wait') AND next_run_at<=clock_timestamp()) OR (state='running' AND lease_until<=clock_timestamp()) ORDER BY coalesce(next_run_at,lease_until),created_at,id LIMIT 1 FOR UPDATE SKIP LOCKED`))
		if errors.Is(e, sql.ErrNoRows) {
			return nil
		}
		if e != nil {
			return e
		}
		now, e = dbClock(ctx, tx)
		if e != nil {
			return e
		}
		if j.Meta.State == correction.Running {
			return correctionJobFailure(ctx, tx, j, "deadline", now, true)
		}
		attempt := j.Meta.Attempt
		if attempt == 0 || j.Meta.State == correction.RetryWait {
			attempt++
		}
		if attempt > 8 || j.Meta.Sequence >= correction.MaxSequence {
			return correction.ErrConflict
		}
		token := j.Token + 1
		until := now.Add(30 * time.Second)
		seq := j.Meta.Sequence + 1
		if _, e = tx.ExecContext(ctx, `UPDATE correction_jobs SET state='running',sequence=$2,attempt=$3,lease_token=$4,lease_until=$5,next_run_at=NULL WHERE id=$1`, j.Meta.ID, seq, attempt, token, until); e != nil {
			return e
		}
		if e = correctionEvent(ctx, tx, "job", j.Meta.ID, "job_claimed", "", j.Meta.CaseID, nil, seq, map[string]any{"epoch": j.Meta.Epoch, "attempt": attempt, "token": token}); e != nil {
			return e
		}
		out = &correction.Lease{JobID: j.Meta.ID, Token: token, Until: until.UTC(), Epoch: j.Meta.Epoch, Attempt: attempt}
		return nil
	})
	if e != nil {
		return nil, e
	}
	return out, nil
}
func (s *Store) RenewCorrectionLease(ctx context.Context, l correction.Lease) (correction.Lease, error) {
	if !correctionValidLease(l) {
		return correction.Lease{}, auth.ErrInvalidInput
	}
	var out correction.Lease
	e := s.correctionSystemTx(ctx, false, func(ctx context.Context, tx *sql.Tx, _ time.Time) error {
		j, e := correctionReadJob(ctx, tx, l.JobID, true)
		if e != nil {
			return e
		}
		now, e := dbClock(ctx, tx)
		if e != nil {
			return e
		}
		if !correctionLeaseMatches(j, l, now) {
			return correction.ErrLeaseLost
		}
		if j.Meta.Sequence >= correction.MaxSequence {
			return correction.ErrConflict
		}
		until := now.Add(30 * time.Second)
		seq := j.Meta.Sequence + 1
		if _, e = tx.ExecContext(ctx, `UPDATE correction_jobs SET sequence=$2,lease_until=$3 WHERE id=$1`, j.Meta.ID, seq, until); e != nil {
			return e
		}
		if e = correctionEvent(ctx, tx, "job", j.Meta.ID, "job_renewed", "", j.Meta.CaseID, nil, seq, map[string]any{"token": j.Token}); e != nil {
			return e
		}
		out = l
		out.Until = until.UTC()
		return nil
	})
	return out, e
}
func (s *Store) FinishCorrectionJob(ctx context.Context, l correction.Lease, state correction.JobState, class string) error {
	if !correctionValidLease(l) || (state != correction.Queued && state != correction.Succeeded && state != correction.RetryWait && state != correction.JobFailed) || (state == correction.Queued || state == correction.Succeeded) && class != "" || (state == correction.RetryWait || state == correction.JobFailed) && !correctionValidErrorClass(class) {
		return auth.ErrInvalidInput
	}
	return s.correctionSystemTx(ctx, false, func(ctx context.Context, tx *sql.Tx, _ time.Time) error {
		j, e := correctionReadJob(ctx, tx, l.JobID, true)
		if e != nil {
			return e
		}
		now, e := dbClock(ctx, tx)
		if e != nil {
			return e
		}
		if !correctionLeaseMatches(j, l, now) {
			return correction.ErrLeaseLost
		}
		if state == correction.RetryWait || state == correction.JobFailed {
			return correctionJobFailure(ctx, tx, j, class, now, false)
		}
		if j.Meta.Sequence >= correction.MaxSequence {
			return correction.ErrConflict
		}
		seq := j.Meta.Sequence + 1
		kind := "job_succeeded"
		var next any
		if state == correction.Queued {
			kind = "job_continued"
			next = now
		}
		if _, e = tx.ExecContext(ctx, `UPDATE correction_jobs SET state=$2,sequence=$3,lease_until=NULL,next_run_at=$4,error_class=NULL WHERE id=$1`, j.Meta.ID, state, seq, next); e != nil {
			return e
		}
		return correctionEvent(ctx, tx, "job", j.Meta.ID, kind, "", j.Meta.CaseID, nil, seq, map[string]any{"processedCount": j.Meta.ProcessedCount, "epoch": j.Meta.Epoch})
	})
}
func (s *Store) RetryCorrectionJob(ctx context.Context, a question.Access, id string, in correction.RetryInput) (correction.Envelope[correction.Receipt], error) {
	var out correction.Envelope[correction.Receipt]
	if !question.ValidID(id) || correction.ValidateRetry(in) != nil {
		return out, auth.ErrInvalidInput
	}
	action := string(correction.RetryJobAction)
	resource := "jobs:" + id
	digest, e := correctionCommandDigest(action, resource, in)
	if e != nil {
		return out, e
	}
	ctx = correctionWithCommand(ctx, action, resource, a.IdempotencyKey)
	e = s.correctionTx(ctx, a, correction.RetryJobAction, nil, func(ctx context.Context, tx *sql.Tx, u auth.User, now time.Time) error {
		out.ActorID = u.ID
		receipt, found, e := correctionReplay(ctx, tx, u.ID, action, resource, a.IdempotencyKey, digest)
		if e != nil {
			return e
		}
		if found {
			out.Data = receipt
			return nil
		}
		j, e := correctionReadJob(ctx, tx, id, true)
		if e != nil {
			return e
		}
		if j.Meta.Sequence != in.ExpectedSequence || j.Meta.Sequence >= correction.MaxSequence || (j.Meta.State != correction.JobFailed && j.Meta.State != correction.RetryWait) {
			return correction.ErrConflict
		}
		seq := j.Meta.Sequence + 1
		epoch := j.Meta.Epoch + 1
		now, e = dbClock(ctx, tx)
		if e != nil {
			return e
		}
		if _, e = tx.ExecContext(ctx, `UPDATE correction_jobs SET state='queued',sequence=$2,epoch=$3,attempt=0,lease_token=lease_token+1,lease_until=NULL,next_run_at=$4 WHERE id=$1`, id, seq, epoch, now); e != nil {
			return e
		}
		if e = correctionEvent(ctx, tx, "job", id, "job_retry", u.ID, j.Meta.CaseID, nil, seq, map[string]any{"epoch": epoch, "priorErrorClass": j.Meta.ErrorClass}); e != nil {
			return e
		}
		j, e = correctionReadJob(ctx, tx, id, false)
		if e != nil {
			return e
		}
		out.Data = correction.Receipt{Status: 200, Job: &j.Meta}
		if e = correctionConsumeRate(ctx, tx, u.ID, "retry", now); e != nil {
			return e
		}
		return correctionRemember(ctx, tx, u.ID, action, resource, a.IdempotencyKey, digest, out.Data)
	})
	if e != nil {
		return correction.Envelope[correction.Receipt]{}, e
	}
	return out, nil
}

var _ correction.WorkerRepository = (*Store)(nil)
