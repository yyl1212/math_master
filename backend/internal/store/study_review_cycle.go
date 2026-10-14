package store

import (
	"context"
	"database/sql"
	"github.com/yyl1212/math_master/backend/internal/knowledgeadmin"
	"github.com/yyl1212/math_master/backend/internal/study"
	"github.com/yyl1212/math_master/backend/internal/studyreview"
	"time"
)

func startReviewCycle(ctx context.Context, tx *sql.Tx, owner string, k knowledgeadmin.Knowledge, r knowledgeadmin.ManagedRecord, now time.Time) (string, error) {
	if r.FirstCompletedAt == nil {
		return "", studyreview.ErrConflict
	}
	if r.State == study.Reviewing && r.ActiveReviewID != nil {
		return *r.ActiveReviewID, nil
	}
	if r.State != study.Completed || r.Sequence >= study.MaxSequence {
		return "", studyreview.ErrConflict
	}
	next, e := study.TransitionState(r.State, study.StartReview)
	if e != nil {
		return "", studyreview.ErrConflict
	}
	id, e := shareID()
	if e != nil {
		return "", e
	}
	r.State = next
	r.Sequence++
	r.ActiveReviewID = &id
	if e = saveManagedRecord(ctx, tx, owner, r); e != nil {
		return "", e
	}
	_, e = tx.ExecContext(ctx, `INSERT INTO managed_study_events(owner_user_id,knowledge_id,knowledge_ref,topic_keys,kind,review_id,recorded_at) VALUES($1,$2,$3,$4,'review-started',$5,$6)`, owner, k.ID, knowledgeJSON(k.Ref), knowledgeJSON(k.TopicKeys), id, now)
	return id, e
}
func finishReviewCycle(ctx context.Context, tx *sql.Tx, owner string, k knowledgeadmin.Knowledge, r knowledgeadmin.ManagedRecord, reviewID string, now time.Time) error {
	if r.State == study.Completed && r.LastReviewID != nil && *r.LastReviewID == reviewID && r.LastReviewRef != nil && *r.LastReviewRef == k.Ref {
		return nil
	}
	if r.State != study.Reviewing || r.ActiveReviewID == nil || *r.ActiveReviewID != reviewID || r.Sequence >= study.MaxSequence {
		return studyreview.ErrConflict
	}
	next, e := study.TransitionState(r.State, study.FinishReview)
	if e != nil {
		return studyreview.ErrConflict
	}
	r.State = next
	r.Sequence++
	r.ActiveReviewID = nil
	r.LastReviewID = &reviewID
	r.LastReviewRef = &k.Ref
	r.LastReviewedAt = &now
	if e = saveManagedRecord(ctx, tx, owner, r); e != nil {
		return e
	}
	_, e = tx.ExecContext(ctx, `INSERT INTO managed_study_events(owner_user_id,knowledge_id,knowledge_ref,topic_keys,kind,review_id,recorded_at) VALUES($1,$2,$3,$4,'review-finished',$5,$6)`, owner, k.ID, knowledgeJSON(k.Ref), knowledgeJSON(k.TopicKeys), reviewID, now)
	return e
}
