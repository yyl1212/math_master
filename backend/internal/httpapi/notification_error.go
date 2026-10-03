package httpapi

import (
	"net/http"
)

func notificationError(w http.ResponseWriter, r *http.Request, e error) { correctionError(w, r, e) }
