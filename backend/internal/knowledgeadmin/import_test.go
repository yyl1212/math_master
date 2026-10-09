package knowledgeadmin

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"github.com/yyl1212/math_master/backend/internal/auth"
	"io"
	"strings"
	"testing"
)

type importRepository struct {
	Repository
	deny    error
	lastSHA string
	calls   int
}

func (r *importRepository) KnowledgePreflight(context.Context, Access) (auth.User, error) {
	return auth.User{ID: "admin"}, r.deny
}
func (r *importRepository) PreviewManagedImport(_ context.Context, _ Access, d SourceDocument, sha string) (Preview, error) {
	r.calls++
	r.lastSHA = sha
	return Preview{ImportID: "preview", InputSHA256: sha, Items: []PreviewItem{}, Counts: ImportCounts{CreatedKnowledge: len(d.KnowledgePoints)}}, nil
}
func (r *importRepository) ApplyManagedImport(_ context.Context, _ Access, id string, i ApplyInput) (Receipt, error) {
	return Receipt{OperationID: id, Items: []ItemReceipt{}}, nil
}

type countedReader struct{ calls int }

func (r *countedReader) Read(b []byte) (int, error) { r.calls++; return 0, io.EOF }
func TestManagedImportBoundsAndRetry(t *testing.T) {
	repo := &importRepository{}
	s := NewService(repo)
	data := sourceBytes(t)
	p, e := s.Preview(context.Background(), Access{}, bytes.NewReader(data))
	if e != nil || p.InputSHA256 != DigestBytes(data) || repo.calls != 1 {
		t.Fatal(p, e)
	}
	repo.deny = auth.ErrForbidden
	reader := &countedReader{}
	if _, e = s.Preview(context.Background(), Access{}, reader); !errors.Is(e, auth.ErrForbidden) || reader.calls != 0 {
		t.Fatal("body read before authorization", e)
	}
	repo.deny = nil
	s.uploadSlot <- struct{}{}
	if _, e = s.Preview(context.Background(), Access{}, bytes.NewReader(data)); !errors.Is(e, ErrBusy) {
		t.Fatal("slot unbounded", e)
	}
	<-s.uploadSlot
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, e = s.Preview(ctx, Access{}, bytes.NewReader(data)); !errors.Is(e, context.Canceled) {
		t.Fatal("cancellation ignored", e)
	}
	d := sourceMap(t)
	point := d["knowledge_points"].([]any)[0]
	d["knowledge_points"] = make([]any, 100)
	for n := range d["knowledge_points"].([]any) {
		d["knowledge_points"].([]any)[n] = point
	}
	raw, _ := json.Marshal(d)
	if _, e = s.Preview(context.Background(), Access{}, bytes.NewReader(raw)); e != nil {
		t.Fatal("100 rejected", e)
	}
	d["knowledge_points"] = append(d["knowledge_points"].([]any), point)
	raw, _ = json.Marshal(d)
	if _, e = s.Preview(context.Background(), Access{}, bytes.NewReader(raw)); e == nil {
		t.Fatal("101 accepted")
	}
	padding := strings.NewReader(strings.Repeat(" ", MaxSourceBytes-len(data)))
	if _, e = s.Preview(context.Background(), Access{}, io.MultiReader(bytes.NewReader(data), padding)); e != nil {
		t.Fatal("64MiB boundary", e)
	}
	if _, e = s.Preview(context.Background(), Access{}, io.MultiReader(bytes.NewReader(data), strings.NewReader(strings.Repeat(" ", MaxSourceBytes-len(data)+1)))); e == nil {
		t.Fatal("over limit accepted")
	}
	if len(s.uploadSlot) != 0 {
		t.Fatal("slot leaked")
	}
}
func TestManagedManifestChecks(t *testing.T) {
	data := sourceBytes(t)
	d, e := DecodeSource(bytes.NewReader(data))
	if e != nil {
		t.Fatal(e)
	}
	records := []any{}
	for n, p := range d.KnowledgePoints {
		sha, _ := jsonDigest(p)
		records = append(records, map[string]any{"source_id": d.Source.SourceID, "id": p.ID, "version": p.Version, "path": "source.json", "json_pointer": "/knowledge_points/" + string(rune('0'+n)), "record_sha256": sha})
	}
	m := map[string]any{"format": "math-master-knowledge-manifest", "schema_version": "1.0", "created_at": "2026-10-09T00:00:00Z", "record_digest_algorithm": "jcs-rfc8785-sha256", "files": []any{map[string]any{"path": "source.json", "sha256": DigestBytes(data), "bytes": len(data), "source_id": d.Source.SourceID, "dataset_version": d.DatasetVersion, "record_count": len(d.KnowledgePoints)}}, "records": records, "summary": map[string]any{"records_total": 2, "ready_for_conversion": 2, "pending_records": 0, "other_type_records": 0, "specific_topic_assignment_pairs": 2, "section_other_assignment_pairs": 0, "project_other_records": 0, "mathematical_publication_approved": false}}
	snapshot, _ := jsonDigest(map[string]any{"files": m["files"], "records": m["records"]})
	m["snapshot_id"] = snapshot
	b, _ := json.Marshal(m)
	if e = VerifyManifestFile(b, "source.json", data); e != nil {
		t.Fatal(e)
	}
	if e = VerifyManifestFile(b, "../source.json", data); e == nil {
		t.Fatal("unsafe path")
	}
	if e = VerifyManifestFile(b, "source.json", append(data, ' ')); e == nil {
		t.Fatal("changed file accepted")
	}
	records[0].(map[string]any)["record_sha256"] = strings.Repeat("b", 64)
	snapshot, _ = jsonDigest(map[string]any{"files": m["files"], "records": m["records"]})
	m["snapshot_id"] = snapshot
	b, _ = json.Marshal(m)
	if VerifyManifestFile(b, "source.json", data) == nil {
		t.Fatal("forged record digest accepted")
	}
}
func TestManagedImportRejectsUnsafeJCSNumbers(t *testing.T) {
	repo := &importRepository{}
	s := NewService(repo)
	raw := mutatedSource(t, func(d, p map[string]any) { p["extensions"] = json.RawMessage(`{"value":9007199254740993}`) })
	if _, e := s.Preview(context.Background(), Access{}, bytes.NewReader(raw)); e == nil {
		t.Fatal("unsafe integer silently rounds before evidence digest")
	}
	raw = mutatedSource(t, func(d, p map[string]any) { p["extensions"] = json.RawMessage(`{"value":1e309}`) })
	if _, e := s.Preview(context.Background(), Access{}, bytes.NewReader(raw)); e == nil {
		t.Fatal("non-finite value accepted")
	}
}
