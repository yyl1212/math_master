package store_test

import (
	"bytes"
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"syscall"
	"testing"
	"time"

	"github.com/yyl1212/math_master/backend/internal/auth"
	"github.com/yyl1212/math_master/backend/internal/testutil"
)

const cutoverCapacityReceiptEnv = "TOPIC_CUTOVER_CAPACITY_RECEIPT"

var capacityDatabaseName = regexp.MustCompile(`^math_master_test_[0-9a-f]{16}$`)
var capacitySourceHash = regexp.MustCompile(`^[0-9a-f]{64}$`)

type capacityReceipt struct {
	Version   int    `json:"version"`
	Database  string `json:"database"`
	Nonce     string `json:"nonce"`
	SourceSHA string `json:"sourceSha"`
}

func readCapacityReceipt(path string) (capacityReceipt, error) {
	var out capacityReceipt
	info, e := os.Lstat(path)
	if e != nil || !info.Mode().IsRegular() || info.Mode().Perm() != 0600 || info.Size() > 2048 {
		return out, errors.New("invalid private capacity receipt")
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok || int(stat.Uid) != os.Getuid() {
		return out, errors.New("invalid capacity receipt owner")
	}
	raw, e := os.ReadFile(path)
	if e != nil {
		return out, e
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if e = decoder.Decode(&out); e != nil {
		return out, errors.New("invalid capacity receipt")
	}
	var trailing any
	if decoder.Decode(&trailing) != io.EOF || out.Version != 1 || !capacityDatabaseName.MatchString(out.Database) || !capacitySourceHash.MatchString(out.SourceSHA) {
		return out, errors.New("invalid capacity receipt")
	}
	// Canonical bytes also reject duplicate keys and case aliases.
	canonical, _ := json.Marshal(out)
	if !bytes.Equal(raw, canonical) {
		return out, errors.New("noncanonical capacity receipt")
	}
	if !regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`).MatchString(out.Nonce) {
		return out, errors.New("invalid capacity nonce")
	}
	return out, nil
}
func capacitySourceDigest(t *testing.T, db *sql.DB) string {
	t.Helper()
	var digest string
	if e := db.QueryRow(`SELECT encode(sha256(convert_to(COALESCE(string_agg(seal_sha256,'' ORDER BY id),''),'UTF8')),'hex') FROM learning_events`).Scan(&digest); e != nil {
		t.Fatal("capacity source digest failed")
	}
	return digest
}
func persistCapacityFixture(t *testing.T, db *sql.DB, path string) {
	t.Helper()
	if _, e := os.Lstat(path); !os.IsNotExist(e) {
		t.Fatal("capacity receipt already exists")
	}
	var source string
	if e := db.QueryRow("SELECT current_database()").Scan(&source); e != nil || !capacityDatabaseName.MatchString(source) {
		t.Fatal("invalid capacity source database")
	}
	id, e := auth.NewID(rand.Reader)
	if e != nil {
		t.Fatal("capacity identity failed")
	}
	name := "math_master_test_" + id[:8] + id[9:13] + id[14:18]
	nonce, e := auth.NewID(rand.Reader)
	if e != nil {
		t.Fatal("capacity identity failed")
	}
	receipt := capacityReceipt{Version: 1, Database: name, Nonce: nonce, SourceSHA: capacitySourceDigest(t, db)}
	if _, e = db.Exec(`CREATE TABLE topic_cutover_capacity_fixture(singleton boolean PRIMARY KEY CHECK(singleton),nonce uuid NOT NULL,source_sha text NOT NULL)`); e != nil {
		t.Fatal("capacity identity table failed")
	}
	if _, e = db.Exec(`INSERT INTO topic_cutover_capacity_fixture VALUES(true,$1,$2)`, nonce, receipt.SourceSHA); e != nil {
		t.Fatal("capacity identity failed")
	}
	adminConfig, _, e := testutil.IsolatedConfigs(os.Getenv("TEST_DATABASE_URL"), name)
	if e != nil {
		t.Fatal("unsafe capacity database configuration")
	}
	admin := testutil.OpenVerified(adminConfig)
	defer admin.Close()
	if e = db.Close(); e != nil {
		t.Fatal("capacity source close failed")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	if _, e = admin.ExecContext(ctx, `CREATE DATABASE "`+name+`" TEMPLATE "`+source+`"`); e != nil {
		t.Fatal("capacity database clone failed")
	}
	handedOff := false
	defer func() {
		if !handedOff {
			_, _ = admin.ExecContext(context.Background(), `DROP DATABASE "`+name+`" WITH (FORCE)`)
		}
	}()
	raw, _ := json.Marshal(receipt)
	file, e := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if e != nil {
		t.Fatal("capacity receipt create failed")
	}
	_, writeError := file.Write(raw)
	closeError := file.Close()
	if writeError != nil || closeError != nil {
		_ = os.Remove(path)
		t.Fatal("capacity receipt write failed")
	}
	handedOff = true
}
func openCapacityFixture(t *testing.T, path string) (*sql.DB, capacityReceipt) {
	t.Helper()
	receipt, e := readCapacityReceipt(path)
	if e != nil {
		t.Fatal("capacity receipt unavailable")
	}
	_, work, e := testutil.IsolatedConfigs(os.Getenv("TEST_DATABASE_URL"), receipt.Database)
	if e != nil {
		t.Fatal("unsafe capacity database configuration")
	}
	db := testutil.OpenVerified(work)
	t.Cleanup(func() { db.Close() })
	if e := verifyCapacityIdentity(context.Background(), db, receipt); e != nil {
		t.Fatal("capacity database identity mismatch")
	}
	return db, receipt
}
func verifyCapacityIdentity(ctx context.Context, db *sql.DB, receipt capacityReceipt) error {
	var nonce, digest string
	if e := db.QueryRowContext(ctx, `SELECT nonce::text,source_sha FROM topic_cutover_capacity_fixture WHERE singleton`).Scan(&nonce, &digest); e != nil || nonce != receipt.Nonce || digest != receipt.SourceSHA {
		return errors.New("capacity database identity mismatch")
	}
	return nil
}
func cleanupCapacityFixture(t *testing.T, path string) {
	t.Helper()
	if _, e := os.Lstat(path); os.IsNotExist(e) {
		return
	}
	db, receipt := openCapacityFixture(t, path)
	adminConfig, _, e := testutil.IsolatedConfigs(os.Getenv("TEST_DATABASE_URL"), receipt.Database)
	if e != nil {
		t.Fatal("unsafe capacity cleanup")
	}
	admin := testutil.OpenVerified(adminConfig)
	defer admin.Close()
	db.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if _, e = admin.ExecContext(ctx, `DROP DATABASE "`+receipt.Database+`" WITH (FORCE)`); e != nil {
		t.Fatal("capacity cleanup failed")
	}
	if e = os.Remove(path); e != nil {
		t.Fatal("capacity receipt cleanup failed")
	}
}
func TestTopicCutoverCapacityCleanup(t *testing.T) {
	path := os.Getenv(cutoverCapacityReceiptEnv)
	if path == "" {
		t.Fatal("capacity receipt path required")
	}
	cleanupCapacityFixture(t, path)
}
func TestTopicCutoverReceiptPrivateAndExact(t *testing.T) {
	path := filepath.Join(t.TempDir(), "receipt.json")
	receipt := capacityReceipt{Version: 1, Database: "math_master_test_0123456789abcdef", Nonce: "11111111-1111-4111-8111-111111111111", SourceSHA: string(bytes.Repeat([]byte("a"), 64))}
	raw, _ := json.Marshal(receipt)
	if e := os.WriteFile(path, raw, 0600); e != nil {
		t.Fatal(e)
	}
	if _, e := readCapacityReceipt(path); e != nil {
		t.Fatal(e)
	}
	if e := os.Chmod(path, 0644); e != nil {
		t.Fatal(e)
	}
	if _, e := readCapacityReceipt(path); e == nil {
		t.Fatal("public receipt accepted")
	}
	os.Chmod(path, 0600)
	for _, bad := range []string{string(raw[:len(raw)-1]) + `,"database":"math_master_test_0123456789abcdef"}`, string(raw) + `{}`, `{"version":1,"database":"production","nonce":"11111111-1111-4111-8111-111111111111","sourceSha":"` + receipt.SourceSHA + `"}`} {
		os.WriteFile(path, []byte(bad), 0600)
		if _, e := readCapacityReceipt(path); e == nil {
			t.Fatal("invalid receipt accepted")
		}
	}
	os.WriteFile(path, raw, 0600)
	link := path + ".link"
	os.Symlink(path, link)
	if _, e := readCapacityReceipt(link); e == nil {
		t.Fatal("linked receipt accepted")
	}
}

func TestTopicCutoverStageIdentityRejectsOtherFixture(t *testing.T) {
	db := testutil.Database(t)
	if _, e := db.Exec(`CREATE TABLE topic_cutover_capacity_fixture(singleton boolean PRIMARY KEY,nonce uuid,source_sha text)`); e != nil {
		t.Fatal(e)
	}
	nonce := "11111111-1111-4111-8111-111111111111"
	digest := string(bytes.Repeat([]byte("a"), 64))
	if _, e := db.Exec(`INSERT INTO topic_cutover_capacity_fixture VALUES(true,$1,$2)`, nonce, digest); e != nil {
		t.Fatal(e)
	}
	receipt := capacityReceipt{Nonce: nonce, SourceSHA: digest}
	if e := verifyCapacityIdentity(context.Background(), db, receipt); e != nil {
		t.Fatal(e)
	}
	receipt.Nonce = "22222222-2222-4222-8222-222222222222"
	if e := verifyCapacityIdentity(context.Background(), db, receipt); e == nil {
		t.Fatal("another fixture nonce accepted")
	}
	receipt.Nonce = nonce
	receipt.SourceSHA = string(bytes.Repeat([]byte("b"), 64))
	if e := verifyCapacityIdentity(context.Background(), db, receipt); e == nil {
		t.Fatal("another source digest accepted")
	}
	var rows int
	if e := db.QueryRow(`SELECT count(*) FROM topic_cutover_capacity_fixture`).Scan(&rows); e != nil || rows != 1 {
		t.Fatal("identity rejection changed fixture")
	}
}
