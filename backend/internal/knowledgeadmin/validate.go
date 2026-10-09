package knowledgeadmin

import (
	"encoding/json"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/text"
	"github.com/yyl1212/math_master/schemas"
	"net/url"
	"regexp"
	"sort"
	"strings"
	"sync"
)

type ClassificationIndex struct{ Specific, Other map[string]bool }

var indexOnce sync.Once
var classificationIndex ClassificationIndex
var indexError error

func LoadClassificationIndex() (ClassificationIndex, error) {
	indexOnce.Do(func() {
		b, e := schemas.Files.ReadFile("msc2020-classification-codes.json")
		if e != nil || DigestBytes(b) != "412c7e175f5b0ea7a4da7ef3e3d281438d363b69d38eb287a8d7c67a969bd07b" {
			indexError = ErrNotConfigured
			return
		}
		var d struct {
			Specific []string `json:"specific_codes"`
			Other    []string `json:"other_codes"`
		}
		if json.Unmarshal(b, &d) != nil || len(d.Specific) != 4969 || len(d.Other) != 534 {
			indexError = ErrNotConfigured
			return
		}
		classificationIndex = ClassificationIndex{map[string]bool{}, map[string]bool{}}
		for _, k := range d.Specific {
			classificationIndex.Specific[k] = true
		}
		for _, k := range d.Other {
			classificationIndex.Other[k] = true
		}
	})
	return classificationIndex, indexError
}

var placeholder = regexp.MustCompile(`(?i)^(?:unknown|unspecified|undefined|tbd|tba|tbc|pending|needs_review|pending_context|needs_semantic_review|unverified|unavailable|not available|not known|未知|待定|待确认|待核对|待核实|未填写|未确定|未核实|不详|不明|暂缺|待补充|未指定|未评估)$`)
var identityKeys = map[string]bool{"id": true, "source_id": true, "dataset_id": true, "record_id": true, "target_id": true, "target_source_id": true, "work_family_id": true}

func knownValues(v any, key, path string) error {
	switch x := v.(type) {
	case nil:
		return invalid(path)
	case string:
		if strings.ContainsRune(x, 0) || !identityKeys[key] && placeholder.MatchString(strings.TrimSpace(x)) {
			return invalid(path)
		}
	case map[string]any:
		for k, v := range x {
			if strings.ContainsRune(k, 0) {
				return invalid(path)
			}
			if e := knownValues(v, k, strings.TrimSuffix(path, "/")+"/"+k); e != nil {
				return e
			}
		}
	case []any:
		for _, v := range x {
			if e := knownValues(v, key, path); e != nil {
				return e
			}
		}
	}
	return nil
}
func safeSourceURL(s string) bool {
	if s == "" {
		return true
	}
	u, e := url.Parse(s)
	return e == nil && u.Scheme == "https" && u.Hostname() != "" && u.User == nil && !strings.ContainsAny(s, "\\\x00\r\n")
}

var unsafeMacro = regexp.MustCompile(`(?i)\\(?:href|url|html[a-z]*|input|include[a-z]*|def|gdef|edef|xdef|let|newcommand|renewcommand|providecommand|newenvironment|renewenvironment|usepackage|require|write|openout|read|catcode|csname)\b`)

func safeMarkdown(s string) bool {
	if unsafeMacro.MatchString(s) {
		return false
	}
	tree := goldmark.New().Parser().Parse(text.NewReader([]byte(s)))
	safe := true
	_ = ast.Walk(tree, func(n ast.Node, enter bool) (ast.WalkStatus, error) {
		if !enter {
			return ast.WalkContinue, nil
		}
		switch n := n.(type) {
		case *ast.RawHTML, *ast.HTMLBlock, *ast.Image:
			safe = false
		case *ast.Link:
			if !safeSourceURL(string(n.Destination)) {
				safe = false
			}
		}
		return ast.WalkContinue, nil
	})
	return safe
}
func ValidatePoint(p SourcePoint, index ClassificationIndex) error {
	b, e := json.Marshal(p)
	if e != nil {
		return invalid("/")
	}
	v, e := StrictJSON(b)
	if e != nil {
		return e
	}
	if e = validateSchema(v, true); e != nil {
		return e
	}
	if e = knownValues(v, "", "/"); e != nil {
		return e
	}
	if len(p.ID) > 512 {
		return invalid("/id")
	}
	topics := map[string]bool{}
	for _, c := range p.MSCCodes {
		if !index.Specific[c] && !index.Other[c] {
			return invalid("/msc_codes")
		}
		topics[c] = true
	}
	evidence := map[string]bool{}
	m := v.(map[string]any)
	for _, ev := range p.ClassificationEvidence {
		if evidence[ev.MSCCode] || !topics[ev.MSCCode] || index.Specific[ev.MSCCode] != (ev.Kind == "specific") {
			return invalid("/classification_evidence")
		}
		evidence[ev.MSCCode] = true
		for _, f := range ev.EvidenceFields {
			if !hasEvidence(m[f]) {
				return invalid("/" + f)
			}
		}
	}
	if len(evidence) != len(topics) {
		return invalid("/classification_evidence")
	}
	for c := range topics {
		if index.Other[c] {
			for t := range topics {
				if index.Specific[t] && t[:3] == c[:3] {
					return invalid("/msc_codes")
				}
			}
		}
	}
	if p.ProjectOther != nil {
		for _, f := range p.ProjectOther.EvidenceFields {
			if !hasEvidence(m[f]) {
				return invalid("/project_other")
			}
		}
	}
	if p.OriginalBinding != nil {
		v := p.OriginalBinding
		if v.RecordID != p.ID || !SafeRelativePath(v.LocalRelativePath) {
			return invalid("/original_binding")
		}
	}
	for _, v := range p.Provenance {
		if !safeSourceURL(v.URL) {
			return invalid("/provenance/url")
		}
	}
	fields := map[string][]string{"statement": {p.Statement}, "conditions": p.Conditions, "scope": {p.Scope}, "system": {p.System}, "proof": {p.Proof}, "equations": p.Equations, "examples": p.Examples, "counterexamples": p.Counterexamples, "common_misconceptions": p.CommonMisconceptions}
	for _, v := range p.Explanations {
		fields["explanations"] = append(fields["explanations"], v.Body)
	}
	for k, values := range fields {
		for _, s := range values {
			if !safeMarkdown(s) {
				return &DecodeError{"UNSAFE_MARKUP", "/" + k}
			}
		}
	}
	return nil
}
func hasEvidence(v any) bool {
	switch x := v.(type) {
	case string:
		return strings.TrimSpace(x) != ""
	case []any:
		return len(x) > 0
	}
	return false
}
func SafeRelativePath(s string) bool {
	if s == "" || strings.HasPrefix(s, "/") || strings.ContainsAny(s, "\\\x00\r\n") || len(s) > 1 && s[1] == ':' {
		return false
	}
	for _, p := range strings.Split(s, "/") {
		if p == "" || p == "." || p == ".." {
			return false
		}
	}
	return true
}
func TopicKeys(p SourcePoint) []string {
	if p.ClassificationMode == "project_other" {
		return []string{"project:other"}
	}
	out := append([]string{}, p.MSCCodes...)
	sort.Strings(out)
	return out
}
func ValidateCurrent(i CurrentInput) error {
	if i.ExternalID != i.Point.ID || i.ExternalID == "" {
		return invalid("/externalId")
	}
	ix, e := LoadClassificationIndex()
	if e != nil {
		return e
	}
	if e = ValidatePoint(i.Point, ix); e != nil {
		return e
	}
	if i.TopicKeys != nil {
		if len(i.TopicKeys) == 0 || len(i.TopicKeys) > 32 {
			return invalid("/topicKeys")
		}
		seen := map[string]bool{}
		for _, t := range i.TopicKeys {
			if seen[t] || !ix.Specific[t] && !ix.Other[t] && t != "project:other" {
				return invalid("/topicKeys")
			}
			seen[t] = true
		}
	}
	if len(i.Sources) == 0 {
		return invalid("/sources")
	}
	for _, s := range i.Sources {
		if s.SourceID == "" || s.Title == "" || s.Citation == "" || s.URL != nil && !safeSourceURL(*s.URL) {
			return invalid("/sources")
		}
	}
	return nil
}
func validateRelations(d SourceDocument) error {
	ids := map[string]SourcePoint{}
	for _, p := range d.KnowledgePoints {
		ids[p.ID] = p
	}
	graph := map[string][]string{}
	for _, p := range d.KnowledgePoints {
		seen := map[string]bool{}
		for _, r := range p.Relations {
			k := r.Kind + "/" + r.TargetSourceID + "/" + r.TargetID
			if seen[k] || r.TargetID == p.ID {
				return invalid("/relations")
			}
			seen[k] = true
			if r.Status == "confirmed" && r.TargetSourceID == d.Source.SourceID {
				target, ok := ids[r.TargetID]
				if !ok || target.Version != r.TargetVersion {
					return invalid("/relations")
				}
				if r.Kind == "prerequisite" {
					graph[p.ID] = append(graph[p.ID], r.TargetID)
				}
			}
		}
	}
	visiting, done := map[string]bool{}, map[string]bool{}
	var visit func(string) bool
	visit = func(id string) bool {
		if visiting[id] {
			return false
		}
		if done[id] {
			return true
		}
		visiting[id] = true
		for _, t := range graph[id] {
			if !visit(t) {
				return false
			}
		}
		visiting[id] = false
		done[id] = true
		return true
	}
	for id := range ids {
		if !visit(id) {
			return invalid("/relations")
		}
	}
	return nil
}

func CurrentTopicKeys(i CurrentInput) []string {
	if i.TopicKeys == nil {
		return TopicKeys(i.Point)
	}
	out := append([]string{}, i.TopicKeys...)
	sort.Strings(out)
	return out
}
