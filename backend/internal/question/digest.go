package question

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"reflect"
	"sort"
)

// canonicalValue owns its copy: nil arrays become explicit [] without mutating a caller.
func canonicalValue(v reflect.Value) reflect.Value {
	if !v.IsValid() {
		return v
	}
	switch v.Kind() {
	case reflect.Pointer:
		if v.IsNil() {
			return reflect.Zero(v.Type())
		}
		out := reflect.New(v.Type().Elem())
		out.Elem().Set(canonicalValue(v.Elem()))
		return out
	case reflect.Slice:
		out := reflect.MakeSlice(v.Type(), v.Len(), v.Len())
		for i := 0; i < v.Len(); i++ {
			out.Index(i).Set(canonicalValue(v.Index(i)))
		}
		return out
	case reflect.Struct:
		out := reflect.New(v.Type()).Elem()
		for i := 0; i < v.NumField(); i++ {
			out.Field(i).Set(canonicalValue(v.Field(i)))
		}
		return out
	default:
		return v
	}
}
func canonical(purpose string, body any) ([]byte, string, error) {
	v := canonicalValue(reflect.ValueOf(body)).Interface()
	raw, e := json.Marshal(struct {
		Purpose string `json:"purpose"`
		Body    any    `json:"body"`
	}{purpose, v})
	if e != nil {
		return nil, "", e
	}
	h := sha256.Sum256(raw)
	return raw, hex.EncodeToString(h[:]), nil
}
func sortBody(body QuestionBody) QuestionBody {
	body = canonicalValue(reflect.ValueOf(body)).Interface().(QuestionBody)
	sort.Slice(body.Units, func(i, j int) bool { return refLess(body.Units[i], body.Units[j]) })
	sort.Slice(body.Coverage, func(i, j int) bool { return refLess(body.Coverage[i].Knowledge, body.Coverage[j].Knowledge) })
	sort.Slice(body.Assets, func(i, j int) bool { return body.Assets[i].ID < body.Assets[j].ID })
	return body
}
func refLess(a, b Ref) bool {
	if a.ID != b.ID {
		return a.ID < b.ID
	}
	return a.Version < b.Version
}
func CanonicalPackage(p QuestionPackage) ([]byte, string, error) {
	p = canonicalValue(reflect.ValueOf(p)).Interface().(QuestionPackage)
	sort.Slice(p.Templates, func(i, j int) bool {
		return refLess(Ref{ID: p.Templates[i].ID, Version: p.Templates[i].Version}, Ref{ID: p.Templates[j].ID, Version: p.Templates[j].Version})
	})
	for j := range p.Templates {
		t := &p.Templates[j]
		sort.Slice(t.Units, func(i, k int) bool { return refLess(t.Units[i], t.Units[k]) })
		sort.Slice(t.Coverage, func(i, k int) bool { return refLess(t.Coverage[i].Knowledge, t.Coverage[k].Knowledge) })
		sort.Slice(t.Assets, func(i, k int) bool { return t.Assets[i].ID < t.Assets[k].ID })
	}
	sort.Slice(p.FixedQuestions, func(i, j int) bool { return p.FixedQuestions[i].ID < p.FixedQuestions[j].ID })
	for i := range p.FixedQuestions {
		p.FixedQuestions[i].Body = sortBody(p.FixedQuestions[i].Body)
	}
	sort.Slice(p.Blueprints, func(i, j int) bool { return p.Blueprints[i].ID < p.Blueprints[j].ID })
	for j := range p.Blueprints {
		sources := p.Blueprints[j].Sources
		sort.Slice(sources, func(i, k int) bool {
			if sources[i].Kind != sources[k].Kind {
				return sources[i].Kind < sources[k].Kind
			}
			return refLess(sources[i].Ref, sources[k].Ref)
		})
	}
	return canonical("question-package-v1", p)
}
func CanonicalInstance(i Instance) ([]byte, string, error) {
	return canonical("question-instance-body-v1", struct {
		Identity         Ref              `json:"identity"`
		Origin           string           `json:"origin"`
		Template         *Identity        `json:"template"`
		Parameters       []ParameterValue `json:"parameters"`
		GeneratorVersion *int             `json:"generatorVersion"`
		VerifierVersion  *int             `json:"verifierVersion"`
		Body             QuestionBody     `json:"body"`
	}{Ref{ID: i.Identity.ID, Version: i.Identity.Version}, i.Origin, i.Template, i.Parameters, i.GeneratorVersion, i.VerifierVersion, sortBody(i.Body)})
}
func CanonicalValidation(input DraftInput, sealed SealedPackage) ([]byte, string, error) {
	return canonical("question-validation-v1", struct {
		Input  DraftInput    `json:"input"`
		Sealed SealedPackage `json:"sealed"`
	}{input, sealed})
}
func CanonicalFrozen(body FrozenBody, instances []Instance) ([]byte, string, error) {
	body.FrozenDigest = ""
	return canonical("question-submission-v1", FrozenPayload{Body: body, Instances: instances})
}
