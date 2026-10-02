package question

import (
	"bytes"
	"encoding/json"
	"github.com/santhosh-tekuri/jsonschema/v6"
	"github.com/yyl1212/math_master/schemas"
	"io"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"unicode/utf8"
)

const MaxEnvelopeBytes = 4 * 1024 * 1024
const MaxPackageBytes = 2 * 1024 * 1024

// SHA purpose framing is metadata, separate from the 2 MiB logical package payload.
const PackageCanonicalOverhead = len(`{"purpose":"question-package-v1","body":}`)
const MaxCanonicalPackageBytes = MaxPackageBytes + PackageCanonicalOverhead
const MaxResponseBytes = 4 * 1024 * 1024

var integerLexeme = regexp.MustCompile(`^-?(0|[1-9][0-9]*)$`)
var contractOnce sync.Once
var packageContract, draftContract *jsonschema.Schema
var contractError error

func loadContracts() {
	raw, e := schemas.Files.ReadFile("question-package.schema.json")
	if e != nil {
		contractError = e
		return
	}
	doc, e := jsonschema.UnmarshalJSON(bytes.NewReader(raw))
	if e != nil {
		contractError = e
		return
	}
	compiler := jsonschema.NewCompiler()
	url := "https://math-master.local/schemas/question-package.schema.json"
	if e = compiler.AddResource(url, doc); e != nil {
		contractError = e
		return
	}
	packageContract, e = compiler.Compile(url)
	if e != nil {
		contractError = e
		return
	}
	draftContract, contractError = compiler.Compile(url + "#/$defs/DraftInput")
}
func DecodePackage(r io.Reader) (QuestionPackage, error) {
	var p QuestionPackage
	raw, e := readJSON(r, MaxPackageBytes)
	if e != nil {
		return p, e
	}
	e = decodeContract(raw, false, &p)
	return p, e
}
func DecodeDraft(r io.Reader) (DraftInput, error) {
	var input DraftInput
	raw, e := readJSON(r, MaxEnvelopeBytes)
	if e != nil {
		return input, e
	}
	if e = decodeContract(raw, true, &input); e != nil {
		return input, e
	}
	var fields map[string]json.RawMessage
	if json.Unmarshal(raw, &fields) != nil {
		return input, ErrInvalid
	}
	if len(fields["questionPackage"]) > MaxPackageBytes {
		return input, ErrLimitExceeded
	}
	return input, nil
}
func readJSON(r io.Reader, limit int) ([]byte, error) {
	raw, e := io.ReadAll(io.LimitReader(r, int64(limit)+1))
	if e != nil {
		return nil, ErrInvalid
	}
	if len(raw) > limit {
		return nil, ErrLimitExceeded
	}
	if !utf8.Valid(raw) || !json.Valid(raw) || !validEscapes(raw) {
		return nil, ErrInvalid
	}
	d := json.NewDecoder(bytes.NewReader(raw))
	d.UseNumber()
	if e = walkJSON(d, 0); e != nil {
		return nil, e
	}
	if _, e = d.Token(); e != io.EOF {
		return nil, ErrInvalid
	}
	return raw, nil
}
func decodeContract(raw []byte, draft bool, out any) error {
	contractOnce.Do(loadContracts)
	if contractError != nil {
		return ErrNotConfigured
	}
	doc, e := jsonschema.UnmarshalJSON(bytes.NewReader(raw))
	if e != nil {
		return ErrInvalid
	}
	contract := packageContract
	if draft {
		contract = draftContract
	}
	if contract.Validate(doc) != nil {
		return ErrInvalid
	}
	d := json.NewDecoder(bytes.NewReader(raw))
	d.DisallowUnknownFields()
	if d.Decode(out) != nil {
		return ErrInvalid
	}
	return nil
}

// walkJSON checks the original token stream before any lossy numeric conversion.
func walkJSON(d *json.Decoder, depth int) error {
	if depth > 32 {
		return ErrInvalid
	}
	token, e := d.Token()
	if e != nil {
		return ErrInvalid
	}
	switch v := token.(type) {
	case string:
		if strings.ContainsRune(v, 0) {
			return ErrInvalid
		}
	case json.Number:
		if !integerLexeme.MatchString(string(v)) {
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
				key, ok := k.(string)
				if !ok || strings.ContainsRune(key, 0) {
					return ErrInvalid
				}
				fold := strings.ToLower(key)
				if seen[fold] {
					return ErrInvalid
				}
				seen[fold] = true
				if e = walkJSON(d, depth+1); e != nil {
					return e
				}
			}
			end, e := d.Token()
			if e != nil || end != json.Delim('}') {
				return ErrInvalid
			}
		case '[':
			for d.More() {
				if e = walkJSON(d, depth+1); e != nil {
					return e
				}
			}
			end, e := d.Token()
			if e != nil || end != json.Delim(']') {
				return ErrInvalid
			}
		default:
			return ErrInvalid
		}
	}
	return nil
}

// JSON validity is established before examining escape lengths.
func validEscapes(raw []byte) bool {
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
			n, e := strconv.ParseUint(string(raw[i+1:i+5]), 16, 16)
			if e != nil {
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
				low, e := strconv.ParseUint(string(raw[i+2:i+6]), 16, 16)
				if e != nil || low < 0xdc00 || low > 0xdfff {
					return false
				}
				i += 6
			}
		}
	}
	return true
}
