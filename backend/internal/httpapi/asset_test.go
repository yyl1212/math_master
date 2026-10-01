package httpapi

import (
	"context"
	"errors"
	"github.com/yyl1212/math_master/backend/internal/store"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
)

type assetFakeReader struct {
	fakeReader
	data []byte
}

func (f *assetFakeReader) GetPublishedAsset(context.Context, string) ([]byte, error) {
	return f.data, f.err
}
func TestPublicAssetResponseContract(t *testing.T) {
	digest := strings.Repeat("a", 64)
	data := []byte(`<svg xmlns="http://www.w3.org/2000/svg"><title>Original</title></svg>`)
	for _, method := range []string{"GET", "HEAD"} {
		f := &assetFakeReader{data: data}
		w := httptest.NewRecorder()
		NewHandler(f, nil).ServeHTTP(w, httptest.NewRequest(method, "/api/v1/assets/"+digest, nil))
		if w.Code != 200 {
			t.Fatalf("%s: expected published SVG, got %d", method, w.Code)
		}
		for k, want := range map[string]string{"Content-Type": "image/svg+xml", "X-Content-Type-Options": "nosniff", "Cache-Control": "no-store", "Content-Security-Policy": "sandbox; default-src 'none'", "Content-Length": strconv.Itoa(len(data))} {
			if w.Header().Get(k) != want {
				t.Fatal(k, w.Header().Get(k))
			}
		}
		if method == "HEAD" && w.Body.Len() != 0 {
			t.Fatal("HEAD returned bytes")
		}
		if method == "GET" && w.Body.String() != string(data) {
			t.Fatal("SVG bytes changed")
		}
	}
	for _, tc := range []struct {
		err    error
		status int
	}{{store.ErrNotFound, 404}, {store.ErrUnavailable, 503}, {errors.New("private-secret"), 500}} {
		w := request(NewHandler(&assetFakeReader{fakeReader: fakeReader{err: tc.err}}, nil), "/api/v1/assets/"+digest)
		if w.Code != tc.status || strings.Contains(w.Body.String(), "private-secret") || w.Header().Get("Cache-Control") != "no-store" {
			t.Fatal(w.Code, w.Body.String())
		}
	}
	for _, bad := range []string{"ABC", strings.Repeat("A", 64), strings.Repeat("a", 63)} {
		w := request(NewHandler(&assetFakeReader{data: data}, nil), "/api/v1/assets/"+bad)
		if w.Code != 404 {
			t.Fatal("bad digest accepted", bad)
		}
	}
}
