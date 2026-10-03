package store

import (
	"context"
	"database/sql"
	"github.com/yyl1212/math_master/backend/internal/learning"
	"github.com/yyl1212/math_master/backend/internal/question"
	"testing"
)

func TestCorrectionProjectionInvalidation(t *testing.T) {
	tx := new(sql.Tx)
	ctx := learningWithProjection(context.Background(), tx, "owner")
	p := learningProjectionFor(ctx, tx, "owner")
	a := question.Identity{ID: "a"}
	b := question.Identity{ID: "b"}
	p.passes[a] = false
	p.evidence[a] = learning.EvidenceView{}
	p.passes[b] = true
	learningInvalidateProjection(ctx, tx, "owner", a)
	if _, ok := p.passes[a]; ok {
		t.Fatal("stale pass memo")
	}
	if _, ok := p.evidence[a]; ok {
		t.Fatal("stale qualification memo")
	}
	if !p.passes[b] {
		t.Fatal("unrelated memo lost")
	}
}
