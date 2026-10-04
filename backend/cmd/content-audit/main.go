package main

import (
	"context"
	"github.com/yyl1212/math_master/backend/internal/cli"
	"os"
)

func main() { os.Exit(cli.RunContentAudit(context.Background(), os.Args[1:], os.Stdout, os.Stderr)) }
