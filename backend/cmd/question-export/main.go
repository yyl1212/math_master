package main

import (
	"context"
	"github.com/yyl1212/math_master/backend/internal/cli"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	ctx, c := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer c()
	os.Exit(cli.RunQuestion(ctx, "export", os.Args[1:], os.Stdout, os.Stderr))
}
