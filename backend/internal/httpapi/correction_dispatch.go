package httpapi

import (
	"context"
	"github.com/yyl1212/math_master/backend/internal/auth"
	"github.com/yyl1212/math_master/backend/internal/correction"
	"github.com/yyl1212/math_master/backend/internal/question"
	"net/http"
)

func dispatchCorrection(ctx context.Context, r *http.Request, route correctionRoute, a question.Access, q correction.Query, evidence correction.EvidenceRef, s *correction.Service) (any, int, error) {
	if route.SHA != "" {
		v, e := s.ReadOwnCorrectionAsset(ctx, a, route.ID, route.SHA)
		return v, 200, e
	}
	ref := correction.PlanRef{ID: route.ID, Version: route.Version}
	switch route.Action {
	case correction.ListCasesAction:
		v, e := s.ListCorrectionCases(ctx, a, q)
		return v, 200, e
	case correction.ReadCaseAction:
		v, e := s.ReadCorrectionCase(ctx, a, route.ID)
		return v, 200, e
	case correction.ListPlansAction:
		v, e := s.ListCorrectionPlans(ctx, a, route.ID, q)
		return v, 200, e
	case correction.ReadPlanAction:
		v, e := s.ReadCorrectionPlan(ctx, a, ref, route.Detail)
		return v, 200, e
	case correction.ListJobsAction:
		v, e := s.ListCorrectionJobs(ctx, a, route.ID, q)
		return v, 200, e
	case correction.ListOwnAction:
		v, e := s.ListOwnCorrections(ctx, a, evidence, q)
		return v, 200, e
	case correction.ReadOwnAction, correction.ReadOwnDetailAction:
		v, e := s.ReadOwnCorrection(ctx, a, route.ID, route.Detail)
		return v, 200, e
	case correction.CreateCaseAction, correction.CreatePlanAction, correction.UpdatePlanAction, correction.SubmitPlanAction, correction.DecidePlanAction, correction.RetryJobAction:
		in, e := readCorrectionInput(r.Body, route.Action)
		if e != nil {
			return nil, 0, e
		}
		var v correction.Envelope[correction.Receipt]
		switch route.Action {
		case correction.CreateCaseAction:
			v, e = s.CreateCorrectionCase(ctx, a, in.(correction.CaseInput))
		case correction.CreatePlanAction:
			v, e = s.CreateCorrectionPlan(ctx, a, route.ID, in.(correction.PlanInput))
		case correction.UpdatePlanAction:
			v, e = s.UpdateCorrectionPlan(ctx, a, ref, in.(correction.PlanInput))
		case correction.SubmitPlanAction:
			v, e = s.SubmitCorrectionPlan(ctx, a, ref, in.(correction.SubmitInput))
		case correction.DecidePlanAction:
			v, e = s.DecideCorrectionPlan(ctx, a, ref, in.(correction.DecisionInput))
		case correction.RetryJobAction:
			v, e = s.RetryCorrectionJob(ctx, a, route.ID, in.(correction.RetryInput))
		}
		return v, v.Data.Status, e
	}
	return nil, 0, auth.ErrNotFound
}
