package cli

import (
	"context"
	"errors"
	"flag"
	"github.com/yyl1212/math_master/backend/internal/auth"
	"golang.org/x/term"
	"io"
	"os"
	"strings"
)

type AdminInitializer interface {
	Initialize(context.Context, string, string, string) (auth.User, error)
}

func RunAdminInit(ctx context.Context, args []string, input io.Reader, stdout, stderr io.Writer, initializer AdminInitializer) int {
	f := flag.NewFlagSet("admin-init", flag.ContinueOnError)
	f.SetOutput(io.Discard)
	var username string
	var stdin bool
	f.StringVar(&username, "username", "", "")
	f.BoolVar(&stdin, "password-stdin", false, "")
	if f.Parse(args) != nil || f.NArg() != 0 {
		return failure(stderr, "INVALID_ARGUMENT")
	}
	normalized, err := auth.ValidateUsername(username)
	if err != nil {
		return failure(stderr, "INVALID_ARGUMENT")
	}
	var raw []byte
	if stdin {
		raw, err = io.ReadAll(io.LimitReader(input, 515))
		if err != nil || len(raw) > 514 {
			return failure(stderr, "INVALID_INPUT")
		}
		if len(raw) > 0 && raw[len(raw)-1] == '\n' {
			raw = raw[:len(raw)-1]
			if len(raw) > 0 && raw[len(raw)-1] == '\r' {
				raw = raw[:len(raw)-1]
			}
		}
		if strings.ContainsAny(string(raw), "\r\n") {
			return failure(stderr, "INVALID_INPUT")
		}
	} else {
		terminal, ok := input.(*os.File)
		if !ok || !term.IsTerminal(int(terminal.Fd())) {
			return failure(stderr, "PASSWORD_INPUT_REQUIRED")
		}
		_, _ = io.WriteString(stderr, "Password: ")
		raw, err = term.ReadPassword(int(terminal.Fd()))
		_, _ = io.WriteString(stderr, "\n")
		if err != nil {
			return failure(stderr, "INVALID_INPUT")
		}
	}
	password := string(raw)
	if auth.ValidateNewPassword(password) != nil {
		return failure(stderr, "INVALID_INPUT")
	}
	if initializer == nil {
		return failure(stderr, "SERVICE_UNAVAILABLE")
	}
	user, err := initializer.Initialize(ctx, normalized, password, "admin-init")
	if err != nil {
		if errors.Is(err, auth.ErrAlreadyInitialized) {
			return failure(stderr, "ALREADY_INITIALIZED")
		}
		if errors.Is(err, auth.ErrUsernameUnavailable) {
			return failure(stderr, "USERNAME_UNAVAILABLE")
		}
		return failure(stderr, "SERVICE_UNAVAILABLE")
	}
	write(stdout, map[string]any{"status": "initialized", "user": user})
	return 0
}
