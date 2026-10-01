package main

import (
	"context"
	"crypto/rand"
	"database/sql"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/yyl1212/math_master/backend/internal/auth"
	"github.com/yyl1212/math_master/backend/internal/config"
	"github.com/yyl1212/math_master/backend/internal/httpapi"
	"github.com/yyl1212/math_master/backend/internal/publication"
	"github.com/yyl1212/math_master/backend/internal/store"
)

func main() {
	c, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}
	db, err := sql.Open("pgx", c.DatabaseURL)
	if err != nil {
		log.Fatal("database configuration failed")
	}
	defer db.Close()
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	repo := store.New(db)
	options := httpapi.AuthOptions{PublicOrigin: c.PublicOrigin, Production: c.AppEnv == "production"}
	if c.PublicOrigin != "" {
		hasher := auth.NewArgon2Hasher(rand.Reader)
		options.Accounts, err = auth.NewService(repo, hasher, rand.Reader)
		if err != nil {
			log.Fatal("account initialization failed")
		}
		options.Admin = auth.NewAdminService(repo, hasher, rand.Reader)
		go func() {
			ticker := time.NewTicker(time.Minute)
			defer ticker.Stop()
			for {
				select {
				case <-ctx.Done():
					return
				case <-ticker.C:
					if _, err := options.Accounts.Cleanup(ctx); err != nil && ctx.Err() == nil {
						log.Print("account cleanup failed")
					}
				}
			}
		}()
	}
	readyCtx, readyCancel := context.WithTimeout(ctx, 8*time.Second)
	contentConfigured, contentErr := httpapi.ContentReady(readyCtx, db)
	readyCancel()
	if contentErr != nil {
		log.Print("content configuration check failed")
	}
	options.Content = &httpapi.ContentOptions{Service: publication.NewService(repo), PublicOrigin: c.PublicOrigin, Production: c.AppEnv == "production", Configured: contentConfigured}
	srv := &http.Server{Addr: c.HTTPAddr, Handler: httpapi.NewApplicationHandler(repo, db, options), ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 15 * time.Second, WriteTimeout: 15 * time.Second, IdleTimeout: time.Minute}
	go func() {
		<-ctx.Done()
		shutdown, cancel := context.WithTimeout(context.Background(), c.ShutdownTimeout)
		defer cancel()
		if srv.Shutdown(shutdown) != nil {
			_ = srv.Close()
		}
	}()
	log.Printf("HTTP service listening on %s", c.HTTPAddr)
	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatal("HTTP service failed")
	}
}
