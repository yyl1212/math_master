package contentaudit

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"github.com/yyl1212/math_master/backend/internal/question"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"time"
)

func hashBytes(b []byte) string { h := sha256.Sum256(b); return hex.EncodeToString(h[:]) }

type fileReadResult struct {
	bytes []byte
	err   error
}
type contextReader struct {
	ctx    context.Context
	reader io.Reader
}

func (r contextReader) Read(p []byte) (int, error) {
	if e := r.ctx.Err(); e != nil {
		return 0, e
	}
	if len(p) > 32<<10 {
		p = p[:32<<10]
	}
	n, e := r.reader.Read(p)
	if canceled := r.ctx.Err(); canceled != nil {
		return 0, canceled
	}
	return n, e
}
func readBounded(ctx context.Context, path string, limit int) ([]byte, error) {
	return readRegular(ctx, path, int64(limit), nil)
}

// The worker makes file-system operations cancellable to the caller, including open/stat.
// Close also interrupts reads where supported. Only regular files enter the reader.
func readRegular(ctx context.Context, path string, limit int64, declared *int64) ([]byte, error) {
	if e := ctx.Err(); e != nil {
		return nil, e
	}
	if limit < 0 || declared != nil && (*declared < 0 || *declared > limit) {
		return nil, ErrLimit
	}
	ctx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()
	done := make(chan fileReadResult)
	go func() {
		b, e := readRegularFile(ctx, path, limit, declared)
		select {
		case done <- fileReadResult{b, e}:
		case <-ctx.Done():
		}
	}()
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case r := <-done:
		return r.bytes, r.err
	}
}
func readRegularFile(ctx context.Context, path string, limit int64, declared *int64) ([]byte, error) {
	before, e := os.Lstat(path)
	if e != nil {
		return nil, e
	}
	if !before.Mode().IsRegular() {
		return nil, ErrInvalid
	}
	if before.Size() > limit {
		return nil, ErrLimit
	}
	if declared != nil && before.Size() != *declared {
		return nil, ErrInvalid
	}
	if e = ctx.Err(); e != nil {
		return nil, e
	}
	f, e := os.Open(path)
	if e != nil {
		return nil, e
	}
	defer f.Close()
	stop := context.AfterFunc(ctx, func() { f.Close() })
	defer stop()
	opened, e := f.Stat()
	if e != nil {
		return nil, e
	}
	if !opened.Mode().IsRegular() || !os.SameFile(before, opened) || before.Size() != opened.Size() || !before.ModTime().Equal(opened.ModTime()) {
		return nil, ErrInvalid
	}
	capBytes := limit
	if declared != nil {
		capBytes = *declared
	}
	b, e := io.ReadAll(io.LimitReader(contextReader{ctx, f}, capBytes+1))
	if e != nil {
		return nil, e
	}
	if int64(len(b)) > capBytes {
		return nil, ErrLimit
	}
	if declared != nil && int64(len(b)) != *declared {
		return nil, ErrInvalid
	}
	after, e := f.Stat()
	if e != nil {
		return nil, e
	}
	current, e := os.Lstat(path)
	if e != nil {
		return nil, e
	}
	if !current.Mode().IsRegular() || !os.SameFile(before, after) || !os.SameFile(before, current) || before.Size() != after.Size() || before.Size() != current.Size() || !before.ModTime().Equal(after.ModTime()) || !before.ModTime().Equal(current.ModTime()) {
		return nil, ErrInvalid
	}
	return b, ctx.Err()
}
func ReadEvidence(ctx context.Context, path string) (AcceptanceEvidence, error) {
	var evidence AcceptanceEvidence
	raw, e := readBounded(ctx, path, MaxMetadataBytes)
	if e != nil {
		return evidence, e
	}
	e = DecodeEvidence(bytes.NewReader(raw), &evidence)
	if e == nil {
		e = ctx.Err()
	}
	return evidence, e
}
func DecodeSourceMap(r io.Reader, out *SourceMap) error {
	return strictDecode(r, MaxSourceMapBytes, out)
}
func DecodeSourceReport(r io.Reader, out *SourceReport) error {
	return strictDecode(r, MaxMetadataBytes, out)
}
func DecodeEvidence(r io.Reader, out *AcceptanceEvidence) error {
	return strictDecode(r, MaxMetadataBytes, out)
}
func safePath(p string) bool {
	if !filepath.IsLocal(p) || strings.ContainsAny(p, "\\\x00") {
		return false
	}
	for _, s := range strings.Split(p, "/") {
		if s == "" || s == "." || s == ".." {
			return false
		}
	}
	return true
}
func fixedPath(root, path string) (string, error) {
	if !safePath(path) {
		return "", ErrInvalid
	}
	current := root
	parts := append([]string{""}, strings.Split(path, "/")...)
	for i, part := range parts {
		current = filepath.Join(current, part)
		s, e := os.Lstat(current)
		if e != nil {
			return "", e
		}
		if s.Mode()&os.ModeSymlink != 0 || i < len(parts)-1 && !s.IsDir() || i == len(parts)-1 && !s.Mode().IsRegular() {
			return "", ErrInvalid
		}
	}
	return current, nil
}

// Manifest has its original fields; Files keeps the original JSON spelling for the JavaScript snapshot identity.
type snapshotManifest struct {
	SchemaVersion         int             `json:"schemaVersion"`
	SnapshotID            string          `json:"snapshotId"`
	CreatedAt             string          `json:"createdAt"`
	SourceRoot            string          `json:"sourceRoot"`
	PackageCount          int             `json:"packageCount"`
	PrimaryFiles          []string        `json:"primaryFiles"`
	Files                 json.RawMessage `json:"files"`
	Changes               json.RawMessage `json:"changes"`
	SourceIndexMismatches []string        `json:"sourceIndexMismatches"`
}
type snapshotFile struct {
	Path      string `json:"path"`
	SizeBytes int64  `json:"sizeBytes"`
	SHA256    string `json:"sha256"`
}

func LoadSources(ctx context.Context, snapshotDir, reportFile, mapFile string) (SourceBundle, error) {
	var out SourceBundle
	rb, e := readBounded(ctx, reportFile, MaxMetadataBytes)
	if e != nil {
		return out, e
	}
	if e = DecodeSourceReport(bytes.NewReader(rb), &out.Report); e != nil {
		return out, e
	}
	mb, e := readBounded(ctx, mapFile, MaxSourceMapBytes)
	if e != nil {
		return out, e
	}
	if e = DecodeSourceMap(bytes.NewReader(mb), &out.Mapping); e != nil {
		return out, e
	}
	out.ReportSHA = hashBytes(rb)
	out.SnapshotID = out.Report.SnapshotID
	r, m := out.Report, out.Mapping
	if r.SchemaVersion != 1 || r.PolicyVersion != 1 || r.PublicationApproved || m.SchemaVersion != 1 || m.PolicyVersion != r.PolicyVersion || m.SnapshotID != r.SnapshotID || m.SourceReportSHA256 != out.ReportSHA || !question.ValidSHA(r.SnapshotID) {
		return out, ErrInvalid
	}
	mp, e := fixedPath(snapshotDir, "manifest.json")
	if e != nil {
		return out, e
	}
	raw, e := readBounded(ctx, mp, MaxMetadataBytes)
	if e != nil {
		return out, e
	}
	if e = uniqueJSON(raw); e != nil {
		return out, e
	}
	var manifest snapshotManifest
	d := json.NewDecoder(bytes.NewReader(raw))
	d.DisallowUnknownFields()
	if d.Decode(&manifest) != nil {
		return out, ErrInvalid
	}
	var compact bytes.Buffer
	if json.Compact(&compact, manifest.Files) != nil || manifest.SchemaVersion != 1 || manifest.SnapshotID != r.SnapshotID || hashBytes(compact.Bytes()) != r.SnapshotID {
		return out, ErrInvalid
	}
	var files []snapshotFile
	if json.Unmarshal(manifest.Files, &files) != nil || len(files) > 10000 {
		return out, ErrLimit
	}
	fm := map[string]snapshotFile{}
	for _, f := range files {
		if !safePath(f.Path) || !question.ValidSHA(f.SHA256) || f.SizeBytes < 0 {
			return out, ErrInvalid
		}
		if _, ok := fm[f.Path]; ok {
			return out, ErrInvalid
		}
		fm[f.Path] = f
	}
	type corpus struct {
		DatasetID string `json:"dataset_id"`
		Knowledge []struct {
			ID string `json:"id"`
		} `json:"knowledge_points"`
	}
	selected := map[string]SelectedSource{}
	records := map[string]map[string]bool{}
	for _, s := range r.SelectedFiles {
		if e = ctx.Err(); e != nil {
			return out, e
		}
		f, ok := fm[s.Path]
		if !ok || f.SHA256 != s.SHA256 || !safePath(s.Path) {
			return out, ErrInvalid
		}
		if _, exists := selected[s.Path]; exists {
			return out, ErrInvalid
		}
		path, e := fixedPath(filepath.Join(snapshotDir, "files"), s.Path)
		if e != nil {
			return out, e
		}
		raw, e := readRegular(ctx, path, MaxSourceFileBytes, &f.SizeBytes)
		if e != nil {
			return out, e
		}
		if hashBytes(raw) != f.SHA256 {
			return out, ErrInvalid
		}
		var c corpus
		e = json.Unmarshal(raw, &c)
		if e != nil || c.DatasetID != s.DatasetID || len(c.Knowledge) != len(s.RecordIDs) {
			return out, ErrInvalid
		}
		ids := map[string]bool{}
		for _, k := range c.Knowledge {
			if k.ID == "" || ids[k.ID] {
				return out, ErrInvalid
			}
			ids[k.ID] = true
		}
		for _, id := range s.RecordIDs {
			if !ids[id] {
				return out, ErrInvalid
			}
		}
		selected[s.Path] = s
		records[s.Path] = ids
	}
	blocked := false
	for _, i := range r.Issues {
		if !safePath(i.Path) || i.Code == "" {
			return out, ErrInvalid
		}
		if i.BlocksSelected {
			blocked = true
		}
	}
	if r.Ready == blocked {
		return out, ErrInvalid
	}
	sourceIDs := map[string]bool{}
	for _, s := range m.Sources {
		picked, ok := selected[s.Path]
		if !ok || !records[s.Path][s.RecordID] || s.DatasetID != picked.DatasetID || s.FileSHA256 != picked.SHA256 || !question.ValidMathID(s.ID) || sourceIDs[s.ID] || !(s.Use == "fact_check" || s.Use == "background" || s.Use == "original_derivation") {
			return out, ErrInvalid
		}
		sourceIDs[s.ID] = true
	}
	objects := map[string]bool{}
	for _, o := range m.Objects {
		key := objectKey(ObjectIdentity{Kind: o.Kind, ID: o.ID, Version: o.Version})
		if objects[key] || o.Origin != "original" || !question.ValidSHA(o.SHA256) || !question.ValidInstanceID(o.ID) || len(o.SourceIDs) == 0 {
			return out, ErrInvalid
		}
		if o.Kind == "asset" {
			if o.Version != nil {
				return out, ErrInvalid
			}
		} else if o.Version == nil || *o.Version < 1 {
			return out, ErrInvalid
		}
		for _, id := range o.SourceIDs {
			if !sourceIDs[id] {
				return out, ErrInvalid
			}
		}
		objects[key] = true
	}
	return out, ctx.Err()
}

// uniqueJSON applies the shared duplicate-key/UTF-8/NUL/int32 scanner without imposing the source corpus shape.
func uniqueJSON(raw []byte) error {
	var out snapshotScan
	return question.DecodeStrictJSON(bytes.NewReader(raw), MaxMetadataBytes, &out)
}

type snapshotScan struct {
	SchemaVersion         int              `json:"schemaVersion"`
	SnapshotID            string           `json:"snapshotId"`
	CreatedAt             *string          `json:"createdAt"`
	SourceRoot            *string          `json:"sourceRoot"`
	PackageCount          *int             `json:"packageCount"`
	PrimaryFiles          *[]string        `json:"primaryFiles"`
	Files                 []snapshotFile   `json:"files"`
	Changes               *snapshotChanges `json:"changes"`
	SourceIndexMismatches *[]string        `json:"sourceIndexMismatches"`
}
type snapshotChanges struct {
	Added    []string `json:"added"`
	Modified []string `json:"modified"`
	Missing  []string `json:"missing"`
}

func strictDecode(r io.Reader, limit int, out any) error {
	raw, e := io.ReadAll(io.LimitReader(r, int64(limit)+1))
	if e != nil {
		return e
	}
	if len(raw) > limit {
		return ErrLimit
	}
	if e = question.DecodeStrictJSON(bytes.NewReader(raw), limit, out); e != nil {
		return e
	}
	var valid func(reflect.Value) bool
	valid = func(v reflect.Value) bool {
		switch v.Kind() {
		case reflect.Pointer:
			return v.IsNil() || valid(v.Elem())
		case reflect.Struct:
			for i := 0; i < v.NumField(); i++ {
				if !valid(v.Field(i)) {
					return false
				}
			}
		case reflect.Slice:
			for i := 0; i < v.Len(); i++ {
				if !valid(v.Index(i)) {
					return false
				}
			}
		case reflect.Int, reflect.Int32, reflect.Int64:
			return v.Int() >= -2147483648 && v.Int() <= 2147483647
		}
		return true
	}
	if !valid(reflect.ValueOf(out)) {
		return ErrInvalid
	}
	return nil
}
