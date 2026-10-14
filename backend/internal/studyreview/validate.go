package studyreview

import (
	"github.com/yyl1212/math_master/backend/internal/knowledgeadmin"
	"github.com/yyl1212/math_master/backend/internal/study"
	"github.com/yyl1212/math_master/backend/internal/taxonomy"
	"regexp"
	"strings"
	"unicode/utf8"
)

var topicKey = regexp.MustCompile(`^(?:[0-9]{2}(?:-XX|[A-Z](?:xx|[0-9]{2}))?|project:other)$`)

func ValidateAnswer(body string) error {
	if study.ValidateNote(body) != nil {
		return ErrInvalid
	}
	return nil
}
func validSequence(n int64) bool { return n >= 1 && n < MaxSequence }
func validRating(r Rating) bool  { return r == Forgotten || r == Uncertain || r == Familiar }
func validModeTarget(mode Mode, target Target) bool {
	return (mode == Read && target == Statement) || (mode == Recall && (target == Statement || target == Proof))
}
func ValidateStart(in StartInput) error {
	if !knowledgeadmin.ValidManagedRef(in.Knowledge) || !validModeTarget(in.Mode, in.Target) {
		return ErrInvalid
	}
	return nil
}
func ValidateDraft(in DraftInput) error {
	if !validSequence(in.ExpectedSequence) || in.ExpectedDraftRevision < 0 || in.ExpectedDraftRevision >= MaxSequence {
		return ErrInvalid
	}
	return ValidateAnswer(in.Body)
}
func ValidateCompare(mode Mode, in CompareInput) error {
	if !validSequence(in.ExpectedSequence) || ValidateAnswer(in.Body) != nil {
		return ErrInvalid
	}
	if mode == Read {
		if in.Body != "" || in.Unanswered {
			return ErrInvalid
		}
		return nil
	}
	if mode != Recall || (strings.TrimSpace(in.Body) == "") != in.Unanswered {
		return ErrInvalid
	}
	return nil
}
func ValidateGrade(unanswered bool, in GradeInput) error {
	if !validSequence(in.ExpectedSequence) || !validRating(in.Rating) || (unanswered && in.Rating != Forgotten) {
		return ErrInvalid
	}
	return nil
}
func ValidateAbandon(in AbandonInput) error {
	if !validSequence(in.ExpectedSequence) {
		return ErrInvalid
	}
	return nil
}
func validQueryText(s string, chars, bytes int) bool {
	return taxonomy.ValidText(s) && utf8.RuneCountInString(s) <= chars && len(s) <= bytes
}
func ValidateQueue(q QueueQuery) error {
	if q.Limit < 1 || q.Limit > 50 || q.Offset < 0 || q.Offset > 100000 || !validQueryText(q.Q, 128, 512) || (q.TopicKey != "" && !topicKey.MatchString(q.TopicKey)) {
		return ErrInvalid
	}
	switch q.Bucket {
	case "", "due", "changed", "unplanned", "unfinished", "future", "all":
		return nil
	}
	return ErrInvalid
}
func ValidateHistory(q HistoryQuery) error {
	if q.Limit < 1 || q.Limit > 50 || (q.KnowledgeID != "" && !knowledgeadmin.ValidManagedRef(Ref{ID: q.KnowledgeID, ContentSHA256: strings.Repeat("0", 64), SourceKind: "managed"})) || (q.TopicKey != "" && !topicKey.MatchString(q.TopicKey)) || (q.Mode != "" && q.Mode != Read && q.Mode != Recall) || (q.Rating != "" && !validRating(q.Rating)) || len(q.Cursor) > 512 || !taxonomy.ValidText(q.Cursor) {
		return ErrInvalid
	}
	return nil
}
