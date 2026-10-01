package publication

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/yyl1212/math_master/backend/internal/auth"
	"github.com/yyl1212/math_master/backend/internal/content"
	"reflect"
	"sort"
	"strings"
	"unicode/utf8"
)

func ValidNote(v string) bool {
	return utf8.ValidString(v) && !strings.ContainsRune(v, 0) && len(v) <= 3000 && utf8.RuneCountInString(v) >= 10 && utf8.RuneCountInString(v) <= 1000 && strings.TrimSpace(v) != ""
}
func validStrings(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.String:
		return utf8.ValidString(v.String()) && !strings.ContainsRune(v.String(), 0)
	case reflect.Struct:
		for i := 0; i < v.NumField(); i++ {
			if !validStrings(v.Field(i)) {
				return false
			}
		}
	case reflect.Slice, reflect.Array:
		for i := 0; i < v.Len(); i++ {
			if !validStrings(v.Index(i)) {
				return false
			}
		}
	}
	return true
}
func RelativeSourcePath(v string) bool {
	if v == "" || strings.HasPrefix(v, "/") || strings.ContainsAny(v, "\\\x00") || len(v) > 1 && v[1] == ':' && (v[0] >= 'a' && v[0] <= 'z' || v[0] >= 'A' && v[0] <= 'Z') {
		return false
	}
	for _, part := range strings.Split(v, "/") {
		if strings.TrimSpace(part) == "" || part == "." || part == ".." {
			return false
		}
	}
	return true
}
func DraftAssets(input DraftInput) (map[string][]byte, error) {
	if input.CatalogueVersion < 1 || input.CatalogueVersion > 2147483647 || !validStrings(reflect.ValueOf(input)) {
		return nil, auth.ErrInvalidInput
	}
	p := input.Package
	if len(p.Knowledge) > 100 || len(p.Units) > 200 || len(p.Paths) > 20 || len(p.Assets) > 16 || len(input.AssetBytes) > 16 {
		return nil, ErrContentLimitExceeded
	}
	raw, err := json.Marshal(p)
	if err != nil {
		return nil, ErrContentInvalid
	}
	if len(raw) > 2<<20 {
		return nil, ErrContentLimitExceeded
	}
	raw, err = json.Marshal(input.SourceMap)
	if err != nil {
		return nil, auth.ErrInvalidInput
	}
	if len(raw) > 256<<10 {
		return nil, ErrContentLimitExceeded
	}
	known := map[content.VersionRef]bool{}
	for _, k := range p.Knowledge {
		known[content.VersionRef{ID: k.ID, Version: k.Version}] = true
	}
	for _, link := range input.SourceMap {
		if !known[link.Knowledge] || !ValidSHA(link.BatchSHA256) || !ValidSHA(link.SHA256) || !RelativeSourcePath(link.RelativePath) {
			return nil, auth.ErrInvalidInput
		}
	}
	assets := map[string][]byte{}
	total := 0
	for _, asset := range input.AssetBytes {
		if _, ok := assets[asset.ID]; ok {
			return nil, auth.ErrInvalidInput
		}
		if len(asset.Base64) > base64.StdEncoding.EncodedLen(1<<20) {
			return nil, ErrContentLimitExceeded
		}
		b, err := base64.StdEncoding.Strict().DecodeString(asset.Base64)
		if err != nil || base64.StdEncoding.EncodeToString(b) != asset.Base64 {
			return nil, auth.ErrInvalidInput
		}
		total += len(b)
		if total > 4<<20 {
			return nil, ErrContentLimitExceeded
		}
		assets[asset.ID] = b
	}
	if len(assets) != len(p.Assets) {
		return nil, ErrContentInvalid
	}
	for _, a := range p.Assets {
		if _, ok := assets[a.ID]; !ok {
			return nil, ErrContentInvalid
		}
	}
	raw, err = json.Marshal(input)
	if err != nil {
		return nil, auth.ErrInvalidInput
	}
	if len(raw) > 8<<20 {
		return nil, ErrContentLimitExceeded
	}
	return assets, nil
}

type contentDigestBody struct {
	CatalogueVersion   int                 `json:"catalogueVersion"`
	CatalogueSHA256    string              `json:"catalogueSha256"`
	Package            content.Package     `json:"package"`
	SourceMap          []SourceLink        `json:"sourceMap"`
	AuthorIDs          []string            `json:"authorIds"`
	LegacyUnattributed bool                `json:"legacyUnattributed"`
	Assets             []content.AssetView `json:"assets"`
}

func digestBody(f FrozenBody) (contentDigestBody, error) {
	authors := append([]string{}, f.AuthorIDs...)
	sort.Strings(authors)
	for i, id := range authors {
		if !ValidID(id) || (i > 0 && id == authors[i-1]) {
			return contentDigestBody{}, auth.ErrInvalidInput
		}
	}
	return contentDigestBody{f.CatalogueVersion, f.CatalogueSHA256, f.Package, f.SourceMap, authors, f.LegacyUnattributed, f.Assets}, nil
}

// FrozenBytes is the exact purpose-tagged content payload stored and checked by SQL.
func FrozenBytes(f FrozenBody) ([]byte, error) {
	b, err := digestBody(f)
	if err != nil {
		return nil, err
	}
	return json.Marshal(struct {
		Purpose string `json:"purpose"`
		contentDigestBody
	}{"math-master/frozen-submission/v1", b})
}
func FrozenDigest(f FrozenBody) (string, error) {
	b, err := FrozenBytes(f)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", sha256.Sum256(b)), nil
}
func DraftDigest(d DraftView) (string, error) {
	if !ValidID(d.ID) || d.Revision < 1 {
		return "", auth.ErrInvalidInput
	}
	b, err := digestBody(FrozenBody{CatalogueVersion: d.CatalogueVersion, CatalogueSHA256: d.CatalogueSHA256, Package: d.Package, SourceMap: d.SourceMap, AuthorIDs: d.AuthorIDs, LegacyUnattributed: d.LegacyUnattributed, Assets: d.Assets})
	if err != nil {
		return "", err
	}
	raw, err := json.Marshal(struct {
		Purpose  string            `json:"purpose"`
		ID       string            `json:"id"`
		Revision int64             `json:"revision"`
		Content  contentDigestBody `json:"content"`
	}{"math-master/draft-validation/v1", d.ID, d.Revision, b})
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", sha256.Sum256(raw)), nil
}
func GateFromReport(r content.WorkflowReport) GateReport {
	r = r.Display()
	return GateReport{StructuralErrors: r.StructuralErrors, CompletenessErrors: r.CompletenessErrors, HumanReviewRequirements: r.HumanReviewRequirements, StructuralTotal: r.StructuralTotal, CompletenessTotal: r.CompletenessTotal, HumanReviewTotal: r.HumanReviewTotal, Truncated: r.Truncated, ReadyToSubmit: r.ReadyToSubmit}
}
func HasRole(u auth.User, role auth.Role) bool {
	for _, r := range u.Roles {
		if r == role {
			return true
		}
	}
	return false
}
func ValidateList(q ListQuery, statuses ...string) (ListQuery, error) {
	if q.Limit == 0 {
		q.Limit = 20
	}
	if q.Scope != "" && q.Scope != "mine" && q.Scope != "all" || q.Limit < 1 || q.Limit > 100 || q.Offset < 0 || q.Offset > 100000 {
		return q, auth.ErrInvalidInput
	}
	if q.Status != "" {
		ok := false
		for _, status := range statuses {
			if q.Status == status {
				ok = true
			}
		}
		if !ok {
			return q, auth.ErrInvalidInput
		}
	}
	return q, nil
}
