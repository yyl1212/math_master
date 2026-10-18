package httpapi

import "testing"

func TestNumberedQueryOptIn(t *testing.T) {
	raw, p, e := numberedQuery("q=math&pagination=page&page=3&limit=20", true, 20, 50)
	if e != nil || p.Page != 3 || raw != "limit=20&q=math" {
		t.Fatal(raw, p, e)
	}
	for _, s := range []string{"pagination=page&page=0", "page=1", "pagination=page&page=1&cursor=a", "pagination=wrong"} {
		if _, _, e = numberedQuery(s, true, 20, 50); e == nil {
			t.Fatal(s)
		}
	}
	if _, _, e = numberedQuery("pagination=page", false, 20, 50); e == nil {
		t.Fatal("non-list accepted")
	}
}
