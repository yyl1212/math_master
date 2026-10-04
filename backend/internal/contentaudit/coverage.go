package contentaudit

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/yyl1212/math_master/backend/internal/content"
	"github.com/yyl1212/math_master/backend/internal/question"
	"sort"
	"strings"
)

func objectKey(o ObjectIdentity) string {
	v := "null"
	if o.Version != nil {
		v = fmt.Sprint(*o.Version)
	}
	return o.Kind + ":" + o.ID + ":" + v
}
func identity(kind, id string, v int, sha string) ObjectIdentity {
	return ObjectIdentity{Kind: kind, ID: id, Version: &v, SHA256: sha}
}
func bodyFingerprint(body question.QuestionBody) string {
	b, _ := json.Marshal(body)
	var normalized question.QuestionBody
	_ = json.Unmarshal(b, &normalized)
	sort.Slice(normalized.Coverage, func(i, j int) bool {
		a, b := normalized.Coverage[i].Knowledge, normalized.Coverage[j].Knowledge
		return a.ID < b.ID || (a.ID == b.ID && a.Version < b.Version)
	})
	for i := range normalized.Coverage {
		sort.Ints(normalized.Coverage[i].ObjectiveIndices)
	}
	sort.Slice(normalized.Units, func(i, j int) bool { return normalized.Units[i].ID < normalized.Units[j].ID })
	sort.Slice(normalized.Assets, func(i, j int) bool { return normalized.Assets[i].ID < normalized.Assets[j].ID })
	b, _ = json.Marshal(normalized)
	return hashBytes(b)
}
func matchesSource(bp question.Blueprint, in question.Instance) bool {
	for _, s := range bp.Sources {
		if s.Kind == "instance" && s.Ref.ID == in.Identity.ID && s.Ref.Version == in.Identity.Version {
			return true
		}
		if s.Kind == "template" && in.Template != nil && s.Ref.ID == in.Template.ID && s.Ref.Version == in.Template.Version {
			return true
		}
	}
	return false
}
func goals(in question.Instance, ref content.VersionRef) []int {
	for _, c := range in.Body.Coverage {
		if c.Knowledge == ref {
			return c.ObjectiveIndices
		}
	}
	return []int{}
}
func witness(ctx context.Context, core []int, instances []question.Instance) ([]string, bool, bool, error) {
	if len(instances) > 1000 {
		return nil, false, false, ErrLimit
	}
	candidates := func(removed string) []question.CandidateCoverage {
		out := []question.CandidateCoverage{}
		for _, i := range instances {
			group := "fixed:" + i.Identity.ID
			if i.Template != nil {
				group = "template:" + i.Template.ID + fmt.Sprint(i.Template.Version)
			}
			if group == removed {
				continue
			}
			out = append(out, question.CandidateCoverage{InstanceID: i.Identity.ID, ObjectiveIndices: goals(i, i.Body.Knowledge)})
		}
		return out
	}
	w, ok, e := question.FiveQuestionCover(core, candidates(""))
	if e != nil || !ok {
		return []string{}, ok, false, e
	}
	groups := map[string]bool{}
	for _, i := range instances {
		g := "fixed:" + i.Identity.ID
		if i.Template != nil {
			g = "template:" + i.Template.ID + fmt.Sprint(i.Template.Version)
		}
		groups[g] = true
	}
	for g := range groups {
		if e = ctx.Err(); e != nil {
			return nil, false, false, e
		}
		_, possible, e := question.FiveQuestionCover(core, candidates(g))
		if e != nil {
			return nil, false, false, e
		}
		if !possible {
			return w, true, false, nil
		}
	}
	return w, true, true, nil
}
func EvaluateDraft(ctx context.Context, req Request, facts DraftFacts, sources SourceBundle) (Report, error) {
	if req.Mode != Draft {
		return Report{}, ErrInvalid
	}
	bank := question.Candidate{Templates: []question.Template{}, Blueprints: []question.Blueprint{}, Instances: []question.Instance{}}
	bpHashes := map[content.VersionRef]string{}
	for _, s := range facts.Sealed {
		bank.Templates = append(bank.Templates, s.Package.Templates...)
		bank.Blueprints = append(bank.Blueprints, s.Package.Blueprints...)
		bank.Instances = append(bank.Instances, s.Instances...)
	}
	for _, report := range facts.QuestionReports {
		for _, n := range report.Coverage {
			if n.Blueprint != nil {
				bpHashes[content.VersionRef{ID: n.Blueprint.ID, Version: n.Blueprint.Version}] = n.Blueprint.SHA256
			}
		}
	}
	p := PublishedFacts{CatalogueVersion: facts.Content.CatalogueVersion, CatalogueSHA: facts.References.CatalogueSHA256, Path: facts.Path, PathSHA: content.Digest(facts.Path), Content: facts.Content, Bank: bank}
	return evaluate(ctx, req, p, sources, AcceptanceEvidence{}, bpHashes)
}
func EvaluatePublished(ctx context.Context, req Request, facts PublishedFacts, sources SourceBundle, evidence AcceptanceEvidence) (Report, error) {
	if e := validateEvidence(evidence); e != nil {
		return Report{}, e
	}
	if req.Mode != Published {
		return Report{}, ErrInvalid
	}
	hashes := map[content.VersionRef]string{}
	for _, m := range facts.Bank.Manifest.Members {
		if m.Identity.Kind == "blueprint" {
			hashes[content.VersionRef{ID: m.Identity.ID, Version: m.Identity.Version}] = m.Identity.SHA256
		}
	}
	if len(hashes) == 0 {
		for _, o := range sources.Mapping.Objects {
			if o.Kind == "blueprint" && o.Version != nil {
				hashes[content.VersionRef{ID: o.ID, Version: *o.Version}] = o.SHA256
			}
		}
	}
	return evaluate(ctx, req, facts, sources, evidence, hashes)
}

var requiredLearningChecks = []string{"reading", "pass", "fail", "prerequisites", "review", "practice_exposure", "retake", "feedback_correction"}

func validateEvidence(e AcceptanceEvidence) error {
	if e.SchemaVersion == 0 && (len(e.ReviewAttestations) != 0 || len(e.LearningChecks) != 0) {
		return ErrInvalid
	}
	seenDecisions := map[string]bool{}
	for _, a := range e.ReviewAttestations {
		if a.DecisionID == "" || seenDecisions[a.DecisionID] {
			return ErrInvalid
		}
		seenDecisions[a.DecisionID] = true
	}
	allowed := map[string]bool{}
	for _, name := range requiredLearningChecks {
		allowed[name] = true
	}
	seenChecks := map[string]bool{}
	for _, c := range e.LearningChecks {
		if !allowed[c.Name] || seenChecks[c.Name] || !(c.Result == "passed" || c.Result == "failed" || c.Result == "not_run") || !question.ValidSHA(c.EvidenceSHA256) {
			return ErrInvalid
		}
		seenChecks[c.Name] = true
	}
	return nil
}
func evaluate(ctx context.Context, req Request, p PublishedFacts, s SourceBundle, evidence AcceptanceEvidence, bpHashes map[content.VersionRef]string) (Report, error) {
	var r Report
	if !question.ValidSHA(p.CatalogueSHA) || len(req.CodeSHA) != 40 || strings.Trim(req.CodeSHA, "0123456789abcdef") != "" || !question.ValidMathID(req.Route.ID) || req.Route.Version < 1 || req.At.IsZero() {
		return r, ErrInvalid
	}
	if p.Path.ID != req.Route.ID || p.Path.Version != req.Route.Version || content.Digest(p.Path) != p.PathSHA {
		return r, ErrInvalid
	}
	r = Report{SchemaVersion: 1, Mode: req.Mode, FixtureOnly: req.FixtureOnly || p.FixtureOnly || evidence.FixtureOnly, CreatedAt: req.At.UTC().Format("2006-01-02T15:04:05Z"), CodeSHA: req.CodeSHA, Context: ReportContext{CatalogueVersion: p.CatalogueVersion, CatalogueSHA: p.CatalogueSHA, Route: req.Route, RouteSHA: p.PathSHA, KnowledgeHead: p.KnowledgeHead, QuestionHead: p.QuestionHead}, Source: ReportSource{SnapshotID: s.SnapshotID, PolicyVersion: s.Report.PolicyVersion, ReportSHA: s.ReportSHA, Complete: s.Report.Ready, UnresolvedCount: len(s.Report.Issues)}, Nodes: []NodeReport{}, Reasons: []Reason{}, Quality: Quality{TwoAngles: true, Examples: true, Assets: true}, Conclusion: NotReady}
	km := map[content.VersionRef]content.Knowledge{}
	for _, k := range p.Content.Knowledge {
		ref := content.VersionRef{ID: k.ID, Version: k.Version}
		if _, ok := km[ref]; ok {
			return r, ErrInvalid
		}
		km[ref] = k
	}
	route := map[content.VersionRef]bool{}
	wanted := []ObjectIdentity{identity("path", p.Path.ID, p.Path.Version, p.PathSHA)}
	usedInstances := []question.Instance{}
	seenIdentity, seenBody := map[string]bool{}, map[string]bool{}
	eligible := map[question.Identity]bool{}
	for _, i := range p.EligibleInstances {
		eligible[i] = true
	}
	excluded := map[string]bool{}
	for _, x := range p.Excluded {
		excluded[objectKey(x.Object)] = true
	}
	bpByKnowledge := map[content.VersionRef]question.Blueprint{}
	for _, bp := range p.Bank.Blueprints {
		if _, ok := bpByKnowledge[bp.Knowledge]; ok {
			return r, ErrInvalid
		}
		bpByKnowledge[bp.Knowledge] = bp
	}
	for _, ref := range p.Path.Nodes {
		if e := ctx.Err(); e != nil {
			return r, e
		}
		if route[ref] {
			return r, ErrInvalid
		}
		route[ref] = true
		k, exists := km[ref]
		n := NodeReport{Knowledge: question.Identity{ID: ref.ID, Version: ref.Version, SHA256: content.Digest(k)}, Core: []int{}, FiveWitness: []string{}, Reasons: []Reason{}}
		if !exists {
			n.Reasons = append(n.Reasons, Reason{Code: "KNOWLEDGE_MISSING", Path: ref.ID})
			r.Nodes = append(r.Nodes, n)
			continue
		}
		r.DraftCounts.Knowledge++
		wanted = append(wanted, identity("knowledge", k.ID, k.Version, content.Digest(k)))
		units := 0
		for _, u := range p.Content.Units {
			if u.Knowledge != ref {
				continue
			}
			units++
			wanted = append(wanted, identity("unit", u.ID, u.Version, content.Digest(u)))
			kinds := map[string]bool{}
			for _, a := range u.Angles {
				if strings.TrimSpace(a.Body) != "" {
					kinds[a.Kind] = true
				}
			}
			if len(kinds) < 2 {
				r.Quality.TwoAngles = false
			}
			if len(u.Examples) == 0 || len(u.Counterexamples) == 0 {
				r.Quality.Examples = false
			}
			for _, id := range u.AssetIDs {
				found := false
				for _, a := range p.Content.Assets {
					if a.ID == id && a.Knowledge == ref {
						found = true
						wanted = append(wanted, ObjectIdentity{Kind: "asset", ID: a.ID, SHA256: a.SHA256})
					}
				}
				if !found {
					r.Quality.Assets = false
				}
			}
		}
		if units == 0 {
			r.Quality.TwoAngles = false
			r.Quality.Examples = false
		}
		bp, hasBP := bpByKnowledge[ref]
		if hasBP {
			sha := bpHashes[content.VersionRef{ID: bp.ID, Version: bp.Version}]
			if !question.ValidSHA(sha) {
				return r, ErrInvalid
			}
			bi := question.Identity{ID: bp.ID, Version: bp.Version, SHA256: sha}
			n.Blueprint = &bi
			n.Core = append([]int{}, bp.CoreObjectiveIndices...)
			wanted = append(wanted, identity("blueprint", bp.ID, bp.Version, sha))
		}
		assessment := []question.Instance{}
		rawCandidates := 0
		for _, in := range p.Bank.Instances {
			if in.Body.Knowledge != ref {
				continue
			}
			if req.Mode == Published && !eligible[in.Identity] {
				r.DraftCounts.Excluded++
				continue
			}
			if matchesSource(bp, in) {
				rawCandidates++
				if rawCandidates > 1000 {
					return r, ErrLimit
				}
			}
			key := objectKey(identity("instance", in.Identity.ID, in.Identity.Version, in.Identity.SHA256)) + ":" + in.Identity.SHA256
			fingerprint := bodyFingerprint(in.Body)
			if seenIdentity[key] || seenBody[fingerprint] {
				r.DraftCounts.Duplicates++
				continue
			}
			seenIdentity[key] = true
			seenBody[fingerprint] = true
			n.EffectiveInstances++
			r.DraftCounts.EffectiveInstances++
			usedInstances = append(usedInstances, in)
			if in.Origin == "fixed" {
				r.DraftCounts.FixedInstances++
				wanted = append(wanted, identity("instance", in.Identity.ID, in.Identity.Version, in.Identity.SHA256))
			} else if in.Template != nil {
				r.DraftCounts.GeneratedInstances++
			}
			if hasBP && matchesSource(bp, in) {
				assessment = append(assessment, in)
			}
		}
		n.AssessmentInstances = len(assessment)
		if hasBP {
			var ok bool
			var e error
			n.FiveWitness, ok, n.AfterPracticeWitness, e = witness(ctx, n.Core, assessment)
			if e != nil {
				return r, e
			}
			if !ok {
				n.Reasons = append(n.Reasons, Reason{Code: "FIVE_COVER_MISSING", Path: ref.ID})
			}
			if !n.AfterPracticeWitness {
				n.Reasons = append(n.Reasons, Reason{Code: "EXPOSURE_WITNESS_MISSING", Path: ref.ID})
			}
		} else {
			n.Reasons = append(n.Reasons, Reason{Code: "BLUEPRINT_MISSING", Path: ref.ID})
		}
		if n.EffectiveInstances < 10 {
			n.Reasons = append(n.Reasons, Reason{Code: "NODE_INSTANCES_BELOW_10", Path: ref.ID})
		}
		n.Ready = n.EffectiveInstances >= 10 && n.AssessmentInstances >= 5 && len(n.FiveWitness) == 5 && n.AfterPracticeWitness
		r.Nodes = append(r.Nodes, n)
	}
	activeTemplates := map[question.Identity]bool{}
	for _, in := range usedInstances {
		if in.Template != nil {
			activeTemplates[*in.Template] = true
		}
	}
	templateKeys := map[string]bool{}
	for _, t := range p.Bank.Templates {
		if !route[t.Knowledge] {
			continue
		}
		_, sha, e := question.CanonicalTemplate(t)
		if e != nil {
			return r, e
		}
		o := identity("template", t.ID, t.Version, sha)
		key := objectKey(o)
		if templateKeys[key] {
			continue
		}
		templateKeys[key] = true
		if excluded[key] || req.Mode == Published && !activeTemplates[question.Identity{ID: t.ID, Version: t.Version, SHA256: sha}] {
			r.DraftCounts.Excluded++
			continue
		}
		r.DraftCounts.Templates++
		wanted = append(wanted, o)
	}
	// Mapping omissions are actionable readiness gaps; conflicting identities are corrupt input.
	mappings := map[string]SourceObject{}
	for _, o := range s.Mapping.Objects {
		key := objectKey(ObjectIdentity{Kind: o.Kind, ID: o.ID, Version: o.Version})
		if _, exists := mappings[key]; exists {
			return r, ErrInvalid
		}
		mappings[key] = o
	}
	uniqueWanted := map[string]ObjectIdentity{}
	for _, o := range wanted {
		key := objectKey(o)
		uniqueWanted[key] = o
		m, ok := mappings[key]
		if !ok {
			r.Source.Complete = false
			r.Source.UnresolvedCount++
			continue
		}
		if m.SHA256 != o.SHA256 {
			return r, ErrInvalid
		}
		if m.Origin != "original" || len(m.SourceIDs) == 0 {
			r.Source.Complete = false
		}
	}
	if !r.Source.Complete {
		r.Reasons = append(r.Reasons, Reason{Code: "SOURCE_UNRESOLVED", Path: "/source"})
	}
	technical := r.Source.Complete && r.Quality.TwoAngles && r.Quality.Examples && r.Quality.Assets && r.DraftCounts.Knowledge >= 30 && r.DraftCounts.Templates >= 20 && r.DraftCounts.EffectiveInstances >= 300
	for _, n := range r.Nodes {
		if !n.Ready {
			technical = false
		}
	}
	if !technical {
		r.Reasons = append(r.Reasons, Reason{Code: "TECHNICAL_REQUIREMENTS_NOT_MET", Path: "/"})
		return r, nil
	}
	if req.Mode == Draft {
		r.Conclusion = DraftReady
		return r, nil
	}
	approved := map[string]bool{}
	attested := map[string]bool{}
	for _, a := range evidence.ReviewAttestations {
		attested[a.DecisionID] = a.IndependenceVerified
	}
	for _, a := range p.Approvals {
		key := objectKey(a.Object)
		expected, ok := uniqueWanted[key]
		if !ok || a.Object.SHA256 != expected.SHA256 || a.ReviewerID == "" || len(a.AuthorIDs) == 0 || !a.ChecksComplete || !a.FrozenMatches || !attested[a.Evidence.DecisionID] || !question.ValidSHA(a.Evidence.FrozenDigest) {
			continue
		}
		independent := true
		for _, author := range a.AuthorIDs {
			if author == a.ReviewerID {
				independent = false
			}
		}
		if independent {
			approved[key] = true
		}
	}
	r.Quality.IndependentReview = len(approved) == len(uniqueWanted)
	learningFailed := false
	if evidence.SchemaVersion != 0 {
		if evidence.SchemaVersion != 1 || evidence.CodeSHA != req.CodeSHA || evidence.RouteSHA != p.PathSHA || !headsEqual(evidence.KnowledgeHead, p.KnowledgeHead) || !headsEqual(evidence.QuestionHead, p.QuestionHead) {
			return r, ErrInvalid
		}
		checks := map[string]bool{}
		for _, c := range evidence.LearningChecks {
			checks[c.Name] = c.Result == "passed"
			if c.Result == "failed" {
				learningFailed = true
			}
		}
		r.Quality.LearningComplete = true
		for _, name := range requiredLearningChecks {
			if !checks[name] {
				r.Quality.LearningComplete = false
			}
		}
	}
	if learningFailed {
		r.Conclusion = NotReady
		r.Reasons = append(r.Reasons, Reason{Code: "LEARNING_CHECK_FAILED", Path: "/learning"})
		return r, nil
	}
	if !r.FixtureOnly {
		for _, o := range uniqueWanted {
			if !approved[objectKey(o)] {
				continue
			}
			switch o.Kind {
			case "knowledge":
				r.FormalCounts.Knowledge++
			case "template":
				r.FormalCounts.Templates++
			}
		}
		for _, in := range usedInstances {
			ok := approved[objectKey(identity("instance", in.Identity.ID, in.Identity.Version, in.Identity.SHA256))]
			if in.Template != nil {
				ok = approved[objectKey(identity("template", in.Template.ID, in.Template.Version, in.Template.SHA256))]
			}
			if !ok {
				continue
			}
			r.FormalCounts.EffectiveInstances++
			if in.Origin == "fixed" {
				r.FormalCounts.FixedInstances++
			} else {
				r.FormalCounts.GeneratedInstances++
			}
		}
	}
	if r.Quality.IndependentReview && r.Quality.LearningComplete && !r.FixtureOnly && p.KnowledgeHead != nil && p.QuestionHead != nil && r.FormalCounts.Knowledge >= 30 && r.FormalCounts.Templates >= 20 && r.FormalCounts.EffectiveInstances >= 300 {
		r.Conclusion = Accepted
	} else {
		r.Conclusion = AwaitingReview
		r.Reasons = append(r.Reasons, Reason{Code: "INDEPENDENT_REVIEW_OR_LEARNING_EVIDENCE_REQUIRED", Path: "/review"})
	}
	return r, nil
}
func headsEqual(a, b *Head) bool { return (a == nil && b == nil) || (a != nil && b != nil && *a == *b) }
