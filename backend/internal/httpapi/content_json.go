package httpapi

import (
	"bytes"
	"encoding/json"
	"github.com/yyl1212/math_master/backend/internal/auth"
	"github.com/yyl1212/math_master/backend/internal/publication"
	"io"
	"reflect"
	"strings"
	"unicode/utf8"
)

func contentJSONFields(t reflect.Type) map[string]reflect.Type {
	out := map[string]reflect.Type{}
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		if f.PkgPath != "" {
			continue
		}
		name := strings.Split(f.Tag.Get("json"), ",")[0]
		if f.Anonymous && name == "" && f.Type.Kind() == reflect.Struct {
			for k, v := range contentJSONFields(f.Type) {
				out[k] = v
			}
			continue
		}
		if name != "" && name != "-" {
			out[name] = f.Type
		}
	}
	return out
}
func walkContentJSON(d *json.Decoder, token json.Token, t reflect.Type, depth int) (map[string]bool, error) {
	invalid := func() (map[string]bool, error) { return nil, auth.ErrInvalidInput }
	if token == nil {
		if t != nil && t.Kind() == reflect.Pointer && t.Elem().Kind() == reflect.String {
			return nil, nil
		}
		return invalid()
	}
	if t != nil && t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	if v, ok := token.(string); ok {
		if strings.ContainsRune(v, 0) {
			return invalid()
		}
		return nil, nil
	}
	delimiter, ok := token.(json.Delim)
	if !ok {
		return nil, nil
	}
	if depth > 32 {
		return invalid()
	}
	switch delimiter {
	case '{':
		var fields map[string]reflect.Type
		if t != nil && t.Kind() != reflect.Interface {
			if t.Kind() != reflect.Struct {
				return invalid()
			}
			fields = contentJSONFields(t)
		}
		kindValue := ""
		seen := map[string]bool{}
		folds := map[string]bool{}
		for d.More() {
			keyToken, err := d.Token()
			if err != nil {
				return invalid()
			}
			key, ok := keyToken.(string)
			if !ok || strings.ContainsRune(key, 0) {
				return invalid()
			}
			fold := strings.ToLower(key)
			if folds[fold] {
				return invalid()
			}
			folds[fold] = true
			seen[key] = true
			var child reflect.Type
			if fields != nil {
				var exists bool
				child, exists = fields[key]
				if !exists {
					return invalid()
				}
			}
			value, err := d.Token()
			if err != nil {
				return invalid()
			}
			if key == "kind" {
				kindValue, _ = value.(string)
			}
			if _, err = walkContentJSON(d, value, child, depth+1); err != nil {
				return nil, err
			}
		}
		end, err := d.Token()
		if err != nil || end != json.Delim('}') {
			return invalid()
		}
		if t == reflect.TypeOf(publication.WithdrawalTarget{}) {
			if kindValue == "asset" {
				if len(seen) != 2 || !seen["sha256"] {
					return invalid()
				}
			} else if kindValue == "knowledge" || kindValue == "unit" || kindValue == "path" {
				if len(seen) != 3 || !seen["id"] || !seen["version"] {
					return invalid()
				}
			} else {
				return invalid()
			}
		} else if fields != nil {
			for key := range fields {
				if !seen[key] {
					return invalid()
				}
			}
		}
		return seen, nil
	case '[':
		var child reflect.Type
		if t != nil && t.Kind() != reflect.Interface {
			if t.Kind() != reflect.Slice && t.Kind() != reflect.Array {
				return invalid()
			}
			child = t.Elem()
		}
		for d.More() {
			value, err := d.Token()
			if err != nil {
				return invalid()
			}
			if _, err = walkContentJSON(d, value, child, depth+1); err != nil {
				return nil, err
			}
		}
		end, err := d.Token()
		if err != nil || end != json.Delim(']') {
			return invalid()
		}
		return nil, nil
	}
	return invalid()
}
func contentRequiredFields(dst any) []string {
	switch dst.(type) {
	case *publication.DraftInput:
		return []string{"catalogueVersion", "package", "assetBytes", "sourceMap"}
	case *publication.SaveDraftInput:
		return []string{"catalogueVersion", "package", "assetBytes", "sourceMap", "expectedRevision"}
	case *publication.AdoptInput:
		return []string{"packageId", "packageVersion", "reason"}
	case *publication.ValidateInput:
		return []string{"expectedRevision"}
	case *publication.SubmitInput:
		return []string{"expectedRevision", "expectedDigest"}
	case *publication.ReviewInput:
		return []string{"decision", "checks", "independenceNote", "note"}
	case *publication.PrepareInput:
		return []string{"submissionIds", "expectedHead", "reason"}
	case *publication.ActivateInput:
		return []string{"expectedHead", "expectedManifestSha", "reason"}
	case *publication.WithdrawalPreviewInput:
		return []string{"target"}
	case *publication.WithdrawalInput:
		return []string{"target", "expectedHead", "reason"}
	}
	return nil
}
func decodeContentJSON(reader io.Reader, limit int64, dst any) error {
	raw, err := io.ReadAll(io.LimitReader(reader, limit+1))
	if int64(len(raw)) > limit {
		return errContentPayloadTooLarge
	}
	if err != nil || !utf8.Valid(raw) || !json.Valid(raw) || !validPrivateEscapes(raw) {
		return auth.ErrInvalidInput
	}
	t := reflect.TypeOf(dst)
	if t == nil || t.Kind() != reflect.Pointer || t.Elem().Kind() != reflect.Struct {
		return auth.ErrInvalidInput
	}
	d := json.NewDecoder(bytes.NewReader(raw))
	d.UseNumber()
	first, err := d.Token()
	if err != nil || first != json.Delim('{') {
		return auth.ErrInvalidInput
	}
	seen, err := walkContentJSON(d, first, t.Elem(), 1)
	if err != nil {
		return err
	}
	for _, name := range contentRequiredFields(dst) {
		if !seen[name] {
			return auth.ErrInvalidInput
		}
	}
	if _, err = d.Token(); err != io.EOF {
		return auth.ErrInvalidInput
	}
	d = json.NewDecoder(bytes.NewReader(raw))
	d.DisallowUnknownFields()
	if d.Decode(dst) != nil {
		return auth.ErrInvalidInput
	}
	return nil
}
