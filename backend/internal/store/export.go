package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"github.com/yyl1212/math_master/backend/internal/catalogue"
	"github.com/yyl1212/math_master/backend/internal/content"
)

func rowError(e error) error {
	if errors.Is(e, sql.ErrNoRows) {
		return ErrNotFound
	}
	return e
}
func (s *Store) ExportPackage(ctx context.Context, id string, version int) (content.Package, error) {
	var b []byte
	var p content.Package
	e := s.db.QueryRowContext(ctx, "SELECT body FROM imported_packages WHERE id=$1 AND version=$2", id, version).Scan(&b)
	if e != nil {
		return p, rowError(e)
	}
	e = json.Unmarshal(b, &p)
	return p, e
}
func (s *Store) ExportCatalogue(ctx context.Context, version int) (catalogue.Catalogue, error) {
	var b []byte
	var c catalogue.Catalogue
	e := s.db.QueryRowContext(ctx, "SELECT body FROM catalogue_versions WHERE version=$1", version).Scan(&b)
	if e != nil {
		return c, rowError(e)
	}
	e = json.Unmarshal(b, &c)
	return c, e
}
func (s *Store) ExportAsset(ctx context.Context, sha string) ([]byte, error) {
	var b []byte
	e := s.db.QueryRowContext(ctx, "SELECT bytes FROM assets WHERE sha256=$1", sha).Scan(&b)
	return b, rowError(e)
}
func (s *Store) PackageCatalogueVersion(ctx context.Context, id string, version int) (int, error) {
	var v int
	e := s.db.QueryRowContext(ctx, "SELECT catalogue_version FROM imported_packages WHERE id=$1 AND version=$2", id, version).Scan(&v)
	return v, rowError(e)
}
