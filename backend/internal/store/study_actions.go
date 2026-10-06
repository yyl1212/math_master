package store

import (
	"context"
	"crypto/rand"
	"database/sql"
	"github.com/yyl1212/math_master/backend/internal/auth"
	"github.com/yyl1212/math_master/backend/internal/study"
	"time"
)

func (s *Store) studyCommand(ctx context.Context, a study.Access, id string, action study.Action, in study.CommandInput, reviewID string) (study.StudyDetail, error) {
	var out study.StudyDetail
	if !study.ValidKnowledgeID(id) || in.Knowledge.ID != id || study.ValidateCommand(in) != nil || (action == study.FinishReview && !study.ValidID(reviewID)) {
		return out, study.ErrInvalid
	}
	var bound any = in
	if action == study.FinishReview {
		bound = study.ReviewInput{CommandInput: in, ReviewID: reviewID}
	}
	e := s.studyTx(ctx, a, action, func(ctx context.Context, tx *sql.Tx, u auth.User, now time.Time) error {
		scope, e := studyScopeTx(ctx, tx)
		if e != nil {
			return e
		}
		current, ok := scope.knowledge[id]
		if !ok {
			return auth.ErrNotFound
		}
		if scope.pair.KnowledgeHead == nil || *scope.pair.KnowledgeHead != in.ExpectedKnowledgeHead || current.KnowledgeRef != in.Knowledge {
			return study.ErrVersionStale
		}
		replay, found, e := studyReplay[study.StudyDetail](ctx, tx, u.ID, a, action, id, bound)
		if e != nil {
			return e
		}
		if found {
			if replay.ActorID != u.ID || replay.Record.KnowledgeID != id {
				return study.ErrNotConfigured
			}
			currentRecord, err := studyReadRecordTx(ctx, tx, u.ID, id, true)
			if err != nil {
				return err
			}
			out = studyDetail(u.ID, currentRecord, scope)
			return nil
		}
		r, e := studyReadRecordTx(ctx, tx, u.ID, id, true)
		if e != nil {
			return e
		}
		changed := true
		event := ""
		var eventReviewID *string
		switch action {
		case study.Begin:
			changed = r.State == study.Unlearned
			event = "started"
			r.LastReadAt = &now
		case study.Complete:
			changed = !(r.CompletedRef != nil && *r.CompletedRef == current.KnowledgeRef && (r.State == study.Completed || r.State == study.Reviewing))
			event = "completed"
		case study.StartReview:
			changed = r.State != study.Reviewing
			event = "review-started"
			if !changed && r.ActiveReviewID == nil {
				return study.ErrNotConfigured
			}
		case study.FinishReview:
			if r.State == study.Completed && r.LastReviewID != nil && *r.LastReviewID == reviewID {
				changed = false
			} else if r.State != study.Reviewing || r.ActiveReviewID == nil || *r.ActiveReviewID != reviewID {
				return study.ErrStateConflict
			}
			event = "review-finished"
		default:
			return study.ErrInvalid
		}
		if changed {
			completed := study.KnowledgeRef{}
			if r.CompletedRef != nil {
				completed = *r.CompletedRef
			}
			next, e := study.ApplyState(r.State, action, completed, current.KnowledgeRef)
			if e != nil {
				return e
			}
			if r.Sequence != in.ExpectedSequence || r.Sequence >= study.MaxSequence {
				return study.ErrStateConflict
			}
			r.Sequence++
			r.State = next
			switch action {
			case study.Begin:
				if r.FirstStartedAt == nil {
					r.FirstStartedAt = &now
				}
			case study.Complete:
				if r.FirstCompletedAt == nil {
					r.FirstCompletedAt = &now
				}
				r.LastCompletedAt = &now
				r.CompletedRef = &current.KnowledgeRef
			case study.StartReview:
				rid, e := auth.NewID(rand.Reader)
				if e != nil {
					return e
				}
				r.ActiveReviewID = &rid
				eventReviewID = &rid
			case study.FinishReview:
				r.ActiveReviewID = nil
				r.LastReviewID = &reviewID
				r.LastReviewRef = &current.KnowledgeRef
				r.LastReviewedAt = &now
				eventReviewID = &reviewID
			}
		}
		if changed || action == study.Begin {
			if e = studySaveRecordTx(ctx, tx, u.ID, r, current.KnowledgeRef, now); e != nil {
				return e
			}
		}
		if changed {
			if e = studyEventTx(ctx, tx, u.ID, a, action, current.KnowledgeRef, scope.pair.TaxonomyVersionID, event, nil, eventReviewID, now); e != nil {
				return e
			}
		}
		out = studyDetail(u.ID, r, scope)
		return studyRemember(ctx, tx, u.ID, a, action, id, bound, out)
	})
	return out, e
}
func (s *Store) BeginStudy(ctx context.Context, a study.Access, id string, in study.CommandInput) (study.StudyDetail, error) {
	return s.studyCommand(ctx, a, id, study.Begin, in, "")
}
func (s *Store) CompleteStudy(ctx context.Context, a study.Access, id string, in study.CommandInput) (study.StudyDetail, error) {
	return s.studyCommand(ctx, a, id, study.Complete, in, "")
}
func (s *Store) StartStudyReview(ctx context.Context, a study.Access, id string, in study.CommandInput) (study.StudyDetail, error) {
	return s.studyCommand(ctx, a, id, study.StartReview, in, "")
}
func (s *Store) FinishStudyReview(ctx context.Context, a study.Access, id string, in study.ReviewInput) (study.StudyDetail, error) {
	return s.studyCommand(ctx, a, id, study.FinishReview, in.CommandInput, in.ReviewID)
}
