package assessment

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"github.com/yyl1212/math_master/backend/internal/question"
)

// CanonicalSeal fixes the complete purpose wrapper. Publication proof and all
// relational invariants are independently enforced by the persistence boundary.
func CanonicalSeal(s Seal) ([]byte, string, error) {
	purpose := ""
	switch s.Kind {
	case "practice":
		purpose = "practice-attempt-v1"
	case "assessment":
		purpose = "assessment-attempt-v1"
	default:
		return nil, "", question.ErrInvalid
	}
	s.Core = append([]int{}, s.Core...)
	s.Items = append([]ItemBinding{}, s.Items...)
	for j := range s.Items {
		i := &s.Items[j]
		i.Coverage = append([]int{}, i.Coverage...)
		i.Units = append([]question.Identity{}, i.Units...)
		i.Assets = append([]question.AssetRef{}, i.Assets...)
	}
	raw, err := json.Marshal(struct {
		Purpose string `json:"purpose"`
		Body    Seal   `json:"body"`
	}{purpose, s})
	if err != nil {
		return nil, "", err
	}
	if len(raw) > 4194304 {
		return nil, "", question.ErrLimitExceeded
	}
	h := sha256.Sum256(raw)
	return raw, hex.EncodeToString(h[:]), nil
}
