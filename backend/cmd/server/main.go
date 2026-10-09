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
	"sync"
	"syscall"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/yyl1212/math_master/backend/internal/auth"
	"github.com/yyl1212/math_master/backend/internal/config"
	"github.com/yyl1212/math_master/backend/internal/correction"
	"github.com/yyl1212/math_master/backend/internal/feedback"
	"github.com/yyl1212/math_master/backend/internal/httpapi"
	"github.com/yyl1212/math_master/backend/internal/knowledgeadmin"
	"github.com/yyl1212/math_master/backend/internal/learning"
	"github.com/yyl1212/math_master/backend/internal/notification"
	"github.com/yyl1212/math_master/backend/internal/publication"
	"github.com/yyl1212/math_master/backend/internal/question"
	"github.com/yyl1212/math_master/backend/internal/store"
	"github.com/yyl1212/math_master/backend/internal/study"
	"github.com/yyl1212/math_master/backend/internal/taxonomy"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}
func run() error {
	c, err := config.Load()
	if err != nil {
		return err
	}
	db, err := sql.Open("pgx", c.DatabaseURL)
	if err != nil {
		return errors.New("database configuration failed")
	}
	defer db.Close()
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	repo := store.NewWithTrustedCodeSHA(db, c.CodeSHA)
	options := httpapi.AuthOptions{ExperienceMode: repo, PublicOrigin: c.PublicOrigin, Production: c.AppEnv == "production"}
	options.Knowledge = &httpapi.KnowledgeOptions{Service: knowledgeadmin.NewService(repo), Current: repo, PublicOrigin: c.PublicOrigin, Production: c.AppEnv == "production"}
	if c.PublicOrigin != "" {
		hasher := auth.NewArgon2Hasher(rand.Reader)
		options.Accounts, err = auth.NewService(repo, hasher, rand.Reader)
		if err != nil {
			return errors.New("account initialization failed")
		}
		options.Admin = auth.NewAdminService(repo, hasher, rand.Reader)
	}
	readyCtx, readyCancel := context.WithTimeout(ctx, 8*time.Second)
	contentConfigured, contentErr := httpapi.ContentReady(readyCtx, db)
	readyCancel()
	if contentErr != nil {
		log.Print("content configuration check failed")
	}
	publicationService := publication.NewService(repo)
	options.Content = &httpapi.ContentOptions{Service: publicationService, PublicOrigin: c.PublicOrigin, Production: c.AppEnv == "production", Configured: contentConfigured}
	options.Taxonomy = &httpapi.TaxonomyOptions{Service: taxonomy.NewService(repo, publicationService), PublicOrigin: c.PublicOrigin, Production: c.AppEnv == "production"}
	questionCtx, questionCancel := context.WithTimeout(ctx, 8*time.Second)
	questionConfigured, questionErr := httpapi.QuestionReady(questionCtx, db)
	questionCancel()
	if questionErr != nil {
		log.Print("question configuration check failed")
	}
	questionService, questionErr := question.NewService(repo, publicationService.AcquireValidation)
	if questionErr != nil {
		return errors.New("question initialization failed")
	}
	options.Question = &httpapi.QuestionOptions{Service: questionService, PublicOrigin: c.PublicOrigin, Production: c.AppEnv == "production", Configured: questionConfigured}
	learningService, learningErr := learning.NewService(repo, publicationService.AcquireValidation)
	if learningErr != nil {
		return errors.New("learning initialization failed")
	}
	options.Learning = &httpapi.LearningOptions{Learning: learningService, PublicOrigin: c.PublicOrigin, Production: c.AppEnv == "production"}
	studyService, studyErr := study.NewService(repo)
	if studyErr != nil {
		return errors.New("study initialization failed")
	}
	options.Study = &httpapi.StudyOptions{Service: studyService, PublicOrigin: c.PublicOrigin, Production: c.AppEnv == "production"}
	feedbackService, feedbackErr := feedback.NewService(repo)
	if feedbackErr != nil {
		return errors.New("feedback initialization failed")
	}
	options.Feedback = &httpapi.FeedbackOptions{Service: feedbackService, PublicOrigin: c.PublicOrigin, Production: c.AppEnv == "production"}
	correctionService, e := correction.NewService(repo)
	if e != nil {
		return errors.New("correction initialization failed")
	}
	notificationService, e := notification.NewService(repo)
	if e != nil {
		return errors.New("notification initialization failed")
	}
	options.Correction = &httpapi.CorrectionOptions{Service: correctionService, PublicOrigin: c.PublicOrigin, Production: c.AppEnv == "production"}
	options.Notification = &httpapi.NotificationOptions{Service: notificationService, PublicOrigin: c.PublicOrigin, Production: c.AppEnv == "production"}
	runner, e := correction.NewRunner(repo, correction.Options{Enabled: c.CorrectionWorkerEnabled, Limit: 50})
	if e != nil {
		return errors.New("correction worker initialization failed")
	}
	var workers sync.WaitGroup
	defer func() { stop(); workers.Wait() }()
	workers.Add(1)
	go func() {
		defer workers.Done()
		if e := runner.Run(ctx); e != nil && ctx.Err() == nil {
			log.Print("correction errorClass=configuration")
		}
	}()
	if options.Accounts != nil {
		workers.Add(1)
		go func() {
			defer workers.Done()
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
	requests := newDrainingHandler(httpapi.NewApplicationHandler(repo, db, options))
	srv := &http.Server{Addr: c.HTTPAddr, Handler: requests, ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 15 * time.Second, WriteTimeout: 15 * time.Second, IdleTimeout: time.Minute}
	workers.Add(1)
	go func() {
		defer workers.Done()
		<-ctx.Done()
		requests.stop()
		shutdown, cancel := context.WithTimeout(context.Background(), c.ShutdownTimeout)
		defer cancel()
		if srv.Shutdown(shutdown) != nil {
			_ = srv.Close()
		}
		requests.wait()
	}()
	log.Printf("HTTP service listening on %s", c.HTTPAddr)
	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return errors.New("HTTP service failed")
	}
	return nil
}

// The gate closes before Shutdown: no Add may race with the final wait. Even
// after forced connection closure, already-entered handlers finish their cleanup.
type drainingHandler struct {
	handler  http.Handler
	mu       sync.Mutex
	stopping bool
	active   sync.WaitGroup
}

func newDrainingHandler(h http.Handler) *drainingHandler { return &drainingHandler{handler: h} }
func (h *drainingHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.mu.Lock()
	if h.stopping {
		h.mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "private, no-store")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Request-ID", "shutdown")
		w.WriteHeader(503)
		_, _ = w.Write([]byte(`{"error":{"code":"SERVICE_UNAVAILABLE","message":"Service temporarily unavailable.","requestId":"shutdown"}}`))
		return
	}
	h.active.Add(1)
	h.mu.Unlock()
	defer h.active.Done()
	h.handler.ServeHTTP(w, r)
}
func (h *drainingHandler) stop() { h.mu.Lock(); h.stopping = true; h.mu.Unlock() }
func (h *drainingHandler) wait() { h.active.Wait() }
