package store

import (
	"context"
	"database/sql"
	"errors"
	"github.com/yyl1212/math_master/backend/internal/auth"
	"github.com/yyl1212/math_master/backend/internal/knowledgeadmin"
	"github.com/yyl1212/math_master/backend/internal/study"
	"github.com/yyl1212/math_master/backend/internal/studyreview"
	"time"
)

func (s *Store) StartReviewSession(ctx context.Context, a studyreview.Access, in studyreview.StartInput) (out studyreview.CommandReceipt, e error) {
	if e = studyreview.ValidateStart(in); e != nil {
		return
	}
	e = s.knowledgeTx(ctx, a, true, false, func(ctx context.Context, tx *sql.Tx, u auth.User) error {
		if e := studyReviewConfigured(ctx, tx); e != nil {
			return e
		}
		r, found, e := reviewReplay(ctx, tx, u.ID, a, "start", in.Knowledge.ID, in)
		if e != nil {
			return e
		}
		if found {
			out = r
			return nil
		}
		k, e := managedStudyCurrent(ctx, tx, in.Knowledge.ID, true)
		if e != nil {
			return e
		}
		if !k.Published || k.Deleted {
			return studyreview.ErrUnavailable
		}
		if k.Ref != in.Knowledge {
			return studyreview.ErrStale
		}
		// Budget checks precede every business write, including the original cycle.
		m, e := studyreview.BuildReference(k.Point, in.Mode, in.Target)
		if e != nil {
			return e
		}
		record, e := readManagedRecord(ctx, tx, u.ID, k.ID, true)
		if e != nil {
			return e
		}
		if record.FirstCompletedAt == nil {
			return studyreview.ErrConflict
		}
		if _, e = readReviewSchedule(ctx, tx, u.ID, k.ID, true); e != nil {
			return e
		}
		var active string
		e = tx.QueryRowContext(ctx, `SELECT session_id::text FROM study_review_sessions WHERE owner_user_id=$1 AND knowledge_id=$2 AND status IN ('working','compared') FOR UPDATE`, u.ID, k.ID).Scan(&active)
		if e == nil {
			return &studyreview.ActiveSessionError{SessionID: active}
		}
		if !errors.Is(e, sql.ErrNoRows) {
			return e
		}
		now, e := dbClock(ctx, tx)
		if e != nil {
			return e
		}
		rid, e := startReviewCycle(ctx, tx, u.ID, k, record, now)
		if e != nil {
			return e
		}
		id, e := shareID()
		if e != nil {
			return e
		}
		current := studyreview.Session{ActorID: u.ID, SessionID: id, KnowledgeID: k.ID, ReviewID: rid, Knowledge: k.Ref, Mode: in.Mode, Target: in.Target, Status: studyreview.Working, Sequence: 1, Title: k.Point.Title, TitleZH: k.Point.TitleZH, KnowledgeType: k.Point.Type, CreatedAt: now, Available: true, Prompt: m.Prompt}
		if in.Mode == studyreview.Recall {
			empty := ""
			current.Answer = &empty
		} else {
			current.Reference = &studyreview.ExpectedReference{Sections: m.Expected, SHA256: m.SHA256}
		}
		if e = studyreview.ValidateSessionBudget(current); e != nil {
			return e
		}
		_, e = tx.ExecContext(ctx, `INSERT INTO study_review_sessions(session_id,owner_user_id,knowledge_id,review_id,knowledge_ref,mode,target,answer,created_at) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9)`, id, u.ID, k.ID, rid, knowledgeJSON(k.Ref), in.Mode, in.Target, current.Answer, now)
		if e != nil {
			return e
		}
		_, e = tx.ExecContext(ctx, `INSERT INTO study_review_snapshots(session_id,owner_user_id,knowledge_id,title,title_zh,knowledge_type,prompt,expected,sha256) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9)`, id, u.ID, k.ID, k.Point.Title, k.Point.TitleZH, k.Point.Type, knowledgeJSON(m.Prompt), knowledgeJSON(m.Expected), m.SHA256)
		if e != nil {
			return e
		}
		out, e = reviewEventReceipt(ctx, tx, a, current, "start", k.ID, in, now)
		return e
	})
	if e != nil {
		out = studyreview.CommandReceipt{}
	}
	return
}

type reviewMutation func(context.Context, *sql.Tx, knowledgeadmin.Knowledge, knowledgeadmin.ManagedRecord, *studyreview.Session, studyreview.FrozenMaterial, *studyreview.Schedule, time.Time) error

// A successful old receipt is confirmed before current content/state gates. The owned
// knowledge/record/schedule/session are then locked in that order for a new command.
func (s *Store) reviewCommand(ctx context.Context, a studyreview.Access, id, action string, input any, expected int64, mutate reviewMutation) (out studyreview.CommandReceipt, e error) {
	if !study.ValidID(id) {
		return out, studyreview.ErrInvalid
	}
	e = s.knowledgeTx(ctx, a, true, false, func(ctx context.Context, tx *sql.Tx, u auth.User) error {
		if e := studyReviewConfigured(ctx, tx); e != nil {
			return e
		}
		receipt, found, e := reviewReplay(ctx, tx, u.ID, a, action, id, input)
		if e != nil {
			return e
		}
		if found {
			out = receipt
			return nil
		}
		var kid string
		e = tx.QueryRowContext(ctx, `SELECT knowledge_id FROM study_review_sessions WHERE owner_user_id=$1 AND session_id=$2`, u.ID, id).Scan(&kid)
		if errors.Is(e, sql.ErrNoRows) {
			return studyreview.ErrNotFound
		}
		if e != nil {
			return e
		}
		k, e := managedStudyCurrent(ctx, tx, kid, true)
		if e != nil {
			return e
		}
		record, e := readManagedRecord(ctx, tx, u.ID, kid, true)
		if e != nil {
			return e
		}
		schedule, e := readReviewSchedule(ctx, tx, u.ID, kid, true)
		if e != nil {
			return e
		}
		current, m, e := readReviewRaw(ctx, tx, u.ID, id, true)
		if e != nil {
			return e
		}
		if current.Sequence != expected || current.Sequence >= studyreview.MaxSequence || current.Status == studyreview.Completed || current.Status == studyreview.Abandoned {
			return studyreview.ErrConflict
		}
		if action != "abandon" {
			if !k.Published || k.Deleted {
				return studyreview.ErrUnavailable
			}
			if current.Knowledge != k.Ref {
				return studyreview.ErrStale
			}
		}
		now, e := dbClock(ctx, tx)
		if e != nil {
			return e
		}
		if e = mutate(ctx, tx, k, record, &current, m, schedule, now); e != nil {
			return e
		}
		// Validate the complete readable shape too, including the currently visible
		// prompt, reference and schedule, before committing any business change.
		budget := current
		budget.Prompt = m.Prompt
		budget.Schedule = schedule
		if budget.Mode == studyreview.Read || budget.Status == studyreview.Compared || budget.Status == studyreview.Completed {
			budget.Reference = &studyreview.ExpectedReference{Sections: m.Expected, SHA256: m.SHA256}
		}
		if current.Grade != nil {
			budget.Schedule = &current.Grade.Schedule
		}
		if e = studyreview.ValidateSessionBudget(budget); e != nil {
			return e
		}
		current.Sequence++
		var grade any
		if current.Grade != nil {
			grade = knowledgeJSON(current.Grade)
		}
		_, e = tx.ExecContext(ctx, `UPDATE study_review_sessions SET status=$3,sequence=$4,draft_revision=$5,answer=$6,unanswered=$7,compared_at=$8,graded_at=$9,abandoned_at=$10,draft_updated_at=$11,grade=$12 WHERE owner_user_id=$1 AND session_id=$2`, u.ID, id, current.Status, current.Sequence, current.DraftRevision, current.Answer, current.Unanswered, current.ComparedAt, current.GradedAt, current.AbandonedAt, current.DraftUpdatedAt, grade)
		if e != nil {
			return e
		}
		if action == "grade" {
			v := current.Grade.Schedule
			_, e = tx.ExecContext(ctx, `INSERT INTO study_review_schedules(owner_user_id,knowledge_id,stage,due_at,last_ref,last_rating,sequence,graded_at,source_session_id) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9) ON CONFLICT(owner_user_id,knowledge_id) DO UPDATE SET stage=excluded.stage,due_at=excluded.due_at,last_ref=excluded.last_ref,last_rating=excluded.last_rating,sequence=excluded.sequence,graded_at=excluded.graded_at,source_session_id=excluded.source_session_id`, u.ID, kid, v.Stage, v.DueAt, knowledgeJSON(v.LastRef), v.LastRating, v.Sequence, v.GradedAt, id)
			if e != nil {
				return e
			}
		}
		out, e = reviewEventReceipt(ctx, tx, a, current, action, id, input, now)
		return e
	})
	if e != nil {
		out = studyreview.CommandReceipt{}
	}
	return
}
func (s *Store) SaveReviewDraft(ctx context.Context, a studyreview.Access, id string, in studyreview.DraftInput) (studyreview.CommandReceipt, error) {
	if e := studyreview.ValidateDraft(in); e != nil {
		return studyreview.CommandReceipt{}, e
	}
	return s.reviewCommand(ctx, a, id, "draft", in, in.ExpectedSequence, func(_ context.Context, _ *sql.Tx, _ knowledgeadmin.Knowledge, _ knowledgeadmin.ManagedRecord, current *studyreview.Session, _ studyreview.FrozenMaterial, _ *studyreview.Schedule, now time.Time) error {
		if current.Mode != studyreview.Recall || current.Status != studyreview.Working || current.DraftRevision != in.ExpectedDraftRevision || current.DraftRevision >= studyreview.MaxSequence {
			return studyreview.ErrConflict
		}
		current.Answer = &in.Body
		current.DraftRevision++
		current.DraftUpdatedAt = &now
		return studyreview.ValidateSessionBudget(*current)
	})
}
func (s *Store) CompareReviewSession(ctx context.Context, a studyreview.Access, id string, in studyreview.CompareInput) (studyreview.CommandReceipt, error) {
	if studyreview.ValidateCompare(studyreview.Recall, in) != nil && studyreview.ValidateCompare(studyreview.Read, in) != nil {
		return studyreview.CommandReceipt{}, studyreview.ErrInvalid
	}
	return s.reviewCommand(ctx, a, id, "compare", in, in.ExpectedSequence, func(_ context.Context, _ *sql.Tx, _ knowledgeadmin.Knowledge, _ knowledgeadmin.ManagedRecord, current *studyreview.Session, m studyreview.FrozenMaterial, _ *studyreview.Schedule, now time.Time) error {
		if current.Status != studyreview.Working || current.DraftRevision >= studyreview.MaxSequence {
			return studyreview.ErrConflict
		}
		if e := studyreview.ValidateCompare(current.Mode, in); e != nil {
			return e
		}
		if current.Mode == studyreview.Recall {
			current.Answer = &in.Body
		}
		current.Unanswered = in.Unanswered
		current.Status = studyreview.Compared
		current.ComparedAt = &now
		current.DraftRevision++
		current.Prompt = m.Prompt
		current.Reference = &studyreview.ExpectedReference{Sections: m.Expected, SHA256: m.SHA256}
		return studyreview.ValidateSessionBudget(*current)
	})
}
func (s *Store) GradeReviewSession(ctx context.Context, a studyreview.Access, id string, in studyreview.GradeInput) (studyreview.CommandReceipt, error) {
	if e := studyreview.ValidateGrade(false, in); e != nil {
		return studyreview.CommandReceipt{}, e
	}
	return s.reviewCommand(ctx, a, id, "grade", in, in.ExpectedSequence, func(ctx context.Context, tx *sql.Tx, k knowledgeadmin.Knowledge, r knowledgeadmin.ManagedRecord, current *studyreview.Session, _ studyreview.FrozenMaterial, previous *studyreview.Schedule, now time.Time) error {
		if current.Status != studyreview.Compared {
			return studyreview.ErrConflict
		}
		if e := studyreview.ValidateGrade(current.Unanswered, in); e != nil {
			return e
		}
		next, e := studyreview.NextSchedule(previous, current.Knowledge, in.Rating, now)
		if e != nil {
			return e
		}
		if e = finishReviewCycle(ctx, tx, current.ActorID, k, r, current.ReviewID, now); e != nil {
			return e
		}
		current.Status = studyreview.Completed
		current.GradedAt = &now
		current.Grade = &studyreview.GradeResult{Rating: in.Rating, GradedAt: now, Schedule: next}
		return nil
	})
}
func (s *Store) AbandonReviewSession(ctx context.Context, a studyreview.Access, id string, in studyreview.AbandonInput) (studyreview.CommandReceipt, error) {
	if e := studyreview.ValidateAbandon(in); e != nil {
		return studyreview.CommandReceipt{}, e
	}
	return s.reviewCommand(ctx, a, id, "abandon", in, in.ExpectedSequence, func(_ context.Context, _ *sql.Tx, _ knowledgeadmin.Knowledge, _ knowledgeadmin.ManagedRecord, current *studyreview.Session, _ studyreview.FrozenMaterial, _ *studyreview.Schedule, now time.Time) error {
		current.Status = studyreview.Abandoned
		current.AbandonedAt = &now
		return nil
	})
}
