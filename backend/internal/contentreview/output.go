package contentreview

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
)

type outputHooks struct {
	Before     func(string) error
	AfterClose func(string) error
}

func validateOutput(files []ExportFile) error {
	if len(files) > MaxFiles {
		return ErrLimit
	}
	total := 0
	seen := map[string]bool{}
	for _, f := range files {
		if !safePath(f.Path) || seen[f.Path] {
			return ErrInvalid
		}
		seen[f.Path] = true
		if len(f.Bytes) > MaxFileBytes {
			return ErrLimit
		}
		total += len(f.Bytes)
		if total > MaxTotalBytes {
			return ErrLimit
		}
	}
	return nil
}
func canonicalDir(p string) error {
	if !filepath.IsAbs(p) || filepath.Clean(p) != p {
		return ErrInvalid
	}
	actual, e := filepath.EvalSymlinks(p)
	if e != nil {
		return e
	}
	if actual != p {
		return ErrInvalid
	}
	s, e := os.Lstat(p)
	if e != nil {
		return e
	}
	if !s.IsDir() || s.Mode()&os.ModeSymlink != 0 {
		return ErrInvalid
	}
	return nil
}

func writePrivate(ctx context.Context, out string, files []ExportFile, hooks outputHooks) (result error) {
	if e := ctx.Err(); e != nil {
		return e
	}
	if e := validateOutput(files); e != nil {
		return e
	}
	if !filepath.IsAbs(out) || filepath.Clean(out) != out || out == "/" {
		return ErrInvalid
	}
	parentPath, base := filepath.Dir(out), filepath.Base(out)
	if e := canonicalDir(parentPath); e != nil {
		return e
	}
	parent, e := os.OpenRoot(parentPath)
	if e != nil {
		return e
	}
	defer parent.Close()
	before, e := os.Lstat(parentPath)
	if e != nil {
		return e
	}
	opened, e := parent.Stat(".")
	if e != nil || !os.SameFile(before, opened) {
		return ErrInvalid
	}
	// Mkdir is the atomic new-directory reservation; an existing name is never reused.
	if e = parent.Mkdir(base, 0700); e != nil {
		return e
	}
	own, e := parent.Lstat(base)
	if e != nil {
		return e
	}
	success := false
	defer func() {
		if !success {
			current, e := parent.Lstat(base)
			if e == nil && os.SameFile(own, current) {
				if cleanup := parent.RemoveAll(base); cleanup != nil && result == nil {
					result = cleanup
				}
			}
		}
	}()
	root, e := parent.OpenRoot(base)
	if e != nil {
		return e
	}
	defer root.Close()
	actual, e := root.Stat(".")
	if e != nil || !os.SameFile(own, actual) {
		return ErrInvalid
	}
	dirs := map[string]bool{".": true}
	for _, file := range files {
		if e = ctx.Err(); e != nil {
			return e
		}
		dir := path.Dir(file.Path)
		parts := strings.Split(dir, "/")
		current := ""
		for _, part := range parts {
			if part == "." {
				continue
			}
			if current == "" {
				current = part
			} else {
				current += "/" + part
			}
			if !dirs[current] {
				if e = root.Mkdir(current, 0700); e != nil {
					return e
				}
				dirs[current] = true
			}
			s, e := root.Lstat(current)
			if e != nil || !s.IsDir() || s.Mode()&os.ModeSymlink != 0 {
				return ErrInvalid
			}
		}
		if hooks.Before != nil {
			if e = hooks.Before(file.Path); e != nil {
				return e
			}
		}
		if e = ctx.Err(); e != nil {
			return e
		}
		f, e := root.OpenFile(file.Path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
		if e != nil {
			return e
		}
		e = writeFile(ctx, f, file.Bytes)
		closeErr := f.Close()
		if e != nil {
			return e
		}
		if closeErr != nil {
			return closeErr
		}
		if hooks.AfterClose != nil {
			if e = hooks.AfterClose(file.Path); e != nil {
				return e
			}
		}
	}
	if e = ctx.Err(); e != nil {
		return e
	}
	if e = canonicalDir(parentPath); e != nil {
		return e
	}
	after, e := os.Lstat(parentPath)
	if e != nil || !os.SameFile(before, after) {
		return ErrInvalid
	}
	current, e := parent.Lstat(base)
	if e != nil || !os.SameFile(own, current) {
		return ErrInvalid
	}
	success = true
	return nil
}
func writeFile(ctx context.Context, f *os.File, b []byte) error {
	s, e := f.Stat()
	if e != nil {
		return e
	}
	if !s.Mode().IsRegular() {
		return ErrInvalid
	}
	if e = f.Chmod(0600); e != nil {
		return e
	}
	for offset := 0; offset < len(b); {
		if e = ctx.Err(); e != nil {
			return e
		}
		end := min(offset+(64<<10), len(b))
		n, e := f.Write(b[offset:end])
		if e != nil {
			return e
		}
		if n != end-offset {
			return io.ErrShortWrite
		}
		offset = end
	}
	if e = ctx.Err(); e != nil {
		return e
	}
	return f.Sync()
}
func WriteBundle(ctx context.Context, out string, b Bundle) error {
	expected := map[string]FileEntry{}
	if b.Manifest.SchemaVersion != 1 {
		return ErrInvalid
	}
	for _, f := range b.Manifest.Files {
		if !safePath(f.Path) || f.Path == "review-manifest.json" || f.Bytes < 0 {
			return ErrInvalid
		}
		if _, ok := expected[f.Path]; ok {
			return ErrInvalid
		}
		expected[f.Path] = f
	}
	for _, f := range b.Files {
		if f.Path == "review-manifest.json" {
			return ErrInvalid
		}
		if want, ok := expected[f.Path]; ok {
			if want.Bytes != len(f.Bytes) || want.SHA256 != digestBytes(f.Bytes) {
				return ErrInvalid
			}
			delete(expected, f.Path)
		} else if f.Path != "review-register.json" && !strings.HasPrefix(f.Path, "register/") {
			return ErrInvalid
		}
	}
	if len(expected) != 0 {
		return ErrInvalid
	}
	raw, e := json.Marshal(b.Manifest)
	if e != nil {
		return e
	}
	files := append([]ExportFile{}, b.Files...)
	files = append(files, ExportFile{"review-manifest.json", raw})
	return writePrivate(ctx, out, files, outputHooks{})
}
func WriteVerification(ctx context.Context, out string, v Verification) error {
	if v.SchemaVersion != 1 || !(v.Conclusion == "evidence_ready" || v.Conclusion == "not_ready" || v.Conclusion == "awaiting_review") {
		return ErrInvalid
	}
	files := append([]ExportFile{}, v.Files...)
	hasEvidence := false
	entries := []FileEntry{}
	for _, f := range files {
		if f.Path == "verification.json" {
			return ErrInvalid
		}
		if f.Path == "acceptance-evidence.json" {
			hasEvidence = true
			if v.Evidence == nil || v.Conclusion == "awaiting_review" {
				return ErrInvalid
			}
			raw, e := json.Marshal(v.Evidence)
			if e != nil || !bytes.Equal(raw, f.Bytes) || v.Evidence.FixtureOnly != v.FixtureOnly {
				return ErrInvalid
			}
		}
		entries = append(entries, entry(f))
	}
	if hasEvidence != (v.Evidence != nil) || v.Conclusion == "evidence_ready" && !hasEvidence {
		return ErrInvalid
	}
	report := struct {
		Verification
		Files []FileEntry `json:"files"`
	}{v, entries}
	raw, e := json.Marshal(report)
	if e != nil {
		return e
	}
	files = append(files, ExportFile{"verification.json", raw})
	return writePrivate(ctx, out, files, outputHooks{})
}
