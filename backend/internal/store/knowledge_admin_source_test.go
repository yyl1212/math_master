package store_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/yyl1212/math_master/backend/internal/knowledgeadmin"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
)

// Optional real source acceptance uses the same strict decoder and authenticated import service.
// Only the random disposable database created by the fixture is mutated; no source bodies are committed.
func TestManagedDownloadedSourceAcceptance(t *testing.T) {
	root := os.Getenv("MANAGED_IMPORT_TEST_DIR")
	if root == "" {
		t.Skip("real source package not supplied")
	}
	raw, e := os.ReadFile(filepath.Join(root, "hefferon.part-0001.json"))
	if e != nil {
		t.Fatal(e)
	}
	manifest, e := os.ReadFile(filepath.Join(root, "manifest.json"))
	if e != nil {
		t.Fatal(e)
	}
	if e = knowledgeadmin.VerifyManifestFile(manifest, "hefferon.part-0001.json", raw); e != nil {
		t.Fatal(e)
	}
	doc, e := knowledgeadmin.DecodeSource(bytes.NewReader(raw))
	if e != nil {
		t.Fatal(e)
	}
	if len(doc.KnowledgePoints) != 100 {
		t.Fatalf("expected exactly 100 points, got %d", len(doc.KnowledgePoints))
	}
	f := newKnowledgeAdminFixture(t)
	f.ActivateManaged()
	service := knowledgeadmin.NewService(f.repo)
	a := f.Access("admin_a", "real-100-preview")
	preview, e := service.Preview(f.ctx, a, bytes.NewReader(raw))
	if e != nil {
		t.Fatal(e)
	}
	if preview.InputSHA256 != knowledgeadmin.DigestBytes(raw) || preview.Counts.CreatedKnowledge != 100 || preview.Counts.InvalidItems != 0 || preview.Counts.Conflicts != 0 {
		t.Fatalf("preview: %+v", preview.Counts)
	}
	indexes := make([]int, 100)
	for i := range indexes {
		indexes[i] = i
	}
	a = f.Access("admin_a", "real-100-apply")
	in := knowledgeadmin.ApplyInput{SelectedIndexes: indexes, Publish: true, PreviewToken: preview.PreviewToken}
	receipt, e := service.Apply(f.ctx, a, preview.ImportID, in)
	if e != nil {
		t.Fatal(e)
	}
	if receipt.Counts.CreatedKnowledge != 100 || len(receipt.Items) != 100 {
		t.Fatalf("receipt: %+v", receipt.Counts)
	}
	replay, e := service.Apply(f.ctx, a, preview.ImportID, in)
	if e != nil || !reflect.DeepEqual(receipt, replay) {
		t.Fatal("same-key replay", e)
	}
	memberships := 0
	for _, point := range doc.KnowledgePoints {
		id := knowledgeadmin.KnowledgeID(point.ID)
		current, e := f.repo.ReadManagedKnowledge(f.ctx, f.Access("admin_a", "read-"+point.ID), id)
		if e != nil {
			t.Fatal(e)
		}
		expectedTopics := append([]string{}, point.MSCCodes...)
		sort.Strings(expectedTopics)
		if !reflect.DeepEqual(point, current.Point) || !reflect.DeepEqual(expectedTopics, current.TopicKeys) {
			t.Fatalf("point/classification not preserved: %s pointEqual=%v wantTopics=%v actualTopics=%v", point.ID, reflect.DeepEqual(point, current.Point), point.MSCCodes, current.TopicKeys)
		}
		public, e := f.repo.ReadCurrentKnowledge(f.ctx, id)
		if e != nil || public.Point["statement"] != point.Statement || !reflect.DeepEqual(public.TopicKeys, expectedTopics) {
			t.Fatalf("public point %s: %v", point.ID, e)
		}
		if _, leaked := public.Point["original_binding"]; leaked {
			t.Fatal("binding exposed")
		}
		memberships += len(point.MSCCodes)
	}
	if f.count("SELECT count(*) FROM managed_knowledge") != 100 || f.count("SELECT count(*) FROM managed_knowledge_topics WHERE active") != memberships {
		t.Fatal("row/membership counts")
	}
	k, e := f.repo.ReadManagedKnowledge(f.ctx, f.Access("admin_a", "read-before-correction"), knowledgeadmin.KnowledgeID(doc.KnowledgePoints[0].ID))
	if e != nil {
		t.Fatal(e)
	}
	edited := knowledgeadmin.CurrentInput{ExternalID: k.ExternalID, Point: k.Point, Sources: k.Sources}
	edited.Point.Statement += " Isolated import correction preservation check."
	k, e = f.repo.UpdateManagedKnowledge(f.ctx, f.Access("admin_a", "real-100-correction"), k.ID, k.EditToken, edited)
	if e != nil {
		t.Fatal(e)
	}
	second, e := service.Preview(f.ctx, f.Access("admin_a", "real-100-second-preview"), bytes.NewReader(raw))
	if e != nil {
		t.Fatal(e)
	}
	if second.Counts.SkippedItems != 100 || second.Counts.CreatedKnowledge != 0 || second.Counts.Conflicts != 0 || second.Counts.InvalidItems != 0 {
		t.Fatalf("repeat: %+v", second.Counts)
	}
	repeated, e := service.Apply(f.ctx, f.Access("admin_a", "real-100-second-apply"), second.ImportID, knowledgeadmin.ApplyInput{SelectedIndexes: []int{}, Publish: true, PreviewToken: second.PreviewToken})
	if e != nil || repeated.Counts.SkippedItems != 100 {
		t.Fatal("repeat apply", e)
	}
	after, e := f.repo.ReadManagedKnowledge(f.ctx, f.Access("admin_a", "read-after-correction"), k.ID)
	if e != nil || after.Point.Statement != edited.Point.Statement {
		t.Fatal("correction overwritten", e)
	}
	if f.count("SELECT count(*) FROM managed_knowledge") != 100 || f.count("SELECT count(*) FROM managed_knowledge_topics WHERE active") != memberships {
		t.Fatal("duplicate import rows")
	}
	expected := map[string]map[string]bool{}
	for _, point := range doc.KnowledgePoints {
		for _, code := range point.MSCCodes {
			for _, prefix := range []string{code[:2], code[:3], code} {
				if expected[prefix] == nil {
					expected[prefix] = map[string]bool{}
				}
				expected[prefix][knowledgeadmin.KnowledgeID(point.ID)] = true
			}
		}
	}
	for prefix, ids := range expected {
		page, e := f.repo.ListCurrentKnowledge(f.ctx, knowledgeadmin.Query{TopicKey: prefix, Limit: 100})
		if e != nil {
			t.Fatal(e)
		}
		if e = validateSourceTopicPage(prefix, ids, page); e != nil {
			t.Fatal(e)
		}
	}

	result, _ := json.Marshal(map[string]any{"ok": true, "points": 100, "memberships": memberships, "created": receipt.Counts, "repeat": repeated.Counts, "manualCorrectionPreserved": true, "originalBindingLocallyVerified": false})
	t.Log(string(result))
}

func validateSourceTopicPage(prefix string, expected map[string]bool, page knowledgeadmin.Page[knowledgeadmin.PublicKnowledge]) error {
	if page.Total != len(expected) || len(page.Items) != len(expected) {
		return fmt.Errorf("%s incomplete page: total=%d rows=%d expected=%d", prefix, page.Total, len(page.Items), len(expected))
	}
	seen := map[string]bool{}
	for _, item := range page.Items {
		if !expected[item.ID] || seen[item.ID] {
			return fmt.Errorf("%s unexpected or duplicate ID %s", prefix, item.ID)
		}
		seen[item.ID] = true
		matches := false
		for _, key := range item.TopicKeys {
			if strings.HasPrefix(key, prefix) {
				matches = true
			}
		}
		if !matches {
			return fmt.Errorf("%s wrong membership for %s", prefix, item.ID)
		}
	}
	return nil
}

func TestManagedSourceTopicPageCompleteness(t *testing.T) {
	expected := map[string]bool{"a": true, "b": true}
	good := knowledgeadmin.PublicKnowledge{ID: "a", TopicKeys: []string{"15A06"}}
	other := knowledgeadmin.PublicKnowledge{ID: "b", TopicKeys: []string{"15A03"}}
	for _, c := range []struct {
		name string
		page knowledgeadmin.Page[knowledgeadmin.PublicKnowledge]
		bad  bool
	}{
		{"complete", knowledgeadmin.Page[knowledgeadmin.PublicKnowledge]{Total: 2, Items: []knowledgeadmin.PublicKnowledge{good, other}}, false},
		{"missing", knowledgeadmin.Page[knowledgeadmin.PublicKnowledge]{Total: 1, Items: []knowledgeadmin.PublicKnowledge{good}}, true},
		{"wrong-total", knowledgeadmin.Page[knowledgeadmin.PublicKnowledge]{Total: 3, Items: []knowledgeadmin.PublicKnowledge{good, other}}, true},
		{"duplicate", knowledgeadmin.Page[knowledgeadmin.PublicKnowledge]{Total: 2, Items: []knowledgeadmin.PublicKnowledge{good, good}}, true},
		{"unexpected", knowledgeadmin.Page[knowledgeadmin.PublicKnowledge]{Total: 2, Items: []knowledgeadmin.PublicKnowledge{good, {ID: "c", TopicKeys: []string{"15A06"}}}}, true},
		{"wrong-classification", knowledgeadmin.Page[knowledgeadmin.PublicKnowledge]{Total: 2, Items: []knowledgeadmin.PublicKnowledge{good, {ID: "b", TopicKeys: []string{"97F40"}}}}, true},
	} {
		t.Run(c.name, func(t *testing.T) {
			if e := validateSourceTopicPage("15A", expected, c.page); (e != nil) != c.bad {
				t.Fatalf("bad=%v error=%v", c.bad, e)
			}
		})
	}
}
