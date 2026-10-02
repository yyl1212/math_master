package question

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"reflect"
	"sort"
	"strings"
)

// DecodeStrictJSON preserves original integer/UTF-8 rules and requires the exact DTO shape.
func DecodeStrictJSON(r io.Reader, limit int, out any) error {
	if limit < 1 || limit > MaxEnvelopeBytes {
		return ErrInvalid
	}
	raw, err := readJSON(r, limit)
	if err != nil {
		return err
	}
	target := reflect.TypeOf(out)
	if target == nil || target.Kind() != reflect.Pointer {
		return ErrInvalid
	}
	var value any
	d := json.NewDecoder(bytes.NewReader(raw))
	d.UseNumber()
	if d.Decode(&value) != nil || !strictShape(value, target.Elem()) {
		return ErrInvalid
	}
	d = json.NewDecoder(bytes.NewReader(raw))
	d.DisallowUnknownFields()
	if d.Decode(out) != nil {
		return ErrInvalid
	}
	return nil
}
func strictFields(t reflect.Type) map[string]reflect.Type {
	fields := map[string]reflect.Type{}
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		tag := strings.Split(f.Tag.Get("json"), ",")[0]
		if tag == "-" {
			continue
		}
		if f.Anonymous && tag == "" {
			for k, v := range strictFields(f.Type) {
				fields[k] = v
			}
			continue
		}
		if tag == "" {
			tag = f.Name
		}
		fields[tag] = f.Type
	}
	return fields
}
func strictShape(v any, t reflect.Type) bool {
	if t.Kind() == reflect.Pointer {
		return v == nil || strictShape(v, t.Elem())
	}
	if v == nil {
		return false
	}
	switch t.Kind() {
	case reflect.Struct:
		obj, ok := v.(map[string]any)
		if !ok {
			return false
		}
		fields := strictFields(t)
		if len(obj) != len(fields) {
			return false
		}
		for key, field := range fields {
			value, exists := obj[key]
			if !exists || !strictShape(value, field) {
				return false
			}
		}
		return true
	case reflect.Slice:
		arr, ok := v.([]any)
		if !ok {
			return false
		}
		for _, value := range arr {
			if !strictShape(value, t.Elem()) {
				return false
			}
		}
		return true
	case reflect.String:
		_, ok := v.(string)
		return ok
	case reflect.Bool:
		_, ok := v.(bool)
		return ok
	case reflect.Int, reflect.Int64, reflect.Int32:
		_, ok := v.(json.Number)
		return ok
	default:
		return false
	}
}
func DecodeArchive(r io.Reader) (Archive, error) {
	var a Archive
	if err := DecodeStrictJSON(r, MaxEnvelopeBytes, &a); err != nil {
		return a, err
	}
	raw, _ := json.Marshal(a.Envelope)
	if _, err := DecodeDraft(bytes.NewReader(raw)); err != nil {
		return a, err
	}
	if !ValidSHA(a.PackageSHA) {
		return a, ErrInvalid
	}
	for _, id := range a.SourceResponsibility.AuthorIDs {
		if !ValidID(id) {
			return a, ErrInvalid
		}
	}
	return a, nil
}
func UsedEngineVersions(p QuestionPackage, instances []Instance) ([]int, []int) {
	generators, verifiers := map[int]bool{}, map[int]bool{}
	for _, t := range p.Templates {
		generators[t.Engine.GeneratorVersion] = true
		verifiers[t.Engine.VerifierVersion] = true
	}
	for _, i := range instances {
		if i.Body.Witness != nil {
			verifiers[i.Body.Witness.Engine.VerifierVersion] = true
		}
	}
	keys := func(values map[int]bool) []int {
		out := []int{}
		for v := range values {
			out = append(out, v)
		}
		sort.Ints(out)
		return out
	}
	return keys(generators), keys(verifiers)
}
func ValidateArchive(ctx context.Context, a Archive, refs ReferenceSnapshot) (SealedPackage, ValidationReport, error) {
	var empty SealedPackage
	var report ValidationReport
	raw, err := json.Marshal(a)
	if err != nil {
		return empty, report, ErrInvalid
	}
	if _, err = DecodeArchive(bytes.NewReader(raw)); err != nil {
		return empty, report, err
	}
	sealed, report, err := ValidateAndSeal(ctx, a.Envelope, refs)
	if err != nil {
		return empty, report, err
	}
	if sealed.PackageSHA != a.PackageSHA || len(a.Instances) != len(sealed.Instances) {
		return empty, report, ErrImmutableConflict
	}
	expected := map[string]Instance{}
	for _, i := range sealed.Instances {
		expected[i.Identity.ID] = i
	}
	seen := map[string]bool{}
	for _, i := range a.Instances {
		if err := ctx.Err(); err != nil {
			return empty, report, err
		}
		correct, ok := expected[i.Identity.ID]
		if !ok || seen[i.Identity.ID] || i.Identity != correct.Identity {
			return empty, report, ErrImmutableConflict
		}
		seen[i.Identity.ID] = true
		left, _, err := CanonicalInstance(i)
		if err != nil {
			return empty, report, err
		}
		right, _, err := CanonicalInstance(correct)
		if err != nil || !bytes.Equal(left, right) {
			return empty, report, ErrImmutableConflict
		}
		if err = VerifyInstance(i); err != nil {
			return empty, report, err
		}
	}
	g, v := UsedEngineVersions(sealed.Package, sealed.Instances)
	if !reflect.DeepEqual(g, a.GeneratorVersions) || !reflect.DeepEqual(v, a.VerifierVersions) {
		return empty, report, ErrInvalid
	}
	return sealed, report, nil
}
