package e2etest

import (
	"context"
	"database/sql"
	"github.com/yyl1212/math_master/backend/internal/taxonomy"
)

type taxonomyControlState struct {
	Pair        taxonomy.PairRef `json:"pair"`
	Nodes       int              `json:"nodes"`
	Primary     [3]int           `json:"primary"`
	Auxiliary   int              `json:"auxiliary"`
	Other       int              `json:"other"`
	Assignments int              `json:"assignments"`
}

func readTaxonomyControl(ctx context.Context, db *sql.DB) (taxonomyControlState, error) {
	var out taxonomyControlState
	tx, e := db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true, Isolation: sql.LevelRepeatableRead})
	if e != nil {
		return out, e
	}
	defer tx.Rollback()
	var kh, th, version sql.NullString
	e = tx.QueryRowContext(ctx, "SELECT (SELECT snapshot_id FROM publication_heads),(SELECT release_id::text FROM taxonomy_heads),(SELECT r.taxonomy_version_id FROM taxonomy_heads h JOIN taxonomy_releases r ON r.id=h.release_id)").Scan(&kh, &th, &version)
	if e != nil {
		return out, e
	}
	if kh.Valid {
		out.Pair.KnowledgeHead = &kh.String
	}
	if th.Valid {
		out.Pair.TaxonomyHead = &th.String
	}
	out.Pair.TaxonomyVersionID = version.String
	e = tx.QueryRowContext(ctx, `SELECT count(*),count(*) FILTER(WHERE kind='primary' AND level=1),count(*) FILTER(WHERE kind='primary' AND level=2),count(*) FILTER(WHERE kind='primary' AND level=3),count(*) FILTER(WHERE kind='auxiliary'),count(*) FILTER(WHERE kind='other') FROM taxonomy_nodes WHERE taxonomy_version_id=$1`, version.String).Scan(&out.Nodes, &out.Primary[0], &out.Primary[1], &out.Primary[2], &out.Auxiliary, &out.Other)
	if e != nil {
		return out, e
	}
	e = tx.QueryRowContext(ctx, "SELECT count(*) FROM taxonomy_release_assignments WHERE release_id=$1::uuid", th).Scan(&out.Assignments)
	if e != nil {
		return out, e
	}
	return out, tx.Commit()
}
