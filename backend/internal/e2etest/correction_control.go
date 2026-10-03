package e2etest

import (
	"context"
	"database/sql"
	"errors"
	"github.com/yyl1212/math_master/backend/internal/auth"
	"github.com/yyl1212/math_master/backend/internal/correction"
	"github.com/yyl1212/math_master/backend/internal/store"
)

type correctionDatabaseState struct {
	Cases           int                  `json:"cases"`
	Plans           int                  `json:"plans"`
	Jobs            int                  `json:"jobs"`
	Results         int                  `json:"results"`
	Notifications   int                  `json:"notifications"`
	Reads           int                  `json:"reads"`
	Rates           int                  `json:"rates"`
	Grants          int                  `json:"grants"`
	CaseID          *string              `json:"caseId"`
	Plan            *correction.PlanRef  `json:"plan"`
	ResultID        *string              `json:"resultId"`
	AttemptID       *string              `json:"attemptId"`
	ActiveAttemptID *string              `json:"activeAttemptId"`
	Mappings        []correction.Mapping `json:"mappings"`
	OriginalSHA     string               `json:"originalSha"`
}

func correctionState(ctx context.Context, db *sql.DB) (correctionDatabaseState, error) {
	v := correctionDatabaseState{Mappings: []correction.Mapping{}}
	e := db.QueryRowContext(ctx, `SELECT (SELECT count(*) FROM correction_cases),(SELECT count(*) FROM correction_plans),(SELECT count(*) FROM correction_jobs),(SELECT count(*) FROM correction_results),(SELECT count(*) FROM notifications),(SELECT count(*) FROM notification_reads),(SELECT count(*) FROM correction_rate_limits),(SELECT count(*) FROM correction_events WHERE kind='qualification_granted')`).Scan(&v.Cases, &v.Plans, &v.Jobs, &v.Results, &v.Notifications, &v.Reads, &v.Rates, &v.Grants)
	if e != nil {
		return v, e
	}
	e = db.QueryRowContext(ctx, `SELECT id FROM correction_cases ORDER BY created_at DESC,id DESC LIMIT 1`).Scan(&v.CaseID)
	if e != nil && e != sql.ErrNoRows {
		return v, e
	}
	var p correction.PlanRef
	e = db.QueryRowContext(ctx, `SELECT id,version FROM correction_plans ORDER BY created_at DESC,id DESC LIMIT 1`).Scan(&p.ID, &p.Version)
	if e == nil {
		v.Plan = &p
	} else if e != sql.ErrNoRows {
		return v, e
	}
	e = db.QueryRowContext(ctx, `SELECT id FROM correction_results ORDER BY created_at DESC,id DESC LIMIT 1`).Scan(&v.ResultID)
	if e != nil && e != sql.ErrNoRows {
		return v, e
	}
	e = db.QueryRowContext(ctx, `SELECT a.id,encode(sha256(a.seal_bytes||convert_to(to_jsonb(r)::text,'UTF8')||convert_to((SELECT jsonb_agg(to_jsonb(x) ORDER BY position)::text FROM assessment_answers x WHERE x.attempt_id=a.id),'UTF8')),'hex') FROM assessment_attempts a JOIN assessment_results r ON r.attempt_id=a.id WHERE a.state='submitted' ORDER BY a.terminal_at,a.id LIMIT 1`).Scan(&v.AttemptID, &v.OriginalSHA)
	if e != nil && e != sql.ErrNoRows {
		return v, e
	}
	e = db.QueryRowContext(ctx, `SELECT id FROM assessment_attempts WHERE state='active' LIMIT 1`).Scan(&v.ActiveAttemptID)
	if e != nil && e != sql.ErrNoRows {
		return v, e
	}
	v.Mappings, e = correctionMappings(ctx, db)
	return v, e
}
func correctionChange(ctx context.Context, db *sql.DB, s *store.Store, accounts *auth.Service, root, scene string) (bool, error) {
	switch scene {
	case "correction-withdraw-instance":
		return learningChange(ctx, db, s, accounts, root, "learning-withdraw-first-assessed")
	case "correction-publish-equivalent":
		return true, publishCorrectionEquivalent(ctx, db, s, accounts, root)
	case "correction-rule-case":
		a, e := fixtureAccess(ctx, accounts, "content_admin", true)
		if e != nil {
			return true, e
		}
		_, e = s.CreateCorrectionCase(ctx, a, correction.CaseInput{Kind: correction.GradingRuleCase, Rule: &correction.RuleScope{Kind: "all", RuleVersion: 1}})
		return true, e
	case "correction-approve-empty", "correction-approve-equivalent":
		return true, correctionApprove(ctx, db, s, accounts, scene == "correction-approve-equivalent")
	case "correction-active-replacement", "correction-active-rule":
		_, e := correctionAttempt(ctx, db, s, accounts)
		return true, e
	case "correction-run":
		if _, e := s.BackfillCorrections(ctx, 50); e != nil {
			return true, e
		}
		for n := 0; n < 10; n++ {
			l, e := s.ClaimCorrectionJob(ctx)
			if e != nil {
				return true, e
			}
			if l == nil {
				return true, nil
			}
			if _, e = s.ProcessCorrectionJob(ctx, *l, 50); e != nil {
				return true, e
			}
		}
		return true, errors.New("bounded correction fixture did not drain")
	case "correction-revoke-owner":
		v, d, e := accounts.Context(ctx, auth.Cookies{})
		if e != nil {
			return true, e
		}
		_, login, e := accounts.Login(ctx, auth.Cookies{Preauth: d.SetPreauth}, v.CSRFToken, auth.LoginInput{Username: "auth_learner", Password: fixturePassword}, "correction-revoke-login")
		if e != nil {
			return true, e
		}
		cookies := auth.Cookies{Session: login.SetSession}
		v, _, e = accounts.Context(ctx, cookies)
		if e != nil {
			return true, e
		}
		_, e = accounts.Logout(ctx, cookies, v.CSRFToken, true, "correction-owner-revoked")
		return true, e
	}
	return false, nil
}
