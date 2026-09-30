package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/yyl1212/math_master/backend/internal/content"
	"time"
)

type ImportResult struct {
	PackageID       string `json:"packageId"`
	Version         int    `json:"version"`
	SHA256          string `json:"sha256"`
	AlreadyImported bool   `json:"alreadyImported"`
}

func body(v any) string { b, _ := json.Marshal(v); return string(b) }
func (s *Store) ImportDraft(ctx context.Context, v content.ValidatedPackage) (ImportResult, error) {
	if !v.Verify() {
		return ImportResult{}, ErrInvalidPackage
	}
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	p, c := v.Package(), v.Catalogue()
	result := ImportResult{PackageID: p.ID, Version: p.Version, SHA256: v.SHA256()}
	tx, e := s.db.BeginTx(ctx, nil)
	if e != nil {
		return result, e
	}
	defer tx.Rollback()
	// One transaction lock serializes overlapping immutable versions, including different packages.
	if _, e = tx.ExecContext(ctx, "SELECT pg_advisory_xact_lock(1296127048)"); e != nil {
		return result, e
	}
	var prior string
	e = tx.QueryRowContext(ctx, "SELECT sha256 FROM imported_packages WHERE id=$1 AND version=$2", p.ID, p.Version).Scan(&prior)
	if e == nil {
		if prior != v.SHA256() {
			return result, ErrImmutableConflict
		}
		result.AlreadyImported = true
		return result, tx.Commit()
	}
	if !errors.Is(e, sql.ErrNoRows) {
		return result, e
	}
	e = tx.QueryRowContext(ctx, "SELECT sha256 FROM catalogue_versions WHERE version=$1", c.Version).Scan(&prior)
	switch {
	case e == nil:
		if prior != v.CatalogueSHA256() {
			return result, ErrImmutableConflict
		}
	case errors.Is(e, sql.ErrNoRows):
		if _, e = tx.ExecContext(ctx, "INSERT INTO catalogue_versions(version,sha256,body) VALUES($1,$2,$3)", c.Version, v.CatalogueSHA256(), body(c)); e != nil {
			return result, e
		}
		for _, d := range c.Domains {
			if _, e = tx.ExecContext(ctx, "INSERT INTO domains(catalogue_version,id,position,body) VALUES($1,$2,$3,$4)", c.Version, d.ID, d.Order, body(d)); e != nil {
				return result, e
			}
			for _, t := range d.Topics {
				if _, e = tx.ExecContext(ctx, "INSERT INTO topics(catalogue_version,id,domain_id,body) VALUES($1,$2,$3,$4)", c.Version, t.ID, d.ID, body(t)); e != nil {
					return result, e
				}
			}
		}
		for _, d := range c.Domains {
			for _, target := range d.RelatedDomainIDs {
				if _, e = tx.ExecContext(ctx, "INSERT INTO domain_relations VALUES($1,$2,$3)", c.Version, d.ID, target); e != nil {
					return result, e
				}
			}
		}
	default:
		return result, e
	}
	insertVersion := func(table, id string, version int, obj any, extra []any) error {
		sha := content.Digest(obj)
		var existing string
		e := tx.QueryRowContext(ctx, fmt.Sprintf("SELECT sha256 FROM %s WHERE id=$1 AND version=$2", table), id, version).Scan(&existing)
		if e == nil {
			if existing != sha {
				return ErrImmutableConflict
			}
			return nil
		}
		if !errors.Is(e, sql.ErrNoRows) {
			return e
		}
		query := fmt.Sprintf("INSERT INTO %s(id,version,sha256,body) VALUES($1,$2,$3,$4)", table)
		args := []any{id, version, sha, body(obj)}
		if table == "unit_versions" {
			query = "INSERT INTO unit_versions(id,version,sha256,body,knowledge_id,knowledge_version) VALUES($1,$2,$3,$4,$5,$6)"
			args = append(args, extra...)
		}
		_, e = tx.ExecContext(ctx, query, args...)
		return e
	}
	for _, k := range p.Knowledge {
		if _, e = tx.ExecContext(ctx, "INSERT INTO knowledge(id) VALUES($1) ON CONFLICT DO NOTHING", k.ID); e != nil {
			return result, e
		}
		if e = insertVersion("knowledge_versions", k.ID, k.Version, k, nil); e != nil {
			return result, e
		}
	}
	for _, k := range p.Knowledge {
		for _, rel := range k.Relations {
			if _, e = tx.ExecContext(ctx, "INSERT INTO knowledge_relations VALUES($1,$2,$3,$4,$5) ON CONFLICT DO NOTHING", k.ID, k.Version, rel.Kind, rel.Target.ID, rel.Target.Version); e != nil {
				return result, e
			}
		}
	}
	for _, u := range p.Units {
		if e = insertVersion("unit_versions", u.ID, u.Version, u, []any{u.Knowledge.ID, u.Knowledge.Version}); e != nil {
			return result, e
		}
	}
	for _, path := range p.Paths {
		if e = insertVersion("path_versions", path.ID, path.Version, path, nil); e != nil {
			return result, e
		}
		for i, n := range path.Nodes {
			if _, e = tx.ExecContext(ctx, "INSERT INTO path_nodes VALUES($1,$2,$3,$4,$5) ON CONFLICT DO NOTHING", path.ID, path.Version, i, n.ID, n.Version); e != nil {
				return result, e
			}
		}
	}
	for _, a := range p.Assets {
		b, _ := v.AssetBytes(a.ID)
		if _, e = tx.ExecContext(ctx, "INSERT INTO assets VALUES($1,$2) ON CONFLICT DO NOTHING", a.SHA256, b); e != nil {
			return result, e
		}
	}
	if _, e = tx.ExecContext(ctx, "INSERT INTO imported_packages VALUES($1,$2,$3,$4,$5,$6)", p.ID, p.Version, v.SHA256(), c.Version, v.CatalogueSHA256(), body(p)); e != nil {
		return result, e
	}
	member := func(kind, id string, version int, sha string) error {
		var k, u, path, asset any
		switch kind {
		case "knowledge":
			k = id
		case "unit":
			u = id
		case "path":
			path = id
		case "asset":
			asset = sha
		}
		_, e := tx.ExecContext(ctx, "INSERT INTO package_members VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9)", p.ID, p.Version, kind, id, version, k, u, path, asset)
		return e
	}
	for _, k := range p.Knowledge {
		if e = member("knowledge", k.ID, k.Version, ""); e != nil {
			return result, e
		}
	}
	for _, u := range p.Units {
		if e = member("unit", u.ID, u.Version, ""); e != nil {
			return result, e
		}
	}
	for _, path := range p.Paths {
		if e = member("path", path.ID, path.Version, ""); e != nil {
			return result, e
		}
	}
	for _, a := range p.Assets {
		if e = member("asset", a.ID, 1, a.SHA256); e != nil {
			return result, e
		}
	}
	snapshot := "draft-" + v.SHA256()
	if _, e = tx.ExecContext(ctx, "INSERT INTO publication_snapshots VALUES($1,$2,'draft')", snapshot, c.Version); e != nil {
		return result, e
	}
	if _, e = tx.ExecContext(ctx, "INSERT INTO publication_members SELECT $1,package_id,package_version,kind,id,version,'active' FROM package_members WHERE package_id=$2 AND package_version=$3", snapshot, p.ID, p.Version); e != nil {
		return result, e
	}
	return result, tx.Commit()
}
