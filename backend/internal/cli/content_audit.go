package cli

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"flag"
	"fmt"
	"github.com/yyl1212/math_master/backend/internal/content"
	"github.com/yyl1212/math_master/backend/internal/contentaudit"
	"github.com/yyl1212/math_master/backend/internal/question"
	"github.com/yyl1212/math_master/backend/internal/store"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// RunContentAudit is a local operator command; it does not publish or expose an HTTP API.
func RunContentAudit(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	ctx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()
	fail := func(code string, status int) int { fmt.Fprintln(stderr, code); return status }
	flags := flag.NewFlagSet("content-audit", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	var mode, route, snapshot, sourceReport, sourceMap, codeSHA, out, root, databaseEnv, evidenceFile string
	var version int
	var fixture bool
	for name, target := range map[string]*string{"mode": &mode, "route": &route, "snapshot": &snapshot, "source-report": &sourceReport, "source-map": &sourceMap, "code-sha": &codeSHA, "out": &out, "root": &root, "database-env": &databaseEnv, "evidence": &evidenceFile} {
		flags.StringVar(target, name, "", "")
	}
	flags.IntVar(&version, "version", 0, "")
	flags.BoolVar(&fixture, "fixture-only", false, "")
	seen := map[string]bool{}
	for _, a := range args {
		if strings.HasPrefix(a, "-") {
			name := strings.SplitN(strings.TrimLeft(a, "-"), "=", 2)[0]
			if seen[name] {
				return fail("INVALID_ARGUMENT", 2)
			}
			seen[name] = true
		}
	}
	if flags.Parse(args) != nil || flags.NArg() != 0 || (mode != "draft" && mode != "published") || !question.ValidMathID(route) || version < 1 || version > 2147483647 || len(codeSHA) != 40 || strings.Trim(codeSHA, "0123456789abcdef") != "" || snapshot == "" || sourceReport == "" || sourceMap == "" || out == "" || (mode == "draft" && (root == "" || databaseEnv != "" || evidenceFile != "")) || (mode == "published" && (root != "" || !regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`).MatchString(databaseEnv))) {
		return fail("INVALID_ARGUMENT", 2)
	}
	if _, e := os.Lstat(out); !errors.Is(e, os.ErrNotExist) {
		return fail("OUTPUT_IO", 1)
	}
	parent, e := filepath.Abs(filepath.Dir(out))
	if e != nil {
		return fail("OUTPUT_IO", 1)
	}
	resolved, e := filepath.EvalSymlinks(parent)
	if e != nil || resolved != parent {
		return fail("OUTPUT_IO", 1)
	}
	classify := func(e error) int {
		if errors.Is(e, context.Canceled) || errors.Is(e, context.DeadlineExceeded) || errors.Is(e, contentaudit.ErrLimit) || errors.Is(e, question.ErrLimitExceeded) || errors.Is(e, content.ErrLimit) {
			return fail("AUDIT_BUDGET_FAILURE", 1)
		}
		if errors.Is(e, contentaudit.ErrInvalid) || errors.Is(e, question.ErrInvalid) || errors.Is(e, store.ErrNotFound) {
			return fail("AUDIT_INVALID", 2)
		}
		return fail("AUDIT_INPUT_OR_DATABASE_FAILURE", 1)
	}
	sources, e := contentaudit.LoadSources(ctx, snapshot, sourceReport, sourceMap)
	if e != nil {
		return classify(e)
	}
	req := contentaudit.Request{Mode: contentaudit.Mode(mode), Route: content.VersionRef{ID: route, Version: version}, FixtureOnly: fixture, CodeSHA: codeSHA, At: time.Now().UTC()}
	var report contentaudit.Report
	if mode == "draft" {
		input, e := contentaudit.LoadDraft(ctx, root)
		if e != nil {
			return classify(e)
		}
		facts, e := contentaudit.CheckDraft(ctx, input)
		if e != nil {
			return classify(e)
		}
		report, e = contentaudit.EvaluateDraft(ctx, req, facts, sources)
		if e != nil {
			return classify(e)
		}
	} else {
		raw := os.Getenv(databaseEnv)
		u, e := url.Parse(raw)
		if e != nil || raw == "" || (u.Scheme != "postgres" && u.Scheme != "postgresql") || u.Hostname() == "" || u.Path == "" || u.Path == "/" {
			return fail("INVALID_CONFIG", 1)
		}
		db, e := sql.Open("pgx", raw)
		if e != nil {
			return fail("INVALID_CONFIG", 1)
		}
		defer db.Close()
		db.SetMaxOpenConns(2)
		db.SetMaxIdleConns(2)
		facts, e := store.New(db).ReadContentAudit(ctx, req.Route)
		if e != nil {
			return classify(e)
		}
		var evidence contentaudit.AcceptanceEvidence
		if evidenceFile != "" {
			f, e := os.Open(evidenceFile)
			if e != nil {
				return fail("EVIDENCE_IO", 1)
			}
			e = contentaudit.DecodeEvidence(f, &evidence)
			f.Close()
			if e != nil {
				return fail("EVIDENCE_INVALID", 2)
			}
		}
		report, e = contentaudit.EvaluatePublished(ctx, req, facts, sources, evidence)
		if e != nil {
			return classify(e)
		}
	}
	if e = ctx.Err(); e != nil {
		return classify(e)
	}
	var j, m bytes.Buffer
	if e = contentaudit.WriteReport(&j, &m, report); e != nil {
		return classify(e)
	}
	// Mkdir is the atomic reservation; cleanup touches only this newly created directory.
	if e = os.Mkdir(out, 0700); e != nil {
		return fail("OUTPUT_IO", 1)
	}
	complete := false
	defer func() {
		if !complete {
			os.RemoveAll(out)
		}
	}()
	for _, f := range []struct {
		name string
		b    []byte
	}{{"report.json", j.Bytes()}, {"report.md", m.Bytes()}} {
		p, e := os.OpenFile(filepath.Join(out, f.name), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
		if e != nil {
			return fail("OUTPUT_IO", 1)
		}
		_, e = p.Write(f.b)
		closeErr := p.Close()
		if e != nil || closeErr != nil {
			return fail("OUTPUT_IO", 1)
		}
	}
	complete = true
	fmt.Fprintf(stdout, "conclusion=%s fixtureOnly=%t draftKnowledge=%d draftInstances=%d formalKnowledge=%d formalInstances=%d\n", report.Conclusion, report.FixtureOnly, report.DraftCounts.Knowledge, report.DraftCounts.EffectiveInstances, report.FormalCounts.Knowledge, report.FormalCounts.EffectiveInstances)
	switch report.Conclusion {
	case contentaudit.Accepted, contentaudit.DraftReady:
		return 0
	case contentaudit.AwaitingReview:
		return 3
	default:
		return 2
	}
}
