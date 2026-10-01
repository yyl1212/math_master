package httpapi

import (
	"bytes"
	"encoding/json"
	"github.com/yyl1212/math_master/backend/internal/auth"
	"io"
	"reflect"
	"strconv"
	"strings"
	"unicode/utf8"
)

const privateBodyLimit = 8 * 1024

// encoding/json replaces unpaired surrogate escapes; reject them before decoding.
func validPrivateEscapes(raw []byte) bool {
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
func walkPrivateJSON(dec *json.Decoder, first json.Token, depth int, allowed map[string]bool) error {
	if first == nil || depth > 8 {
		return auth.ErrInvalidInput
	}
	delim, ok := first.(json.Delim)
	if !ok {
		return nil
	}
	switch delim {
	case '{':
		seen := make(map[string]bool)
		for dec.More() {
			token, err := dec.Token()
			if err != nil {
				return auth.ErrInvalidInput
			}
			key, ok := token.(string)
			if !ok {
				return auth.ErrInvalidInput
			}
			fold := strings.ToLower(key)
			if seen[fold] || (allowed != nil && !allowed[key]) {
				return auth.ErrInvalidInput
			}
			seen[fold] = true
			value, err := dec.Token()
			if err != nil {
				return auth.ErrInvalidInput
			}
			if err = walkPrivateJSON(dec, value, depth+1, nil); err != nil {
				return err
			}
		}
		end, err := dec.Token()
		if err != nil || end != json.Delim('}') {
			return auth.ErrInvalidInput
		}
	case '[':
		for dec.More() {
			value, err := dec.Token()
			if err != nil {
				return auth.ErrInvalidInput
			}
			if err = walkPrivateJSON(dec, value, depth+1, nil); err != nil {
				return err
			}
		}
		end, err := dec.Token()
		if err != nil || end != json.Delim(']') {
			return auth.ErrInvalidInput
		}
	default:
		return auth.ErrInvalidInput
	}
	return nil
}
func decodePrivateJSON(reader io.Reader, dst any) error {
	raw, err := io.ReadAll(io.LimitReader(reader, privateBodyLimit+1))
	if err != nil || len(raw) > privateBodyLimit || !utf8.Valid(raw) || !json.Valid(raw) {
		return auth.ErrInvalidInput
	}
	if !validPrivateEscapes(raw) {
		return auth.ErrInvalidInput
	}
	typ := reflect.TypeOf(dst)
	if typ == nil || typ.Kind() != reflect.Pointer || typ.Elem().Kind() != reflect.Struct {
		return auth.ErrInvalidInput
	}
	allowed := make(map[string]bool)
	typ = typ.Elem()
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		name := strings.Split(field.Tag.Get("json"), ",")[0]
		if name != "" && name != "-" {
			allowed[name] = true
		}
	}
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	first, err := dec.Token()
	if err != nil || first != json.Delim('{') {
		return auth.ErrInvalidInput
	}
	if err = walkPrivateJSON(dec, first, 1, allowed); err != nil {
		return err
	}
	if _, err = dec.Token(); err != io.EOF {
		return auth.ErrInvalidInput
	}
	dec = json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	if dec.Decode(dst) != nil {
		return auth.ErrInvalidInput
	}
	return nil
}
