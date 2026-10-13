package notesharing

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"github.com/yyl1212/math_master/backend/internal/knowledgeadmin"
	"github.com/yyl1212/math_master/backend/internal/study"
	"strings"
	"time"
)

type Cursor struct {
	At    time.Time `json:"at"`
	ID    string    `json:"id"`
	Scope string    `json:"scope"`
}

func QueryScope(kind, owner string, q Query) string {
	b, _ := json.Marshal([]string{kind, owner, q.Q, q.TopicKey, q.Status})
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}
func EncodeCursor(at time.Time, id, scope string) string {
	b, _ := json.Marshal(Cursor{at.UTC(), id, scope})
	return base64.RawURLEncoding.EncodeToString(b)
}
func DecodeCursor(raw, scope string) (Cursor, error) {
	var c Cursor
	if raw == "" {
		return c, nil
	}
	if len(raw) > 512 {
		return c, ErrInvalid
	}
	b, e := base64.RawURLEncoding.DecodeString(raw)
	if e != nil {
		return c, ErrInvalid
	}
	if Decode(strings.NewReader(string(b)), &c) != nil || c.Scope != scope || c.At.IsZero() || !(study.ValidID(c.ID) || knowledgeadmin.ValidManagedRef(knowledgeadmin.Ref{ID: c.ID, ContentSHA256: strings.Repeat("0", 64), SourceKind: "managed"})) {
		return c, ErrInvalid
	}
	return c, nil
}
func TopicPrefix(key string) string { return strings.TrimSuffix(strings.TrimSuffix(key, "-XX"), "xx") }
