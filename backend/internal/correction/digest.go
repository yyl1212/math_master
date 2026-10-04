package correction

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"github.com/yyl1212/math_master/backend/internal/auth"
	"reflect"
	"strings"
	"unicode/utf8"
)

func Canonical(purpose string, value any) ([]byte, string, error) {
	switch purpose {
	case "correction-case-v1", "correction-plan-v1", "correction-result-v1", "correction-command-v1", "notification-command-v1":
	default:
		return nil, "", auth.ErrInvalidInput
	}
	if !canonicalStrings(reflect.ValueOf(value), 0) {
		return nil, "", auth.ErrInvalidInput
	}
	b, e := json.Marshal(struct {
		Purpose string `json:"purpose"`
		Body    any    `json:"body"`
	}{purpose, value})
	if e != nil {
		return nil, "", e
	}
	h := sha256.Sum256(b)
	return b, hex.EncodeToString(h[:]), nil
}
func canonicalStrings(v reflect.Value, depth int) bool {
	if depth > 64 {
		return false
	}
	if !v.IsValid() {
		return true
	}
	switch v.Kind() {
	case reflect.Interface, reflect.Pointer:
		if v.IsNil() {
			return true
		}
		return canonicalStrings(v.Elem(), depth+1)
	case reflect.String:
		s := v.String()
		return utf8.ValidString(s) && !strings.ContainsRune(s, 0)
	case reflect.Map:
		it := v.MapRange()
		for it.Next() {
			if !canonicalStrings(it.Key(), depth+1) || !canonicalStrings(it.Value(), depth+1) {
				return false
			}
		}
	case reflect.Slice, reflect.Array:
		for n := 0; n < v.Len(); n++ {
			if !canonicalStrings(v.Index(n), depth+1) {
				return false
			}
		}
	case reflect.Struct:
		for n := 0; n < v.NumField(); n++ {
			if v.Type().Field(n).IsExported() && !canonicalStrings(v.Field(n), depth+1) {
				return false
			}
		}
	}
	return true
}
