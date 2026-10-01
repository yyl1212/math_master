package httpapi

import (
	"github.com/yyl1212/math_master/backend/internal/store"
	"net/http"
	"regexp"
	"strconv"
)

var assetDigestPattern = regexp.MustCompile(`^[a-f0-9]{64}$`)

type headWriter struct{ http.ResponseWriter }

func (w headWriter) Write(b []byte) (int, error) { return len(b), nil }
func publicAsset(w http.ResponseWriter, r *http.Request, reader Reader) {
	if r.Method == http.MethodHead {
		w = headWriter{w}
	}
	digest := r.PathValue("sha256")
	if !assetDigestPattern.MatchString(digest) {
		apiError(w, r.Header.Get("X-Request-ID"), store.ErrNotFound)
		return
	}
	data, e := reader.GetPublishedAsset(r.Context(), digest)
	if e != nil {
		apiError(w, r.Header.Get("X-Request-ID"), e)
		return
	}
	w.Header().Set("Content-Type", "image/svg+xml")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Security-Policy", "sandbox; default-src 'none'")
	w.Header().Set("Content-Length", strconv.Itoa(len(data)))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
}
