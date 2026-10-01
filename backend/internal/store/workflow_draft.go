package store

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/yyl1212/math_master/backend/internal/auth"
	"github.com/yyl1212/math_master/backend/internal/catalogue"
	"github.com/yyl1212/math_master/backend/internal/content"
	"github.com/yyl1212/math_master/backend/internal/publication"
	"sort"
	"time"
)

func workflowRowError(err error) error {
	if errors.Is(err, sql.ErrNoRows) {
		return auth.ErrNotFound
	}
	return err
}
func workflowID() (string, error) { return auth.NewID(rand.Reader) }
func workflowSHA(v any) string    { return content.Digest(v) }
func workflowRequestSHA(id string, v any) string {
	return workflowSHA(struct {
		ID    string `json:"id"`
		Input any    `json:"input"`
	}{id, v})
}
func workflowJSONReplay[T any](s *Store, ctx context.Context, tx *sql.Tx, u auth.User, a publication.Access, action publication.Action, id string, input any) (T, bool, error) {
	var out T
	raw, found, err := s.workflowReplay(ctx, tx, u.ID, string(action), a.IdempotencyKey, workflowRequestSHA(id, input))
	if err != nil || !found {
		return out, found, err
	}
	if json.Unmarshal(raw, &out) != nil {
		return out, false, auth.ErrUnavailable
	}
	return out, true, nil
}
func workflowJSONRemember[T any](s *Store, ctx context.Context, tx *sql.Tx, u auth.User, a publication.Access, action publication.Action, id string, input any, out T) error {
	raw, err := json.Marshal(out)
	if err != nil {
		return err
	}
	return s.workflowRemember(ctx, tx, u.ID, string(action), a.IdempotencyKey, workflowRequestSHA(id, input), raw)
}
func workflowEvent(ctx context.Context, tx *sql.Tx, u auth.User, a publication.Access, action publication.Action, kind, id, before, after, beforeState, afterState, reason string, now time.Time) error {
	eventID, err := workflowID()
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO content_workflow_events(id,actor_user_id,action,object_kind,object_id,before_digest,after_digest,before_state,after_state,reason,request_id,created_at) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)`, eventID, u.ID, string(action), kind, id, before, after, beforeState, afterState, reason, a.RequestID, now)
	return err
}
func workflowCatalogue(ctx context.Context, tx *sql.Tx, version int) (catalogue.Catalogue, string, error) {
	var c catalogue.Catalogue
	var raw []byte
	var sha string
	err := tx.QueryRowContext(ctx, `SELECT body,sha256 FROM catalogue_versions WHERE version=$1`, version).Scan(&raw, &sha)
	if err != nil {
		return c, "", workflowRowError(err)
	}
	if json.Unmarshal(raw, &c) != nil || content.Digest(c) != sha {
		return c, "", auth.ErrUnavailable
	}
	return c, sha, nil
}
func workflowAssetViews(p content.Package) []content.AssetView {
	v := make([]content.AssetView, 0, len(p.Assets))
	for _, a := range p.Assets {
		v = append(v, content.AssetView{ID: a.ID, SHA256: a.SHA256, Author: a.Author, License: a.License, Attribution: a.Attribution, Knowledge: a.Knowledge})
	}
	return v
}
func workflowMemoryAssets(b map[string][]byte) content.AssetReader {
	return func(ctx context.Context, a content.Asset) ([]byte, error) {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		raw, ok := b[a.ID]
		if !ok {
			return nil, content.ErrValidation
		}
		return raw, nil
	}
}
func (s *Store) workflowAuthors(ctx context.Context, tx *sql.Tx, owner, base string, p content.Package, inherited []string) ([]string, error) {
	authors := map[string]bool{owner: true}
	for _, id := range inherited {
		authors[id] = true
	}
	addRows := func(q string, args ...any) error {
		rows, err := tx.QueryContext(ctx, q, args...)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var id string
			if err = rows.Scan(&id); err != nil {
				return err
			}
			authors[id] = true
		}
		return rows.Err()
	}
	if base != "" {
		if err := addRows(`SELECT a.user_id::text FROM content_submission_authors a JOIN content_submissions s ON s.id=a.submission_id WHERE s.id=$1 AND s.sealed`, base); err != nil {
			return nil, err
		}
	}
	members := []publication.MemberIdentity{}
	for _, k := range p.Knowledge {
		members = append(members, publication.MemberIdentity{Kind: "knowledge", ID: k.ID, Version: k.Version, SHA256: content.Digest(k)})
	}
	for _, u := range p.Units {
		members = append(members, publication.MemberIdentity{Kind: "unit", ID: u.ID, Version: u.Version, SHA256: content.Digest(u)})
	}
	for _, path := range p.Paths {
		members = append(members, publication.MemberIdentity{Kind: "path", ID: path.ID, Version: path.Version, SHA256: content.Digest(path)})
	}
	for _, a := range p.Assets {
		members = append(members, publication.MemberIdentity{Kind: "asset", ID: a.ID, Version: 1, SHA256: a.SHA256})
	}
	for _, m := range members {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if err := addRows(`SELECT DISTINCT a.user_id::text FROM content_submission_members m JOIN content_submission_authors a ON a.submission_id=m.submission_id JOIN content_submissions s ON s.id=m.submission_id WHERE s.sealed AND m.kind=$1 AND m.id=$2 AND m.version=$3 AND m.sha256=$4`, m.Kind, m.ID, m.Version, m.SHA256); err != nil {
			return nil, err
		}
	}
	out := make([]string, 0, len(authors))
	for id := range authors {
		out = append(out, id)
	}
	sort.Strings(out)
	return out, nil
}
func (s *Store) readWorkflowDraft(ctx context.Context, tx *sql.Tx, u auth.User, id string, ownerOnly bool) (publication.DraftView, string, error) {
	var d publication.DraftView
	var pkg, sources, authors, gate []byte
	var base string
	var created, updated time.Time
	err := tx.QueryRowContext(ctx, `SELECT w.id::text,w.owner_user_id::text,w.catalogue_version,c.sha256,w.revision,w.status,w.package,w.source_map,w.author_ids,w.legacy_unattributed,w.gate,COALESCE(w.base_submission_id::text,''),w.created_at,w.updated_at FROM content_workspaces w JOIN catalogue_versions c ON c.version=w.catalogue_version WHERE w.id=$1`, id).Scan(&d.ID, &d.OwnerID, &d.CatalogueVersion, &d.CatalogueSHA256, &d.Revision, &d.Status, &pkg, &sources, &authors, &d.LegacyUnattributed, &gate, &base, &created, &updated)
	if err != nil {
		return d, base, workflowRowError(err)
	}
	if d.OwnerID != u.ID && (ownerOnly || !publication.HasRole(u, auth.RoleAdmin)) {
		return publication.DraftView{}, base, auth.ErrNotFound
	}
	if len(pkg) > 4<<20 || len(sources) > 512<<10 {
		return d, base, publication.ErrContentLimitExceeded
	}
	for _, pair := range []struct {
		raw []byte
		out any
	}{{pkg, &d.Package}, {sources, &d.SourceMap}, {authors, &d.AuthorIDs}, {gate, &d.Gate}} {
		if json.Unmarshal(pair.raw, pair.out) != nil {
			return d, base, auth.ErrUnavailable
		}
	}
	d.AuthorIDs, err = s.workflowAuthors(ctx, tx, d.OwnerID, base, d.Package, d.AuthorIDs)
	if err != nil {
		return d, base, err
	}
	d.Assets = workflowAssetViews(d.Package)
	d.CreatedAt = created.UTC().Format(time.RFC3339)
	d.UpdatedAt = updated.UTC().Format(time.RFC3339)
	d.Gate.Digest, err = publication.DraftDigest(d)
	return d, base, err
}
func (s *Store) writeWorkflowAssets(ctx context.Context, tx *sql.Tx, id string, p content.Package, bytes map[string][]byte) error {
	if _, err := tx.ExecContext(ctx, `DELETE FROM content_workspace_assets WHERE workspace_id=$1`, id); err != nil {
		return err
	}
	for _, a := range p.Assets {
		if _, err := tx.ExecContext(ctx, `INSERT INTO content_workspace_assets(workspace_id,asset_id,sha256,bytes) VALUES($1,$2,$3,$4)`, id, a.ID, a.SHA256, bytes[a.ID]); err != nil {
			return err
		}
	}
	return nil
}
func workflowNullableID(id string) any {
	if id == "" {
		return nil
	}
	return id
}
func (s *Store) createWorkflowDraft(ctx context.Context, tx *sql.Tx, u auth.User, a publication.Access, input publication.DraftInput, base string, inherited []string, legacy bool, now time.Time) (publication.DraftView, error) {
	var d publication.DraftView
	bytes, err := publication.DraftAssets(input)
	if err != nil {
		return d, err
	}
	c, sha, err := workflowCatalogue(ctx, tx, input.CatalogueVersion)
	if err != nil {
		return d, err
	}
	report, err := content.ValidateEditable(ctx, c, input.Package, workflowMemoryAssets(bytes))
	if err != nil {
		return d, err
	}
	id, err := workflowID()
	if err != nil {
		return d, err
	}
	authors, err := s.workflowAuthors(ctx, tx, u.ID, base, input.Package, inherited)
	if err != nil {
		return d, err
	}
	d = publication.DraftView{ID: id, OwnerID: u.ID, CatalogueVersion: input.CatalogueVersion, CatalogueSHA256: sha, Revision: 1, Status: "editing", Package: input.Package, SourceMap: input.SourceMap, AuthorIDs: authors, LegacyUnattributed: legacy, Assets: workflowAssetViews(input.Package), Gate: publication.GateFromReport(report), CreatedAt: now.UTC().Format(time.RFC3339), UpdatedAt: now.UTC().Format(time.RFC3339)}
	d.Gate.Digest, err = publication.DraftDigest(d)
	if err != nil {
		return d, err
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO content_workspaces(id,owner_user_id,catalogue_version,package,source_map,author_ids,legacy_unattributed,base_submission_id,revision,status,gate,created_at,updated_at) VALUES($1,$2,$3,$4,$5,$6,$7,$8,1,'editing',$9,$10,$10)`, id, u.ID, d.CatalogueVersion, body(d.Package), body(d.SourceMap), body(d.AuthorIDs), legacy, workflowNullableID(base), body(d.Gate), now)
	if err != nil {
		return d, err
	}
	err = s.writeWorkflowAssets(ctx, tx, id, input.Package, bytes)
	return d, err
}
func (s *Store) CreateDraft(ctx context.Context, a publication.Access, input publication.DraftInput) (publication.DraftView, error) {
	var out publication.DraftView
	err := s.workflowTx(ctx, a, publication.CreateDraftAction, nil, func(ctx context.Context, tx *sql.Tx, u auth.User, now time.Time) error {
		prior, found, err := workflowJSONReplay[publication.DraftView](s, ctx, tx, u, a, publication.CreateDraftAction, "", input)
		if err != nil {
			return err
		}
		if found {
			out = prior
			return nil
		}
		out, err = s.createWorkflowDraft(ctx, tx, u, a, input, "", nil, false, now)
		if err != nil {
			return err
		}
		if err = workflowEvent(ctx, tx, u, a, publication.CreateDraftAction, "draft", out.ID, "", out.Gate.Digest, "", "editing:1", "", now); err != nil {
			return err
		}
		return workflowJSONRemember(s, ctx, tx, u, a, publication.CreateDraftAction, "", input, out)
	})
	return out, err
}
func (s *Store) ReadDraft(ctx context.Context, a publication.Access, id string) (publication.DraftView, error) {
	var out publication.DraftView
	if !publication.ValidID(id) {
		return out, auth.ErrInvalidInput
	}
	err := s.workflowReadTx(ctx, a, publication.ReadDraftAction, func(ctx context.Context, tx *sql.Tx, u auth.User) error {
		var err error
		out, _, err = s.readWorkflowDraft(ctx, tx, u, id, false)
		return err
	})
	return out, err
}
func (s *Store) SaveDraft(ctx context.Context, a publication.Access, id string, input publication.SaveDraftInput) (publication.DraftView, error) {
	var out publication.DraftView
	if !publication.ValidID(id) {
		return out, auth.ErrInvalidInput
	}
	err := s.workflowTx(ctx, a, publication.SaveDraftAction, nil, func(ctx context.Context, tx *sql.Tx, u auth.User, now time.Time) error {
		prior, base, err := s.readWorkflowDraft(ctx, tx, u, id, true)
		if err != nil {
			return err
		}
		replay, found, err := workflowJSONReplay[publication.DraftView](s, ctx, tx, u, a, publication.SaveDraftAction, id, input)
		if err != nil {
			return err
		}
		if found {
			out = replay
			return nil
		}
		if prior.Status != "editing" || input.ExpectedRevision != prior.Revision {
			return publication.ErrDraftConflict
		}
		bytes, err := publication.DraftAssets(input.DraftInput)
		if err != nil {
			return err
		}
		c, sha, err := workflowCatalogue(ctx, tx, input.CatalogueVersion)
		if err != nil {
			return err
		}
		report, err := content.ValidateEditable(ctx, c, input.Package, workflowMemoryAssets(bytes))
		if err != nil {
			return err
		}
		authors, err := s.workflowAuthors(ctx, tx, u.ID, base, input.Package, prior.AuthorIDs)
		if err != nil {
			return err
		}
		out = prior
		out.Revision++
		out.Package = input.Package
		out.CatalogueVersion = input.CatalogueVersion
		out.CatalogueSHA256 = sha
		out.SourceMap = input.SourceMap
		out.AuthorIDs = authors
		out.Assets = workflowAssetViews(input.Package)
		out.Gate = publication.GateFromReport(report)
		out.UpdatedAt = now.UTC().Format(time.RFC3339)
		out.Gate.Digest, err = publication.DraftDigest(out)
		if err != nil {
			return err
		}
		result, err := tx.ExecContext(ctx, `UPDATE content_workspaces SET catalogue_version=$2,package=$3,source_map=$4,author_ids=$5,revision=revision+1,gate=$6,updated_at=$7 WHERE id=$1 AND owner_user_id=$8 AND status='editing' AND revision=$9`, id, input.CatalogueVersion, body(input.Package), body(input.SourceMap), body(authors), body(out.Gate), now, u.ID, input.ExpectedRevision)
		if err != nil {
			return err
		}
		n, err := result.RowsAffected()
		if err != nil {
			return err
		}
		if n != 1 {
			return publication.ErrDraftConflict
		}
		if err = s.writeWorkflowAssets(ctx, tx, id, input.Package, bytes); err != nil {
			return err
		}
		if err = workflowEvent(ctx, tx, u, a, publication.SaveDraftAction, "draft", id, prior.Gate.Digest, out.Gate.Digest, fmt.Sprintf("editing:%d", prior.Revision), fmt.Sprintf("editing:%d", out.Revision), "", now); err != nil {
			return err
		}
		return workflowJSONRemember(s, ctx, tx, u, a, publication.SaveDraftAction, id, input, out)
	})
	return out, err
}
func (s *Store) ReadDraftAsset(ctx context.Context, a publication.Access, id, sha string) ([]byte, error) {
	var out []byte
	if !publication.ValidID(id) || !publication.ValidSHA(sha) {
		return nil, auth.ErrInvalidInput
	}
	err := s.workflowReadTx(ctx, a, publication.ReadDraftAssetAction, func(ctx context.Context, tx *sql.Tx, u auth.User) error {
		if _, _, err := s.readWorkflowDraft(ctx, tx, u, id, false); err != nil {
			return err
		}
		return workflowRowError(tx.QueryRowContext(ctx, `SELECT bytes FROM content_workspace_assets WHERE workspace_id=$1 AND sha256=$2 LIMIT 1`, id, sha).Scan(&out))
	})
	return out, err
}
func (s *Store) ListDrafts(ctx context.Context, a publication.Access, q publication.ListQuery) (publication.Page[publication.DraftSummary], error) {
	q, err := publication.ValidateList(q, "editing", "submitted")
	out := publication.Page[publication.DraftSummary]{Items: []publication.DraftSummary{}, Limit: q.Limit, Offset: q.Offset}
	if err != nil {
		return out, err
	}
	err = s.workflowReadTx(ctx, a, publication.ListDraftsAction, func(ctx context.Context, tx *sql.Tx, u auth.User) error {
		all := q.Scope == "all"
		if all && !publication.HasRole(u, auth.RoleAdmin) {
			return auth.ErrForbidden
		}
		where := `WHERE ($1 OR owner_user_id=$2) AND ($3='' OR status=$3)`
		if err := tx.QueryRowContext(ctx, `SELECT count(*) FROM content_workspaces `+where, all, u.ID, q.Status).Scan(&out.Total); err != nil {
			return err
		}
		rows, err := tx.QueryContext(ctx, `SELECT id::text,owner_user_id::text,package->>'id',(package->>'version')::integer,status,catalogue_version,revision,COALESCE((gate->>'structuralTotal')::integer,0),COALESCE((gate->>'completenessTotal')::integer,0),created_at,updated_at FROM content_workspaces `+where+` ORDER BY created_at DESC,id DESC LIMIT $4 OFFSET $5`, all, u.ID, q.Status, q.Limit, q.Offset)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var item publication.DraftSummary
			var created, updated time.Time
			if err = rows.Scan(&item.ID, &item.OwnerID, &item.PackageID, &item.PackageVersion, &item.Status, &item.CatalogueVersion, &item.Revision, &item.StructuralTotal, &item.CompletenessTotal, &created, &updated); err != nil {
				return err
			}
			item.CreatedAt = created.UTC().Format(time.RFC3339)
			item.UpdatedAt = updated.UTC().Format(time.RFC3339)
			out.Items = append(out.Items, item)
		}
		return rows.Err()
	})
	return out, err
}
func workflowImportedInput(ctx context.Context, tx *sql.Tx, pkgID string, version int) (publication.DraftInput, error) {
	var out publication.DraftInput
	var raw []byte
	var size int
	err := tx.QueryRowContext(ctx, `SELECT catalogue_version,octet_length(body::text) FROM imported_packages WHERE id=$1 AND version=$2`, pkgID, version).Scan(&out.CatalogueVersion, &size)
	if err != nil {
		return out, workflowRowError(err)
	}
	if size > 4<<20 {
		return out, publication.ErrContentLimitExceeded
	}
	if err = tx.QueryRowContext(ctx, `SELECT body FROM imported_packages WHERE id=$1 AND version=$2`, pkgID, version).Scan(&raw); err != nil {
		return out, err
	}
	if json.Unmarshal(raw, &out.Package) != nil {
		return out, auth.ErrUnavailable
	}
	out.SourceMap = []publication.SourceLink{}
	out.AssetBytes = []publication.AssetInput{}
	encoded, err := json.Marshal(out.Package)
	if err != nil {
		return out, err
	}
	if len(encoded) > 2<<20 || len(out.Package.Knowledge) > 100 || len(out.Package.Units) > 200 || len(out.Package.Paths) > 20 || len(out.Package.Assets) > 16 {
		return out, publication.ErrContentLimitExceeded
	}
	total := 0
	for _, a := range out.Package.Assets {
		var b []byte
		if err = tx.QueryRowContext(ctx, `SELECT bytes FROM assets WHERE sha256=$1`, a.SHA256).Scan(&b); err != nil {
			return out, err
		}
		total += len(b)
		if total > 4<<20 {
			return out, publication.ErrContentLimitExceeded
		}
		out.AssetBytes = append(out.AssetBytes, publication.AssetInput{ID: a.ID, Base64: base64.StdEncoding.EncodeToString(b)})
	}
	return out, nil
}
func (s *Store) AdoptDraft(ctx context.Context, a publication.Access, input publication.AdoptInput) (publication.DraftView, error) {
	var out publication.DraftView
	if !publication.ValidNote(input.Reason) || input.PackageVersion < 1 || input.PackageVersion > 2147483647 {
		return out, auth.ErrInvalidInput
	}
	err := s.workflowTx(ctx, a, publication.AdoptDraftAction, nil, func(ctx context.Context, tx *sql.Tx, u auth.User, now time.Time) error {
		replay, found, err := workflowJSONReplay[publication.DraftView](s, ctx, tx, u, a, publication.AdoptDraftAction, "", input)
		if err != nil {
			return err
		}
		if found {
			out = replay
			return nil
		}
		in, err := workflowImportedInput(ctx, tx, input.PackageID, input.PackageVersion)
		if err != nil {
			return err
		}
		var base string
		var raw []byte
		legacy := true
		var inherited []string
		err = tx.QueryRowContext(ctx, `SELECT id::text,frozen_body FROM content_submissions WHERE sealed AND package_id=$1 AND package_version=$2 ORDER BY created_at DESC,id DESC LIMIT 1`, input.PackageID, input.PackageVersion).Scan(&base, &raw)
		if err == nil {
			var frozen publication.FrozenBody
			if json.Unmarshal(raw, &frozen) != nil {
				return auth.ErrUnavailable
			}
			in.SourceMap = frozen.SourceMap
			inherited = frozen.AuthorIDs
			legacy = frozen.LegacyUnattributed
		} else if !errors.Is(err, sql.ErrNoRows) {
			return err
		}
		out, err = s.createWorkflowDraft(ctx, tx, u, a, in, base, inherited, legacy, now)
		if err != nil {
			return err
		}
		if err = workflowEvent(ctx, tx, u, a, publication.AdoptDraftAction, "draft", out.ID, "", out.Gate.Digest, "", "editing:1", input.Reason, now); err != nil {
			return err
		}
		return workflowJSONRemember(s, ctx, tx, u, a, publication.AdoptDraftAction, "", input, out)
	})
	return out, err
}
