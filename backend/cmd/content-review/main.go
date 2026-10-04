package main

import (
	"context"
	"github.com/yyl1212/math_master/backend/internal/cli"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	os.Exit(cli.RunContentReview(ctx, os.Args[1:], os.Stdout, os.Stderr))
}
