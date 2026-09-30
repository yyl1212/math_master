package store

import (
	"context"
	"encoding/json"
	"github.com/yyl1212/math_master/backend/internal/catalogue"
	"sort"
	"strings"
)

func has(ids []string, id string) bool {
	for _, v := range ids {
		if v == id {
			return true
		}
	}
	return false
}
func (st *publicationState) summary(d catalogue.Domain) catalogue.DomainSummary {
	v := catalogue.DomainSummary{Domain: d, ContentStatus: "planned"}
	for id, k := range st.knowledge {
		if st.eligible[id] && has(k.DomainIDs, d.ID) {
			v.PublishedKnowledgeCount++
		}
	}
	if v.PublishedKnowledgeCount > 0 {
		v.ContentStatus = "published"
	}
	return v
}

const domainSearch = `catalogue_version=$1 AND (body->>'name' ILIKE $2 ESCAPE '\' OR body->>'nameZh' ILIKE $2 ESCAPE '\' OR EXISTS(SELECT 1 FROM topics t WHERE t.catalogue_version=domains.catalogue_version AND t.domain_id=domains.id AND (t.body->>'name' ILIKE $2 ESCAPE '\' OR t.body->>'nameZh' ILIKE $2 ESCAPE '\')))`

func (s *Store) ListDomains(ctx context.Context, q string, limit, offset int) (items []catalogue.DomainSummary, total int, e error) {
	items = []catalogue.DomainSummary{}
	pattern := "%" + strings.NewReplacer("\\", "\\\\", "%", "\\%", "_", "\\_").Replace(q) + "%"
	e = s.withPublication(ctx, func(st *publicationState) error {
		if e := st.tx.QueryRowContext(st.ctx, "SELECT count(*) FROM domains WHERE "+domainSearch, st.catalogueVersion, pattern).Scan(&total); e != nil {
			return e
		}
		rows, e := st.tx.QueryContext(st.ctx, "SELECT body FROM domains WHERE "+domainSearch+" ORDER BY position LIMIT $3 OFFSET $4", st.catalogueVersion, pattern, limit, offset)
		if e != nil {
			return e
		}
		defer rows.Close()
		for rows.Next() {
			var b []byte
			if e = rows.Scan(&b); e != nil {
				return e
			}
			var d catalogue.Domain
			if e = json.Unmarshal(b, &d); e != nil {
				return e
			}
			items = append(items, st.summary(d))
		}
		return rows.Err()
	})
	return
}
func (s *Store) GetDomain(ctx context.Context, id string) (v catalogue.DomainDetail, e error) {
	e = s.withPublication(ctx, func(st *publicationState) error {
		var b []byte
		if e := st.tx.QueryRowContext(st.ctx, "SELECT body FROM domains WHERE catalogue_version=$1 AND id=$2", st.catalogueVersion, id).Scan(&b); e != nil {
			return rowError(e)
		}
		var d catalogue.Domain
		if e := json.Unmarshal(b, &d); e != nil {
			return e
		}
		v = catalogue.DomainDetail{DomainSummary: st.summary(d), Paths: []catalogue.PathSummary{}}
		for id, p := range st.paths {
			if !has(p.DomainIDs, d.ID) {
				continue
			}
			if _, ok := st.pathView(id); ok {
				v.Paths = append(v.Paths, catalogue.PathSummary{ID: p.ID, Version: p.Version, Title: p.Title, TitleZh: p.TitleZh})
			}
		}
		sort.Slice(v.Paths, func(i, j int) bool { return v.Paths[i].ID < v.Paths[j].ID })
		return nil
	})
	return
}
