package content

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"
	"unicode/utf8"

	"github.com/santhosh-tekuri/jsonschema/v6"
	"github.com/yyl1212/math_master/backend/internal/catalogue"
	"github.com/yyl1212/math_master/schemas"
)

const MaxPackageBytes = 10 * 1024 * 1024

type DecodeError struct {
	Code string `json:"code"`
	Path string `json:"path"`
}

func (e *DecodeError) Error() string { return e.Code + " at " + e.Path }

var schemaOnce sync.Once
var compiled map[string]*jsonschema.Schema
var schemaErr error

func contracts() {
	compiled = map[string]*jsonschema.Schema{}
	for _, name := range []string{"catalogue.schema.json", "content-package.schema.json"} {
		b, e := schemas.Files.ReadFile(name)
		if e != nil {
			schemaErr = e
			return
		}
		doc, e := jsonschema.UnmarshalJSON(bytes.NewReader(b))
		if e != nil {
			schemaErr = e
			return
		}
		c := jsonschema.NewCompiler()
		url := "https://math-master.local/schemas/" + name
		if e = c.AddResource(url, doc); e != nil {
			schemaErr = e
			return
		}
		compiled[name], e = c.Compile(url)
		if e != nil {
			schemaErr = e
			return
		}
	}
}
func DecodePackage(r io.Reader) (Package, error) {
	var p Package
	err := decode(r, "content-package.schema.json", &p)
	return p, err
}
func DecodeCatalogue(r io.Reader) (catalogue.Catalogue, error) {
	var c catalogue.Catalogue
	err := decode(r, "catalogue.schema.json", &c)
	return c, err
}

func decode(r io.Reader, name string, out any) error {
	return decodeLimit(r, name, out, MaxPackageBytes)
}
func decodeLimit(r io.Reader, name string, out any, limit int) error {
	data, err := io.ReadAll(io.LimitReader(r, int64(limit)+1))
	if err != nil {
		return &DecodeError{"READ_FAILED", "/"}
	}
	if len(data) > limit {
		return &DecodeError{"INPUT_TOO_LARGE", "/"}
	}
	if !utf8.Valid(data) {
		return &DecodeError{"INVALID_UTF8", "/"}
	}
	d := json.NewDecoder(bytes.NewReader(data))
	d.UseNumber()
	if err = uniqueValue(d, "", 0); err != nil {
		return err
	}
	if _, err = d.Token(); !errors.Is(err, io.EOF) {
		return &DecodeError{"TRAILING_JSON", "/"}
	}
	doc, err := jsonschema.UnmarshalJSON(bytes.NewReader(data))
	if err != nil {
		return &DecodeError{"INVALID_JSON", "/"}
	}
	schemaOnce.Do(contracts)
	if schemaErr != nil {
		return &DecodeError{"SCHEMA_UNAVAILABLE", "/"}
	}
	if err = compiled[name].Validate(doc); err != nil {
		var ve *jsonschema.ValidationError
		if errors.As(err, &ve) {
			for len(ve.Causes) > 0 {
				ve = ve.Causes[0]
			}
			return &DecodeError{"INVALID_FIELD", "/" + strings.Join(ve.InstanceLocation, "/")}
		}
		return &DecodeError{"INVALID_FIELD", "/"}
	}
	if err = json.Unmarshal(data, out); err != nil {
		return &DecodeError{"INVALID_FIELD", "/"}
	}
	return nil
}
func uniqueValue(d *json.Decoder, path string, depth int) error {
	if depth > 32 {
		return &DecodeError{"JSON_TOO_DEEP", path}
	}
	token, err := d.Token()
	if err != nil {
		return &DecodeError{"INVALID_JSON", path}
	}
	delim, ok := token.(json.Delim)
	if !ok {
		return nil
	}
	switch delim {
	case '{':
		seen := map[string]bool{}
		for d.More() {
			key, e := d.Token()
			if e != nil {
				return &DecodeError{"INVALID_JSON", path}
			}
			name, ok := key.(string)
			if !ok {
				return &DecodeError{"INVALID_JSON", path}
			}
			child := path + "/" + strings.ReplaceAll(strings.ReplaceAll(name, "~", "~0"), "/", "~1")
			if seen[name] {
				return &DecodeError{"DUPLICATE_FIELD", child}
			}
			seen[name] = true
			if e = uniqueValue(d, child, depth+1); e != nil {
				return e
			}
		}
	case '[':
		for i := 0; d.More(); i++ {
			if e := uniqueValue(d, fmt.Sprintf("%s/%d", path, i), depth+1); e != nil {
				return e
			}
		}
	default:
		return &DecodeError{"INVALID_JSON", path}
	}
	if _, err = d.Token(); err != nil {
		return &DecodeError{"INVALID_JSON", path}
	}
	return nil
}
