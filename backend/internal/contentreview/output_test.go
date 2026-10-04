package contentreview

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

func outputRoot(t *testing.T) string {
	t.Helper()
	p, e := filepath.EvalSymlinks(t.TempDir())
	if e != nil {
		t.Fatal(e)
	}
	return p
}
func smallBundle() Bundle {
	f := ExportFile{"data.json", []byte("safe")}
	return Bundle{Manifest: ReviewManifest{SchemaVersion: 1, Files: []FileEntry{entry(f)}}, Files: []ExportFile{f}}
}
func TestReviewOutputNoOverwrite(t *testing.T) {
	root := outputRoot(t)
	b := smallBundle()
	for _, kind := range []string{"dir", "file", "symlink", "fifo"} {
		t.Run(kind, func(t *testing.T) {
			out := filepath.Join(root, kind)
			switch kind {
			case "dir":
				fixtureWrite(t, filepath.Join(out, "old"), []byte("old"))
			case "file":
				fixtureWrite(t, out, []byte("old"))
			case "symlink":
				if e := os.Symlink(root, out); e != nil {
					t.Fatal(e)
				}
			case "fifo":
				if e := makeFIFO(out); e != nil {
					t.Fatal(e)
				}
			}
			before, e := os.Lstat(out)
			if e != nil {
				t.Fatal(e)
			}
			if e = WriteBundle(context.Background(), out, b); e == nil {
				t.Fatal("existing output overwritten")
			}
			after, e := os.Lstat(out)
			if e != nil || !os.SameFile(before, after) {
				t.Fatal("old output changed", e)
			}
		})
	}
	out := filepath.Join(root, "new")
	if e := WriteBundle(context.Background(), out, b); e != nil {
		t.Fatal(e)
	}
	if e := filepath.Walk(out, func(p string, info os.FileInfo, e error) error {
		if e != nil {
			return e
		}
		want := os.FileMode(0600)
		if info.IsDir() {
			want = 0700
		}
		if info.Mode().Perm() != want {
			t.Fatalf("mode %s=%o", p, info.Mode().Perm())
		}
		return nil
	}); e != nil {
		t.Fatal(e)
	}
	if got, e := os.ReadFile(filepath.Join(out, "data.json")); e != nil || !bytes.Equal(got, b.Files[0].Bytes) {
		t.Fatal("bytes changed")
	}
}
func TestReviewOutputFailureCleanup(t *testing.T) {
	root := outputRoot(t)
	old := filepath.Join(root, "old")
	fixtureWrite(t, old, []byte("old"))
	oldSHA := sha([]byte("old"))
	for _, stage := range []string{"write", "close", "cancel"} {
		t.Run(stage, func(t *testing.T) {
			out := filepath.Join(root, stage)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			hooks := outputHooks{Before: func(name string) error {
				if stage == "cancel" {
					cancel()
				}
				if stage == "write" {
					return errors.New("injected write failure")
				}
				return nil
			}, AfterClose: func(string) error {
				if stage == "close" {
					return errors.New("injected close failure")
				}
				return nil
			}}
			if e := writePrivate(ctx, out, []ExportFile{{"sub/part", []byte("data")}, {"review-manifest.json", []byte("final")}}, hooks); e == nil {
				t.Fatal("failure accepted")
			}
			if _, e := os.Lstat(out); !os.IsNotExist(e) {
				t.Fatal("partial output remains")
			}
			b, e := os.ReadFile(old)
			if e != nil || sha(b) != oldSHA {
				t.Fatal("unrelated file changed")
			}
		})
	}
}
func TestReviewOutputLimits(t *testing.T) {
	block := bytes.Repeat([]byte("a"), MaxFileBytes)
	boundary := []ExportFile{{"body", block}}
	if e := validateOutput(boundary); e != nil {
		t.Fatal("legal file size", e)
	}
	if e := validateOutput([]ExportFile{{"body", append(block, 'a')}}); !errors.Is(e, ErrLimit) {
		t.Fatal("file size+1", e)
	}
	many := []ExportFile{}
	for i := 0; i < MaxFiles; i++ {
		many = append(many, ExportFile{fmt.Sprintf("parts/%03d", i), []byte("a")})
	}
	if e := validateOutput(many); e != nil {
		t.Fatal(e)
	}
	many = append(many, ExportFile{"extra", []byte("a")})
	if e := validateOutput(many); !errors.Is(e, ErrLimit) {
		t.Fatal("file count+1", e)
	}
	total := []ExportFile{}
	for i := 0; i < MaxTotalBytes/MaxFileBytes; i++ {
		total = append(total, ExportFile{fmt.Sprintf("large/%02d", i), block})
	}
	if e := validateOutput(total); e != nil {
		t.Fatal("legal total", e)
	}
	total = append(total, ExportFile{"extra", []byte("a")})
	if e := validateOutput(total); !errors.Is(e, ErrLimit) {
		t.Fatal("total+1", e)
	}
	for _, files := range [][]ExportFile{{{"../escape", []byte("a")}}, {{"/escape", []byte("a")}}, {{"same", []byte("a")}, {"same", []byte("b")}}} {
		if validateOutput(files) == nil {
			t.Fatal("unsafe path accepted")
		}
	}
	root := outputRoot(t)
	link := filepath.Join(root, "link")
	if e := os.Symlink(root, link); e != nil {
		t.Fatal(e)
	}
	if e := writePrivate(context.Background(), filepath.Join(link, "escape"), boundary, outputHooks{}); e == nil {
		t.Fatal("symlink parent accepted")
	}
	start := time.Now()
	if e := writePrivate(context.Background(), filepath.Join(root, "maximum-file"), boundary, outputHooks{}); e != nil {
		t.Fatal(e)
	}
	t.Logf("maximum legal file bytes=%d elapsed=%s", MaxFileBytes, time.Since(start))
	b := smallBundle()
	b.Files[0].Bytes = []byte("wrong")
	if e := WriteBundle(context.Background(), filepath.Join(root, "mismatch"), b); e == nil {
		t.Fatal("hash mismatch accepted")
	}
}

func makeFIFO(p string) error { return exec.Command("mkfifo", p).Run() }
