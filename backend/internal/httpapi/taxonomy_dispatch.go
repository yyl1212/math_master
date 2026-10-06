package httpapi

import (
	"context"
	"encoding/json"
	"github.com/yyl1212/math_master/backend/internal/auth"
	"github.com/yyl1212/math_master/backend/internal/publication"
	"github.com/yyl1212/math_master/backend/internal/taxonomy"
	"io"
)

func jsonMarshalTaxonomy(v any) (json.RawMessage, error) {
	raw, e := json.Marshal(v)
	if e != nil {
		return nil, e
	}
	if len(raw) > 2<<20 {
		return nil, taxonomy.ErrLimit
	}
	return json.RawMessage(raw), nil
}
func dispatchPublicTaxonomy(ctx context.Context, s *taxonomy.Service, r taxonomyRoute, q taxonomy.Query) (any, error) {
	switch r.kind {
	case "readExperience":
		mode, e := s.ReadExperienceMode(ctx)
		return map[string]taxonomy.ExperienceMode{"mode": mode}, e
	case "listTopics":
		return s.ListTopics(ctx, q)
	case "readTopic":
		return s.ReadTopic(ctx, r.id)
	case "listKnowledge":
		return s.ListTopicKnowledge(ctx, r.id, q)
	}
	return nil, auth.ErrNotFound
}
func dispatchManagedTaxonomy(ctx context.Context, s *taxonomy.Service, r taxonomyRoute, q taxonomy.Query, a publication.Access, body io.Reader) (any, error) {
	switch r.kind {
	case "readDraft":
		return s.ReadDraftTopics(ctx, a, r.id)
	case "readSubmission":
		return s.ReadSubmissionTopics(ctx, a, r.id)
	case "readRelease":
		return s.ReadTopicRelease(ctx, a, r.id)
	case "listReleases":
		return s.ListTopicReleases(ctx, a, q)
	case "saveDraft":
		var v taxonomy.DraftTopicInput
		if e := decodeTaxonomyJSON(body, &v); e != nil {
			return nil, e
		}
		return s.SaveDraftTopics(ctx, a, r.id, v)
	case "prepareRelease":
		var v taxonomy.PrepareInput
		if e := decodeTaxonomyJSON(body, &v); e != nil {
			return nil, e
		}
		return s.PrepareTopicRelease(ctx, a, v)
	case "activateRelease":
		var v taxonomy.ActivateInput
		if e := decodeTaxonomyJSON(body, &v); e != nil {
			return nil, e
		}
		return s.ActivateTopicRelease(ctx, a, r.id, v)
	}
	return nil, auth.ErrNotFound
}
