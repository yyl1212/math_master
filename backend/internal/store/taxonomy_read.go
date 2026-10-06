package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"github.com/yyl1212/math_master/backend/internal/taxonomy"
	"sort"
	"strings"
)

type taxonomyState struct {
	pair      taxonomy.PairRef
	nodes     map[string]taxonomy.TopicNode
	knowledge map[string]taxonomy.KnowledgeSummary
	counts    map[string]map[string]bool
}

func loadTaxonomyState(st *publicationState) (*taxonomyState, error) {
	if e := taxonomyConfigured(st.ctx, st.tx); e != nil {
		return nil, e
	}
	out := &taxonomyState{nodes: map[string]taxonomy.TopicNode{}, knowledge: map[string]taxonomy.KnowledgeSummary{}, counts: map[string]map[string]bool{}}
	var release, version string
	var kh sql.NullString
	e := st.tx.QueryRowContext(st.ctx, "SELECT r.id::text,r.taxonomy_version_id,r.knowledge_publication_id FROM taxonomy_heads h JOIN taxonomy_releases r ON r.id=h.release_id AND r.status='published' WHERE h.singleton").Scan(&release, &version, &kh)
	if errors.Is(e, sql.ErrNoRows) {
		return nil, taxonomy.ErrNotConfigured
	}
	if e != nil {
		return nil, e
	}
	var current sql.NullString
	e = st.tx.QueryRowContext(st.ctx, "SELECT snapshot_id FROM publication_heads WHERE singleton").Scan(&current)
	if e != nil && !errors.Is(e, sql.ErrNoRows) {
		return nil, e
	}
	if kh.Valid != current.Valid || (kh.Valid && kh.String != current.String) {
		return nil, taxonomy.ErrHeadStale
	}
	out.pair = taxonomy.PairRef{TaxonomyHead: &release, TaxonomyVersionID: version}
	if kh.Valid {
		v := kh.String
		out.pair.KnowledgeHead = &v
	}
	rows, e := st.tx.QueryContext(st.ctx, "SELECT body FROM taxonomy_nodes WHERE taxonomy_version_id=$1", version)
	if e != nil {
		return nil, e
	}
	var nodes []taxonomy.TopicNode
	for rows.Next() {
		var b []byte
		var n taxonomy.TopicNode
		if e = rows.Scan(&b); e != nil {
			rows.Close()
			return nil, e
		}
		if e = json.Unmarshal(b, &n); e != nil {
			rows.Close()
			return nil, e
		}
		nodes = append(nodes, n)
		out.nodes[n.ID] = n
	}
	e = rows.Err()
	rows.Close()
	if e != nil {
		return nil, e
	}
	if e = taxonomy.ValidateCatalogue(nodes); e != nil {
		return nil, e
	}
	rows, e = st.tx.QueryContext(st.ctx, "SELECT a.knowledge_id,a.knowledge_version,a.knowledge_sha,a.topic_id,k.sha256 FROM taxonomy_release_assignments a JOIN knowledge_versions k ON k.id=a.knowledge_id AND k.version=a.knowledge_version WHERE a.release_id=$1", release)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	for rows.Next() {
		var id, sha, topic, actualSHA string
		var version int
		if e = rows.Scan(&id, &version, &sha, &topic, &actualSHA); e != nil {
			return nil, e
		}
		k, exists := st.knowledge[id]
		if !exists || !st.eligible[id] || k.Version != version || sha != actualSHA {
			continue
		}
		if _, exists = out.nodes[topic]; !exists {
			return nil, taxonomy.ErrInvalid
		}
		item, exists := out.knowledge[id]
		if !exists {
			item = taxonomy.KnowledgeSummary{KnowledgeRef: taxonomy.KnowledgeRef{ID: id, Version: version, SHA256: sha}, Title: k.Title, TitleZh: k.TitleZh, TopicIDs: []string{}}
		}
		item.TopicIDs = append(item.TopicIDs, topic)
		out.knowledge[id] = item
		current := topic
		for depth := 0; depth < 3; depth++ {
			if out.counts[current] == nil {
				out.counts[current] = map[string]bool{}
			}
			out.counts[current][id] = true
			n := out.nodes[current]
			if n.ParentID == nil {
				break
			}
			current = *n.ParentID
		}
	}
	if e = rows.Err(); e != nil {
		return nil, e
	}
	for id, k := range out.knowledge {
		sort.Strings(k.TopicIDs)
		out.knowledge[id] = k
	}
	return out, nil
}
func (s *taxonomyState) summary(n taxonomy.TopicNode) taxonomy.TopicSummary {
	out := taxonomy.TopicSummary{TopicNode: n, Ancestors: []taxonomy.TopicNode{}, PublishedKnowledgeCount: len(s.counts[n.ID])}
	parent := n.ParentID
	for parent != nil {
		p := s.nodes[*parent]
		out.Ancestors = append([]taxonomy.TopicNode{p}, out.Ancestors...)
		parent = p.ParentID
	}
	for _, other := range s.nodes {
		if other.ParentID != nil && *other.ParentID == n.ID {
			out.HasChildren = true
			break
		}
	}
	return out
}
func topicQuery(q taxonomy.Query) (taxonomy.Query, error) {
	if !taxonomy.ValidText(q.Q) || len(q.Q) > 512 || q.Limit < 0 || q.Limit > 100 || q.Offset < 0 || q.Offset > 100000 || q.Level < 0 || q.Level > 3 {
		return q, taxonomy.ErrInvalid
	}
	if q.ParentID != "" && !strings.HasPrefix(q.ParentID, "msc-") {
		return q, taxonomy.ErrInvalid
	}
	if q.Kind == "" {
		q.Kind = "primary"
	}
	if q.Kind != "primary" && q.Kind != "auxiliary" && q.Kind != "other" {
		return q, taxonomy.ErrInvalid
	}
	if q.Limit == 0 {
		q.Limit = 20
	}
	if q.Level == 0 && q.ParentID == "" && q.Q == "" {
		q.Level = 1
		if q.Kind == "auxiliary" {
			q.Level = 2
		} else if q.Kind == "other" {
			q.Level = 3
		}
	}
	return q, nil
}
func (s *Store) ListTopics(ctx context.Context, q taxonomy.Query) (taxonomy.Page[taxonomy.TopicSummary], error) {
	out := taxonomy.Page[taxonomy.TopicSummary]{Items: []taxonomy.TopicSummary{}}
	q, e := topicQuery(q)
	if e != nil {
		return out, e
	}
	out.Limit = q.Limit
	out.Offset = q.Offset
	e = s.withPublication(ctx, func(st *publicationState) error {
		state, e := loadTaxonomyState(st)
		if e != nil {
			return e
		}
		out.Pair = state.pair
		if q.ParentID != "" {
			if _, ok := state.nodes[q.ParentID]; !ok {
				return ErrNotFound
			}
		}
		var found []taxonomy.TopicNode
		needle := strings.ToLower(q.Q)
		titleMatches := map[string]bool{}
		if needle != "" {
			for _, k := range state.knowledge {
				if strings.Contains(strings.ToLower(k.Title+" "+k.TitleZh), needle) {
					for _, id := range k.TopicIDs {
						for depth := 0; depth < 3; depth++ {
							titleMatches[id] = true
							n := state.nodes[id]
							if n.ParentID == nil {
								break
							}
							id = *n.ParentID
						}
					}
				}
			}
		}

		for _, n := range state.nodes {
			if n.Kind != q.Kind || (q.Level != 0 && n.Level != q.Level) || (q.ParentID != "" && (n.ParentID == nil || *n.ParentID != q.ParentID)) {
				continue
			}
			if needle != "" && !titleMatches[n.ID] && !strings.Contains(strings.ToLower(n.Code+" "+n.Name+" "+n.NameZh), needle) {
				continue
			}
			found = append(found, n)
		}
		sort.Slice(found, func(i, j int) bool { return found[i].Code < found[j].Code })
		out.Total = len(found)
		start := q.Offset
		if start > len(found) {
			start = len(found)
		}
		end := start + q.Limit
		if end > len(found) {
			end = len(found)
		}
		for _, n := range found[start:end] {
			out.Items = append(out.Items, state.summary(n))
		}
		return nil
	})
	return out, taxonomyError(e)
}
func (s *Store) ReadTopic(ctx context.Context, id string) (taxonomy.TopicDetail, error) {
	var out taxonomy.TopicDetail
	e := s.withPublication(ctx, func(st *publicationState) error {
		state, e := loadTaxonomyState(st)
		if e != nil {
			return e
		}
		n, ok := state.nodes[id]
		if !ok {
			return ErrNotFound
		}
		out = taxonomy.TopicDetail{Summary: state.summary(n), Pair: state.pair}
		return nil
	})
	return out, taxonomyError(e)
}
func (s *Store) ListTopicKnowledge(ctx context.Context, id string, q taxonomy.Query) (taxonomy.Page[taxonomy.KnowledgeSummary], error) {
	out := taxonomy.Page[taxonomy.KnowledgeSummary]{Items: []taxonomy.KnowledgeSummary{}}
	q, e := topicQuery(q)
	if e != nil {
		return out, e
	}
	out.Limit = q.Limit
	out.Offset = q.Offset
	e = s.withPublication(ctx, func(st *publicationState) error {
		state, e := loadTaxonomyState(st)
		if e != nil {
			return e
		}
		out.Pair = state.pair
		if _, ok := state.nodes[id]; !ok {
			return ErrNotFound
		}
		var found []taxonomy.KnowledgeSummary
		needle := strings.ToLower(q.Q)
		for kid := range state.counts[id] {
			k := state.knowledge[kid]
			if needle == "" || strings.Contains(strings.ToLower(k.Title+" "+k.TitleZh+" "+k.ID), needle) {
				found = append(found, k)
			}
		}
		sort.Slice(found, func(i, j int) bool { return found[i].ID < found[j].ID })
		out.Total = len(found)
		start := q.Offset
		if start > len(found) {
			start = len(found)
		}
		end := start + q.Limit
		if end > len(found) {
			end = len(found)
		}
		out.Items = append(out.Items, found[start:end]...)
		return nil
	})
	return out, taxonomyError(e)
}
