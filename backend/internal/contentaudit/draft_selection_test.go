package contentaudit

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/yyl1212/math_master/backend/internal/content"
)

// Catches reading a guessed/default file instead of the explicitly selected version.
func selectedFixture(t *testing.T) (string, DraftSelection) {
	t.Helper()
	repo, err := filepath.Abs("../../..")
	if err != nil {
		t.Fatal(err)
	}
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	s := DraftSelection{CataloguePath: "content/catalogue/domains.json", ContentPath: "content/packages/elementary-foundations.v1.json", AssetsRoot: "content/assets", QuestionPaths: []string{}}
	for _, theme := range []string{"numbers", "operations", "fractions", "decimals", "ratios"} {
		s.QuestionPaths = append(s.QuestionPaths, "content/questions/elementary-foundations-"+theme+".v1.json")
	}
	paths := append([]string{s.CataloguePath, s.ContentPath}, s.QuestionPaths...)
	raw, err := os.ReadFile(filepath.Join(repo, s.ContentPath))
	if err != nil {
		t.Fatal(err)
	}
	var p content.Package
	if err = json.Unmarshal(raw, &p); err != nil {
		t.Fatal(err)
	}
	for _, a := range p.Assets {
		paths = append(paths, filepath.Join(s.AssetsRoot, a.Path))
	}
	for _, path := range paths {
		b, e := os.ReadFile(filepath.Join(repo, path))
		if e != nil {
			t.Fatal(e)
		}
		selectedWrite(t, filepath.Join(root, path), b)
	}
	return root, s
}
func selectedWrite(t *testing.T, path string, b []byte) {
	t.Helper()
	if e := os.MkdirAll(filepath.Dir(path), 0700); e != nil {
		t.Fatal(e)
	}
	if e := os.WriteFile(path, b, 0600); e != nil {
		t.Fatal(e)
	}
}

func TestSelectedDraftMixedVersions(t *testing.T) {
	root, s := selectedFixture(t)
	raw, e := os.ReadFile(filepath.Join(root, s.QuestionPaths[0]))
	if e != nil {
		t.Fatal(e)
	}
	var p map[string]any
	if e = json.Unmarshal(raw, &p); e != nil {
		t.Fatal(e)
	}
	p["version"] = 2
	raw, e = json.Marshal(p)
	if e != nil {
		t.Fatal(e)
	}
	s.QuestionPaths[0] = strings.Replace(s.QuestionPaths[0], ".v1.json", ".v2.json", 1)
	selectedWrite(t, filepath.Join(root, s.QuestionPaths[0]), raw)
	got, e := LoadSelectedDraft(context.Background(), root, s)
	if e != nil {
		t.Fatal(e)
	}
	if len(got.Input.Questions) != 5 || len(got.Files) != 16 {
		t.Fatalf("all selected inputs required: questions=%d files=%d", len(got.Input.Questions), len(got.Files))
	}
	if got.Input.Questions[0].Version != 2 || got.Input.Questions[1].Version != 1 {
		t.Fatal("mixed versions were guessed")
	}
	if !bytes.Equal(got.Files[2].Bytes, raw) || got.Files[2].Path != s.QuestionPaths[0] {
		t.Fatal("digest/export capture differs from selected version")
	}
	if _, e = CheckSelectedDraft(context.Background(), got); e != nil {
		t.Fatal(e)
	}
	legacy, e := LoadDraft(context.Background(), root)
	if e != nil || legacy.Questions[0].Version != 1 {
		t.Fatal("legacy default changed", e)
	}
}

// Catches reopening SVGs after their digest/identity has already been captured.
func TestSelectedDraftCapturedAssets(t *testing.T) {
	root, s := selectedFixture(t)
	got, e := LoadSelectedDraft(context.Background(), root, s)
	if e != nil {
		t.Fatal(e)
	}
	if len(got.Input.Content.Assets) == 0 {
		t.Fatal("captured content missing")
	}
	a := got.Input.Content.Assets[0]
	original := append([]byte(nil), got.Assets[a.ID]...)
	selectedWrite(t, filepath.Join(root, s.AssetsRoot, a.Path), append(append([]byte(nil), original...), '\n'))
	if _, e = CheckSelectedDraft(context.Background(), got); e != nil {
		t.Fatal("captured asset was reopened", e)
	}
	if !bytes.Equal(got.Assets[a.ID], original) {
		t.Fatal("captured SVG changed")
	}
	if _, e = CheckDraft(context.Background(), got.Input); e == nil {
		t.Fatal("changed on-disk SVG should fail original digest check")
	}
	got.Assets[a.ID] = []byte("<svg>wrong</svg>")
	if _, e = CheckSelectedDraft(context.Background(), got); e == nil {
		t.Fatal("modified captured asset accepted")
	}
}
func TestSelectedDraftCapturedValues(t *testing.T) {
	root, s := selectedFixture(t)
	got, e := LoadSelectedDraft(context.Background(), root, s)
	if e != nil {
		t.Fatal(e)
	}
	if len(got.Input.Content.Knowledge) == 0 {
		t.Fatal("captured knowledge missing")
	}
	got.Input.Content.Knowledge[0].Statement += " changed after capture"
	if _, e = CheckSelectedDraft(context.Background(), got); e == nil {
		t.Fatal("typed body differs from captured bytes")
	}
}

func TestSelectedDraftUnsafeInputs(t *testing.T) {
	cases := []struct {
		name   string
		mutate func(*testing.T, *string, *DraftSelection)
	}{
		{"absolute", func(t *testing.T, r *string, s *DraftSelection) { s.CataloguePath = filepath.Join(*r, s.CataloguePath) }},
		{"traversal", func(t *testing.T, r *string, s *DraftSelection) { s.ContentPath = "../outside.json" }},
		{"dot", func(t *testing.T, r *string, s *DraftSelection) { s.ContentPath = "./" + s.ContentPath }},
		{"duplicate-path", func(t *testing.T, r *string, s *DraftSelection) { s.QuestionPaths[1] = s.QuestionPaths[0] }},
		{"duplicate-id", func(t *testing.T, r *string, s *DraftSelection) {
			b, _ := os.ReadFile(filepath.Join(*r, s.QuestionPaths[0]))
			selectedWrite(t, filepath.Join(*r, s.QuestionPaths[1]), b)
		}},
		{"four-packages", func(t *testing.T, r *string, s *DraftSelection) { s.QuestionPaths = s.QuestionPaths[:4] }},
		{"six-packages", func(t *testing.T, r *string, s *DraftSelection) {
			s.QuestionPaths = append(s.QuestionPaths, s.QuestionPaths[0])
		}},
		{"missing", func(t *testing.T, r *string, s *DraftSelection) {
			s.QuestionPaths[0] = "content/questions/missing.json"
		}},
		{"directory", func(t *testing.T, r *string, s *DraftSelection) { s.ContentPath = "content/packages" }},
		{"root-symlink", func(t *testing.T, r *string, s *DraftSelection) {
			link := filepath.Join(filepath.Dir(*r), filepath.Base(*r)+"-link")
			if e := os.Symlink(*r, link); e != nil {
				t.Fatal(e)
			}
			t.Cleanup(func() { os.Remove(link) })
			*r = link
		}},
		{"middle-symlink", func(t *testing.T, r *string, s *DraftSelection) {
			old := filepath.Join(*r, "content")
			if e := os.Rename(old, old+"-real"); e != nil {
				t.Fatal(e)
			}
			if e := os.Symlink(old+"-real", old); e != nil {
				t.Fatal(e)
			}
		}},
		{"file-symlink", func(t *testing.T, r *string, s *DraftSelection) {
			old := filepath.Join(*r, s.QuestionPaths[0])
			if e := os.Rename(old, old+".real"); e != nil {
				t.Fatal(e)
			}
			if e := os.Symlink(old+".real", old); e != nil {
				t.Fatal(e)
			}
		}},
		{"assets-directory-symlink", func(t *testing.T, r *string, s *DraftSelection) {
			old := filepath.Join(*r, s.AssetsRoot)
			if e := os.Rename(old, old+"-real"); e != nil {
				t.Fatal(e)
			}
			if e := os.Symlink(old+"-real", old); e != nil {
				t.Fatal(e)
			}
		}},
		{"fifo", func(t *testing.T, r *string, s *DraftSelection) {
			p := filepath.Join(*r, s.QuestionPaths[0])
			if e := os.Remove(p); e != nil {
				t.Fatal(e)
			}
			if b, e := exec.Command("mkfifo", p).CombinedOutput(); e != nil {
				t.Fatalf("mkfifo %s: %v", b, e)
			}
		}},
		{"duplicate-json", func(t *testing.T, r *string, s *DraftSelection) {
			p := filepath.Join(*r, s.QuestionPaths[0])
			b, _ := os.ReadFile(p)
			b = append([]byte(`{"version":1,`), b[1:]...)
			selectedWrite(t, p, b)
		}},
		{"bad-utf8", func(t *testing.T, r *string, s *DraftSelection) {
			selectedWrite(t, filepath.Join(*r, s.ContentPath), []byte{0xff})
		}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			r, s := selectedFixture(t)
			c.mutate(t, &r, &s)
			if _, e := LoadSelectedDraft(context.Background(), r, s); e == nil {
				t.Fatal("unsafe selection accepted")
			}
		})
	}
}

func TestSelectedDraftReadBoundaries(t *testing.T) {
	root, e := filepath.EvalSymlinks(t.TempDir())
	if e != nil {
		t.Fatal(e)
	}
	selectedWrite(t, filepath.Join(root, "file"), []byte("abc"))
	b, e := ReadFileUnder(context.Background(), root, "file", 3)
	if e != nil || string(b) != "abc" {
		t.Fatal("legal limit refused", e)
	}
	selectedWrite(t, filepath.Join(root, "file"), []byte("abcd"))
	if _, e = ReadFileUnder(context.Background(), root, "file", 3); !errors.Is(e, ErrLimit) {
		t.Fatal("size+1 accepted", e)
	}
	if _, e = ReadFileUnder(context.Background(), "/dev", "null", 3); e == nil {
		t.Fatal("device accepted")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, e = ReadFileUnder(ctx, root, "file", 8); !errors.Is(e, context.Canceled) {
		t.Fatal("canceled read continued", e)
	}
}
func TestCapturedSourcesSameBytes(t *testing.T) {
	root, rp, mp := sourceFixture(t)
	root, e := filepath.EvalSymlinks(root)
	if e != nil {
		t.Fatal(e)
	}
	rb, e := os.ReadFile(rp)
	if e != nil {
		t.Fatal(e)
	}
	mb, e := os.ReadFile(mp)
	if e != nil {
		t.Fatal(e)
	}
	want, e := LoadSources(context.Background(), root, rp, mp)
	if e != nil {
		t.Fatal(e)
	}
	selectedWrite(t, rp, []byte("modified"))
	selectedWrite(t, mp, []byte("modified"))
	got, e := LoadSourcesFromBytes(context.Background(), root, rb, mb)
	if e != nil {
		t.Fatal(e)
	}
	if !reflect.DeepEqual(got, want) || got.ReportSHA != hashBytes(rb) {
		t.Fatal("source parse/hash reread another file")
	}
	bad := append(append([]byte(nil), rb...), byte(' '))
	if _, e = LoadSourcesFromBytes(context.Background(), root, bad, mb); e == nil {
		t.Fatal("raw report digest changed without mapping change")
	}
	selectedWrite(t, filepath.Join(root, "files/main.json"), []byte(`{"dataset_id":"set","knowledge_points":[{"id":"wrong"}]}`))
	if _, e = LoadSourcesFromBytes(context.Background(), root, rb, mb); e == nil {
		t.Fatal("changed source accepted")
	}
}
