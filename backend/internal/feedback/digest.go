package feedback

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"github.com/yyl1212/math_master/backend/internal/auth"
	"strings"
	"unicode/utf8"
)

func CanonicalCommand(action Action, resource string, input any) ([]byte, string, error) {
	if !IsWrite(action) || !utf8.ValidString(resource) || resource == "" || len(resource) > 512 || strings.ContainsRune(resource, 0) {
		return nil, "", auth.ErrInvalidInput
	}
	switch action {
	case CreateAction:
		in, ok := input.(CreateInput)
		if !ok || ValidateCreate(in) != nil {
			return nil, "", auth.ErrInvalidInput
		}
	case ReplyAction:
		in, ok := input.(ReplyInput)
		if !ok || ValidateReply(in) != nil {
			return nil, "", auth.ErrInvalidInput
		}
	case TransitionAction:
		in, ok := input.(TransitionInput)
		if !ok || ValidateTransitionInput(in) != nil {
			return nil, "", auth.ErrInvalidInput
		}
	}
	raw, e := json.Marshal(struct {
		Purpose  string `json:"purpose"`
		Action   Action `json:"action"`
		Resource string `json:"resource"`
		Input    any    `json:"input"`
	}{"feedback-command-v1", action, resource, input})
	if e != nil {
		return nil, "", e
	}
	h := sha256.Sum256(raw)
	return raw, hex.EncodeToString(h[:]), nil
}
