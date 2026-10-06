package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"github.com/yyl1212/math_master/backend/internal/auth"
	"github.com/yyl1212/math_master/backend/internal/study"
	"github.com/yyl1212/math_master/backend/internal/taxonomy"
	"sort"
	"strings"
	"time"
	"unicode/utf8"
)

type studyOwned struct {
	record study.StudyRecord
	ref    study.KnowledgeRef
}

func studyOwnedTx(ctx context.Context, tx *sql.Tx, actor string) (map[string]studyOwned, error) {
	out := map[string]studyOwned{}
	rows, e := tx.QueryContext(ctx, "SELECT knowledge_id,body,last_known_ref FROM study_records WHERE owner_user_id=$1", actor)
	if e != nil {
		return out, e
	}
	defer rows.Close()
	for rows.Next() {
		var id string
		var body, ref []byte
		if e = rows.Scan(&id, &body, &ref); e != nil {
			return out, e
		}
		var item studyOwned
		if json.Unmarshal(body, &item.record) != nil || json.Unmarshal(ref, &item.ref) != nil || item.record.KnowledgeID != id || item.ref.ID != id {
			return out, study.ErrNotConfigured
		}
		out[id] = item
	}
	return out, rows.Err()
}
func studyNodesTx(ctx context.Context, tx *sql.Tx, version string) (map[string]taxonomy.TopicNode, error) {
	out := map[string]taxonomy.TopicNode{}
	rows, e := tx.QueryContext(ctx, "SELECT body FROM taxonomy_nodes WHERE taxonomy_version_id=$1", version)
	if e != nil {
		return out, e
	}
	defer rows.Close()
	for rows.Next() {
		var raw []byte
		var n taxonomy.TopicNode
		if e = rows.Scan(&raw); e != nil {
			return out, e
		}
		if json.Unmarshal(raw, &n) != nil || !taxonomy.ValidNode(n) {
			return out, study.ErrNotConfigured
		}
		out[n.ID] = n
	}
	return out, rows.Err()
}
func studyMembership(scope studyScope, nodes map[string]taxonomy.TopicNode) map[string]map[string]bool {
	out := map[string]map[string]bool{}
	for id, k := range scope.knowledge {
		for _, leaf := range k.TopicIDs {
			for topic, depth := leaf, 0; topic != "" && depth < 3; depth++ {
				if out[topic] == nil {
					out[topic] = map[string]bool{}
				}
				out[topic][id] = true
				n, ok := nodes[topic]
				if !ok || n.ParentID == nil {
					break
				}
				topic = *n.ParentID
			}
		}
	}
	return out
}
func studyBaselineTx(ctx context.Context, tx *sql.Tx, actor string, nodes map[string]taxonomy.TopicNode) (map[string]map[string]bool, bool, error) {
	var head string
	e := tx.QueryRowContext(ctx, `SELECT taxonomy_head::text FROM study_events WHERE owner_user_id=$1 AND source_kind='native' AND kind IN ('started','completed','review-started','review-finished') ORDER BY recorded_at,id LIMIT 1`, actor).Scan(&head)
	if e == sql.ErrNoRows {
		return nil, false, nil
	}
	if e != nil {
		return nil, false, e
	}
	scope := studyScope{knowledge: map[string]taxonomy.KnowledgeSummary{}}
	rows, e := tx.QueryContext(ctx, `SELECT a.knowledge_id,a.topic_id FROM taxonomy_release_assignments a JOIN taxonomy_releases r ON r.id=a.release_id JOIN publication_members m ON m.snapshot_id=r.knowledge_publication_id AND m.kind='knowledge' AND m.id=a.knowledge_id AND m.version=a.knowledge_version AND m.availability='active' WHERE a.release_id=$1 ORDER BY a.knowledge_id,a.topic_id`, head)
	if e != nil {
		return nil, false, e
	}
	defer rows.Close()
	for rows.Next() {
		var id, topic string
		if e = rows.Scan(&id, &topic); e != nil {
			return nil, false, e
		}
		k := scope.knowledge[id]
		k.ID = id
		k.TopicIDs = append(k.TopicIDs, topic)
		scope.knowledge[id] = k
	}
	return studyMembership(scope, nodes), true, rows.Err()
}
func studyQuery(q study.ListQuery) (study.ListQuery, error) {
	if q.Limit == 0 {
		q.Limit = 20
	}
	if q.Limit < 1 || q.Limit > 100 || q.Offset < 0 || q.Offset > 100000 || !utf8.ValidString(q.Q) || strings.ContainsRune(q.Q, 0) || len(q.Q) > 512 || (q.State != "" && !study.ValidState(study.State(q.State))) || (q.TopicID != "" && !study.ValidKnowledgeID(q.TopicID)) {
		return q, study.ErrInvalid
	}
	return q, nil
}
func studyOwnedDetail(actor string, id string, scope studyScope, owned map[string]studyOwned) study.StudyDetail {
	r := study.StudyRecord{KnowledgeID: id, State: study.Unlearned}
	item, exists := owned[id]
	if exists {
		r = item.record
	}
	out := studyDetail(actor, r, scope)
	if exists && out.CurrentKnowledge != nil && r.CompletedRef == nil && r.LastReviewRef == nil {
		out.MaterialChanged = item.ref != out.CurrentKnowledge.KnowledgeRef
	}
	return out
}
func studyCompleted(r study.StudyRecord) bool {
	return r.FirstCompletedAt != nil && (r.State == study.Completed || r.State == study.Reviewing)
}
func (s *Store) ReadStudyOverview(ctx context.Context, a study.Access) (study.Overview, error) {
	var out study.Overview
	e := s.studyReadTx(ctx, a, func(ctx context.Context, tx *sql.Tx, u auth.User) error {
		scope, e := studyScopeTx(ctx, tx)
		if e != nil {
			return e
		}
		owned, e := studyOwnedTx(ctx, tx, u.ID)
		if e != nil {
			return e
		}
		out = study.Overview{ActorID: u.ID, Pair: scope.pair, Total: len(scope.knowledge), Reminders: []study.ContentReminder{}}
		if e = tx.QueryRowContext(ctx, "SELECT experience_mode FROM topic_learning_state WHERE singleton").Scan(&out.Mode); e != nil {
			return e
		}
		for id, k := range scope.knowledge {
			r := studyOwnedDetail(u.ID, id, scope, owned)
			if studyCompleted(r.Record) {
				out.Completed++
			}
			if r.Record.State == study.Learning {
				out.Learning++
			}
			if r.Record.State == study.Reviewing {
				out.Reviewing++
			}
			if r.MaterialChanged {
				out.MaterialChanged++
			}
			if len(k.TopicIDs) == 0 {
				out.Unclassified++
			}
		}
		for id, item := range owned {
			if _, ok := scope.knowledge[id]; !ok && item.record.FirstStartedAt != nil {
				out.Unavailable++
			}
		}
		out.Reminders, e = studyRemindersTx(ctx, tx, u.ID, scope, owned)
		return e
	})
	return out, e
}
func (s *Store) ListStudyTopics(ctx context.Context, a study.Access, q study.ListQuery) (study.Page[study.TopicProgress], error) {
	out := study.Page[study.TopicProgress]{Items: []study.TopicProgress{}}
	q, e := studyQuery(q)
	if e != nil {
		return out, e
	}
	e = s.studyReadTx(ctx, a, func(ctx context.Context, tx *sql.Tx, u auth.User) error {
		scope, e := studyScopeTx(ctx, tx)
		if e != nil {
			return e
		}
		nodes, e := studyNodesTx(ctx, tx, scope.pair.TaxonomyVersionID)
		if e != nil {
			return e
		}
		if q.TopicID != "" {
			if _, ok := nodes[q.TopicID]; !ok {
				return auth.ErrNotFound
			}
		}
		owned, e := studyOwnedTx(ctx, tx, u.ID)
		if e != nil {
			return e
		}
		current := studyMembership(scope, nodes)
		baseline, known, e := studyBaselineTx(ctx, tx, u.ID, nodes)
		if e != nil {
			return e
		}
		ids := []string{}
		for id, n := range nodes {
			if q.TopicID == id || q.TopicID == "" && n.Kind == "primary" && n.Level == 1 {
				ids = append(ids, id)
			}
		}
		sort.Strings(ids)
		out.ActorID = u.ID
		out.Total = len(ids)
		out.Limit = q.Limit
		out.Offset = q.Offset
		for i := q.Offset; i < len(ids) && i < q.Offset+q.Limit; i++ {
			id := ids[i]
			p := study.TopicProgress{TopicID: id, Total: len(current[id])}
			for kid := range current[id] {
				r := owned[kid].record
				if studyCompleted(r) {
					p.Completed++
				}
				if r.State == study.Learning {
					p.Learning++
				}
				if r.State == study.Reviewing {
					p.Reviewing++
				}
				if known && !baseline[id][kid] {
					p.Added++
				}
			}
			if known {
				for kid := range baseline[id] {
					if !current[id][kid] {
						p.Removed++
					}
				}
			}
			if p.Total > 0 {
				ratio := float64(p.Completed) / float64(p.Total)
				p.CompletedRatio = &ratio
			}
			out.Items = append(out.Items, p)
		}
		return nil
	})
	return out, e
}
func (s *Store) ListStudyKnowledge(ctx context.Context, a study.Access, q study.ListQuery) (study.Page[study.StudyDetail], error) {
	out := study.Page[study.StudyDetail]{Items: []study.StudyDetail{}}
	q, e := studyQuery(q)
	if e != nil {
		return out, e
	}
	e = s.studyReadTx(ctx, a, func(ctx context.Context, tx *sql.Tx, u auth.User) error {
		scope, e := studyScopeTx(ctx, tx)
		if e != nil {
			return e
		}
		owned, e := studyOwnedTx(ctx, tx, u.ID)
		if e != nil {
			return e
		}
		var members map[string]bool
		if q.TopicID != "" {
			nodes, e := studyNodesTx(ctx, tx, scope.pair.TaxonomyVersionID)
			if e != nil {
				return e
			}
			if _, ok := nodes[q.TopicID]; !ok {
				return auth.ErrNotFound
			}
			members = studyMembership(scope, nodes)[q.TopicID]
		}
		all := map[string]bool{}
		for id := range scope.knowledge {
			all[id] = true
		}
		for id := range owned {
			all[id] = true
		}
		ids := []string{}
		for id := range all {
			item := studyOwnedDetail(u.ID, id, scope, owned)
			if q.ReviewOnly && item.Record.FirstStartedAt == nil {
				continue
			}
			if q.State != "" && string(item.Record.State) != q.State {
				continue
			}
			if q.TopicID != "" && !members[id] {
				continue
			}
			title := id
			if item.CurrentKnowledge != nil {
				title = item.CurrentKnowledge.Title + " " + item.CurrentKnowledge.TitleZh
			}
			if q.Q != "" && !strings.Contains(strings.ToLower(title), strings.ToLower(q.Q)) {
				continue
			}
			ids = append(ids, id)
		}
		sort.Strings(ids)
		out.ActorID = u.ID
		out.Total = len(ids)
		out.Limit = q.Limit
		out.Offset = q.Offset
		for i := q.Offset; i < len(ids) && i < q.Offset+q.Limit; i++ {
			out.Items = append(out.Items, studyOwnedDetail(u.ID, ids[i], scope, owned))
		}
		return nil
	})
	return out, e
}
func (s *Store) ReadStudyKnowledge(ctx context.Context, a study.Access, id string) (study.StudyDetail, error) {
	var out study.StudyDetail
	if !study.ValidKnowledgeID(id) {
		return out, study.ErrInvalid
	}
	e := s.studyReadTx(ctx, a, func(ctx context.Context, tx *sql.Tx, u auth.User) error {
		scope, e := studyScopeTx(ctx, tx)
		if e != nil {
			return e
		}
		owned, e := studyOwnedTx(ctx, tx, u.ID)
		if e != nil {
			return e
		}
		if _, ok := scope.knowledge[id]; !ok {
			if _, own := owned[id]; !own {
				return auth.ErrNotFound
			}
		}
		out = studyOwnedDetail(u.ID, id, scope, owned)
		return nil
	})
	return out, e
}
func studyAcknowledged(r study.StudyRecord, ref *study.KnowledgeRef, at time.Time) bool {
	if ref == nil {
		return false
	}
	return r.CompletedRef != nil && *r.CompletedRef == *ref && r.LastCompletedAt != nil && !r.LastCompletedAt.Before(at) || r.LastReviewRef != nil && *r.LastReviewRef == *ref && r.LastReviewedAt != nil && !r.LastReviewedAt.Before(at)
}
