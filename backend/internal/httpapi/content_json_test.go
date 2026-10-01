package httpapi

import (
	"errors"
	"github.com/yyl1212/math_master/backend/internal/publication"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestContentJSONExactBoundary(t *testing.T) {
	good := `{"submissionIds":["11111111-1111-4111-8111-111111111111"],"expectedHead":null,"reason":"A valid technical reason."}`
	for _, limit := range []int64{8192, 8388608} {
		var out publication.PrepareInput
		at := good + strings.Repeat(" ", int(limit)-len(good))
		if err := decodeContentJSON(strings.NewReader(at), limit, &out); err != nil || out.ExpectedHead != nil {
			t.Fatal("exact raw-byte limit rejected", limit, err)
		}
		if err := decodeContentJSON(strings.NewReader(at+" "), limit, &out); !errors.Is(err, errContentPayloadTooLarge) {
			t.Fatal("limit+1 not 413", limit, err)
		}
	}
	bad := []string{`{"submissionIds":[],"reason":"Valid reason text."}`, `{"SubmissionIds":[],"expectedHead":null,"reason":"Valid reason text."}`, `{"submissionIds":[],"expectedHead":null,"expectedHead":null,"reason":"Valid reason text."}`, `{"submissionIds":[],"expectedHead":null,"ExpectedHead":null,"reason":"Valid reason text."}`, `{"submissionIds":[],"expectedHead":null,"reason":"\ud800"}`, `{"submissionIds":[],"expectedHead":null,"reason":"\udc00"}`, `{"submissionIds":[],"expectedHead":null,"reason":"\u0000"}`, `{"submissionIds":null,"expectedHead":null,"reason":"Valid reason text."}`, `{"submissionIds":[],"expectedHead":null,"reason":"Valid reason text.","authorIds":[]}`, good + ` {}`}
	for i, input := range bad {
		var out publication.PrepareInput
		if decodeContentJSON(strings.NewReader(input), 8192, &out) == nil {
			t.Fatal("invalid control accepted", i)
		}
	}
	var nested struct {
		Payload any `json:"payload"`
	}
	for _, depth := range []int{32, 33} {
		raw := `{"payload":` + strings.Repeat("[", depth-1) + `0` + strings.Repeat("]", depth-1) + `}`
		err := decodeContentJSON(strings.NewReader(raw), 8192, &nested)
		if (depth == 32) != (err == nil) {
			t.Fatal("incorrect depth boundary", depth, err)
		}
	}
	var d publication.DraftInput
	for _, input := range []string{`{"catalogueVersion":1,"package":{"schemaVersion":1,"id":"fixture","version":1,"knowledge":[],"units":[],"paths":[],"assets":[],"authorIds":[]},"assetBytes":[],"sourceMap":[]}`, `{"catalogueVersion":1,"package":{"schemaVersion":1,"ID":"fixture"},"assetBytes":[],"sourceMap":[]}`} {
		if decodeContentJSON(strings.NewReader(input), 8<<20, &d) == nil {
			t.Fatal("nested unknown/case alias accepted")
		}
	}
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/api/v1/content/drafts", nil)
	r.Header.Set("X-Request-ID", "technical-request")
	exact := strings.Repeat("x", (4<<20)-2)
	contentResponse(w, r, 200, exact)
	if w.Code != 200 || w.Body.Len() != 4<<20 {
		t.Fatal("exact response cap rejected", w.Code, w.Body.Len())
	}
	w = httptest.NewRecorder()
	contentResponse(w, r, 200, exact+"x")
	if w.Code != 422 || strings.Contains(w.Body.String(), exact[:100]) {
		t.Fatal("oversize private response leaked", w.Code)
	}
}
