package store

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/yyl1212/math_master/backend/internal/content"
	"io"
	"net"
	"sort"
	"strings"
	"time"
)

type publicationState struct {
	tx               *sql.Tx
	ctx              context.Context
	catalogueVersion int
	knowledge        map[string]content.Knowledge
	units            []content.Unit
	paths            map[string]content.Path
	assets           map[string]content.Asset
	eligible         map[string]bool
}

func readError(e error) error {
	if e == nil {
		return nil
	}
	var n net.Error
	var pe *pgconn.PgError
	if errors.As(e, &n) || errors.Is(e, context.DeadlineExceeded) || errors.Is(e, driver.ErrBadConn) || errors.Is(e, sql.ErrConnDone) || errors.Is(e, io.EOF) || (errors.As(e, &pe) && (strings.HasPrefix(pe.Code, "08") || pe.Code == "57P01" || pe.Code == "57P02" || pe.Code == "57P03")) {
		return ErrUnavailable
	}
	return e
}
func (s *Store) withPublication(ctx context.Context, fn func(*publicationState) error) error {
	ctx, c := context.WithTimeout(ctx, 3*time.Second)
	defer c()
	tx, e := s.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true, Isolation: sql.LevelRepeatableRead})
	if e != nil {
		return readError(e)
	}
	defer tx.Rollback()
	st := &publicationState{tx: tx, ctx: ctx, knowledge: map[string]content.Knowledge{}, paths: map[string]content.Path{}, assets: map[string]content.Asset{}, eligible: map[string]bool{}, units: []content.Unit{}}
	var head string
	e = tx.QueryRowContext(ctx, "SELECT s.id,s.catalogue_version FROM publication_heads h JOIN publication_snapshots s ON s.id=h.snapshot_id WHERE h.singleton AND s.status='published'").Scan(&head, &st.catalogueVersion)
	if errors.Is(e, sql.ErrNoRows) {
		e = tx.QueryRowContext(ctx, "SELECT COALESCE(MAX(version),0) FROM catalogue_versions").Scan(&st.catalogueVersion)
	} else if e == nil {
		load := func(q string, accept func([]byte) error) error {
			rows, e := tx.QueryContext(ctx, q, head)
			if e != nil {
				return e
			}
			defer rows.Close()
			for rows.Next() {
				var b []byte
				if e = rows.Scan(&b); e != nil {
					return e
				}
				if e = accept(b); e != nil {
					return e
				}
			}
			return rows.Err()
		}
		e = load("SELECT k.body FROM publication_members m JOIN knowledge_versions k ON k.id=m.id AND k.version=m.version WHERE m.snapshot_id=$1 AND m.kind='knowledge' AND m.availability='active' ORDER BY m.id", func(b []byte) error {
			var k content.Knowledge
			e := json.Unmarshal(b, &k)
			st.knowledge[k.ID] = k
			return e
		})
		if e == nil {
			e = load("SELECT u.body FROM publication_members m JOIN unit_versions u ON u.id=m.id AND u.version=m.version WHERE m.snapshot_id=$1 AND m.kind='unit' AND m.availability='active' ORDER BY m.id", func(b []byte) error {
				var u content.Unit
				e := json.Unmarshal(b, &u)
				st.units = append(st.units, u)
				return e
			})
		}
		if e == nil {
			e = load("SELECT p.body FROM publication_members m JOIN path_versions p ON p.id=m.id AND p.version=m.version WHERE m.snapshot_id=$1 AND m.kind='path' AND m.availability='active' ORDER BY m.id", func(b []byte) error { var p content.Path; e := json.Unmarshal(b, &p); st.paths[p.ID] = p; return e })
		}
		if e == nil {
			e = load("SELECT a.value FROM publication_members m JOIN imported_packages p ON p.id=m.package_id AND p.version=m.package_version CROSS JOIN LATERAL jsonb_array_elements(p.body->'assets') a(value) WHERE m.snapshot_id=$1 AND m.kind='asset' AND m.availability='active' AND a.value->>'id'=m.id ORDER BY m.id", func(b []byte) error { var a content.Asset; e := json.Unmarshal(b, &a); st.assets[a.ID] = a; return e })
		}
		colors := map[string]int{}
		var valid func(string) bool
		valid = func(id string) bool {
			if colors[id] == 1 {
				return false
			}
			if colors[id] == 2 {
				return st.eligible[id]
			}
			k, ok := st.knowledge[id]
			if !ok {
				return false
			}
			colors[id] = 1
			ok = true
			for _, r := range k.Relations {
				if r.Kind == "prerequisite" {
					target, present := st.knowledge[r.Target.ID]
					if !present || target.Version != r.Target.Version || !valid(r.Target.ID) {
						ok = false
					}
				}
			}
			colors[id] = 2
			st.eligible[id] = ok
			return ok
		}
		for id := range st.knowledge {
			valid(id)
		}
	}
	if e != nil {
		return readError(e)
	}
	if e = fn(st); e != nil {
		return readError(e)
	}
	return readError(tx.Commit())
}
func (st *publicationState) knowledgeView(id string) (content.KnowledgeView, bool) {
	k, ok := st.knowledge[id]
	if !ok || !st.eligible[id] {
		return content.KnowledgeView{}, false
	}
	rels := []content.Relation{}
	for _, r := range k.Relations {
		target, ok := st.knowledge[r.Target.ID]
		if ok && st.eligible[r.Target.ID] && target.Version == r.Target.Version {
			rels = append(rels, r)
		}
	}
	k.Relations = rels
	v := content.KnowledgeView{Knowledge: k, Units: []content.Unit{}, Assets: []content.AssetView{}}
	assets := map[string]bool{}
	ids := []string{}
	for id, a := range st.assets {
		if a.Knowledge.ID == k.ID && a.Knowledge.Version == k.Version {
			assets[id] = true
			ids = append(ids, id)
		}
	}
	sort.Strings(ids)
	for _, id := range ids {
		a := st.assets[id]
		v.Assets = append(v.Assets, content.AssetView{ID: a.ID, SHA256: a.SHA256, Author: a.Author, License: a.License, Attribution: a.Attribution, Knowledge: a.Knowledge})
	}
	for _, u := range st.units {
		if u.Knowledge.ID != k.ID || u.Knowledge.Version != k.Version {
			continue
		}
		valid := true
		for _, id := range u.AssetIDs {
			if !assets[id] {
				valid = false
			}
		}
		if valid {
			v.Units = append(v.Units, u)
		}
	}
	return v, true
}
func (st *publicationState) pathView(id string) (content.PathView, bool) {
	p, ok := st.paths[id]
	if !ok {
		return content.PathView{}, false
	}
	v := content.PathView{Path: p, Knowledge: []content.KnowledgeView{}}
	nodes := map[string]int{}
	for _, n := range p.Nodes {
		nodes[n.ID] = n.Version
	}
	for _, n := range p.Nodes {
		k, ok := st.knowledgeView(n.ID)
		if !ok || k.Knowledge.Version != n.Version {
			return content.PathView{}, false
		}
		for _, r := range k.Knowledge.Relations {
			if r.Kind == "prerequisite" && nodes[r.Target.ID] != r.Target.Version {
				return content.PathView{}, false
			}
		}
		v.Knowledge = append(v.Knowledge, k)
	}
	return v, true
}
func (s *Store) GetPublishedKnowledge(ctx context.Context, id string) (v content.KnowledgeView, e error) {
	e = s.withPublication(ctx, func(st *publicationState) error {
		var ok bool
		v, ok = st.knowledgeView(id)
		if !ok {
			return ErrNotFound
		}
		return nil
	})
	return
}
func (s *Store) GetPublishedPath(ctx context.Context, id string) (v content.PathView, e error) {
	e = s.withPublication(ctx, func(st *publicationState) error {
		var ok bool
		v, ok = st.pathView(id)
		if !ok {
			return ErrNotFound
		}
		return nil
	})
	return
}
