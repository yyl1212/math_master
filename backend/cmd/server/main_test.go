package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestCorrectionServerDrainWaitsForHandlers(t *testing.T) {
	entered, release := make(chan struct{}), make(chan struct{})
	h := newDrainingHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { close(entered); <-release; w.WriteHeader(204) }))
	requestDone := make(chan struct{})
	go func() {
		h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/api/v1/learning", nil))
		close(requestDone)
	}()
	<-entered
	h.stop()
	closed := make(chan struct{})
	go func() { h.wait(); close(closed) }()
	select {
	case <-closed:
		t.Fatal("database could close before handler transaction finished")
	case <-time.After(10 * time.Millisecond):
	}
	late := httptest.NewRecorder()
	h.ServeHTTP(late, httptest.NewRequest("GET", "/api/v1/learning", nil))
	if late.Code != 503 || late.Header().Get("Cache-Control") != "private, no-store" {
		t.Fatal("late request entered database", late.Code)
	}
	close(release)
	<-closed
	<-requestDone
}
