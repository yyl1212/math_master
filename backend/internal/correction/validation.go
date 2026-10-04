package correction

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"github.com/yyl1212/math_master/backend/internal/auth"
	"github.com/yyl1212/math_master/backend/internal/question"
	"io"
	"reflect"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

func ValidSequence(n int64) bool { return n >= 1 && n <= MaxSequence }
func ValidPlanRef(p PlanRef) bool {
	return question.ValidID(p.ID) && p.Version >= 1 && p.Version <= MaxVersion
}
func ValidEvidence(r EvidenceRef, enrollment bool) bool {
	if !question.ValidID(r.ID) {
		return false
	}
	switch r.Kind {
	case LearningEventEvidence, PracticeEvidence, AssessmentEvidence:
		return true
	case EnrollmentEvidence:
		return enrollment
	}
	return false
}
func ValidIdentity(i question.Identity, instance bool) bool {
	return i.Version >= 1 && i.Version <= MaxVersion && question.ValidSHA(i.SHA256) && (instance && question.ValidInstanceID(i.ID) || !instance && question.ValidMathID(i.ID))
}
func ValidText(s string, min, max int) bool {
	return utf8.ValidString(s) && !strings.ContainsRune(s, 0) && utf8.RuneCountInString(s) >= min && utf8.RuneCountInString(s) <= max && (min == 0 && s == "" || strings.TrimSpace(s) != "")
}
func ValidateCase(in CaseInput) error {
	switch in.Kind {
	case WithdrawalCase:
		if in.Rule != nil || in.Withdrawal == nil || !question.ValidID(in.Withdrawal.ID) || (in.Withdrawal.Space != "content" && in.Withdrawal.Space != "question") {
			return auth.ErrInvalidInput
		}
	case GradingRuleCase:
		if in.Withdrawal != nil || in.Rule == nil || in.Rule.RuleVersion < 1 || in.Rule.RuleVersion > MaxVersion {
			return auth.ErrInvalidInput
		}
		r := in.Rule
		if r.Kind == "all" {
			if r.Knowledge != nil {
				return auth.ErrInvalidInput
			}
		} else if r.Kind != "knowledge" || r.Knowledge == nil || !ValidIdentity(*r.Knowledge, false) {
			return auth.ErrInvalidInput
		}
	default:
		return auth.ErrInvalidInput
	}
	return nil
}
func ValidatePlan(in PlanInput) error {
	if in.AlgorithmVersion != 1 || !ValidText(in.Reason, 1, 4000) || in.Mappings == nil || len(in.Mappings) > 50 || in.ExpectedSequence != nil && !ValidSequence(*in.ExpectedSequence) || in.Parent != nil && !ValidPlanRef(*in.Parent) {
		return auth.ErrInvalidInput
	}
	seen := map[question.Identity]bool{}
	for _, m := range in.Mappings {
		if !ValidIdentity(m.Original, true) || !ValidIdentity(m.Replacement.Identity, true) || !question.ValidID(m.OriginalPublicationID) || !question.ValidID(m.Replacement.PublicationID) || seen[m.Original] {
			return auth.ErrInvalidInput
		}
		seen[m.Original] = true
	}
	return nil
}
func ValidateSubmit(in SubmitInput) error {
	if !ValidSequence(in.ExpectedSequence) {
		return auth.ErrInvalidInput
	}
	return nil
}
func ValidateDecision(in DecisionInput) error {
	if !ValidSequence(in.ExpectedSequence) || (in.Decision != "approve" && in.Decision != "reject") || !ValidText(in.Reason, 1, 4000) {
		return auth.ErrInvalidInput
	}
	return nil
}
func ValidateRetry(in RetryInput) error { return ValidateSubmit(SubmitInput{in.ExpectedSequence}) }

type ListCursor struct {
	Version   int       `json:"version"`
	CreatedAt time.Time `json:"createdAt"`
	ID        string    `json:"id"`
}

func EncodeCursor(c ListCursor) (string, error) {
	if c.Version != 1 || c.CreatedAt.IsZero() || !question.ValidID(c.ID) {
		return "", auth.ErrInvalidInput
	}
	b, e := json.Marshal(c)
	if e != nil {
		return "", e
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}
func DecodeCursor(raw string) (ListCursor, error) {
	var c ListCursor
	if len(raw) == 0 || len(raw) > 512 {
		return c, auth.ErrInvalidInput
	}
	b, e := base64.RawURLEncoding.Strict().DecodeString(raw)
	if e != nil || strictDecode(b, &c) != nil {
		return c, auth.ErrInvalidInput
	}
	again, e := EncodeCursor(c)
	if e != nil || again != raw {
		return c, auth.ErrInvalidInput
	}
	return c, nil
}
func ValidateQuery(q Query) error {
	if q.Limit < 0 || q.Limit > 50 {
		return auth.ErrInvalidInput
	}
	if q.Cursor != "" {
		_, e := DecodeCursor(q.Cursor)
		return e
	}
	return nil
}

// DTO roundtripping proves exact key casing and the presence of nullable keys.
func strictDecode(raw []byte, out any) error {
	if len(raw) > MaxRequestBytes || !utf8.Valid(raw) || !json.Valid(raw) || !validCorrectionEscapes(raw) {
		return auth.ErrInvalidInput
	}
	d := json.NewDecoder(bytes.NewReader(raw))
	d.UseNumber()
	token, e := d.Token()
	if e != nil || walkJSON(d, token, 0) != nil {
		return auth.ErrInvalidInput
	}
	if _, e = d.Token(); e != io.EOF {
		return auth.ErrInvalidInput
	}
	d = json.NewDecoder(bytes.NewReader(raw))
	d.DisallowUnknownFields()
	if d.Decode(out) != nil {
		return auth.ErrInvalidInput
	}
	var got, want any
	encoded, e := json.Marshal(out)
	if e != nil || json.Unmarshal(raw, &got) != nil || json.Unmarshal(encoded, &want) != nil || !reflect.DeepEqual(got, want) {
		return auth.ErrInvalidInput
	}
	return nil
}
func walkJSON(d *json.Decoder, t json.Token, depth int) error {
	if depth > 16 {
		return auth.ErrInvalidInput
	}
	delimiter, ok := t.(json.Delim)
	if !ok {
		return nil
	}
	switch delimiter {
	case '{':
		seen := map[string]bool{}
		for d.More() {
			k, e := d.Token()
			if e != nil {
				return e
			}
			key, ok := k.(string)
			if !ok || seen[strings.ToLower(key)] {
				return auth.ErrInvalidInput
			}
			seen[strings.ToLower(key)] = true
			v, e := d.Token()
			if e != nil {
				return e
			}
			if e = walkJSON(d, v, depth+1); e != nil {
				return e
			}
		}
		end, e := d.Token()
		if e != nil || end != json.Delim('}') {
			return auth.ErrInvalidInput
		}
	case '[':
		for d.More() {
			v, e := d.Token()
			if e != nil {
				return e
			}
			if e = walkJSON(d, v, depth+1); e != nil {
				return e
			}
		}
		end, e := d.Token()
		if e != nil || end != json.Delim(']') {
			return auth.ErrInvalidInput
		}
	default:
		return auth.ErrInvalidInput
	}
	return nil
}
func (in *CaseInput) UnmarshalJSON(raw []byte) error {
	type plain CaseInput
	var p plain
	if e := strictDecode(raw, &p); e != nil {
		return e
	}
	*in = CaseInput(p)
	return nil
}
func (in *RuleScope) UnmarshalJSON(raw []byte) error {
	type plain RuleScope
	var p plain
	if e := strictDecode(raw, &p); e != nil {
		return e
	}
	*in = RuleScope(p)
	return nil
}
func (in *PlanInput) UnmarshalJSON(raw []byte) error {
	type plain PlanInput
	var p plain
	if e := strictDecode(raw, &p); e != nil {
		return e
	}
	*in = PlanInput(p)
	return nil
}

// encoding/json replaces unpaired surrogate escapes; reject them before decoding.
func validCorrectionEscapes(raw []byte) bool {
	for i := 0; i < len(raw); i++ {
		if raw[i] != '"' {
			continue
		}
		i++
		for i < len(raw) && raw[i] != '"' {
			if raw[i] != '\\' {
				i++
				continue
			}
			i++
			if raw[i] != 'u' {
				i++
				continue
			}
			n, err := strconv.ParseUint(string(raw[i+1:i+5]), 16, 16)
			if err != nil {
				return false
			}
			i += 5
			if n >= 0xdc00 && n <= 0xdfff {
				return false
			}
			if n >= 0xd800 && n <= 0xdbff {
				if i+6 > len(raw) || raw[i] != '\\' || raw[i+1] != 'u' {
					return false
				}
				low, err := strconv.ParseUint(string(raw[i+2:i+6]), 16, 16)
				if err != nil || low < 0xdc00 || low > 0xdfff {
					return false
				}
				i += 6
			}
		}
	}
	return true
}
