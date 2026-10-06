package taxonomy

import (
	"context"
	"github.com/yyl1212/math_master/backend/internal/auth"
	"github.com/yyl1212/math_master/backend/internal/publication"
)

type ReleasePage struct {
	Items  []ReleaseView `json:"items"`
	Total  int           `json:"total"`
	Limit  int           `json:"limit"`
	Offset int           `json:"offset"`
	Pair   PairRef       `json:"pair"`
}
type ManagementRepository interface {
	ReadDraftTopics(context.Context, publication.Access, string) (DraftTopicView, error)
	SaveDraftTopics(context.Context, publication.Access, string, DraftTopicInput) (DraftTopicView, error)
	ReadSubmissionTopics(context.Context, publication.Access, string) (DraftTopicView, error)
	ReadTopicRelease(context.Context, publication.Access, string) (ReleaseView, error)
	ListTopicReleases(context.Context, publication.Access, Query) (ReleasePage, error)
	PrepareTopicRelease(context.Context, publication.Access, PrepareInput) (ReleaseView, error)
	ActivateTopicRelease(context.Context, publication.Access, string, ActivateInput) (ReleaseView, error)
}

func (s *Service) management() (ManagementRepository, error) {
	r, ok := s.repo.(ManagementRepository)
	if !ok {
		return nil, ErrNotConfigured
	}
	return r, nil
}
func (s *Service) Preflight(ctx context.Context, a publication.Access, action publication.Action) (auth.User, error) {
	return s.guard.Preflight(ctx, a, action)
}
func (s *Service) AcquireValidation(ctx context.Context) (func(), error) {
	return s.guard.AcquireValidation(ctx)
}
func (s *Service) ReadDraftTopics(ctx context.Context, a publication.Access, id string) (DraftTopicView, error) {
	r, e := s.management()
	if e != nil {
		return DraftTopicView{}, e
	}
	return r.ReadDraftTopics(ctx, a, id)
}
func (s *Service) SaveDraftTopics(ctx context.Context, a publication.Access, id string, in DraftTopicInput) (DraftTopicView, error) {
	r, e := s.management()
	if e != nil {
		return DraftTopicView{}, e
	}
	return r.SaveDraftTopics(ctx, a, id, in)
}
func (s *Service) ReadSubmissionTopics(ctx context.Context, a publication.Access, id string) (DraftTopicView, error) {
	r, e := s.management()
	if e != nil {
		return DraftTopicView{}, e
	}
	return r.ReadSubmissionTopics(ctx, a, id)
}
func (s *Service) ReadTopicRelease(ctx context.Context, a publication.Access, id string) (ReleaseView, error) {
	r, e := s.management()
	if e != nil {
		return ReleaseView{}, e
	}
	return r.ReadTopicRelease(ctx, a, id)
}
func (s *Service) ListTopicReleases(ctx context.Context, a publication.Access, q Query) (ReleasePage, error) {
	r, e := s.management()
	if e != nil {
		return ReleasePage{}, e
	}
	return r.ListTopicReleases(ctx, a, q)
}
func (s *Service) PrepareTopicRelease(ctx context.Context, a publication.Access, in PrepareInput) (ReleaseView, error) {
	r, e := s.management()
	if e != nil {
		return ReleaseView{}, e
	}
	return r.PrepareTopicRelease(ctx, a, in)
}
func (s *Service) ActivateTopicRelease(ctx context.Context, a publication.Access, id string, in ActivateInput) (ReleaseView, error) {
	r, e := s.management()
	if e != nil {
		return ReleaseView{}, e
	}
	return r.ActivateTopicRelease(ctx, a, id, in)
}
