package contentaudit

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"

	"github.com/yyl1212/math_master/backend/internal/content"
	"github.com/yyl1212/math_master/backend/internal/question"
)

// DraftSelection chooses exact files; it never infers versions from names or mtimes.
type DraftSelection struct {
	CataloguePath string
	ContentPath   string
	QuestionPaths []string
	AssetsRoot    string
}
type CapturedFile struct {
	Path  string
	Bytes []byte
}
type SelectedDraft struct {
	Input  DraftInput
	Files  []CapturedFile
	Assets map[string][]byte
}

func canonicalDirectory(path string) error {
	if !filepath.IsAbs(path) || filepath.Clean(path) != path {
		return ErrInvalid
	}
	resolved, e := filepath.EvalSymlinks(path)
	if e != nil {
		return e
	}
	if resolved != path {
		return ErrInvalid
	}
	s, e := os.Lstat(path)
	if e != nil {
		return e
	}
	if !s.IsDir() || s.Mode()&os.ModeSymlink != 0 {
		return ErrInvalid
	}
	return nil
}
func directoryUnder(root, relative string) (string, error) {
	if !safePath(relative) {
		return "", ErrInvalid
	}
	p := root
	for _, part := range strings.Split(relative, "/") {
		p = filepath.Join(p, part)
		s, e := os.Lstat(p)
		if e != nil {
			return "", e
		}
		if !s.IsDir() || s.Mode()&os.ModeSymlink != 0 {
			return "", ErrInvalid
		}
	}
	return p, nil
}

// ReadFileUnder shares the original cancellable, identity-checked normal-file reader.
// The anchored root also prevents parent-renaming races from escaping containment.
func ReadFileUnder(ctx context.Context, root, relative string, limit int) ([]byte, error) {
	if e := ctx.Err(); e != nil {
		return nil, e
	}
	if e := canonicalDirectory(root); e != nil {
		return nil, e
	}
	p, e := fixedPath(root, relative)
	if e != nil {
		return nil, e
	}
	before, e := os.Lstat(root)
	if e != nil {
		return nil, e
	}
	anchor, e := os.OpenRoot(root)
	if e != nil {
		return nil, e
	}
	defer anchor.Close()
	opened, e := anchor.Stat(".")
	if e != nil {
		return nil, e
	}
	if !os.SameFile(before, opened) {
		return nil, ErrInvalid
	}
	b, e := readRegularOpened(ctx, p, int64(limit), nil, func(string) (*os.File, error) { return anchor.Open(relative) })
	if e != nil {
		return nil, e
	}
	if e = canonicalDirectory(root); e != nil {
		return nil, e
	}
	after, e := os.Lstat(root)
	if e != nil {
		return nil, e
	}
	if !os.SameFile(before, after) {
		return nil, ErrInvalid
	}
	if _, e = fixedPath(root, relative); e != nil {
		return nil, e
	}
	return b, ctx.Err()
}
func strictCaptured(raw []byte, limit int, out any) error {
	if e := question.DecodeStrictJSON(bytes.NewReader(raw), limit, out); e != nil {
		return ErrInvalid
	}
	return nil
}
func LoadSelectedDraft(ctx context.Context, root string, s DraftSelection) (SelectedDraft, error) {
	out := SelectedDraft{Files: []CapturedFile{}, Assets: map[string][]byte{}}
	if e := ctx.Err(); e != nil {
		return out, e
	}
	if len(s.QuestionPaths) != 5 {
		return out, ErrInvalid
	}
	if e := canonicalDirectory(root); e != nil {
		return out, e
	}
	assetsRoot, e := directoryUnder(root, s.AssetsRoot)
	if e != nil {
		return out, e
	}
	out.Input.AssetsRoot = assetsRoot
	seen := map[string]bool{}
	capture := func(path string, limit int) ([]byte, error) {
		if seen[path] {
			return nil, ErrInvalid
		}
		seen[path] = true
		raw, e := ReadFileUnder(ctx, root, path, limit)
		if e != nil {
			return nil, e
		}
		out.Files = append(out.Files, CapturedFile{Path: path, Bytes: raw})
		return raw, nil
	}
	raw, e := capture(s.CataloguePath, MaxMetadataBytes)
	if e != nil {
		return out, e
	}
	if e = strictCaptured(raw, MaxMetadataBytes, &out.Input.Catalogue); e != nil {
		return out, e
	}
	out.Input.Catalogue, e = content.DecodeCatalogue(bytes.NewReader(raw))
	if e != nil {
		return out, e
	}
	raw, e = capture(s.ContentPath, content.MaxWorkflowPackageBytes)
	if e != nil {
		return out, e
	}
	if e = strictCaptured(raw, content.MaxWorkflowPackageBytes, &out.Input.Content); e != nil {
		return out, e
	}
	out.Input.Content, e = content.DecodePackage(bytes.NewReader(raw))
	if e != nil {
		return out, e
	}
	out.Input.Questions = []question.QuestionPackage{}
	ids := map[string]bool{}
	for _, path := range s.QuestionPaths {
		raw, e = capture(path, question.MaxPackageBytes)
		if e != nil {
			return out, e
		}
		p, e := question.DecodePackage(bytes.NewReader(raw))
		if e != nil {
			return out, e
		}
		if ids[p.ID] {
			return out, ErrInvalid
		}
		ids[p.ID] = true
		out.Input.Questions = append(out.Input.Questions, p)
	}
	if len(out.Input.Content.Assets) > 16 {
		return out, ErrLimit
	}
	for _, a := range out.Input.Content.Assets {
		if _, ok := out.Assets[a.ID]; ok {
			return out, ErrInvalid
		}
		raw, e = capture(filepath.Join(s.AssetsRoot, a.Path), 1<<20)
		if e != nil {
			return out, e
		}
		out.Assets[a.ID] = raw
	}
	return out, ctx.Err()
}
func CheckSelectedDraft(ctx context.Context, s SelectedDraft) (DraftFacts, error) {
	if e := ctx.Err(); e != nil {
		return DraftFacts{}, e
	}
	if len(s.Input.Questions) != 5 || len(s.Files) != 7+len(s.Input.Content.Assets) || len(s.Assets) != len(s.Input.Content.Assets) {
		return DraftFacts{}, ErrInvalid
	}
	seen := map[string]bool{}
	for _, f := range s.Files {
		if !safePath(f.Path) || seen[f.Path] {
			return DraftFacts{}, ErrInvalid
		}
		seen[f.Path] = true
	}
	c, e := content.DecodeCatalogue(bytes.NewReader(s.Files[0].Bytes))
	if e != nil || content.Digest(c) != content.Digest(s.Input.Catalogue) {
		return DraftFacts{}, ErrInvalid
	}
	p, e := content.DecodePackage(bytes.NewReader(s.Files[1].Bytes))
	if e != nil || content.Digest(p) != content.Digest(s.Input.Content) {
		return DraftFacts{}, ErrInvalid
	}
	for i, raw := range s.Files[2:7] {
		q, e := question.DecodePackage(bytes.NewReader(raw.Bytes))
		if e != nil || content.Digest(q) != content.Digest(s.Input.Questions[i]) {
			return DraftFacts{}, ErrInvalid
		}
	}
	for i, a := range s.Input.Content.Assets {
		if !bytes.Equal(s.Files[7+i].Bytes, s.Assets[a.ID]) {
			return DraftFacts{}, ErrInvalid
		}
	}
	return checkDraftWithReader(ctx, s.Input, func(ctx context.Context, a content.Asset) ([]byte, error) {
		if e := ctx.Err(); e != nil {
			return nil, e
		}
		b, ok := s.Assets[a.ID]
		if !ok {
			return nil, ErrInvalid
		}
		return bytes.Clone(b), nil
	})
}
func LoadSourcesFromBytes(ctx context.Context, snapshot string, reportRaw, mapRaw []byte) (SourceBundle, error) {
	var out SourceBundle
	if e := ctx.Err(); e != nil {
		return out, e
	}
	if e := canonicalDirectory(snapshot); e != nil {
		return out, e
	}
	if e := DecodeSourceReport(bytes.NewReader(reportRaw), &out.Report); e != nil {
		return out, e
	}
	if e := DecodeSourceMap(bytes.NewReader(mapRaw), &out.Mapping); e != nil {
		return out, e
	}
	return loadSourcesFromDecoded(ctx, snapshot, reportRaw, out)
}
