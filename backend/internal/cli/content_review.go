package cli

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/yyl1212/math_master/backend/internal/content"
	"github.com/yyl1212/math_master/backend/internal/contentaudit"
	"github.com/yyl1212/math_master/backend/internal/contentreview"
	"github.com/yyl1212/math_master/backend/internal/publication"
	"github.com/yyl1212/math_master/backend/internal/question"
)

const ContentReviewMaxCommandDuration = 120 * time.Second

// RunContentReview performs bounded offline file operations; it never opens a database or publishes.
func RunContentReview(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	ctx, cancel := context.WithTimeout(ctx, ContentReviewMaxCommandDuration)
	defer cancel()
	fail := func(code string, status int) int { fmt.Fprintln(stderr, code); return status }
	classify := func(e error) int {
		if errors.Is(e, context.Canceled) || errors.Is(e, context.DeadlineExceeded) || errors.Is(e, contentreview.ErrLimit) || errors.Is(e, contentaudit.ErrLimit) || errors.Is(e, question.ErrLimitExceeded) || errors.Is(e, content.ErrLimit) || errors.Is(e, publication.ErrContentLimitExceeded) {
			return fail("REVIEW_BUDGET_FAILURE", 1)
		}
		if errors.Is(e, contentreview.ErrInvalid) || errors.Is(e, contentaudit.ErrInvalid) || errors.Is(e, question.ErrInvalid) || errors.Is(e, content.ErrValidation) || errors.Is(e, publication.ErrContentInvalid) {
			return fail("REVIEW_INVALID", 2)
		}
		return fail("REVIEW_IO_FAILURE", 1)
	}
	if e := ctx.Err(); e != nil {
		return classify(e)
	}
	if len(args) == 0 || (args[0] != "prepare" && args[0] != "verify-evidence") {
		return fail("INVALID_ARGUMENT", 2)
	}
	command := args[0]
	flags := flag.NewFlagSet("content-review", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	var root, inputManifest, snapshot, sourceReport, sourceMap, route, codeSHA, out, evidenceRoot, reviewManifest, input string
	var version int
	var fixture bool
	flags.StringVar(&out, "out", "", "")
	if command == "prepare" {
		for name, target := range map[string]*string{"root": &root, "input-manifest": &inputManifest, "snapshot": &snapshot, "source-report": &sourceReport, "source-map": &sourceMap, "route": &route, "code-sha": &codeSHA} {
			flags.StringVar(target, name, "", "")
		}
		flags.IntVar(&version, "version", 0, "")
		flags.BoolVar(&fixture, "fixture-only", false, "")
	} else {
		flags.StringVar(&evidenceRoot, "evidence-root", "", "")
		flags.StringVar(&reviewManifest, "review-manifest", "", "")
		flags.StringVar(&input, "input", "", "")
	}
	seen := map[string]bool{}
	for _, a := range args[1:] {
		if strings.HasPrefix(a, "-") {
			name := strings.SplitN(strings.TrimLeft(a, "-"), "=", 2)[0]
			if seen[name] {
				return fail("INVALID_ARGUMENT", 2)
			}
			seen[name] = true
		}
	}
	if flags.Parse(args[1:]) != nil || flags.NArg() != 0 || !reviewAbsolute(out) {
		return fail("INVALID_ARGUMENT", 2)
	}
	if command == "prepare" {
		if !reviewAbsolute(root) || !reviewAbsolute(inputManifest) || !reviewAbsolute(snapshot) || !reviewAbsolute(sourceReport) || !reviewAbsolute(sourceMap) || !question.ValidMathID(route) || version < 1 || version > 2147483647 || len(codeSHA) != 40 || strings.Trim(codeSHA, "0123456789abcdef") != "" {
			return fail("INVALID_ARGUMENT", 2)
		}
	} else if !reviewAbsolute(evidenceRoot) || !reviewAbsolute(reviewManifest) || !reviewAbsolute(input) {
		return fail("INVALID_ARGUMENT", 2)
	}
	if _, e := os.Lstat(out); !errors.Is(e, os.ErrNotExist) {
		return fail("REVIEW_IO_FAILURE", 1)
	}
	parent := filepath.Dir(out)
	canonical, e := filepath.EvalSymlinks(parent)
	if e != nil || canonical != parent {
		return fail("REVIEW_IO_FAILURE", 1)
	}
	writeSummary := func(v any) int {
		if e := json.NewEncoder(stdout).Encode(v); e != nil {
			return fail("REVIEW_IO_FAILURE", 1)
		}
		return 0
	}
	if command == "prepare" {
		manifestRaw, e := readReviewAbsolute(ctx, inputManifest, contentreview.MaxManifestBytes)
		if e != nil {
			return classify(e)
		}
		var manifest contentreview.InputManifest
		if e = question.DecodeOperationalJSON(bytes.NewReader(manifestRaw), contentreview.MaxManifestBytes, &manifest); e != nil {
			return classify(e)
		}
		selected, e := contentaudit.LoadSelectedDraft(ctx, root, manifest.Selection())
		if e != nil {
			return classify(e)
		}
		reportRaw, e := readReviewAbsolute(ctx, sourceReport, contentaudit.MaxMetadataBytes)
		if e != nil {
			return classify(e)
		}
		mapRaw, e := readReviewAbsolute(ctx, sourceMap, contentaudit.MaxSourceMapBytes)
		if e != nil {
			return classify(e)
		}
		sources, e := contentaudit.LoadSourcesFromBytes(ctx, snapshot, reportRaw, mapRaw)
		if e != nil {
			return classify(e)
		}
		bundle, e := contentreview.Prepare(ctx, contentreview.PrepareInput{CodeSHA: codeSHA, Route: content.VersionRef{ID: route, Version: version}, Manifest: manifest, ManifestRaw: manifestRaw, Selected: selected, Sources: sources, SourceReportRaw: reportRaw, SourceMapRaw: mapRaw, FixtureOnly: fixture})
		if e != nil {
			return classify(e)
		}
		if e = contentreview.WriteBundle(ctx, out, bundle); e != nil {
			return classify(e)
		}
		raw, e := json.Marshal(bundle.Manifest)
		if e != nil {
			return classify(e)
		}
		return writeSummary(struct {
			Status         string `json:"status"`
			Objects        int    `json:"objects"`
			Sources        int    `json:"sources"`
			FixtureOnly    bool   `json:"fixtureOnly"`
			ManifestSHA256 string `json:"manifestSHA256"`
		}{"prepared", len(bundle.Manifest.Objects), len(bundle.Manifest.Sources), bundle.Manifest.FixtureOnly, fmt.Sprintf("%x", sha256.Sum256(raw))})
	}
	manifestRaw, e := readReviewAbsolute(ctx, reviewManifest, contentreview.MaxFileBytes)
	if e != nil {
		return classify(e)
	}
	raw, e := readReviewAbsolute(ctx, input, contentreview.MaxFileBytes)
	if e != nil {
		return classify(e)
	}
	var evidence contentreview.EvidenceInput
	if e = question.DecodeOperationalJSON(bytes.NewReader(raw), contentreview.MaxFileBytes, &evidence); e != nil {
		return classify(e)
	}
	verified, e := contentreview.VerifyEvidenceFromBytes(ctx, evidenceRoot, manifestRaw, evidence)
	if e != nil {
		return classify(e)
	}
	if e = contentreview.WriteVerification(ctx, out, verified); e != nil {
		return classify(e)
	}
	if code := writeSummary(struct {
		Status         string `json:"status"`
		FixtureOnly    bool   `json:"fixtureOnly"`
		ManifestSHA256 string `json:"manifestSHA256"`
	}{verified.Conclusion, verified.FixtureOnly, verified.ManifestSHA256}); code != 0 {
		return code
	}
	switch verified.Conclusion {
	case "evidence_ready":
		return 0
	case "awaiting_review":
		return 3
	default:
		return 2
	}
}
func reviewAbsolute(p string) bool { return filepath.IsAbs(p) && filepath.Clean(p) == p && p != "/" }
func readReviewAbsolute(ctx context.Context, p string, limit int) ([]byte, error) {
	if !reviewAbsolute(p) {
		return nil, contentreview.ErrInvalid
	}
	return contentaudit.ReadFileUnder(ctx, filepath.Dir(p), filepath.Base(p), limit)
}
