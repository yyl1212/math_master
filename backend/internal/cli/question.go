package cli

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"flag"
	"github.com/yyl1212/math_master/backend/internal/question"
	"github.com/yyl1212/math_master/backend/internal/store"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"time"
)

func questionFailure(w io.Writer, err error) int {
	code := "DATABASE_FAILURE"
	status := 1
	switch {
	case errors.Is(err, question.ErrInvalid):
		code = "QUESTION_INVALID"
		status = 2
	case errors.Is(err, question.ErrNotReady):
		code = "QUESTION_NOT_READY"
		status = 2
	case errors.Is(err, question.ErrImmutableConflict):
		code = "IMMUTABLE_CONFLICT"
		status = 2
	case errors.Is(err, question.ErrVersionConflict):
		code = "VERSION_CONFLICT"
		status = 2
	case errors.Is(err, question.ErrLimitExceeded):
		code = "QUESTION_LIMIT_EXCEEDED"
		status = 2
	case errors.Is(err, question.ErrNotConfigured):
		code = "QUESTION_BANK_NOT_CONFIGURED"
	}
	failure(w, code)
	return status
}
func RunQuestion(ctx context.Context, command string, args []string, stdout, stderr io.Writer) int {
	ctx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()
	flags := flag.NewFlagSet(command, flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	var archiveFile, referenceFile, id, out string
	var version int
	switch command {
	case "check", "import":
		flags.StringVar(&archiveFile, "archive", "", "")
		if command == "check" {
			flags.StringVar(&referenceFile, "references", "", "")
		}
	case "export":
		flags.StringVar(&id, "id", "", "")
		flags.IntVar(&version, "version", 0, "")
		flags.StringVar(&out, "out", "", "")
	default:
		return failure(stderr, "INVALID_COMMAND")
	}
	if flags.Parse(args) != nil || flags.NArg() != 0 {
		return failure(stderr, "INVALID_ARGUMENT")
	}
	var archive question.Archive
	if command == "check" || command == "import" {
		if archiveFile == "" || command == "check" && referenceFile == "" {
			return failure(stderr, "INVALID_ARGUMENT")
		}
		file, err := os.Open(archiveFile)
		if err != nil {
			return failure(stderr, "INPUT_IO")
		}
		archive, err = question.DecodeArchive(file)
		file.Close()
		if err != nil {
			return questionFailure(stderr, err)
		}
	}
	if command == "check" {
		file, err := os.Open(referenceFile)
		if err != nil {
			return failure(stderr, "INPUT_IO")
		}
		var refs question.ReferenceSnapshot
		err = question.DecodeStrictJSON(file, question.MaxEnvelopeBytes, &refs)
		file.Close()
		if err != nil {
			return questionFailure(stderr, err)
		}
		sealed, report, err := question.ValidateArchive(ctx, archive, refs)
		if err != nil {
			if len(report.StructuralErrors)+len(report.CompletenessErrors) > 0 {
				write(stdout, report)
			}
			return questionFailure(stderr, err)
		}
		write(stdout, map[string]any{"status": "draft", "packageSha": sealed.PackageSHA, "instances": len(sealed.Instances), "publicApproval": false, "validation": report})
		return 0
	}
	if command == "export" && (!question.ValidMathID(id) || version < 1 || version > 2147483647 || out == "") {
		return failure(stderr, "INVALID_ARGUMENT")
	}
	raw := os.Getenv("DATABASE_URL")
	uri, err := url.Parse(raw)
	if err != nil || raw == "" || (uri.Scheme != "postgres" && uri.Scheme != "postgresql") || uri.Hostname() == "" || uri.Path == "" || uri.Path == "/" {
		return failure(stderr, "INVALID_CONFIG")
	}
	db, err := sql.Open("pgx", raw)
	if err != nil {
		return failure(stderr, "INVALID_CONFIG")
	}
	defer db.Close()
	db.SetMaxOpenConns(2)
	repo := store.New(db)
	if command == "import" {
		result, err := repo.ImportQuestionDraft(ctx, archive)
		if err != nil {
			return questionFailure(stderr, err)
		}
		write(stdout, result)
		return 0
	}
	archive, err = repo.ExportQuestionArchive(ctx, id, version)
	if err != nil {
		return questionFailure(stderr, err)
	}
	if err = os.Mkdir(out, 0700); err != nil {
		return failure(stderr, "OUTPUT_IO")
	}
	complete := false
	defer func() {
		if !complete {
			_ = os.RemoveAll(out)
		}
	}()
	bytes, err := json.Marshal(archive)
	if err != nil {
		return failure(stderr, "OUTPUT_IO")
	}
	if err = os.WriteFile(filepath.Join(out, "archive.json"), append(bytes, '\n'), 0600); err != nil {
		return failure(stderr, "OUTPUT_IO")
	}
	complete = true
	write(stdout, map[string]string{"status": "exported", "approval": "none"})
	return 0
}
