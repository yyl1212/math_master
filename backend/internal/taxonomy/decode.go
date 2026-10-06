package taxonomy

import (
	"bytes"
	"encoding/json"
	"github.com/santhosh-tekuri/jsonschema/v6"
	"github.com/yyl1212/math_master/schemas"
	"io"
	"strconv"
	"strings"
	"sync"
	"unicode/utf8"
)

var catalogueContract *jsonschema.Schema
var catalogueOnce sync.Once
var catalogueContractErr error

func validJSONEscapes(raw []byte) bool {
	inside := false
	for i := 0; i < len(raw); i++ {
		if raw[i] == '"' {
			inside = !inside
			continue
		}
		if !inside || raw[i] != '\\' {
			continue
		}
		i++
		if i >= len(raw) {
			return false
		}
		if raw[i] != 'u' {
			continue
		}
		if i+4 >= len(raw) {
			return false
		}
		v, e := strconv.ParseUint(string(raw[i+1:i+5]), 16, 16)
		if e != nil {
			return false
		}
		i += 4
		if v >= 0xd800 && v <= 0xdbff {
			if i+6 >= len(raw) || raw[i+1] != '\\' || raw[i+2] != 'u' {
				return false
			}
			next, e := strconv.ParseUint(string(raw[i+3:i+7]), 16, 16)
			if e != nil || next < 0xdc00 || next > 0xdfff {
				return false
			}
			i += 6
		} else if v >= 0xdc00 && v <= 0xdfff {
			return false
		}
	}
	return true
}
func strictWalk(d *json.Decoder, depth int) error {
	if depth > 64 {
		return ErrInvalid
	}
	token, e := d.Token()
	if e != nil {
		return ErrInvalid
	}
	switch v := token.(type) {
	case string:
		if !ValidText(v) {
			return ErrInvalid
		}
	case json.Delim:
		switch v {
		case '{':
			seen := map[string]bool{}
			for d.More() {
				k, e := d.Token()
				if e != nil {
					return ErrInvalid
				}
				name, ok := k.(string)
				if !ok || !ValidText(name) {
					return ErrInvalid
				}
				name = strings.ToLower(name)
				if seen[name] {
					return ErrInvalid
				}
				seen[name] = true
				if e = strictWalk(d, depth+1); e != nil {
					return e
				}
			}
		case '[':
			for d.More() {
				if e = strictWalk(d, depth+1); e != nil {
					return e
				}
			}
		default:
			return ErrInvalid
		}
		if _, e = d.Token(); e != nil {
			return ErrInvalid
		}
	}
	return nil
}
func DecodeCapturedBatch(r io.Reader) (CapturedBatch, error) {
	var out CapturedBatch
	raw, e := io.ReadAll(io.LimitReader(r, (32<<20)+1))
	if e != nil {
		return out, ErrInvalid
	}
	if len(raw) > 32<<20 {
		return out, ErrLimit
	}
	if !utf8.Valid(raw) || !json.Valid(raw) || !validJSONEscapes(raw) {
		return out, ErrInvalid
	}
	d := json.NewDecoder(bytes.NewReader(raw))
	d.UseNumber()
	if e = strictWalk(d, 0); e != nil {
		return out, e
	}
	if _, e = d.Token(); e != io.EOF {
		return out, ErrInvalid
	}
	catalogueOnce.Do(func() {
		b, e := schemas.Files.ReadFile("topic-catalogue.schema.json")
		if e != nil {
			catalogueContractErr = e
			return
		}
		doc, e := jsonschema.UnmarshalJSON(bytes.NewReader(b))
		if e != nil {
			catalogueContractErr = e
			return
		}
		c := jsonschema.NewCompiler()
		u := "https://math-master.local/schemas/topic-catalogue.schema.json"
		if e = c.AddResource(u, doc); e != nil {
			catalogueContractErr = e
			return
		}
		catalogueContract, catalogueContractErr = c.Compile(u)
	})
	if catalogueContractErr != nil {
		return out, ErrNotConfigured
	}
	generic, e := jsonschema.UnmarshalJSON(bytes.NewReader(raw))
	if e != nil || catalogueContract.Validate(generic) != nil {
		return out, ErrInvalid
	}
	d = json.NewDecoder(bytes.NewReader(raw))
	d.DisallowUnknownFields()
	if e = d.Decode(&out); e != nil {
		return out, ErrInvalid
	}
	if e = ValidateCatalogue(out.Nodes); e != nil {
		return out, e
	}
	return out, nil
}
