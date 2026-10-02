package learning

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"github.com/yyl1212/math_master/backend/internal/question"
)

func CanonicalLearningEvent(e EventSeal) ([]byte, string, error) {
	if e.Kind != "started" && e.Kind != "completed" {
		return nil, "", question.ErrInvalid
	}
	e.Units = append([]question.Identity{}, e.Units...)
	e.Assets = append([]question.AssetRef{}, e.Assets...)
	e.RecordedAt = e.RecordedAt.UTC()
	raw, err := json.Marshal(struct {
		Purpose string    `json:"purpose"`
		Body    EventSeal `json:"body"`
	}{"learning-event-v1", e})
	if err != nil {
		return nil, "", err
	}
	if len(raw) > 4194304 {
		return nil, "", question.ErrLimitExceeded
	}
	h := sha256.Sum256(raw)
	return raw, hex.EncodeToString(h[:]), nil
}
