package taxonomy

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
	"unicode/utf8"
)

var (
	mathID     = regexp.MustCompile(`^[a-z][a-z0-9-]{0,63}$`)
	shaPattern = regexp.MustCompile(`^[a-f0-9]{64}$`)
	topCode    = regexp.MustCompile(`^[0-9]{2}-XX$`)
	middleCode = regexp.MustCompile(`^[0-9]{2}[A-Z]xx$`)
	leafCode   = regexp.MustCompile(`^[0-9]{2}[A-Z][0-9]{2}$`)
	auxCode    = regexp.MustCompile(`^[0-9]{2}-[0-9]{2}$`)
)

func TopicID(code string) string {
	s := strings.ToLower(code)
	if strings.HasSuffix(s, "xx") {
		s = strings.TrimSuffix(s, "xx")
		s = strings.TrimSuffix(s, "-")
	}
	return "msc-" + s
}
func ValidSHA(s string) bool  { return shaPattern.MatchString(s) }
func ValidText(s string) bool { return utf8.ValidString(s) && !strings.ContainsRune(s, 0) }
func SafePath(s string) bool {
	if !ValidText(s) || s == "" || filepath.IsAbs(s) || strings.Contains(s, "\\") || strings.Contains(s, ":") {
		return false
	}
	for _, p := range strings.Split(s, "/") {
		if p == "" || p == "." || p == ".." {
			return false
		}
	}
	return true
}
func ValidNode(n TopicNode) bool {
	if !ValidText(n.Name) || strings.TrimSpace(n.Name) == "" || !ValidText(n.NameZh) || n.ID != TopicID(n.Code) || !mathID.MatchString(n.ID) {
		return false
	}
	var parent string
	switch {
	case n.Kind == "primary" && n.Level == 1 && topCode.MatchString(n.Code):
		return n.ParentID == nil
	case n.Kind == "primary" && n.Level == 2 && middleCode.MatchString(n.Code):
		parent = TopicID(n.Code[:2] + "-XX")
	case n.Kind == "primary" && n.Level == 3 && leafCode.MatchString(n.Code) && !strings.HasSuffix(n.Code, "99"):
		parent = TopicID(n.Code[:3] + "xx")
	case n.Kind == "auxiliary" && n.Level == 2 && auxCode.MatchString(n.Code):
		parent = TopicID(n.Code[:2] + "-XX")
	case n.Kind == "other" && n.Level == 3 && leafCode.MatchString(n.Code) && strings.HasSuffix(n.Code, "99"):
		parent = TopicID(n.Code[:3] + "xx")
	default:
		return false
	}
	return n.ParentID != nil && *n.ParentID == parent
}
func ValidateCatalogue(nodes []TopicNode) error {
	if len(nodes) != 6603 {
		return fmt.Errorf("%w: classification count", ErrInvalid)
	}
	by := map[string]TopicNode{}
	codes := map[string]bool{}
	counts := [5]int{}
	for _, n := range nodes {
		if !ValidNode(n) || codes[n.Code] {
			return fmt.Errorf("%w: node", ErrInvalid)
		}
		if _, exists := by[n.ID]; exists {
			return fmt.Errorf("%w: duplicate", ErrInvalid)
		}
		by[n.ID] = n
		codes[n.Code] = true
		if n.Kind == "primary" {
			counts[n.Level-1]++
		} else if n.Kind == "auxiliary" {
			counts[3]++
		} else {
			counts[4]++
		}
	}
	if counts != [5]int{63, 534, 4969, 503, 534} {
		return fmt.Errorf("%w: kind counts", ErrInvalid)
	}
	for _, n := range nodes {
		if n.ParentID != nil {
			p, ok := by[*n.ParentID]
			if !ok || p.Kind != "primary" || p.Level != n.Level-1 {
				return fmt.Errorf("%w: parent", ErrInvalid)
			}
		}
	}
	return nil
}
func ValidateAssignment(v AssignmentInput) error {
	if !mathID.MatchString(v.Knowledge.ID) || v.Knowledge.Version < 1 || v.Knowledge.Version > 2147483647 || !ValidSHA(v.SourceBatchSHA) || len(v.TopicIDs) == 0 || len(v.TopicIDs) > 16 || len(v.SourceRefs) == 0 || len(v.SourceRefs) > 8 {
		return ErrInvalid
	}
	topics := map[string]bool{}
	for _, topic := range v.TopicIDs {
		code := strings.ToUpper(strings.TrimPrefix(topic, "msc-"))
		if !strings.HasPrefix(topic, "msc-") || !leafCode.MatchString(code) || strings.HasSuffix(code, "99") || topic != TopicID(code) || topics[topic] {
			return ErrInvalid
		}
		topics[topic] = true
	}
	refs := map[SourceRecordRef]bool{}
	for _, r := range v.SourceRefs {
		if !ValidText(r.SourceID) || strings.TrimSpace(r.SourceID) == "" || len(r.SourceID) > 128 || !ValidText(r.WorkFamilyID) || strings.TrimSpace(r.WorkFamilyID) == "" || len(r.WorkFamilyID) > 128 || !ValidText(r.RecordID) || strings.TrimSpace(r.RecordID) == "" || len(r.RecordID) > 512 || !SafePath(r.Path) || len(r.Path) > 1024 || !ValidSHA(r.SHA256) || refs[r] {
			return ErrInvalid
		}
		refs[r] = true
	}
	return nil
}
