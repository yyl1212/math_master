package cli

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"syscall"
	"time"
)

var backupHash = regexp.MustCompile(`^[a-f0-9]{64}$`)

func privateBackupPath(path string, mode os.FileMode, directory bool) bool {
	info, e := os.Lstat(path)
	if e != nil || info.Mode()&os.ModeSymlink != 0 || info.IsDir() != directory || info.Mode().Perm() != mode {
		return false
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	return ok && int(stat.Uid) == os.Geteuid()
}
func backupDigest(ctx context.Context, path string) (string, error) {
	f, e := os.Open(path)
	if e != nil {
		return "", e
	}
	defer f.Close()
	hash := sha256.New()
	buffer := make([]byte, 65536)
	for {
		if e = ctx.Err(); e != nil {
			return "", e
		}
		n, e := f.Read(buffer)
		if n > 0 {
			hash.Write(buffer[:n])
		}
		if e == io.EOF {
			break
		}
		if e != nil {
			return "", e
		}
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}
func validateTopicBackup(ctx context.Context, path, database string) (string, error) {
	invalid := errors.New("invalid topic backup record")
	absolute, e := filepath.Abs(path)
	if e != nil {
		return "", invalid
	}
	resolved, e := filepath.EvalSymlinks(absolute)
	if e != nil || resolved != absolute || !privateBackupPath(absolute, 0700, true) {
		return "", invalid
	}
	for _, name := range []string{"database.dump", "manifest.json", "SHA256SUMS"} {
		if !privateBackupPath(filepath.Join(absolute, name), 0600, false) {
			return "", invalid
		}
	}
	manifestFile := filepath.Join(absolute, "manifest.json")
	info, e := os.Stat(manifestFile)
	if e != nil || info.Size() > 2<<20 {
		return "", invalid
	}
	raw, e := os.ReadFile(manifestFile)
	if e != nil {
		return "", invalid
	}
	var manifest struct {
		SchemaVersion    int       `json:"schemaVersion"`
		SourceCommit     string    `json:"sourceCommit"`
		CreatedAt        time.Time `json:"createdAt"`
		MigrationVersion int       `json:"migrationVersion"`
		DumpSHA          string    `json:"dumpSha256"`
		Database         struct {
			Name string `json:"name"`
		} `json:"database"`
		Tables map[string]json.RawMessage `json:"tables"`
	}
	if json.Unmarshal(raw, &manifest) != nil || manifest.SchemaVersion != 1 || !regexp.MustCompile(`^[a-f0-9]{40}$`).MatchString(manifest.SourceCommit) || manifest.CreatedAt.IsZero() || manifest.CreatedAt.After(time.Now().Add(time.Minute)) || manifest.MigrationVersion < 1 || manifest.MigrationVersion > 12 || manifest.Database.Name != database || !backupHash.MatchString(manifest.DumpSHA) || manifest.Tables == nil {
		return "", invalid
	}
	file, e := os.Open(filepath.Join(absolute, "database.dump"))
	if e != nil {
		return "", invalid
	}
	magic := make([]byte, 5)
	_, e = io.ReadFull(file, magic)
	file.Close()
	if e != nil || string(magic) != "PGDMP" {
		return "", invalid
	}
	dump, e := backupDigest(ctx, filepath.Join(absolute, "database.dump"))
	if e != nil || dump != manifest.DumpSHA {
		return "", invalid
	}
	digest := sha256.Sum256(raw)
	record := hex.EncodeToString(digest[:])
	sumsFile := filepath.Join(absolute, "SHA256SUMS")
	sumInfo, e := os.Stat(sumsFile)
	if e != nil || sumInfo.Size() > 1024 {
		return "", invalid
	}
	sums, e := os.ReadFile(sumsFile)
	if e != nil || strings.TrimSpace(string(sums)) != dump+"  database.dump\n"+record+"  manifest.json" {
		return "", invalid
	}
	return record, nil
}
