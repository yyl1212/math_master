package store

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"github.com/yyl1212/math_master/backend/internal/taxonomy"
	"time"
)

func (s *Store) InstallTaxonomyBatch(ctx context.Context, c taxonomy.CapturedBatch) (taxonomy.Version, error) {
	var out taxonomy.Version
	if e := taxonomy.ValidateCatalogue(c.Nodes); e != nil {
		return out, e
	}
	if c.Manifest.SchemaVersion != 1 || !c.Manifest.Accepted || c.Manifest.Batch < 1 || !taxonomy.ValidSHA(c.Manifest.SnapshotID) || !taxonomy.ValidSHA(c.RawClassificationSHA) || c.Attribution == "" || c.License == "" {
		return out, taxonomy.ErrInvalid
	}
	files, e := json.Marshal(c.Manifest.SourceFiles)
	if e != nil {
		return out, taxonomy.ErrInvalid
	}
	h := sha256.Sum256(files)
	if hex.EncodeToString(h[:]) != c.Manifest.SnapshotID {
		return out, taxonomy.ErrInvalid
	}
	seen := map[string]string{}
	for _, f := range c.Manifest.SourceFiles {
		if !taxonomy.SafePath(f.Path) || f.SizeBytes < 0 || !taxonomy.ValidSHA(f.SHA256) {
			return out, taxonomy.ErrInvalid
		}
		if _, ok := seen[f.Path]; ok {
			return out, taxonomy.ErrInvalid
		}
		seen[f.Path] = f.SHA256
	}
	for _, r := range c.SourceRecordIndex {
		if !taxonomy.SafePath(r.Path) || seen["Knowledge_JSON/"+r.Path] != r.SHA256 || r.SourceID == "" || r.RecordID == "" || r.WorkFamilyID == "" {
			return out, taxonomy.ErrInvalid
		}
	}
	canonical, e := json.Marshal(c)
	if e != nil || len(canonical) > 32<<20 {
		return out, taxonomy.ErrLimit
	}
	digest, e := taxonomy.Digest(c)
	if e != nil {
		return out, e
	}
	out = taxonomy.Version{ID: c.Manifest.SnapshotID, SnapshotID: c.Manifest.SnapshotID, TaxonomySHA: digest, Batch: c.Manifest.Batch}
	ctx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()
	tx, e := s.db.BeginTx(ctx, nil)
	if e != nil {
		return out, taxonomyError(e)
	}
	defer tx.Rollback()
	if e = taxonomyConfigured(ctx, tx); e != nil {
		return out, e
	}
	if _, e = tx.ExecContext(ctx, "SET LOCAL lock_timeout='1s'"); e != nil {
		return out, e
	}
	if _, e = tx.ExecContext(ctx, "SELECT pg_advisory_xact_lock($1)", int64(1296127050)); e != nil {
		return out, e
	}
	var prior string
	e = tx.QueryRowContext(ctx, "SELECT taxonomy_sha FROM taxonomy_versions WHERE id=$1", out.ID).Scan(&prior)
	if e == nil {
		if prior != digest {
			return out, taxonomy.ErrConflict
		}
		return out, tx.Commit()
	}
	if e != sql.ErrNoRows {
		return out, taxonomyError(e)
	}
	if _, e = tx.ExecContext(ctx, "INSERT INTO taxonomy_source_batches(snapshot_id,batch,body) VALUES($1,$2,$3)", out.SnapshotID, out.Batch, string(canonical)); e != nil {
		return out, taxonomyError(e)
	}
	if _, e = tx.ExecContext(ctx, "INSERT INTO taxonomy_versions(id,snapshot_id,taxonomy_sha,raw_classification_sha,attribution,license) VALUES($1,$1,$2,$3,$4,$5)", out.ID, digest, c.RawClassificationSHA, c.Attribution, c.License); e != nil {
		return out, taxonomyError(e)
	}
	nodes, e := json.Marshal(c.Nodes)
	if e != nil {
		return out, e
	}
	_, e = tx.ExecContext(ctx, `INSERT INTO taxonomy_nodes(taxonomy_version_id,id,code,parent_id,kind,level,body)
 SELECT $1,node->>'id',node->>'code',node->>'parentId',node->>'kind',(node->>'level')::integer,node FROM jsonb_array_elements($2::jsonb) node`, out.ID, string(nodes))
	if e != nil {
		return out, taxonomyError(e)
	}
	return out, taxonomyError(tx.Commit())
}
