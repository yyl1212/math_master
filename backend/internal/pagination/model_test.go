package pagination

import (
	"net/url"
	"testing"
)

func TestWindowSafeIntegerAndClamp(t *testing.T) {
	for _, c := range []struct {
		total, page int64
		want        int64
	}{{0, 1, 0}, {41, 2, 20}, {41, 99, 40}, {1000021, 50002, 1000020}} {
		m, e := Window(c.total, c.page, 20)
		if e != nil || m.Offset != c.want {
			t.Fatalf("%+v: %+v %v", c, m, e)
		}
	}
	if _, e := Window(41, 9007199254740992, 20); e == nil {
		t.Fatal("unsafe page accepted")
	}
}
func TestParseExplicitModeAndAnchors(t *testing.T) {
	for _, raw := range []string{"pagination=page&page=0", "pagination=page&page=1&cursor=a", "pagination=page&page=1.2", "pagination=page&page=1&page=2"} {
		v, _ := url.ParseQuery(raw)
		if _, e := Parse(v, 20, 50); e == nil {
			t.Fatal(raw)
		}
	}
	v, _ := url.ParseQuery("pagination=page&page=3&limit=20")
	p, e := Parse(v, 20, 50)
	if e != nil || !p.Enabled || p.Page != 3 {
		t.Fatal(p, e)
	}
}
