package study

import (
	"context"
	"github.com/yyl1212/math_master/backend/internal/auth"
	"reflect"
	"time"
)

const Read Action = "read"

func Authorize(u auth.User, a Action) error {
	if !ValidID(u.ID) {
		return auth.ErrAuthenticationRequired
	}
	switch a {
	case Read, Begin, Complete, StartReview, FinishReview, SaveNote, DeleteNote:
	default:
		return auth.ErrInvalidInput
	}
	if u.MustChangePassword {
		return auth.ErrPasswordChangeRequired
	}
	for _, r := range u.Roles {
		if r == auth.RoleLearner {
			return nil
		}
	}
	return auth.ErrForbidden
}
func Rates(actor string, a Action) ([]auth.RateKey, error) {
	switch a {
	case Read, Begin, Complete, StartReview, FinishReview, SaveNote, DeleteNote:
	default:
		return nil, auth.ErrInvalidInput
	}
	if !ValidID(actor) {
		return nil, auth.ErrInvalidInput
	}
	scope, userScope, global, user := "content_write", "content_write_user", 120, 30
	if a == Read {
		scope, userScope, global, user = "auth_read", "content_read_user", 600, 120
	}
	return []auth.RateKey{{Scope: scope, Limit: global, Window: time.Minute}, {Scope: userScope, Key: actor, Limit: user, Window: time.Minute}}, nil
}

type Service struct{ repo Repository }

func NewService(repo Repository) (*Service, error) {
	if repo == nil {
		return nil, ErrNotConfigured
	}
	v := reflect.ValueOf(repo)
	if v.Kind() == reflect.Pointer && v.IsNil() {
		return nil, ErrNotConfigured
	}
	return &Service{repo}, nil
}
func (s *Service) Preflight(ctx context.Context, a Access, action Action) (auth.User, error) {
	if s == nil || s.repo == nil {
		return auth.User{}, ErrNotConfigured
	}
	u, e := s.repo.StudyPreflight(ctx, a, action)
	if e != nil {
		return auth.User{}, e
	}
	if e = Authorize(u, action); e != nil {
		return auth.User{}, e
	}
	rates, e := Rates(u.ID, action)
	if e == nil {
		e = s.repo.ConsumeRates(ctx, rates)
	}
	return u, e
}

func (s *Service) BeginStudy(ctx context.Context, a Access, id string, in CommandInput) (StudyDetail, error) {
	return s.repo.BeginStudy(ctx, a, id, in)
}
func (s *Service) CompleteStudy(ctx context.Context, a Access, id string, in CommandInput) (StudyDetail, error) {
	return s.repo.CompleteStudy(ctx, a, id, in)
}
func (s *Service) StartStudyReview(ctx context.Context, a Access, id string, in CommandInput) (StudyDetail, error) {
	return s.repo.StartStudyReview(ctx, a, id, in)
}
func (s *Service) FinishStudyReview(ctx context.Context, a Access, id string, in ReviewInput) (StudyDetail, error) {
	return s.repo.FinishStudyReview(ctx, a, id, in)
}
func (s *Service) ReadStudyOverview(ctx context.Context, a Access) (Overview, error) {
	return s.repo.ReadStudyOverview(ctx, a)
}
func (s *Service) ListStudyTopics(ctx context.Context, a Access, q ListQuery) (Page[TopicProgress], error) {
	return s.repo.ListStudyTopics(ctx, a, q)
}
func (s *Service) ListStudyKnowledge(ctx context.Context, a Access, q ListQuery) (Page[StudyDetail], error) {
	return s.repo.ListStudyKnowledge(ctx, a, q)
}
func (s *Service) ReadStudyKnowledge(ctx context.Context, a Access, id string) (StudyDetail, error) {
	return s.repo.ReadStudyKnowledge(ctx, a, id)
}
func (s *Service) ListStudyHistory(ctx context.Context, a Access, q HistoryQuery) (HistoryPage, error) {
	return s.repo.ListStudyHistory(ctx, a, q)
}
func (s *Service) ReadStudyNote(ctx context.Context, a Access, id string) (NoteView, error) {
	return s.repo.ReadStudyNote(ctx, a, id)
}
func (s *Service) SaveStudyNote(ctx context.Context, a Access, id string, in NoteInput) (NoteReceipt, error) {
	return s.repo.SaveStudyNote(ctx, a, id, in)
}
func (s *Service) DeleteStudyNote(ctx context.Context, a Access, id string, in NoteDeleteInput) (NoteReceipt, error) {
	return s.repo.DeleteStudyNote(ctx, a, id, in)
}
