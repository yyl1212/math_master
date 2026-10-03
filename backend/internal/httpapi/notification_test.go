package httpapi

import (
	"context"
	"github.com/yyl1212/math_master/backend/internal/auth"
	"github.com/yyl1212/math_master/backend/internal/correction"
	"github.com/yyl1212/math_master/backend/internal/notification"
	"github.com/yyl1212/math_master/backend/internal/question"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

type notificationHTTPRepo struct {
	notification.Repository
	calls []string
	fault error
}

func (r *notificationHTTPRepo) CorrectionPreflight(context.Context, question.Access, correction.Action) (auth.User, error) {
	return auth.User{ID: contentFixtureID}, r.fault
}
func (r *notificationHTTPRepo) ListNotifications(context.Context, question.Access, notification.Query) (notification.Envelope[notification.Page[notification.Metadata]], error) {
	r.calls = append(r.calls, "list")
	return notification.Envelope[notification.Page[notification.Metadata]]{}, nil
}
func (r *notificationHTTPRepo) ReadNotificationCount(context.Context, question.Access) (notification.Envelope[notification.UnreadCount], error) {
	r.calls = append(r.calls, "count")
	return notification.Envelope[notification.UnreadCount]{}, nil
}
func (r *notificationHTTPRepo) ReadNotification(context.Context, question.Access, string) (notification.Envelope[notification.Metadata], error) {
	r.calls = append(r.calls, "read")
	return notification.Envelope[notification.Metadata]{}, nil
}
func (r *notificationHTTPRepo) MarkNotificationRead(context.Context, question.Access, string) (notification.Envelope[notification.ReadReceipt], error) {
	r.calls = append(r.calls, "markRead")
	return notification.Envelope[notification.ReadReceipt]{Data: notification.ReadReceipt{Status: 200}}, nil
}
func notificationHTTPFixture(t *testing.T, r *notificationHTTPRepo) http.Handler {
	svc, e := notification.NewService(r)
	if e != nil {
		t.Fatal(e)
	}
	return questionHTTPFixture(t, &httpQuestionRepo{}, nil, func(o *AuthOptions) { o.Notification = &NotificationOptions{Service: svc, PublicOrigin: privateOrigin} })
}
func TestNotificationHTTP(t *testing.T) {
	r := &notificationHTTPRepo{}
	h := notificationHTTPFixture(t, r)
	for _, c := range []struct{ Method, Path, Body, Call string }{{"GET", "", "", "list"}, {"GET", "/count", "", "count"}, {"GET", "/" + contentFixtureID, "", "read"}, {"POST", "/" + contentFixtureID + "/read", "{}", "markRead"}} {
		n := len(r.calls)
		w := privateRequest(h, c.Method, "/api/v1/notifications"+c.Path, c.Body, contentHeaders())
		if w.Code != 200 || len(r.calls) != n+1 || r.calls[n] != c.Call || w.Header().Get("Cache-Control") != "private, no-store" {
			t.Fatal(c, w.Code, w.Body.String())
		}
	}
	for _, q := range []string{"?limit=51", "?limit=1&limit=2", "?unknown=1", "/count?limit=1", "/bad", "/" + contentFixtureID + "?detail=true", "/" + contentFixtureID + "/read?x=1"} {
		w := privateRequest(h, "GET", "/api/v1/notifications"+q, "", contentHeaders())
		if w.Code < 400 {
			t.Fatal("strict route", q)
		}
	}
	for _, body := range []string{"null", "[]", "{\"read\":true}", "{} {}"} {
		w := privateRequest(h, "POST", "/api/v1/notifications/"+contentFixtureID+"/read", body, contentHeaders())
		if w.Code != 400 {
			t.Fatal("empty object only", w.Code)
		}
	}
	headers := contentHeaders()
	delete(headers, "X-CSRF-Token")
	if w := privateRequest(h, "POST", "/api/v1/notifications/"+contentFixtureID+"/read", "{}", headers); w.Code != 403 {
		t.Fatal("CSRF", w.Code)
	}
	r.fault = auth.ErrAuthenticationRequired
	if w := privateRequest(h, "GET", "/api/v1/notifications", "", contentHeaders()); w.Code != 401 {
		t.Fatal("current account", w.Code)
	}
}

func TestNotificationHTTPDeadlineIncludesBody(t *testing.T) {
	repo := &notificationHTTPRepo{}
	server := httptest.NewServer(notificationHTTPFixture(t, repo))
	defer server.Close()
	reader, writer := io.Pipe()
	defer reader.Close()
	defer writer.Close()
	req, _ := http.NewRequest("POST", server.URL+"/api/v1/notifications/"+contentFixtureID+"/read", reader)
	for k, v := range contentHeaders() {
		req.Header.Set(k, v)
	}
	go func() { _, _ = writer.Write([]byte("{")) }()
	begin := time.Now()
	res, e := (&http.Client{Timeout: 10 * time.Second}).Do(req)
	if e != nil {
		t.Fatal(e)
	}
	defer res.Body.Close()
	raw, e := io.ReadAll(res.Body)
	if e != nil || res.StatusCode != 503 || time.Since(begin) < 7*time.Second || time.Since(begin) > 10*time.Second || !strings.Contains(string(raw), "SERVICE_UNAVAILABLE") || len(repo.calls) != 0 {
		t.Fatal("notification whole request deadline", e, res.StatusCode)
	}
}
