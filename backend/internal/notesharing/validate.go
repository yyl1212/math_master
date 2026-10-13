package notesharing

import (
	"bytes"
	"encoding/json"
	"github.com/yyl1212/math_master/backend/internal/knowledgeadmin"
	"github.com/yyl1212/math_master/backend/internal/study"
	"github.com/yyl1212/math_master/backend/internal/taxonomy"
	"io"
	"regexp"
	"strings"
	"unicode/utf8"
)

var topic = regexp.MustCompile(`^(?:[0-9]{2}(?:-XX|[A-Z](?:xx|[0-9]{2}))?|project:other)$`)

func ValidSequence(v int64) bool { return v >= 0 && v < study.MaxSequence }
func ValidateSubmit(in SubmitInput) error {
	t := strings.TrimSpace(in.Title)
	if !ValidSequence(in.ExpectedSequence) || in.ExpectedRevision < 1 || in.ExpectedRevision > study.MaxSequence || !taxonomy.ValidText(t) || utf8.RuneCountInString(t) < 1 || utf8.RuneCountInString(t) > 120 {
		return ErrInvalid
	}
	return nil
}
func ValidateWithdraw(in WithdrawInput) error {
	if !study.ValidID(in.SubmissionID) || !ValidSequence(in.ExpectedSequence) {
		return ErrInvalid
	}
	return nil
}
func ValidateDecision(action string, in DecisionInput) error {
	if !ValidSequence(in.ExpectedSequence) {
		return ErrInvalid
	}
	if action == "approve" {
		if in.Reason != "" {
			return ErrInvalid
		}
		return nil
	}
	if action != "return" && action != "hide" {
		return ErrInvalid
	}
	r := strings.TrimSpace(in.Reason)
	n := utf8.RuneCountInString(r)
	if !taxonomy.ValidText(r) || n < 10 || n > 1000 {
		return ErrInvalid
	}
	return nil
}
func ValidateBody(body string) error {
	if strings.TrimSpace(body) == "" || study.ValidateNote(body) != nil {
		return ErrInvalid
	}
	return nil
}
func ValidateQuery(q Query) error {
	if !taxonomy.ValidText(q.Q) || utf8.RuneCountInString(q.Q) > 128 || len(q.Q) > 512 || len(q.Cursor) > 512 || q.Limit < 0 || q.Limit > 50 {
		return ErrInvalid
	}
	if q.TopicKey != "" && !topic.MatchString(q.TopicKey) {
		return ErrInvalid
	}
	if q.Status != "" && !ValidStatus(q.Status) {
		return ErrInvalid
	}
	return nil
}
func ValidStatus(s string) bool {
	switch s {
	case Unshared, Pending, Shared, Returned, Withdrawn, Hidden:
		return true
	}
	return false
}
func Decode(r io.Reader, out any) error {
	b, e := io.ReadAll(io.LimitReader(r, MaxRequestBytes+1))
	if e != nil || len(b) > MaxRequestBytes || !utf8.Valid(b) {
		return ErrInvalid
	}
	if _, e = knowledgeadmin.CanonicalJSON(b); e != nil {
		return ErrInvalid
	}
	d := json.NewDecoder(bytes.NewReader(b))
	d.DisallowUnknownFields()
	if e = d.Decode(out); e != nil {
		return ErrInvalid
	}
	var extra any
	if d.Decode(&extra) != io.EOF {
		return ErrInvalid
	}
	return nil
}
