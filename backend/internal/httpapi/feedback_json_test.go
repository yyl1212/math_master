package httpapi

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"github.com/yyl1212/math_master/backend/internal/feedback"
	"net/http/httptest"
	"net/url"
	"os"
	"testing"
)

func TestFeedbackJSONSharedBoundary(t *testing.T) {
	raw, e := os.ReadFile("../../../api/feedback-boundary-cases.json")
	if e != nil {
		t.Fatal(e)
	}
	var cases []struct {
		Name           string  `json:"name"`
		Route          string  `json:"route"`
		RawBase64      string  `json:"rawBase64"`
		ExpectedValid  bool    `json:"expectedValid"`
		ExpectedStatus *int    `json:"expectedStatus"`
		ExpectedCode   *string `json:"expectedCode"`
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if decoder.Decode(&cases) != nil || len(cases) < 70 {
		t.Fatal("case file")
	}
	seen := map[string]bool{}
	for _, c := range cases {
		if c.Name == "" || seen[c.Name] {
			t.Fatal("duplicate or empty case name")
		}
		seen[c.Name] = true
		t.Run(c.Name, func(t *testing.T) {
			b, e := base64.StdEncoding.Strict().DecodeString(c.RawBase64)
			if e != nil {
				t.Fatal("base64")
			}
			u, e := url.Parse(c.Route)
			if e != nil {
				t.Fatal(e)
			}
			method := "POST"
			if len(b) == 0 {
				method = "GET"
			}
			r, e := routeFeedback(u.Path, method)
			if e == nil {
				_, _, e = feedbackQuery(u.RawQuery, r)
			}
			if e == nil && feedback.IsWrite(r.Action) {
				_, e = DecodeFeedbackInput(b, r.Action)
			}
			if !c.ExpectedValid {
				w := httptest.NewRecorder()
				req := httptest.NewRequest(method, c.Route, nil)
				feedbackError(w, req, e)
				var envelope struct {
					Error struct {
						Code string `json:"code"`
					}
				}
				if json.Unmarshal(w.Body.Bytes(), &envelope) != nil || w.Code != 400 || envelope.Error.Code != "INVALID_REQUEST" {
					t.Fatal("invalid byte status/code", w.Code, envelope.Error.Code)
				}
			}
			if (e == nil) != c.ExpectedValid {
				t.Fatal("boundary mismatch", e)
			}
			if c.ExpectedValid && (c.ExpectedStatus != nil || c.ExpectedCode != nil) || !c.ExpectedValid && (c.ExpectedStatus == nil || *c.ExpectedStatus != 400 || c.ExpectedCode == nil || *c.ExpectedCode != "INVALID_REQUEST") {
				t.Fatal("legal byte acceptance is not HTTP success")
			}
		})
	}
}
