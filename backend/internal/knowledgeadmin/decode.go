package knowledgeadmin

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/dlclark/regexp2"
	"github.com/santhosh-tekuri/jsonschema/v6"
	"github.com/yyl1212/math_master/schemas"
	"io"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf8"
)

type DecodeError struct {
	Code string `json:"code"`
	Path string `json:"path"`
}

func (e *DecodeError) Error() string { return e.Code + " at " + e.Path }
func invalid(path string) error      { return &DecodeError{"INVALID_FIELD", path} }
func bytesReader(b []byte) io.Reader { return bytes.NewReader(b) }

var schemaOnce sync.Once
var sourceSchema, pointSchema *jsonschema.Schema
var schemaError error
var schemaLock sync.Mutex
var regexpFailed bool

type boundedRegexp struct{ re *regexp2.Regexp }

func (re boundedRegexp) String() string { return re.re.String() }
func (re boundedRegexp) MatchString(s string) bool {
	ok, e := re.re.MatchString(s)
	if e != nil {
		regexpFailed = true
	}
	return ok
}
func compileSchemas() {
	b, e := schemas.Files.ReadFile("knowledge-source.schema.json")
	if e != nil {
		schemaError = e
		return
	}
	d, e := jsonschema.UnmarshalJSON(bytes.NewReader(b))
	if e != nil {
		schemaError = e
		return
	}
	c := jsonschema.NewCompiler()
	c.UseRegexpEngine(func(s string) (jsonschema.Regexp, error) {
		r, e := regexp2.Compile(s, regexp2.ECMAScript)
		if e != nil {
			return nil, e
		}
		r.MatchTimeout = 50 * time.Millisecond
		return boundedRegexp{r}, nil
	})
	c.AssertFormat()
	url := "https://math-master.local/schemas/knowledge-source.schema.json"
	if e = c.AddResource(url, d); e != nil {
		schemaError = e
		return
	}
	sourceSchema, e = c.Compile(url)
	if e != nil {
		schemaError = e
		return
	}
	pointSchema, schemaError = c.Compile(url + "#/$defs/record")
}
func validateSchema(d any, point bool) error {
	schemaOnce.Do(compileSchemas)
	if schemaError != nil {
		return &DecodeError{"SCHEMA_UNAVAILABLE", "/"}
	}
	schemaLock.Lock()
	defer schemaLock.Unlock()
	regexpFailed = false
	s := sourceSchema
	if point {
		s = pointSchema
	}
	e := s.Validate(d)
	if regexpFailed {
		return &DecodeError{"REGEX_TIMEOUT", "/"}
	}
	if e != nil {
		var v *jsonschema.ValidationError
		if errors.As(e, &v) {
			for len(v.Causes) > 0 {
				v = v.Causes[0]
			}
			return invalid("/" + strings.Join(v.InstanceLocation, "/"))
		}
		return invalid("/")
	}
	return nil
}
func DecodeSource(r io.Reader) (SourceDocument, error) {
	var d SourceDocument
	b, e := io.ReadAll(io.LimitReader(r, MaxSourceBytes+1))
	if e != nil {
		return d, &DecodeError{"READ_FAILED", "/"}
	}
	if len(b) > MaxSourceBytes {
		return d, &DecodeError{"INPUT_TOO_LARGE", "/"}
	}
	v, e := StrictJSON(b)
	if e != nil {
		return d, e
	}
	if e = validateSchema(v, false); e != nil {
		return d, e
	}
	if e = knownValues(v, "", "/"); e != nil {
		return d, e
	}
	if e = json.Unmarshal(b, &d); e != nil {
		return d, invalid("/")
	}
	if len(d.KnowledgePoints) > MaxSourcePoints {
		return d, invalid("/knowledge_points")
	}
	if len(d.Source.SourceID) > 128 || len(d.Source.WorkFamilyID) > 128 || len(d.DatasetID) > 128 {
		return d, invalid("/source")
	}
	if !safeSourceURL(d.Source.URL) {
		return d, invalid("/source/url")
	}
	index, e := LoadClassificationIndex()
	if e != nil {
		return d, &DecodeError{"SCHEMA_UNAVAILABLE", "/"}
	}
	for i, p := range d.KnowledgePoints {
		if e = ValidatePoint(p, index); e != nil {
			if de, ok := e.(*DecodeError); ok {
				return d, &DecodeError{de.Code, fmt.Sprintf("/knowledge_points/%d%s", i, de.Path)}
			}
			return d, e
		}
	}
	// Repeated IDs are handled by the import preview; they never become multiple entities.
	if e = validateRelations(d); e != nil {
		return d, e
	}
	return d, nil
}
func StrictJSON(b []byte) (any, error) {
	if !utf8.Valid(b) {
		return nil, &DecodeError{"INVALID_UTF8", "/"}
	}
	if e := validEscapes(b); e != nil {
		return nil, e
	}
	d := json.NewDecoder(bytes.NewReader(b))
	d.UseNumber()
	if e := uniqueValue(d, "/", 0); e != nil {
		return nil, e
	}
	if _, e := d.Token(); !errors.Is(e, io.EOF) {
		return nil, &DecodeError{"TRAILING_JSON", "/"}
	}
	d = json.NewDecoder(bytes.NewReader(b))
	d.UseNumber()
	var v any
	if e := d.Decode(&v); e != nil {
		return nil, invalid("/")
	}
	return v, nil
}
func uniqueValue(d *json.Decoder, path string, depth int) error {
	if depth > 32 {
		return &DecodeError{"JSON_TOO_DEEP", path}
	}
	tok, e := d.Token()
	if e != nil {
		return invalid(path)
	}
	delim, ok := tok.(json.Delim)
	if !ok {
		return nil
	}
	switch delim {
	case '{':
		seen := map[string]bool{}
		for d.More() {
			k, e := d.Token()
			if e != nil {
				return invalid(path)
			}
			key, ok := k.(string)
			if !ok {
				return invalid(path)
			}
			p := strings.TrimSuffix(path, "/") + "/" + strings.ReplaceAll(strings.ReplaceAll(key, "~", "~0"), "/", "~1")
			if seen[key] {
				return &DecodeError{"DUPLICATE_FIELD", p}
			}
			seen[key] = true
			if e = uniqueValue(d, p, depth+1); e != nil {
				return e
			}
		}
	case '[':
		for i := 0; d.More(); i++ {
			if e = uniqueValue(d, fmt.Sprintf("%s/%d", strings.TrimSuffix(path, "/"), i), depth+1); e != nil {
				return e
			}
		}
	default:
		return invalid(path)
	}
	_, e = d.Token()
	if e != nil {
		return invalid(path)
	}
	return nil
}
func validEscapes(b []byte) error {
	for i := 0; i < len(b); i++ {
		if b[i] != '"' {
			continue
		}
		i++
		for i < len(b) && b[i] != '"' {
			if b[i] != '\\' {
				i++
				continue
			}
			i++
			if i >= len(b) {
				return invalid("/")
			}
			if b[i] != 'u' {
				i++
				continue
			}
			if i+4 >= len(b) {
				return invalid("/")
			}
			u, e := strconv.ParseUint(string(b[i+1:i+5]), 16, 16)
			if e != nil {
				return invalid("/")
			}
			i += 5
			if u >= 0xd800 && u <= 0xdbff {
				if i+5 >= len(b) || b[i] != '\\' || b[i+1] != 'u' {
					return &DecodeError{"INVALID_UNICODE", "/"}
				}
				lo, e := strconv.ParseUint(string(b[i+2:i+6]), 16, 16)
				if e != nil || lo < 0xdc00 || lo > 0xdfff {
					return &DecodeError{"INVALID_UNICODE", "/"}
				}
				i += 6
			} else if u >= 0xdc00 && u <= 0xdfff {
				return &DecodeError{"INVALID_UNICODE", "/"}
			}
		}
	}
	return nil
}
