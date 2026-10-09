package knowledgeadmin

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"github.com/cyberphone/json-canonicalization/go/src/webpki.org/jsoncanonicalizer"
	"sort"
)

func CanonicalJSON(b []byte) ([]byte, error) { return jsoncanonicalizer.Transform(b) }
func DigestBytes(b []byte) string            { return fmt.Sprintf("%x", sha256.Sum256(b)) }
func KnowledgeID(externalID string) string   { return "k-" + DigestBytes([]byte(externalID))[:56] }
func jsonDigest(v any) (string, error) {
	b, e := json.Marshal(v)
	if e != nil {
		return "", e
	}
	b, e = CanonicalJSON(b)
	if e != nil {
		return "", e
	}
	return DigestBytes(b), nil
}
func SourceCoreSHA(p SourcePoint) (string, error) {
	b, e := json.Marshal(p)
	if e != nil {
		return "", e
	}
	var v map[string]any
	d := json.NewDecoder(bytesReader(b))
	d.UseNumber()
	if e = d.Decode(&v); e != nil {
		return "", e
	}
	for _, k := range []string{"id", "version", "original_binding", "content_origin", "original_type", "provenance", "relations", "msc_codes", "classification_status", "classification_evidence", "classification_mode", "project_other", "extensions"} {
		delete(v, k)
	}
	return jsonDigest(v)
}
func PublicProjection(p SourcePoint) PublicPoint {
	b, _ := json.Marshal(p)
	var v PublicPoint
	_ = json.Unmarshal(b, &v)
	for _, k := range []string{"id", "version", "original_binding", "original_type", "content_origin", "provenance", "extensions"} {
		delete(v, k)
	}
	return v
}
func CurrentSHA(i CurrentInput, topics []string) (string, error) {
	t := append([]string{}, topics...)
	sort.Strings(t)
	return jsonDigest(struct {
		Point   PublicPoint    `json:"point"`
		Sources []PublicSource `json:"sources"`
		Topics  []string       `json:"topics"`
	}{PublicProjection(i.Point), i.Sources, t})
}
