package taxonomy

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sort"
	"strings"
)

func Digest(v any) (string, error) {
	b, e := json.Marshal(v)
	if e != nil {
		return "", e
	}
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:]), nil
}
func CanonicalAssignment(in AssignmentInput) ([]byte, string, error) {
	if e := ValidateAssignment(in); e != nil {
		return nil, "", e
	}
	in.TopicIDs = append([]string(nil), in.TopicIDs...)
	sort.Strings(in.TopicIDs)
	in.SourceRefs = append([]SourceRecordRef(nil), in.SourceRefs...)
	sort.Slice(in.SourceRefs, func(i, j int) bool {
		a, b := in.SourceRefs[i], in.SourceRefs[j]
		return strings.Join([]string{a.SourceID, a.WorkFamilyID, a.RecordID, a.Path, a.SHA256}, "\x00") < strings.Join([]string{b.SourceID, b.WorkFamilyID, b.RecordID, b.Path, b.SHA256}, "\x00")
	})
	raw, e := json.Marshal(in)
	if e != nil {
		return nil, "", e
	}
	h := sha256.Sum256(raw)
	return raw, hex.EncodeToString(h[:]), nil
}
