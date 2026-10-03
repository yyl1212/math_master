package correction

import (
	"bytes"
	"testing"
)

func TestCorrectionDigest(t *testing.T) {
	v := map[string]any{"id": testUUID}
	a, x, e := Canonical("correction-command-v1", v)
	if e != nil {
		t.Fatal(e)
	}
	b, y, e := Canonical("notification-command-v1", v)
	if e != nil {
		t.Fatal(e)
	}
	if x == y || bytes.Equal(a, b) {
		t.Fatal("cross-domain command collision")
	}
	a2, x2, e := Canonical("correction-command-v1", v)
	if e != nil || x != x2 || !bytes.Equal(a, a2) {
		t.Fatal("unstable canonical receipt")
	}
	if _, _, e := Canonical("assessment-seal-v1", v); e == nil {
		t.Fatal("new digest domain accepts original purpose")
	}
	if _, _, e := Canonical("correction-plan-v1", map[string]string{"reason": string([]byte{255})}); e == nil {
		t.Fatal("invalid UTF8 silently rewritten")
	}
}
