package httpapi

import (
	"crypto/sha256"
	"fmt"
	"github.com/yyl1212/math_master/backend/internal/auth"
	"github.com/yyl1212/math_master/backend/internal/content"
	"net/http"
	"strconv"
)

func contentAsset(w http.ResponseWriter, r *http.Request, sha string, data []byte, err error) {
	if r.Context().Err() != nil {
		contentError(w, r, auth.ErrUnavailable)
		return
	}
	if err != nil {
		contentError(w, r, err)
		return
	}
	if len(data) > 1<<20 {
		contentError(w, r, auth.ErrUnavailable)
		return
	}
	if fmt.Sprintf("%x", sha256.Sum256(data)) != sha {
		contentError(w, r, auth.ErrNotFound)
		return
	}
	if content.ValidateSVG(data) != nil {
		contentError(w, r, auth.ErrUnavailable)
		return
	}
	if r.Context().Err() != nil {
		contentError(w, r, auth.ErrUnavailable)
		return
	}
	w.Header().Set("Content-Type", "image/svg+xml")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Cache-Control", "private, no-store")
	w.Header().Set("Content-Security-Policy", "sandbox; default-src 'none'")
	w.Header().Set("Content-Length", strconv.Itoa(len(data)))
	w.WriteHeader(200)
	_, _ = w.Write(data)
}
