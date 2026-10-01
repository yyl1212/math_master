package httpapi

import (
	"crypto/sha256"
	"fmt"
	"github.com/yyl1212/math_master/backend/internal/auth"
	"strings"
	"testing"
)

func TestPrivateContentAsset(t *testing.T) {
	svg := []byte(`<svg xmlns="http://www.w3.org/2000/svg"><title>Original technical fixture</title></svg>`)
	sha := fmt.Sprintf("%x", sha256.Sum256(svg))
	repo := &httpContentRepo{user: auth.User{ID: contentFixtureID, Roles: []auth.Role{auth.RoleEditor}}, boundID: contentFixtureID, asset: svg}
	h := contentHTTPFixture(t, repo, nil)
	for _, scope := range []string{"drafts", "submissions"} {
		w := privateRequest(h, "GET", "/api/v1/content/"+scope+"/"+contentFixtureID+"/assets/"+sha, "", contentHeaders())
		if w.Code != 200 || w.Body.String() != string(svg) || w.Header().Get("Content-Type") != "image/svg+xml" || w.Header().Get("Content-Security-Policy") != "sandbox; default-src 'none'" || w.Header().Get("X-Content-Type-Options") != "nosniff" || w.Header().Get("Cache-Control") != "private, no-store" || len(w.Result().Cookies()) != 0 {
			t.Fatal("SVG response boundary changed", scope, w.Code)
		}
	}
	w := privateRequest(h, "GET", "/api/v1/content/drafts/"+contentOtherID+"/assets/"+sha, "", contentHeaders())
	if w.Code != 404 {
		t.Fatal("other workspace leaked identical SVG bytes")
	}
	w = privateRequest(h, "GET", "/api/v1/content/drafts/"+contentFixtureID+"/assets/"+strings.Repeat("a", 64), "", contentHeaders())
	if w.Code != 404 {
		t.Fatal("wrong digest accepted", w.Code)
	}
	repo.asset = []byte(`<svg xmlns="http://www.w3.org/2000/svg"><script>unsafe()</script></svg>`)
	unsafeSHA := fmt.Sprintf("%x", sha256.Sum256(repo.asset))
	w = privateRequest(h, "GET", "/api/v1/content/drafts/"+contentFixtureID+"/assets/"+unsafeSHA, "", contentHeaders())
	if w.Code != 503 || strings.Contains(w.Body.String(), "unsafe()") {
		t.Fatal("unsafe SVG bytes leaked")
	}
}
