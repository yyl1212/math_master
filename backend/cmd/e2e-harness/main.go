// This separate binary must never be used as the production server.
package main

import (
	"context"
	"github.com/yyl1212/math_master/backend/internal/e2etest"
	"log"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	c := e2etest.Config{TestDatabaseURL: os.Getenv("TEST_DATABASE_URL"), APIAddr: "127.0.0.1:18081", ControlAddr: "127.0.0.1:18082", StateFile: "tests/e2e/runtime.local.json"}
	if e := e2etest.Run(ctx, c); e != nil {
		log.Fatal(e)
	}
}
