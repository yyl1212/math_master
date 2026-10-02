package question

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/yyl1212/math_master/backend/internal/content"
	"github.com/yyl1212/math_master/backend/internal/publication"
	"net/url"
	"reflect"
	"sort"
	"strings"
)

const MaxTemplates = 50
const MaxFixedQuestions = 200
const MaxBlueprints = 100
const MaxGeneratedInstances = 1000
const MaxSourceMapBytes = 256 * 1024

type validationState struct {
	report         ValidationReport
	limit          bool
	objectiveBytes int
	refs           ReferenceSnapshot
	knowledge      map[Ref]FixedKnowledge
	units          map[Ref]FixedUnit
	assets         map[string]content.AssetView
	resolved       map[string]KnownObject
	objectives     map[string]ResolvedObjective
}

func newValidation(refs ReferenceSnapshot) *validationState {
	refs = canonicalValue(reflect.ValueOf(refs)).Interface().(ReferenceSnapshot)
	v := &validationState{report: ValidationReport{StructuralErrors: []Issue{}, CompletenessErrors: []Issue{}, HumanReviewRequirements: []Issue{}, Generation: []GenerationReport{}, Coverage: []CoverageNode{}}, refs: refs, knowledge: map[Ref]FixedKnowledge{}, units: map[Ref]FixedUnit{}, assets: map[string]content.AssetView{}, resolved: map[string]KnownObject{}, objectives: map[string]ResolvedObjective{}}
	for _, k := range refs.Knowledge {
		v.knowledge[Ref{ID: k.Identity.ID, Version: k.Identity.Version}] = k
	}
	for _, u := range refs.Units {
		v.units[Ref{ID: u.Identity.ID, Version: u.Identity.Version}] = u
	}
	for _, a := range refs.Assets {
		v.assets[a.ID] = a
	}
	return v
}
func (v *validationState) issue(kind, code, path string) {
	issue := Issue{Code: code, Path: path, Message: "Question validation failed."}
	switch kind {
	case "structural":
		v.report.StructuralTotal++
		if len(v.report.StructuralErrors) < 100 {
			v.report.StructuralErrors = append(v.report.StructuralErrors, issue)
		}
	case "complete":
		v.report.CompletenessTotal++
		if len(v.report.CompletenessErrors) < 100 {
			v.report.CompletenessErrors = append(v.report.CompletenessErrors, issue)
		}
	case "review":
		v.report.HumanReviewTotal++
		issue.Message = "Independent review is required."
		if len(v.report.HumanReviewRequirements) < 100 {
			v.report.HumanReviewRequirements = append(v.report.HumanReviewRequirements, issue)
		}
	}
}
func (v *validationState) limitAt(path string) {
	v.limit = true
	v.issue("structural", "QUESTION_LIMIT_EXCEEDED", path)
}
func (v *validationState) finish() ValidationReport {
	r := v.report
	remaining := 100 - len(r.StructuralErrors)
	if len(r.CompletenessErrors) > remaining {
		r.CompletenessErrors = r.CompletenessErrors[:remaining]
	}
	remaining -= len(r.CompletenessErrors)
	if len(r.HumanReviewRequirements) > remaining {
		r.HumanReviewRequirements = r.HumanReviewRequirements[:remaining]
	}
	r.Truncated = r.StructuralTotal+r.CompletenessTotal+r.HumanReviewTotal > 100
	r.ReadyToSubmit = r.StructuralTotal == 0 && r.CompletenessTotal == 0
	return r
}
func objectKey(kind string, ref Ref) string {
	return fmt.Sprintf("%s/%s/%d", kind, ref.ID, ref.Version)
}
func (v *validationState) resolveKnowledge(ref Ref, path string) (FixedKnowledge, bool) {
	k, ok := v.knowledge[ref]
	if !ok || !ValidSHA(k.Identity.SHA256) {
		v.issue("complete", "KNOWLEDGE_NOT_PUBLISHED", path)
		return FixedKnowledge{}, false
	}
	key := objectKey("knowledge", ref)
	if _, already := v.resolved[key]; !already {
		v.resolved[key] = KnownObject{Ref: ref, SHA256: k.Identity.SHA256, Kind: "knowledge"}
		for index, text := range k.Objectives {
			objective := ResolvedObjective{Knowledge: k.Identity, ObjectiveIndex: index, Text: text}
			raw, _ := json.Marshal(objective)
			v.objectiveBytes += len(raw) + 1
			if v.objectiveBytes > MaxEnvelopeBytes {
				v.limitAt(path + "/objectives")
				break
			}
			v.objectives[fmt.Sprintf("%s/%d", key, index)] = objective
		}
	}
	return k, true
}
func (v *validationState) checkCoverage(primary Ref, coverage []ObjectiveCoverage, path string) map[Ref]bool {
	allowed := map[Ref]bool{primary: true}
	if len(coverage) > 4 {
		v.limitAt(path)
	}
	_, _ = v.resolveKnowledge(primary, path+"/knowledge")
	seen := map[string]bool{}
	foundPrimary := false
	for i, c := range coverage {
		p := fmt.Sprintf("%s/coverage/%d", path, i)
		if seen[c.Knowledge.ID] {
			v.issue("structural", "DUPLICATE_COVERAGE", p)
		}
		seen[c.Knowledge.ID] = true
		allowed[c.Knowledge] = true
		if c.Knowledge == primary && len(c.ObjectiveIndices) > 0 {
			foundPrimary = true
		}
		k, ok := v.resolveKnowledge(c.Knowledge, p+"/knowledge")
		goals := map[int]bool{}
		if len(c.ObjectiveIndices) == 0 {
			v.issue("complete", "OBJECTIVES_REQUIRED", p)
		}
		for _, goal := range c.ObjectiveIndices {
			if goals[goal] {
				v.issue("structural", "DUPLICATE_OBJECTIVE", p)
			}
			goals[goal] = true
			if !ok {
				continue
			}
			if goal < 0 || goal >= len(k.Objectives) {
				v.issue("complete", "OBJECTIVE_NOT_FOUND", p)
				continue
			}
			key := fmt.Sprintf("%s/%d", objectKey("knowledge", c.Knowledge), goal)
			v.objectives[key] = ResolvedObjective{Knowledge: k.Identity, ObjectiveIndex: goal, Text: k.Objectives[goal]}
		}
	}
	if !foundPrimary {
		v.issue("complete", "PRIMARY_COVERAGE_REQUIRED", path+"/coverage")
	}
	return allowed
}
func (v *validationState) checkReferences(primary Ref, coverage []ObjectiveCoverage, units []Ref, assets []AssetRef, path string) {
	allowed := v.checkCoverage(primary, coverage, path)
	seenUnits := map[Ref]bool{}
	for i, ref := range units {
		p := fmt.Sprintf("%s/units/%d", path, i)
		if seenUnits[ref] {
			v.issue("structural", "DUPLICATE_UNIT", p)
		}
		seenUnits[ref] = true
		unit, ok := v.units[ref]
		if !ok || !ValidSHA(unit.Identity.SHA256) {
			v.issue("complete", "UNIT_NOT_PUBLISHED", p)
			continue
		}
		if !allowed[unit.Knowledge] {
			v.issue("complete", "UNIT_KNOWLEDGE_MISMATCH", p)
			continue
		}
		v.resolved[objectKey("unit", ref)] = KnownObject{Ref: ref, SHA256: unit.Identity.SHA256, Kind: "unit"}
	}
	if len(assets) > 8 {
		v.limitAt(path + "/assets")
	}
	seenAssets := map[string]bool{}
	for i, ref := range assets {
		p := fmt.Sprintf("%s/assets/%d", path, i)
		if seenAssets[ref.ID] {
			v.issue("structural", "DUPLICATE_ASSET", p)
		}
		seenAssets[ref.ID] = true
		asset, ok := v.assets[ref.ID]
		if !ok || !ValidSHA(ref.SHA256) || asset.SHA256 != ref.SHA256 {
			v.issue("complete", "ASSET_NOT_PUBLISHED", p)
			continue
		}
		if !allowed[asset.Knowledge] {
			v.issue("complete", "ASSET_KNOWLEDGE_MISMATCH", p)
			continue
		}
		v.resolved[objectKey("asset", Ref{ID: ref.ID})] = KnownObject{Ref: Ref{ID: ref.ID, Version: 0}, SHA256: asset.SHA256, Kind: "asset"}
	}
}
func (v *validationState) checkSources(sources []content.Source, path string) {
	if len(sources) == 0 {
		v.issue("complete", "SOURCES_REQUIRED", path)
	}
	for i, s := range sources {
		p := fmt.Sprintf("%s/%d", path, i)
		if strings.TrimSpace(s.Author) == "" || strings.TrimSpace(s.License) == "" || strings.TrimSpace(s.Title) == "" || strings.TrimSpace(s.Kind) == "" {
			v.issue("complete", "SOURCE_DETAILS_REQUIRED", p)
		}
		if s.URL != "" {
			u, e := url.Parse(s.URL)
			if e != nil || u.Scheme != "https" || u.Hostname() == "" || u.User != nil || strings.ContainsAny(s.URL, "\\\x00\r\n") {
				v.issue("structural", "UNSAFE_SOURCE_URL", p)
			}
		}
	}
}
func (v *validationState) checkBody(body QuestionBody, path string) {
	v.checkReferences(body.Knowledge, body.Coverage, body.Units, body.Assets, path)
	v.checkSources(body.Sources, path+"/sources")
	if len(body.Prompt)+len(body.Explanation) > MaxQuestionTextBytes {
		v.limitAt(path)
	}
	if strings.TrimSpace(body.Prompt) == "" || strings.TrimSpace(body.Explanation) == "" {
		v.issue("complete", "QUESTION_TEXT_REQUIRED", path)
	}
	if !safeQuestionMarkdown(body.Prompt, body.Assets) || !safeQuestionMarkdown(body.Explanation, body.Assets) {
		v.issue("structural", "UNSAFE_MARKUP", path)
	}
	if len(body.Choices) > 6 {
		v.limitAt(path + "/choices")
	}
}
func frozenSize(body FrozenBody, instances []Instance) (int, error) {
	raw, _, e := CanonicalFrozen(body, instances)
	if e != nil {
		return 0, e
	}
	if len(raw) > MaxEnvelopeBytes {
		return len(raw), ErrLimitExceeded
	}
	return len(raw), nil
}
func ValidateEditable(ctx context.Context, input DraftInput, refs ReferenceSnapshot) (ValidationReport, error) {
	_, report, e := validatePackage(ctx, input, refs)
	if errors.Is(e, ErrLimitExceeded) {
		e = nil
	}
	return report, e
}
func ValidateAndSeal(ctx context.Context, input DraftInput, refs ReferenceSnapshot) (SealedPackage, ValidationReport, error) {
	sealed, report, e := validatePackage(ctx, input, refs)
	if e != nil {
		return SealedPackage{}, report, e
	}
	if !report.ReadyToSubmit {
		for _, issue := range report.StructuralErrors {
			if issue.Code == "QUESTION_LIMIT_EXCEEDED" {
				return SealedPackage{}, report, ErrLimitExceeded
			}
		}
		if report.StructuralTotal > 0 {
			return SealedPackage{}, report, ErrInvalid
		}
		return SealedPackage{}, report, ErrNotReady
	}
	return sealed, report, nil
}
func validatePackage(ctx context.Context, input DraftInput, refs ReferenceSnapshot) (SealedPackage, ValidationReport, error) {
	v := newValidation(refs)
	empty := SealedPackage{}
	if e := ctx.Err(); e != nil {
		return empty, v.finish(), e
	}
	raw, e := json.Marshal(input)
	if e != nil {
		v.issue("structural", "INVALID_PACKAGE", "/")
		return empty, v.finish(), nil
	}
	if len(raw) > MaxEnvelopeBytes {
		v.limitAt("/")
		return empty, v.finish(), nil
	}
	packageRaw, e := json.Marshal(input.QuestionPackage)
	if e != nil {
		v.issue("structural", "INVALID_PACKAGE", "/")
		return empty, v.finish(), nil
	}
	v.report.PackageBytes = len(packageRaw)
	if len(packageRaw) > MaxPackageBytes {
		v.limitAt("/questionPackage")
		return empty, v.finish(), nil
	}
	if _, e = DecodeDraft(bytes.NewReader(raw)); e != nil {
		v.issue("structural", "INVALID_PACKAGE", "/")
		return empty, v.finish(), nil
	}
	sourceRaw, _ := json.Marshal(input.SourceMap)
	if len(sourceRaw) > MaxSourceMapBytes {
		v.limitAt("/sourceMap")
	}
	orderedBytes, _, orderingError := CanonicalPackage(input.QuestionPackage)
	if orderingError != nil {
		return empty, v.finish(), orderingError
	}
	var ordered struct {
		Body QuestionPackage `json:"body"`
	}
	if e := json.Unmarshal(orderedBytes, &ordered); e != nil {
		return empty, v.finish(), e
	}
	p := ordered.Body
	if len(p.Templates) > MaxTemplates {
		v.limitAt("/templates")
	}
	if len(p.FixedQuestions) > MaxFixedQuestions {
		v.limitAt("/fixedQuestions")
	}
	if len(p.Blueprints) > MaxBlueprints {
		v.limitAt("/blueprints")
	}
	if v.limit {
		return empty, v.finish(), nil
	}
	if len(p.Templates)+len(p.FixedQuestions) == 0 {
		v.issue("complete", "QUESTIONS_REQUIRED", "/")
	}
	if input.CatalogueVersion != refs.CatalogueVersion || !ValidSHA(refs.CatalogueSHA256) {
		v.issue("complete", "CATALOGUE_NOT_PUBLISHED", "/catalogueVersion")
	}
	allInstances := []Instance{}
	templateInstances := map[Ref][]Instance{}
	fixedInstances := map[Ref]Instance{}
	memberIDs := map[string]bool{}
	generatedCount := 0
	offerable := map[string]bool{}
	usedKnowledge := map[Ref]bool{}
	rememberKnowledge := func(primary Ref, coverage []ObjectiveCoverage) {
		usedKnowledge[primary] = true
		for _, c := range coverage {
			usedKnowledge[c.Knowledge] = true
		}
	}
	for i, t := range p.Templates {
		if e := ctx.Err(); e != nil {
			return empty, v.finish(), e
		}
		path := fmt.Sprintf("/templates/%d", i)
		beforeIssues := v.report.StructuralTotal + v.report.CompletenessTotal
		if memberIDs[t.ID] {
			v.issue("structural", "DUPLICATE_TEMPLATE", path)
		}
		memberIDs[t.ID] = true
		rememberKnowledge(t.Knowledge, t.Coverage)
		v.checkReferences(t.Knowledge, t.Coverage, t.Units, t.Assets, path)
		v.checkSources(t.Sources, path+"/sources")
		if strings.TrimSpace(t.PromptTemplate) == "" || strings.TrimSpace(t.ExplanationTemplate) == "" {
			v.issue("complete", "QUESTION_TEXT_REQUIRED", path)
		}
		instances, report, e := Generate(ctx, t)
		v.report.Generation = append(v.report.Generation, report)
		if e != nil {
			if ctx.Err() != nil {
				return empty, v.finish(), ctx.Err()
			}
			if errors.Is(e, ErrLimitExceeded) {
				v.limitAt(path)
			} else {
				v.issue("complete", "TEMPLATE_NOT_READY", path)
			}
			v.issue("review", "GENERATION_REVIEW_REQUIRED", path)
			continue
		}
		normalized, _, _ := normalizeTemplate(t)
		p.Templates[i] = normalized
		generatedCount += len(instances)
		if generatedCount > MaxGeneratedInstances {
			v.limitAt("/templates")
		}
		templateInstances[Ref{ID: t.ID, Version: t.Version}] = instances
		allInstances = append(allInstances, instances...)
		for _, instance := range instances {
			offerable[instance.Identity.ID] = v.report.StructuralTotal+v.report.CompletenessTotal == beforeIssues
		}
		v.issue("review", "GENERATION_REVIEW_REQUIRED", path)
	}
	memberIDs = map[string]bool{}
	for i, f := range p.FixedQuestions {
		if e := ctx.Err(); e != nil {
			return empty, v.finish(), e
		}
		path := fmt.Sprintf("/fixedQuestions/%d", i)
		beforeIssues := v.report.StructuralTotal + v.report.CompletenessTotal
		if memberIDs[f.ID] {
			v.issue("structural", "DUPLICATE_INSTANCE", path)
		}
		memberIDs[f.ID] = true
		rememberKnowledge(f.Body.Knowledge, f.Body.Coverage)
		v.checkBody(f.Body, path+"/body")
		body := sortBody(f.Body)
		if body.Witness != nil {
			seenParameters := map[string]bool{}
			parameters := []ParameterValue{}
			validWitness := true
			for _, param := range body.Witness.Parameters {
				if seenParameters[param.Name] {
					v.issue("structural", "DUPLICATE_WITNESS_PARAMETER", path+"/witness")
					validWitness = false
				}
				seenParameters[param.Name] = true
				r, e := parseLiteral(param.Value)
				if e != nil {
					validWitness = false
					v.issue("complete", "INVALID_WITNESS_PARAMETER", path+"/witness")
					continue
				}
				parameters = append(parameters, ParameterValue{Name: param.Name, Value: r.Num().String() + "/" + r.Denom().String()})
			}
			if validWitness {
				sort.Slice(parameters, func(i, j int) bool { return parameters[i].Name < parameters[j].Name })
				body.Witness.Parameters = parameters
			}
		}
		p.FixedQuestions[i].Body = body
		instance := Instance{Identity: Identity{ID: f.ID, Version: f.Version}, Origin: "fixed", Parameters: []ParameterValue{}, Body: body}
		_, instance.Identity.SHA256, e = CanonicalInstance(instance)
		if e != nil {
			return empty, v.finish(), e
		}
		if e := VerifyInstance(instance); e != nil {
			if errors.Is(e, ErrLimitExceeded) {
				v.limitAt(path)
			} else {
				v.issue("complete", "FIXED_ANSWER_NOT_VERIFIED", path)
			}
			v.issue("review", "FIXED_REVIEW_REQUIRED", path)
			continue
		}
		fixedInstances[Ref{ID: f.ID, Version: f.Version}] = instance
		allInstances = append(allInstances, instance)
		offerable[instance.Identity.ID] = v.report.StructuralTotal+v.report.CompletenessTotal == beforeIssues
		v.issue("review", "FIXED_REVIEW_REQUIRED", path)
	}
	sourceKeys := map[string]bool{}
	for i, link := range input.SourceMap {
		path := fmt.Sprintf("/sourceMap/%d", i)
		if !usedKnowledge[link.Knowledge] || !ValidSHA(link.BatchSHA256) || !ValidSHA(link.SHA256) || !publication.RelativeSourcePath(link.RelativePath) {
			v.issue("structural", "INVALID_SOURCE_MAP", path)
		}
		key := fmt.Sprintf("%s/%s/%s/%s", objectKey("knowledge", link.Knowledge), link.BatchSHA256, link.RelativePath, link.SHA256)
		if sourceKeys[key] {
			v.issue("structural", "DUPLICATE_SOURCE_MAP", path)
		}
		sourceKeys[key] = true
	}
	memberIDs = map[string]bool{}
	blueprinted := map[Ref]bool{}
	for i, b := range p.Blueprints {
		if e := ctx.Err(); e != nil {
			return empty, v.finish(), e
		}
		path := fmt.Sprintf("/blueprints/%d", i)
		if memberIDs[b.ID] {
			v.issue("structural", "DUPLICATE_BLUEPRINT", path)
		}
		memberIDs[b.ID] = true
		k, ok := v.resolveKnowledge(b.Knowledge, path+"/knowledge")
		blueprinted[b.Knowledge] = true
		coreValid := len(b.CoreObjectiveIndices) >= 1 && len(b.CoreObjectiveIndices) <= 8
		if len(b.CoreObjectiveIndices) < 1 {
			v.issue("complete", "CORE_OBJECTIVES_REQUIRED", path)
		} else if len(b.CoreObjectiveIndices) > 8 {
			v.limitAt(path + "/coreObjectiveIndices")
		}
		core := map[int]bool{}
		for _, goal := range b.CoreObjectiveIndices {
			if core[goal] {
				coreValid = false
				v.issue("structural", "DUPLICATE_OBJECTIVE", path)
			}
			core[goal] = true
			if !ok || goal < 0 || goal >= len(k.Objectives) {
				coreValid = false
				v.issue("complete", "OBJECTIVE_NOT_FOUND", path)
			}
		}
		if strings.TrimSpace(b.CoverageNote) == "" {
			v.issue("complete", "COVERAGE_NOTE_REQUIRED", path)
		}
		candidates := map[string]Instance{}
		seen := map[string]bool{}
		sourceValid := true
		for j, source := range b.Sources {
			sp := fmt.Sprintf("%s/sources/%d", path, j)
			key := objectKey(source.Kind, source.Ref)
			if seen[key] {
				v.issue("structural", "DUPLICATE_BLUEPRINT_SOURCE", sp)
			}
			seen[key] = true
			if source.Kind == "template" {
				instances, found := templateInstances[source.Ref]
				if !found {
					sourceValid = false
					v.issue("complete", "BLUEPRINT_SOURCE_NOT_FOUND", sp)
				}
				for _, instance := range instances {
					candidates[instance.Identity.ID] = instance
				}
			} else {
				instance, found := fixedInstances[source.Ref]
				if !found {
					sourceValid = false
					v.issue("complete", "BLUEPRINT_SOURCE_NOT_FOUND", sp)
				} else {
					candidates[instance.Identity.ID] = instance
				}
			}
		}
		node := CoverageNode{Knowledge: k.Identity, EffectiveInstances: 0, CoreObjectiveIndices: append([]int{}, b.CoreObjectiveIndices...), CoveredObjectiveIndices: []int{}, Reasons: []Issue{}, SupplementaryObjectiveIndices: []int{}}
		if !ok {
			node.Knowledge = Identity{ID: b.Knowledge.ID, Version: b.Knowledge.Version}
		}
		_, blueprintSHA, _ := canonical("question-blueprint-v1", b)
		identity := Identity{ID: b.ID, Version: b.Version, SHA256: blueprintSHA}
		node.Blueprint = &identity
		pool := []CandidateCoverage{}
		covered := map[int]bool{}
		nodeIDs := map[string]bool{}
		for _, instance := range allInstances {
			goals := instanceObjectives(instance, b.Knowledge)
			if !offerable[instance.Identity.ID] || len(goals) == 0 || nodeIDs[instance.Identity.ID] {
				continue
			}
			nodeIDs[instance.Identity.ID] = true
			node.EffectiveInstances++
			if instance.Origin == "fixed" {
				node.FixedInstances++
			} else {
				node.GeneratedInstances++
			}
		}
		for _, instance := range candidates {
			if !offerable[instance.Identity.ID] {
				sourceValid = false
				continue
			}
			goals := instanceObjectives(instance, b.Knowledge)
			if len(goals) == 0 {
				sourceValid = false
				v.issue("complete", "BLUEPRINT_SOURCE_KNOWLEDGE_MISMATCH", path)
				continue
			}
			pool = append(pool, CandidateCoverage{InstanceID: instance.Identity.ID, ObjectiveIndices: goals})
			for _, goal := range goals {
				covered[goal] = true
			}
		}
		node.AssessmentInstances = len(pool)
		for goal := range covered {
			node.CoveredObjectiveIndices = append(node.CoveredObjectiveIndices, goal)
		}
		sort.Ints(node.CoveredObjectiveIndices)
		for goal := range k.Objectives {
			if !core[goal] {
				node.SupplementaryObjectiveIndices = append(node.SupplementaryObjectiveIndices, goal)
			}
		}
		_, feasible, e := FiveQuestionCover(b.CoreObjectiveIndices, pool)
		node.FiveQuestionFeasible = feasible
		node.Ready = ok && coreValid && sourceValid && feasible && len(core) == len(b.CoreObjectiveIndices)
		if len(pool) > 1000 || node.EffectiveInstances > 1000 {
			v.limitAt(path)
			node.Ready = false
		}
		if e != nil || !node.Ready {
			v.issue("complete", "FIVE_QUESTION_COVER_REQUIRED", path)
			node.Reasons = append(node.Reasons, Issue{Code: "FIVE_QUESTION_COVER_REQUIRED", Path: path, Message: "Five distinct questions must cover every core objective."})
		}
		v.report.Coverage = append(v.report.Coverage, node)
		v.issue("review", "OBJECTIVE_REVIEW_REQUIRED", path)
	}
	for ref := range usedKnowledge {
		if blueprinted[ref] {
			continue
		}
		k, ok := v.knowledge[ref]
		node := CoverageNode{Knowledge: Identity{ID: ref.ID, Version: ref.Version}, CoreObjectiveIndices: []int{}, CoveredObjectiveIndices: []int{}, Reasons: []Issue{{Code: "BLUEPRINT_REQUIRED", Path: "/blueprints", Message: "No assessment blueprint is defined."}}, SupplementaryObjectiveIndices: []int{}}
		if ok {
			node.Knowledge = k.Identity
			for goal := range k.Objectives {
				node.SupplementaryObjectiveIndices = append(node.SupplementaryObjectiveIndices, goal)
			}
		}
		ids := map[string]bool{}
		goals := map[int]bool{}
		for _, instance := range allInstances {
			indices := instanceObjectives(instance, ref)
			if !offerable[instance.Identity.ID] || len(indices) == 0 || ids[instance.Identity.ID] {
				continue
			}
			ids[instance.Identity.ID] = true
			node.EffectiveInstances++
			if instance.Origin == "fixed" {
				node.FixedInstances++
			} else {
				node.GeneratedInstances++
			}
			for _, index := range indices {
				goals[index] = true
			}
		}
		for goal := range goals {
			node.CoveredObjectiveIndices = append(node.CoveredObjectiveIndices, goal)
		}
		sort.Ints(node.CoveredObjectiveIndices)
		if node.EffectiveInstances > 1000 {
			v.limitAt("/coverage")
		}
		v.report.Coverage = append(v.report.Coverage, node)
	}
	sort.SliceStable(v.report.Coverage, func(i, j int) bool {
		a, b := v.report.Coverage[i], v.report.Coverage[j]
		if a.Blueprint != nil && b.Blueprint == nil {
			return true
		}
		if a.Blueprint == nil && b.Blueprint != nil {
			return false
		}
		if a.Blueprint != nil {
			return a.Blueprint.ID < b.Blueprint.ID
		}
		return refLess(Ref{ID: a.Knowledge.ID, Version: a.Knowledge.Version}, Ref{ID: b.Knowledge.ID, Version: b.Knowledge.Version})
	})
	if e := ctx.Err(); e != nil {
		return empty, v.finish(), e
	}
	sort.Slice(allInstances, func(i, j int) bool { return allInstances[i].Identity.ID < allInstances[j].Identity.ID })
	resolved := []KnownObject{}
	for _, object := range v.resolved {
		resolved = append(resolved, object)
	}
	sort.Slice(resolved, func(i, j int) bool {
		a, b := resolved[i], resolved[j]
		if a.Kind != b.Kind {
			return a.Kind < b.Kind
		}
		return refLess(a.Ref, b.Ref)
	})
	objectives := []ResolvedObjective{}
	for _, objective := range v.objectives {
		objectives = append(objectives, objective)
	}
	sort.Slice(objectives, func(i, j int) bool {
		a, b := objectives[i], objectives[j]
		ar, br := Ref{ID: a.Knowledge.ID, Version: a.Knowledge.Version}, Ref{ID: b.Knowledge.ID, Version: b.Knowledge.Version}
		if ar != br {
			return refLess(ar, br)
		}
		return a.ObjectiveIndex < b.ObjectiveIndex
	})
	packageRaw, packageSHA, e := CanonicalPackage(p)
	if e != nil {
		return empty, v.finish(), e
	}
	// Normalized parameter text may grow. The logical payload remains 2 MiB;
	// the fixed purpose framing is counted independently and never changes SHA.
	normalizedBytes := len(packageRaw) - PackageCanonicalOverhead
	if normalizedBytes > MaxPackageBytes {
		v.limitAt("/questionPackage")
		return empty, v.finish(), ErrLimitExceeded
	}
	sealed := SealedPackage{Package: p, PackageSHA: packageSHA, Instances: allInstances, Generation: v.report.Generation, Resolved: resolved, Objectives: objectives, ReferenceSnapshot: v.refs}
	identities := []Identity{}
	for _, i := range allInstances {
		identities = append(identities, i.Identity)
	}
	generators, verifiers := UsedEngineVersions(p, allInstances)
	frozen := FrozenBody{CatalogueVersion: input.CatalogueVersion, CatalogueSHA256: refs.CatalogueSHA256, QuestionPackage: p, SourceMap: input.SourceMap, AuthorIDs: []string{}, Resolved: resolved, Objectives: objectives, Generation: v.report.Generation, InstanceIdentities: identities, Coverage: v.report.Coverage, GeneratorVersions: generators, VerifierVersions: verifiers}
	v.report.FrozenBytes, e = frozenSize(frozen, allInstances)
	if errors.Is(e, ErrLimitExceeded) {
		v.limitAt("/")
	} else if e != nil {
		return empty, v.finish(), e
	}
	_, v.report.Digest, e = CanonicalValidation(input, sealed)
	if e != nil {
		return empty, v.finish(), e
	}
	if v.limit {
		return empty, v.finish(), ErrLimitExceeded
	}
	return sealed, v.finish(), nil
}
func instanceObjectives(instance Instance, knowledge Ref) []int {
	for _, mapping := range instance.Body.Coverage {
		if mapping.Knowledge == knowledge {
			return mapping.ObjectiveIndices
		}
	}
	return nil
}
