package httpapi

import (
	"context"
	"github.com/yyl1212/math_master/backend/internal/auth"
	"github.com/yyl1212/math_master/backend/internal/study"
	"io"
)

func dispatchStudy(ctx context.Context, s *study.Service, r studyRoute, q study.ListQuery, h study.HistoryQuery, a study.Access, body io.Reader, limit int64) (any, error) {
	switch r.kind {
	case "overview":
		return s.ReadStudyOverview(ctx, a)
	case "topics":
		return s.ListStudyTopics(ctx, a, q)
	case "knowledge":
		return s.ListStudyKnowledge(ctx, a, q)
	case "readKnowledge":
		return s.ReadStudyKnowledge(ctx, a, r.id)
	case "history":
		return s.ListStudyHistory(ctx, a, h)
	case "readNote":
		return s.ReadStudyNote(ctx, a, r.id)
	case "saveNote":
		var in study.NoteInput
		if e := decodeStudyJSON(body, limit, &in); e != nil {
			return nil, e
		}
		return s.SaveStudyNote(ctx, a, r.id, in)
	case "deleteNote":
		var in study.NoteDeleteInput
		if e := decodeStudyJSON(body, limit, &in); e != nil {
			return nil, e
		}
		return s.DeleteStudyNote(ctx, a, r.id, in)
	case "finishReview":
		var in study.ReviewInput
		if e := decodeStudyJSON(body, limit, &in); e != nil {
			return nil, e
		}
		return s.FinishStudyReview(ctx, a, r.id, in)
	case "begin", "complete", "startReview":
		var in study.CommandInput
		if e := decodeStudyJSON(body, limit, &in); e != nil {
			return nil, e
		}
		switch r.kind {
		case "begin":
			return s.BeginStudy(ctx, a, r.id, in)
		case "complete":
			return s.CompleteStudy(ctx, a, r.id, in)
		default:
			return s.StartStudyReview(ctx, a, r.id, in)
		}
	}
	return nil, auth.ErrNotFound
}
