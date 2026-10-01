package content

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/yyl1212/math_master/backend/internal/catalogue"
	"net/url"
	"strings"
	"time"
)

var ErrValidation = errors.New("invalid content")
var ErrLimit = errors.New("content capacity exceeded")

const MaxWorkflowPackageBytes = 2 << 20
const MaxWorkflowAssetBytes = 4 << 20
const MaxSnapshotBytes = 32 << 20

type WorkflowReport struct {
	StructuralErrors        []Issue `json:"structuralErrors"`
	CompletenessErrors      []Issue `json:"completenessErrors"`
	HumanReviewRequirements []Issue `json:"humanReviewRequirements"`
	StructuralTotal         int     `json:"structuralTotal"`
	CompletenessTotal       int     `json:"completenessTotal"`
	HumanReviewTotal        int     `json:"humanReviewTotal"`
	Truncated               bool    `json:"truncated"`
	ReadyToSubmit           bool    `json:"readyToSubmit"`
}

func (r WorkflowReport) Display() WorkflowReport {
	remaining := 100
	trim := func(v []Issue) []Issue {
		n := len(v)
		if n > remaining {
			n = remaining
			r.Truncated = true
		}
		remaining -= n
		return append([]Issue{}, v[:n]...)
	}
	r.StructuralErrors = trim(r.StructuralErrors)
	r.CompletenessErrors = trim(r.CompletenessErrors)
	r.HumanReviewRequirements = trim(r.HumanReviewRequirements)
	return r
}
func (r *WorkflowReport) totals() {
	r.StructuralTotal = len(r.StructuralErrors)
	r.CompletenessTotal = len(r.CompletenessErrors)
	r.HumanReviewTotal = len(r.HumanReviewRequirements)
	r.ReadyToSubmit = r.StructuralTotal == 0 && r.CompletenessTotal == 0
}
func reportFrom(r Report) WorkflowReport {
	return WorkflowReport{StructuralErrors: r.Errors, CompletenessErrors: []Issue{}, HumanReviewRequirements: []Issue{}}
}
func workflowLimits(p Package) error {
	if len(p.Knowledge) > 100 || len(p.Units) > 200 || len(p.Paths) > 20 || len(p.Assets) > 16 {
		return ErrLimit
	}
	b, err := json.Marshal(p)
	if err != nil {
		return ErrValidation
	}
	if len(b) > MaxWorkflowPackageBytes {
		return ErrLimit
	}
	return nil
}
func nonblank(v string) bool { return strings.TrimSpace(v) != "" }
func completeness(p Package, r *WorkflowReport) {
	missing := func(path string) {
		r.CompletenessErrors = append(r.CompletenessErrors, Issue{"INCOMPLETE_CONTENT", path, "Complete this field before submitting for independent review."})
	}
	if len(p.Knowledge) == 0 {
		missing("/knowledge")
	}
	units := map[VersionRef]int{}
	for _, u := range p.Units {
		units[u.Knowledge]++
	}
	for i, k := range p.Knowledge {
		base := fmt.Sprintf("/knowledge/%d", i)
		for _, field := range []struct{ name, value string }{{"title", k.Title}, {"titleZh", k.TitleZh}, {"statement", k.Statement}, {"scope", k.Scope}, {"system", k.System}} {
			if !nonblank(field.value) {
				missing(base + "/" + field.name)
			}
		}
		if len(k.Objectives) == 0 {
			missing(base + "/objectives")
		}
		for j, v := range k.Objectives {
			if !nonblank(v) {
				missing(fmt.Sprintf("%s/objectives/%d", base, j))
			}
		}
		if k.Type == "theorem" && !nonblank(k.Proof) {
			missing(base + "/proof")
		}
		if units[VersionRef{k.ID, k.Version}] == 0 {
			missing(base + "/units")
		}
		if len(k.Sources) == 0 {
			missing(base + "/sources")
		}
		for j, s := range k.Sources {
			path := fmt.Sprintf("%s/sources/%d", base, j)
			if !nonblank(s.Author) || !nonblank(s.Title) || !nonblank(s.License) || !nonblank(s.Attribution) {
				missing(path)
			}
			if s.Kind == "external" {
				u, err := url.Parse(s.URL)
				date, e := time.Parse("2006-01-02", s.AccessedAt)
				if err != nil || u.Scheme != "https" || u.Hostname() == "" || u.User != nil || e != nil || date.Format("2006-01-02") != s.AccessedAt {
					missing(path)
				}
			}
		}
	}
	for i, u := range p.Units {
		base := fmt.Sprintf("/units/%d", i)
		kinds := map[string]bool{}
		for j, a := range u.Angles {
			if !nonblank(a.Kind) || !nonblank(a.Body) {
				missing(fmt.Sprintf("%s/angles/%d", base, j))
			} else {
				kinds[a.Kind] = true
			}
		}
		if len(kinds) < 2 {
			missing(base + "/angles")
		}
		if len(u.Examples) == 0 {
			missing(base + "/examples")
		}
		for j, v := range u.Examples {
			if !nonblank(v) {
				missing(fmt.Sprintf("%s/examples/%d", base, j))
			}
		}
	}
	for i, a := range p.Assets {
		if !nonblank(a.Author) || !nonblank(a.License) || !nonblank(a.Attribution) {
			missing(fmt.Sprintf("/assets/%d", i))
		}
	}
	// Human judgement is always outstanding; readiness here means only safe to submit.
	if len(p.Knowledge) > 0 {
		for _, name := range []string{"mathematics", "explanations", "relationships", "sources", "illustrations"} {
			r.HumanReviewRequirements = append(r.HumanReviewRequirements, Issue{"REVIEW_REQUIRED", "/review/" + name, "An independent reviewer must verify this criterion."})
		}
	}
}
func pathOrder(p Package, r *WorkflowReport) {
	km := map[VersionRef]Knowledge{}
	for _, k := range p.Knowledge {
		km[VersionRef{k.ID, k.Version}] = k
	}
	for i, path := range p.Paths {
		positions := map[VersionRef]int{}
		for j, ref := range path.Nodes {
			positions[ref] = j
		}
		for j, ref := range path.Nodes {
			for _, rel := range km[ref].Relations {
				if rel.Kind == "prerequisite" {
					at, ok := positions[rel.Target]
					if !ok || at >= j {
						r.StructuralErrors = append(r.StructuralErrors, Issue{"PATH_ORDER", fmt.Sprintf("/paths/%d/nodes/%d", i, j), "Prerequisites must precede their dependents."})
					}
				}
			}
		}
	}
}
func ValidateWorkflow(ctx context.Context, c catalogue.Catalogue, p Package, reader AssetReader) (ValidatedPackage, WorkflowReport) {
	if err := workflowLimits(p); err != nil {
		return ValidatedPackage{}, WorkflowReport{StructuralErrors: []Issue{{"CONTENT_LIMIT_EXCEEDED", "/", "Content exceeds the authoring limits."}}, CompletenessErrors: []Issue{}, HumanReviewRequirements: []Issue{}, StructuralTotal: 1}
	}
	v, base := validateAndSeal(ctx, c, p, reader, MaxWorkflowPackageBytes, MaxWorkflowAssetBytes, false)
	r := reportFrom(base)
	completeness(p, &r)
	pathOrder(p, &r)
	r.totals()
	if !r.ReadyToSubmit {
		return ValidatedPackage{}, r
	}
	return v, r
}
func ValidateEditable(ctx context.Context, c catalogue.Catalogue, p Package, reader AssetReader) (WorkflowReport, error) {
	if err := ctx.Err(); err != nil {
		return WorkflowReport{}, err
	}
	if err := workflowLimits(p); err != nil {
		return WorkflowReport{}, err
	}
	_, base := validateAndSeal(ctx, c, p, reader, MaxWorkflowPackageBytes, MaxWorkflowAssetBytes, false)
	r := reportFrom(base)
	completeness(p, &r)
	pathOrder(p, &r)
	r.totals()
	if err := ctx.Err(); err != nil {
		return r, err
	}
	for _, issue := range r.StructuralErrors {
		switch issue.Code {
		case "INVALID_PACKAGE", "INVALID_CATALOGUE", "INVALID_ASSET", "UNSAFE_MARKUP", "UNSAFE_SOURCE_URL", "ASSETS_TOO_LARGE", "CANCELLED":
			return r, ErrValidation
		}
	}
	return r, nil
}
