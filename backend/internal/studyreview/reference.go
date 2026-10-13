package studyreview

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"github.com/yyl1212/math_master/backend/internal/knowledgeadmin"
	"github.com/yyl1212/math_master/backend/internal/taxonomy"
	"strings"
	"time"
)

func section(kind string, items ...string) []Section {
	result := make([]string, 0, len(items))
	for _, s := range items {
		if strings.TrimSpace(s) != "" {
			result = append(result, s)
		}
	}
	if len(result) == 0 {
		return nil
	}
	return []Section{{Kind: kind, Items: result}}
}
func BuildReference(point knowledgeadmin.SourcePoint, mode Mode, target Target) (FrozenMaterial, error) {
	m := FrozenMaterial{Prompt: []Section{}, Expected: []Section{}}
	if !validModeTarget(mode, target) {
		return m, ErrInvalid
	}
	if !validQueryText(point.Title, 1024, 4096) || !validQueryText(point.TitleZH, 1024, 4096) || !validKnowledgeType(point.Type) {
		return m, ErrReferenceTooLarge
	}
	statement := []Section{}
	statement = append(statement, section("statement", point.Statement)...)
	statement = append(statement, section("conditions", point.Conditions...)...)
	statement = append(statement, section("scope", point.Scope)...)
	statement = append(statement, section("system", point.System)...)
	statement = append(statement, section("equations", point.Equations...)...)
	proof := append(section("proof", point.Proof), section("proof-scope", point.ProofScope)...)
	switch {
	case mode == Read:
		m.Expected = append(statement, proof...)
	case target == Statement:
		m.Expected = statement
	case target == Proof:
		if strings.TrimSpace(point.Proof) == "" {
			return m, ErrInvalid
		}
		m.Prompt = statement
		m.Expected = proof
	}
	if len(m.Expected) == 0 {
		return m, ErrInvalid
	}
	raw := 0
	for _, sections := range [][]Section{m.Prompt, m.Expected} {
		for _, s := range sections {
			for _, item := range s.Items {
				if !taxonomy.ValidText(item) {
					return m, ErrInvalid
				}
				raw += len(item)
				if raw > MaxMaterialBytes {
					return m, ErrReferenceTooLarge
				}
			}
		}
	}
	canonical, e := json.Marshal(struct {
		Prompt   []Section `json:"prompt"`
		Expected []Section `json:"expected"`
	}{m.Prompt, m.Expected})
	if e != nil {
		return m, ErrInvalid
	}
	canonical, e = knowledgeadmin.CanonicalJSON(canonical)
	if e != nil {
		return m, ErrInvalid
	}
	hash := sha256.Sum256(canonical)
	m.SHA256 = hex.EncodeToString(hash[:])
	prompt, _ := json.Marshal(m.Prompt)
	reference, _ := json.Marshal(ExpectedReference{Sections: m.Expected, SHA256: m.SHA256})
	if len(prompt)+len(reference) > MaxMaterialWireBytes {
		return m, ErrReferenceTooLarge
	}
	// Exercise the maximum metadata envelope, not only initial null fields.
	n := time.Date(9999, 1, 1, 23, 59, 59, 999999999, time.UTC)
	ref := Ref{ID: "k-" + strings.Repeat("f", 56), ContentSHA256: strings.Repeat("f", 64), SourceKind: "managed"}
	sch := Schedule{Stage: 5, DueAt: n, LastRef: ref, LastRating: Forgotten, Sequence: MaxSequence, GradedAt: n}
	envelope := Session{ActorID: strings.Repeat("f", 36), SessionID: strings.Repeat("f", 36), KnowledgeID: ref.ID, ReviewID: strings.Repeat("f", 36), Knowledge: ref, Mode: Recall, Target: Statement, Status: Abandoned, Sequence: MaxSequence, DraftRevision: MaxSequence, Title: point.Title, TitleZH: point.TitleZH, KnowledgeType: point.Type, CreatedAt: n, ComparedAt: &n, GradedAt: &n, AbandonedAt: &n, DraftUpdatedAt: &n, Prompt: []Section{}, Schedule: &sch, Grade: &GradeResult{Rating: Forgotten, GradedAt: n, Schedule: sch}}
	metadata, _ := json.Marshal(envelope)
	if len(metadata) > MaxMetadataWireBytes || len(prompt)+len(reference)+len(metadata)+MaxAnswerWireBytes > MaxResponseBytes {
		return m, ErrReferenceTooLarge
	}
	return m, nil
}
func validKnowledgeType(t string) bool {
	switch t {
	case "concept", "definition", "axiom", "theorem", "corollary", "method", "mathematical-thinking", "other":
		return true
	}
	return false
}

// ValidateSessionBudget checks the exact JSON rule used by the HTTP response writer.
func ValidateSessionBudget(s Session) error {
	if s.Answer != nil && ValidateAnswer(*s.Answer) != nil {
		return ErrInvalid
	}
	b, e := json.Marshal(s)
	if e != nil {
		return ErrInvalid
	}
	if len(b) > MaxResponseBytes {
		return ErrReferenceTooLarge
	}
	return nil
}
func ClipTitle(s string) string {
	r := []rune(s)
	if len(r) > 128 {
		return string(r[:128])
	}
	return s
}
