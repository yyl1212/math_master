package study

import (
	"encoding/json"
	"github.com/yyl1212/math_master/backend/internal/taxonomy"
	"regexp"
	"unicode/utf8"
)

const MaxSequence int64 = 9007199254740991

var mathID = regexp.MustCompile(`^[a-z][a-z0-9-]{0,63}$`)
var uuid = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)

func ValidID(id string) bool          { return uuid.MatchString(id) }
func ValidKnowledgeID(id string) bool { return mathID.MatchString(id) }
func ValidRef(r KnowledgeRef) bool {
	return ValidKnowledgeID(r.ID) && r.Version >= 1 && r.Version <= 2147483647 && taxonomy.ValidSHA(r.SHA256)
}
func ValidateCommand(in CommandInput) error {
	b, e := json.Marshal(in)
	if e != nil || len(b) > 8192 || !ValidRef(in.Knowledge) || !ValidID(in.ExpectedKnowledgeHead) || in.ExpectedSequence < 0 || in.ExpectedSequence > MaxSequence {
		return ErrInvalid
	}
	return nil
}
func ValidateNote(body string) error {
	if !taxonomy.ValidText(body) || len(body) > 64<<10 || utf8.RuneCountInString(body) > 16000 {
		return ErrInvalid
	}
	return nil
}
