package store_test

import (
	"github.com/yyl1212/math_master/backend/internal/correction"
	"github.com/yyl1212/math_master/backend/internal/pagination"
	"github.com/yyl1212/math_master/backend/internal/study"
	"testing"
)

func TestNumberedLegacyHistory(t *testing.T) {
	f := newStudyFixture(t)
	_, e := f.repo.BeginStudy(f.ctx, f.Access("author_a"), f.Ref().ID, f.Command(0))
	if e != nil {
		t.Fatal(e)
	}
	p, e := f.repo.ListStudyHistoryPage(f.ctx, f.Access("author_a"), study.HistoryQuery{}, pagination.Request{Enabled: true, Page: 99, Limit: 1})
	if e != nil || p.Total != 1 || p.Offset != 0 || len(p.Items) != 1 {
		t.Fatal(p, e)
	}
}
func TestNumberedCorrectionArchive(t *testing.T) {
	f := newCorrectionFixture(t)
	f.cReadableResult()
	p, e := f.repo.ListCorrectionCasePage(f.ctx, f.Access("admin_a", false), correction.Query{}, pagination.Request{Enabled: true, Page: 99, Limit: 1})
	if e != nil || p.Data.Total < 1 || len(p.Data.Items) != 1 {
		t.Fatal(p, e)
	}
}
