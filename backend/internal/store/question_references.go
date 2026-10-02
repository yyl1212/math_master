package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"github.com/yyl1212/math_master/backend/internal/auth"
	"github.com/yyl1212/math_master/backend/internal/content"
	"github.com/yyl1212/math_master/backend/internal/question"
	"sort"
)

func questionReferences(ctx context.Context, tx *sql.Tx, input question.DraftInput) (question.ReferenceSnapshot, error) {
	refs := question.ReferenceSnapshot{CatalogueVersion: input.CatalogueVersion, Knowledge: []question.FixedKnowledge{}, Units: []question.FixedUnit{}, Assets: []content.AssetView{}}
	_, sha, err := workflowCatalogue(ctx, tx, input.CatalogueVersion)
	if err != nil {
		return refs, err
	}
	refs.CatalogueSHA256 = sha
	var head string
	var catalogueVersion int
	err = tx.QueryRowContext(ctx, `SELECT s.id,s.catalogue_version FROM publication_heads h JOIN publication_snapshots s ON s.id=h.snapshot_id AND s.status='published' WHERE h.singleton`).Scan(&head, &catalogueVersion)
	if errors.Is(err, sql.ErrNoRows) {
		return refs, nil
	}
	if err != nil {
		return refs, err
	}
	refs.KnowledgeHead = &head
	if catalogueVersion != input.CatalogueVersion {
		return refs, nil
	}
	knowledge := map[question.Ref]bool{}
	units := map[question.Ref]bool{}
	assets := map[string]bool{}
	add := func(k question.Ref, coverage []question.ObjectiveCoverage, us []question.Ref, as []question.AssetRef) {
		knowledge[k] = true
		for _, c := range coverage {
			knowledge[c.Knowledge] = true
		}
		for _, u := range us {
			units[u] = true
		}
		for _, a := range as {
			assets[a.ID] = true
		}
	}
	for _, t := range input.QuestionPackage.Templates {
		add(t.Knowledge, t.Coverage, t.Units, t.Assets)
	}
	for _, i := range input.QuestionPackage.FixedQuestions {
		add(i.Body.Knowledge, i.Body.Coverage, i.Body.Units, i.Body.Assets)
	}
	for _, b := range input.QuestionPackage.Blueprints {
		knowledge[b.Knowledge] = true
	}
	ks := []question.Ref{}
	for k := range knowledge {
		ks = append(ks, k)
	}
	sort.Slice(ks, func(i, j int) bool {
		if ks[i].ID != ks[j].ID {
			return ks[i].ID < ks[j].ID
		}
		return ks[i].Version < ks[j].Version
	})
	rows, err := tx.QueryContext(ctx, `WITH RECURSIVE wanted(id,version) AS (
 SELECT id,version FROM jsonb_to_recordset($2::jsonb) AS r(id text,version integer)
 UNION SELECT r->'target'->>'id',(r->'target'->>'version')::integer FROM wanted w JOIN knowledge_versions k ON k.id=w.id AND k.version=w.version JOIN publication_members m ON m.snapshot_id=$1 AND m.kind='knowledge' AND m.id=k.id AND m.version=k.version AND m.availability='active' CROSS JOIN LATERAL jsonb_array_elements(k.body->'relations') r WHERE r->>'kind'='prerequisite'
 ) SELECT k.id,k.version,k.sha256,k.body->>'title',k.body->>'titleZh',k.body->'objectives',k.body->'relations' FROM wanted w JOIN knowledge_versions k ON k.id=w.id AND k.version=w.version JOIN publication_members m ON m.snapshot_id=$1 AND m.kind='knowledge' AND m.id=k.id AND m.version=k.version AND m.availability='active' WHERE NOT EXISTS(SELECT 1 FROM content_withdrawals cw WHERE cw.kind='knowledge' AND cw.target_id=k.id AND cw.target_version=k.version) ORDER BY k.id,k.version`, head, body(ks))
	if err != nil {
		return refs, err
	}
	known := map[question.Ref]question.FixedKnowledge{}
	relations := map[question.Ref][]content.Relation{}
	for rows.Next() {
		var k question.FixedKnowledge
		var objectives, rel []byte
		if err = rows.Scan(&k.Identity.ID, &k.Identity.Version, &k.Identity.SHA256, &k.Title, &k.TitleZh, &objectives, &rel); err != nil {
			rows.Close()
			return refs, err
		}
		if json.Unmarshal(objectives, &k.Objectives) != nil {
			rows.Close()
			return refs, auth.ErrUnavailable
		}
		var rs []content.Relation
		if json.Unmarshal(rel, &rs) != nil {
			rows.Close()
			return refs, auth.ErrUnavailable
		}
		ref := question.Ref{ID: k.Identity.ID, Version: k.Identity.Version}
		known[ref] = k
		relations[ref] = rs
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return refs, err
	}
	colors := map[question.Ref]int{}
	eligible := map[question.Ref]bool{}
	var valid func(question.Ref) bool
	valid = func(ref question.Ref) bool {
		if colors[ref] == 1 {
			return false
		}
		if colors[ref] == 2 {
			return eligible[ref]
		}
		if _, ok := known[ref]; !ok {
			return false
		}
		colors[ref] = 1
		ok := true
		for _, r := range relations[ref] {
			if r.Kind == "prerequisite" && !valid(r.Target) {
				ok = false
			}
		}
		colors[ref] = 2
		eligible[ref] = ok
		return ok
	}
	for _, k := range ks {
		if valid(k) {
			refs.Knowledge = append(refs.Knowledge, known[k])
		}
	}
	us := []question.Ref{}
	for u := range units {
		us = append(us, u)
	}
	rows, err = tx.QueryContext(ctx, `SELECT u.id,u.version,u.sha256,u.knowledge_id,u.knowledge_version,u.body->'assetIds' FROM jsonb_to_recordset($2::jsonb) r(id text,version integer) JOIN unit_versions u ON u.id=r.id AND u.version=r.version JOIN publication_members m ON m.snapshot_id=$1 AND m.kind='unit' AND m.id=u.id AND m.version=u.version AND m.availability='active'
 WHERE NOT EXISTS(SELECT 1 FROM content_withdrawals cw WHERE cw.kind='unit' AND cw.target_id=u.id AND cw.target_version=u.version)
 AND NOT EXISTS(SELECT 1 FROM jsonb_array_elements_text(u.body->'assetIds') a LEFT JOIN unit_asset_bindings b ON b.unit_id=u.id AND b.unit_version=u.version AND b.asset_id=a LEFT JOIN publication_members am ON am.snapshot_id=$1 AND am.kind='asset' AND am.id=a AND am.availability='active' LEFT JOIN package_members pm ON pm.package_id=am.package_id AND pm.package_version=am.package_version AND pm.kind='asset' AND pm.id=a WHERE b.asset_sha256 IS DISTINCT FROM pm.asset_sha256 OR pm.asset_sha256 IS NULL OR EXISTS(SELECT 1 FROM content_withdrawals cw WHERE cw.kind='asset' AND cw.sha256=pm.asset_sha256)) ORDER BY u.id,u.version`, head, body(us))
	if err != nil {
		return refs, err
	}
	for rows.Next() {
		var u question.FixedUnit
		var ids []byte
		if err = rows.Scan(&u.Identity.ID, &u.Identity.Version, &u.Identity.SHA256, &u.Knowledge.ID, &u.Knowledge.Version, &ids); err != nil {
			rows.Close()
			return refs, err
		}
		if json.Unmarshal(ids, &u.AssetIDs) != nil {
			rows.Close()
			return refs, auth.ErrUnavailable
		}
		if valid(u.Knowledge) {
			refs.Units = append(refs.Units, u)
		}
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return refs, err
	}
	assetIDs := []string{}
	for id := range assets {
		assetIDs = append(assetIDs, id)
	}
	rows, err = tx.QueryContext(ctx, `SELECT a.value FROM publication_members m JOIN imported_packages p ON p.id=m.package_id AND p.version=m.package_version CROSS JOIN LATERAL jsonb_array_elements(p.body->'assets') a(value) WHERE m.snapshot_id=$1 AND m.kind='asset' AND m.availability='active' AND m.id=ANY($2::text[]) AND a.value->>'id'=m.id AND NOT EXISTS(SELECT 1 FROM content_withdrawals cw WHERE cw.kind='asset' AND cw.sha256=a.value->>'sha256') ORDER BY m.id`, head, assetIDs)
	if err != nil {
		return refs, err
	}
	for rows.Next() {
		var raw []byte
		if err = rows.Scan(&raw); err != nil {
			rows.Close()
			return refs, err
		}
		var a content.Asset
		if json.Unmarshal(raw, &a) != nil {
			rows.Close()
			return refs, auth.ErrUnavailable
		}
		if valid(a.Knowledge) {
			refs.Assets = append(refs.Assets, content.AssetView{ID: a.ID, SHA256: a.SHA256, Author: a.Author, License: a.License, Attribution: a.Attribution, Knowledge: a.Knowledge})
		}
	}
	err = rows.Err()
	rows.Close()
	return refs, err
}
