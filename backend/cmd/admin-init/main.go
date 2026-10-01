package main

import (
	"context"
	"crypto/rand"
	"database/sql"
	"github.com/yyl1212/math_master/backend/internal/auth"
	"github.com/yyl1212/math_master/backend/internal/cli"
	"github.com/yyl1212/math_master/backend/internal/config"
	"github.com/yyl1212/math_master/backend/internal/store"
	"os"
	"os/signal"
	"syscall"
)

func run() int {
	ctx, c := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer c()
	cfg, err := config.Load()
	if err != nil {
		_, _ = os.Stderr.WriteString("Invalid administrator initialization configuration.\n")
		return 1
	}
	db, err := sql.Open("pgx", cfg.DatabaseURL)
	if err != nil {
		_, _ = os.Stderr.WriteString("Administrator initialization is unavailable.\n")
		return 1
	}
	defer db.Close()
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	initializer := auth.NewAdminService(store.New(db), auth.NewArgon2Hasher(rand.Reader), rand.Reader)
	return cli.RunAdminInit(ctx, os.Args[1:], os.Stdin, os.Stdout, os.Stderr, initializer)
}
func main() { os.Exit(run()) }
