package knowledgeadmin

import (
	"bytes"
	"context"
	"crypto/sha256"
	"fmt"
	"io"
)

type contextReader struct {
	ctx context.Context
	r   io.Reader
}

func (r contextReader) Read(b []byte) (int, error) {
	if e := r.ctx.Err(); e != nil {
		return 0, e
	}
	return r.r.Read(b)
}
func (s *Service) Preview(ctx context.Context, a Access, r io.Reader) (Preview, error) {
	if e := ctx.Err(); e != nil {
		return Preview{}, e
	}
	if _, e := s.Repository.KnowledgePreflight(ctx, a); e != nil {
		return Preview{}, e
	}
	select {
	case s.uploadSlot <- struct{}{}:
		defer func() { <-s.uploadSlot }()
	default:
		return Preview{}, ErrBusy
	}
	h := sha256.New()
	d, e := DecodeSource(io.TeeReader(contextReader{ctx, r}, h))
	if ce := ctx.Err(); ce != nil {
		return Preview{}, ce
	}
	if e != nil {
		return Preview{}, e
	}
	return s.Repository.PreviewManagedImport(ctx, a, d, fmt.Sprintf("%x", h.Sum(nil)))
}
func (s *Service) Apply(ctx context.Context, a Access, id string, in ApplyInput) (Receipt, error) {
	if e := ctx.Err(); e != nil {
		return Receipt{}, e
	}
	return s.Repository.ApplyManagedImport(ctx, a, id, in)
}

// VerifyManifestFile checks only bytes supplied with this upload. It never opens original_binding paths.
func VerifyManifestFile(manifest []byte, path string, data []byte) error {
	if !SafeRelativePath(path) || len(manifest) > MaxSourceBytes || len(data) > MaxSourceBytes {
		return invalid("/files/path")
	}
	v, e := StrictJSON(manifest)
	if e != nil {
		return e
	}
	if e = validateManifestSchema(v); e != nil {
		return e
	}
	m := v.(map[string]any)
	snapshot, e := jsonDigest(map[string]any{"files": m["files"], "records": m["records"]})
	if e != nil || m["snapshot_id"] != snapshot {
		return invalid("/snapshot_id")
	}
	d, e := DecodeSource(bytes.NewReader(data))
	if e != nil {
		return e
	}
	var file map[string]any
	seenFiles := map[string]bool{}
	for _, part := range m["files"].([]any) {
		f := part.(map[string]any)
		name := f["path"].(string)
		if !SafeRelativePath(name) || seenFiles[name] {
			return invalid("/files/path")
		}
		seenFiles[name] = true
		if name == path {
			file = f
		}
	}
	number := func(v any, want int) bool { return fmt.Sprint(v) == fmt.Sprint(want) }
	if file == nil || file["sha256"] != DigestBytes(data) || !number(file["bytes"], len(data)) || file["source_id"] != d.Source.SourceID || !number(file["dataset_version"], d.DatasetVersion) || !number(file["record_count"], len(d.KnowledgePoints)) {
		return invalid("/files")
	}
	expected := map[string]int{}
	for n := range d.KnowledgePoints {
		expected[fmt.Sprintf("/knowledge_points/%d", n)] = n
	}
	seen := map[string]bool{}
	for _, part := range m["records"].([]any) {
		r := part.(map[string]any)
		rp := r["path"].(string)
		if !SafeRelativePath(rp) || !seenFiles[rp] {
			return invalid("/records/path")
		}
		if rp != path {
			continue
		}
		pointer := r["json_pointer"].(string)
		n, ok := expected[pointer]
		if !ok || seen[pointer] {
			return invalid("/records/json_pointer")
		}
		seen[pointer] = true
		p := d.KnowledgePoints[n]
		sha, e := jsonDigest(p)
		if e != nil || r["record_sha256"] != sha || r["source_id"] != d.Source.SourceID || r["id"] != p.ID || !number(r["version"], p.Version) {
			return invalid("/records")
		}
	}
	if len(seen) != len(expected) {
		return invalid("/records")
	}
	return nil
}
func validateManifestSchema(v any) error {
	schemaOnce.Do(compileSchemas)
	if schemaError != nil {
		return &DecodeError{"SCHEMA_UNAVAILABLE", "/"}
	}
	schemaLock.Lock()
	defer schemaLock.Unlock()
	regexpFailed = false
	if e := manifestSchema.Validate(v); e != nil || regexpFailed {
		return invalid("/manifest")
	}
	return nil
}
