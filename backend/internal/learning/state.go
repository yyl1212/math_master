package learning

import "github.com/yyl1212/math_master/backend/internal/assessment"

func ResolveState(f StateFacts) State {
	if f.LatestReviewFailed || f.HadInvalidatedPass && !f.EffectivePass {
		return NeedsReview
	}
	if f.EffectivePass {
		return Mastered
	}
	if f.Completed {
		return Learned
	}
	if f.Started {
		return Learning
	}
	return Unlearned
}
func EvaluateEvidence(f EvidenceFacts) EvidenceView {
	if !f.Current {
		return EvidenceView{}
	}
	var chosen *assessment.AttemptFact
	for _, p := range f.Passes {
		if p.Knowledge != f.Knowledge || p.Validity != assessment.Effective || p.Outcome != assessment.Passed || p.Passed == nil || !*p.Passed || p.Score == nil || *p.Score < 4 || *p.Score > 5 {
			continue
		}
		if p.Mode != assessment.ModeDiagnostic && (p.Mode != assessment.ModeNode && p.Mode != assessment.ModeReview || f.CompletionEventID == nil) {
			continue
		}
		if chosen == nil || (p.Mode == assessment.ModeDiagnostic && chosen.Mode != assessment.ModeDiagnostic) || (p.Mode == assessment.ModeDiagnostic) == (chosen.Mode == assessment.ModeDiagnostic) && (p.SubmittedAt.After(chosen.SubmittedAt) || p.SubmittedAt.Equal(chosen.SubmittedAt) && p.ID < chosen.ID) {
			copy := p
			chosen = &copy
		}
	}
	if chosen == nil {
		return EvidenceView{}
	}
	out := &QualificationView{Knowledge: f.Knowledge, Kind: "normal", EvidenceAttemptID: chosen.ID, CompletedEventID: f.CompletionEventID, Validity: assessment.Effective}
	if chosen.Mode == assessment.ModeDiagnostic {
		out.Kind = "diagnostic"
		out.CompletedEventID = nil
	}
	return EvidenceView{Qualified: true, Qualification: out}
}
