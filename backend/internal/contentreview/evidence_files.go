package contentreview

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/yyl1212/math_master/backend/internal/content"
	"github.com/yyl1212/math_master/backend/internal/contentaudit"
	"github.com/yyl1212/math_master/backend/internal/publication"
)

var learningCheckNames = []string{"reading", "pass", "fail", "prerequisites", "review", "practice_exposure", "retake", "feedback_correction"}

const MaxLearningAttachments = 64

func validateRelease(r *evidenceReader, m ReviewManifest, rawManifestSHA string, release ReleaseContext) (bool, error) {
	if release.SchemaVersion != 1 || release.CodeSHA != m.CodeSHA || release.CatalogueVersion != m.CatalogueVersion || release.CatalogueSHA256 != m.CatalogueSHA256 || release.Route != m.Route || release.ManifestSHA256 != rawManifestSHA || release.FixtureOnly != m.FixtureOnly {
		return false, ErrInvalid
	}
	ready := true
	for _, head := range []contentaudit.Head{release.KnowledgeHead, release.QuestionHead} {
		if head.ID == "" && head.SHA256 == "" {
			ready = false
			continue
		}
		if !publication.ValidID(head.ID) || !publication.ValidSHA(head.SHA256) {
			return false, ErrInvalid
		}
	}
	build, e := r.signed(release.BuildRecord)
	if e != nil {
		return false, e
	}
	signed, e := r.signed(release.Attestation)
	if e != nil {
		return false, e
	}
	if !build || !signed || strings.TrimSpace(release.AttestedBy) == "" {
		ready = false
	}
	return ready, nil
}

type learningResult struct {
	ready  bool
	failed bool
	checks []contentaudit.LearningCheck
}

func verifyLearning(r *evidenceReader, release ReleaseContext, files []LearningFile) (learningResult, error) {
	out := learningResult{ready: true, checks: []contentaudit.LearningCheck{}}
	if len(files) > len(learningCheckNames) {
		return out, ErrInvalid
	}
	allowed := map[string]bool{}
	for _, name := range learningCheckNames {
		allowed[name] = true
	}
	seen := map[string]contentaudit.LearningCheck{}
	attachments := 0
	for _, file := range files {
		if !allowed[file.Name] {
			return out, ErrInvalid
		}
		if _, ok := seen[file.Name]; ok {
			return out, ErrInvalid
		}
		var record LearningRecord
		if e := r.decode(file.File, MaxFileBytes, &record); e != nil {
			return out, e
		}
		if record.SchemaVersion != 1 || record.Name != file.Name || !(record.Result == "passed" || record.Result == "failed" || record.Result == "not_run") || content.Digest(record.Context) != content.Digest(release.ReleaseIdentity) {
			return out, ErrInvalid
		}
		attachments += len(record.Attachments)
		if attachments > MaxLearningAttachments {
			return out, ErrLimit
		}
		paths := map[string]bool{}
		for _, ref := range record.Attachments {
			if paths[ref.Path] {
				return out, ErrInvalid
			}
			paths[ref.Path] = true
			if _, e := r.read(ref, MaxFileBytes); e != nil {
				return out, e
			}
		}
		signed, e := r.signed(record.Attestation)
		if e != nil {
			return out, e
		}
		if record.ExecutedAt != "" {
			if _, e = time.Parse(time.RFC3339, record.ExecutedAt); e != nil {
				return out, ErrInvalid
			}
		}
		if record.Result == "not_run" || record.ExecutedAt == "" || len(record.Steps) == 0 || strings.TrimSpace(record.AttestedBy) == "" || !signed {
			out.ready = false
		}
		for _, step := range record.Steps {
			if strings.TrimSpace(step.Action) == "" || strings.TrimSpace(step.Expected) == "" || strings.TrimSpace(step.Observed) == "" {
				out.ready = false
			}
		}
		if record.Result == "failed" {
			out.failed = true
		}
		seen[file.Name] = contentaudit.LearningCheck{Name: file.Name, Result: record.Result, EvidenceSHA256: file.File.SHA256}
	}
	if len(seen) != len(learningCheckNames) {
		out.ready = false
	}
	for _, name := range learningCheckNames {
		if check, ok := seen[name]; ok {
			out.checks = append(out.checks, check)
		}
	}
	return out, nil
}
func verificationFiles(ctx context.Context, v Verification, r *evidenceReader) ([]ExportFile, error) {
	if e := ctx.Err(); e != nil {
		return nil, e
	}
	files := []ExportFile{}
	if v.Evidence != nil {
		f, e := encodeFile("acceptance-evidence.json", v.Evidence)
		if e != nil {
			return nil, e
		}
		files = append(files, f)
	}
	type checked struct {
		Ordinal int    `json:"ordinal"`
		Bytes   int    `json:"bytes"`
		SHA256  string `json:"sha256"`
	}
	paths := []string{}
	for p := range r.cache {
		paths = append(paths, p)
	}
	sort.Strings(paths)
	hashes := []checked{}
	for i, p := range paths {
		hashes = append(hashes, checked{i + 1, len(r.cache[p]), digestBytes(r.cache[p])})
	}
	f, e := encodeFile("checked-files.json", hashes)
	if e != nil {
		return nil, e
	}
	files = append(files, f)
	body := fmt.Sprintf("# 离线证据核对\n\n状态：%s\n\nfixtureOnly：%t\n\n复核清单 SHA256：%s\n\n读取文件：%d；合计字节：%d。\n\n本报告只核对文件一致性，不能证明自然人独立性，也不表示正式 accepted。负责人仍须核验构建、人员和实际执行材料；最后由既有 content-audit 重新核对数据库当前资格与双 head。\n", v.Conclusion, v.FixtureOnly, v.ManifestSHA256, len(paths), r.total)
	files = append(files, ExportFile{"verification.md", []byte(body)})
	return files, nil
}
