package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"github.com/yyl1212/math_master/backend/internal/knowledgeadmin"
	"strings"
	"time"
)

var _ knowledgeadmin.Repository = (*Store)(nil)
var _ knowledgeadmin.CurrentRepository = (*Store)(nil)

func (s *Store) currentTx(ctx context.Context, fn func(context.Context, *sql.Tx) error) error {
	ctx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()
	tx, e := s.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true, Isolation: sql.LevelRepeatableRead})
	if e != nil {
		return knowledgeError(e)
	}
	defer tx.Rollback()
	if e = knowledgeConfigured(ctx, tx); e != nil {
		return knowledgeError(e)
	}
	var mode string
	var marker, once bool
	e = tx.QueryRowContext(ctx, `SELECT content_mode,enabled_once,(SELECT managed_knowledge_enabled FROM goose_db_version WHERE version_id=0) FROM knowledge_admin_state WHERE singleton`).Scan(&mode, &once, &marker)
	if e != nil || !once || !marker || mode != "managed" {
		return knowledgeadmin.ErrNotConfigured
	}
	if e = fn(ctx, tx); e != nil {
		return knowledgeError(e)
	}
	return knowledgeError(tx.Commit())
}
func managedPublic(k knowledgeadmin.Knowledge) knowledgeadmin.PublicKnowledge {
	return knowledgeadmin.PublicKnowledge{ID: k.ID, ExternalID: k.ExternalID, TopicKeys: k.TopicKeys, Point: knowledgeadmin.PublicProjection(k.Point), Sources: k.Sources, Ref: k.Ref, UpdatedAt: k.UpdatedAt}
}
func (s *Store) ReadCurrentKnowledge(ctx context.Context, id string) (out knowledgeadmin.PublicKnowledge, e error) {
	e = s.currentTx(ctx, func(ctx context.Context, tx *sql.Tx) error {
		k, e := readManaged(ctx, tx, id, false)
		if e != nil {
			return e
		}
		if !k.Published || k.Deleted {
			return knowledgeadmin.ErrNotFound
		}
		out = managedPublic(k)
		return nil
	})
	return
}
func topicPrefix(k string) string {
	k = strings.TrimPrefix(k, "msc-")
	k = strings.ToUpper(k)
	if strings.HasSuffix(k, "-XX") {
		return strings.TrimSuffix(k, "-XX")
	}
	if strings.HasSuffix(k, "XX") {
		return strings.TrimSuffix(k, "XX")
	}
	if k == "PROJECT:OTHER" || k == "PROJECT-OTHER" {
		return "project:other"
	}
	return k
}
func listCurrent(ctx context.Context, tx *sql.Tx, q knowledgeadmin.Query, owners ...string) (out knowledgeadmin.Page[knowledgeadmin.PublicKnowledge], e error) {
	q, e = normalizeKnowledgeQuery(q)
	if e != nil {
		return
	}
	q.TopicKey = topicPrefix(q.TopicKey)
	out = knowledgeadmin.Page[knowledgeadmin.PublicKnowledge]{Items: []knowledgeadmin.PublicKnowledge{}, Limit: q.Limit, Offset: q.Offset}
	filter := managedFilter + " AND k.published AND k.deleted_at IS NULL"
	args := []any{q.TopicKey, q.Q, q.Type, q.Difficulty}
	if len(owners) > 0 {
		filter += ` AND ($6='' OR coalesce((SELECT state FROM managed_study_records r WHERE r.owner_user_id=$5 AND r.knowledge_id=k.internal_id),'unlearned')=$6) AND (NOT $7 OR EXISTS(SELECT 1 FROM managed_study_records r WHERE r.owner_user_id=$5 AND r.knowledge_id=k.internal_id AND r.state IN ('completed','reviewing')))`
		args = append(args, owners[0], q.State, q.ReviewOnly)
	}
	if e = tx.QueryRowContext(ctx, "SELECT count(*) FROM managed_knowledge k WHERE "+filter, args...).Scan(&out.Total); e != nil {
		return
	}
	rows, e := tx.QueryContext(ctx, "SELECT "+managedColumns+" FROM managed_knowledge k WHERE "+filter+fmt.Sprintf(" ORDER BY updated_at DESC,internal_id LIMIT $%d OFFSET $%d", len(args)+1, len(args)+2), append(args, q.Limit, q.Offset)...)
	if e != nil {
		return
	}
	defer rows.Close()
	for rows.Next() {
		k, err := scanManaged(rows)
		if err != nil {
			return out, err
		}
		item := managedPublic(k)
		for field := range item.Point {
			switch field {
			case "title", "title_zh", "type", "type_reason", "type_other_reason", "learning_difficulty", "classification_mode", "msc_codes", "classification_evidence", "project_other", "tags":
			default:
				delete(item.Point, field)
			}
		}
		out.Items = append(out.Items, item)
	}
	if e = rows.Err(); e != nil {
		return
	}
	if e = rows.Close(); e != nil {
		return
	}
	for n := range out.Items {
		out.Items[n].TopicKeys, e = managedTopics(ctx, tx, out.Items[n].ID)
		if e != nil {
			return
		}
	}
	return
}
func (s *Store) ListCurrentKnowledge(ctx context.Context, q knowledgeadmin.Query) (out knowledgeadmin.Page[knowledgeadmin.PublicKnowledge], e error) {
	if _, e = normalizeKnowledgeQuery(q); e != nil {
		return
	}
	e = s.currentTx(ctx, func(ctx context.Context, tx *sql.Tx) error { var e error; out, e = listCurrent(ctx, tx, q); return e })
	return
}
func currentTopicCount(ctx context.Context, tx *sql.Tx, key string) (int, error) {
	var n int
	e := tx.QueryRowContext(ctx, `SELECT count(DISTINCT k.internal_id) FROM managed_knowledge k JOIN managed_knowledge_topics t ON t.internal_id=k.internal_id AND t.active WHERE k.published AND k.deleted_at IS NULL AND (t.topic_key=$1 OR t.topic_key LIKE $1||'%')`, topicPrefix(key)).Scan(&n)
	return n, e
}
func scanCurrentTopic(row knowledgeScanner) (out knowledgeadmin.CurrentTopic, e error) {
	var id, kind string
	var level int
	e = row.Scan(&id, &out.TopicKey, &out.Title, &out.TitleEn, &kind, &level)
	if e != nil {
		return
	}
	out.Kind = kind
	if kind == "primary" {
		switch level {
		case 1:
			out.Kind = "primary"
		case 2:
			out.Kind = "secondary"
		case 3:
			out.Kind = "specific"
		}
	}
	out.Title, out.TitleEn, e = knowledgeadmin.DirectoryTitles(out.TopicKey, out.TitleEn, out.Title, out.Kind)
	out.Items = knowledgeadmin.Page[knowledgeadmin.PublicKnowledge]{Items: []knowledgeadmin.PublicKnowledge{}, Limit: 20}
	return
}

const currentTopicSelect = `SELECT n.id,n.code,coalesce(nullif(n.body->>'nameZh',''),n.body->>'name',n.code),coalesce(nullif(n.body->>'name',''),nullif(n.body->>'nameZh',''),n.code),n.kind,n.level FROM taxonomy_nodes n JOIN taxonomy_heads h ON h.singleton JOIN taxonomy_releases r ON r.id=h.release_id AND r.status='published' AND r.taxonomy_version_id=n.taxonomy_version_id`

func (s *Store) ListCurrentTopics(ctx context.Context, q knowledgeadmin.Query) (out knowledgeadmin.Page[knowledgeadmin.CurrentTopic], e error) {
	q, e = normalizeKnowledgeQuery(q)
	if e != nil {
		return
	}
	out = knowledgeadmin.Page[knowledgeadmin.CurrentTopic]{Items: []knowledgeadmin.CurrentTopic{}, Limit: q.Limit, Offset: q.Offset}
	e = s.currentTx(ctx, func(ctx context.Context, tx *sql.Tx) error {
		prefix := topicPrefix(q.TopicKey)
		names, e := knowledgeadmin.DirectoryChineseSearch(q.Q)
		if e != nil {
			return e
		}
		namesJSON := knowledgeJSON(names)
		level := 1
		if len(prefix) == 2 {
			level = 2
		} else if len(prefix) == 3 {
			level = 3
		}
		filter := ` WHERE n.kind<>'auxiliary' AND n.level=$1 AND ($2='' OR n.code LIKE $2||'%') AND ($3='' OR n.body->>'nameZh' ILIKE '%'||$3||'%' OR n.body->>'name' ILIKE '%'||$3||'%' OR EXISTS(SELECT 1 FROM taxonomy_nodes hit WHERE hit.taxonomy_version_id=n.taxonomy_version_id AND ($4::jsonb->upper(hit.code)->>'english')=hit.body->>'name' AND ($4::jsonb->upper(hit.code)->>'kind')=(CASE WHEN hit.kind='primary' THEN CASE hit.level WHEN 1 THEN 'primary' WHEN 2 THEN 'secondary' WHEN 3 THEN 'specific' END ELSE hit.kind END) AND (hit.id=n.id OR (n.level=1 AND left(hit.code,2)=left(n.code,2)) OR (n.level=2 AND left(hit.code,3)=left(n.code,3)))))`
		var n int
		if e := tx.QueryRowContext(ctx, `SELECT count(*) FROM taxonomy_nodes n JOIN taxonomy_heads h ON h.singleton JOIN taxonomy_releases r ON r.id=h.release_id AND r.status='published' AND r.taxonomy_version_id=n.taxonomy_version_id`+filter, level, prefix, q.Q, namesJSON).Scan(&n); e != nil {
			return e
		}
		out.Total = n
		if prefix == "" {
			out.Total++
		}
		rows, e := tx.QueryContext(ctx, currentTopicSelect+filter+" ORDER BY n.code LIMIT $5 OFFSET $6", level, prefix, q.Q, namesJSON, q.Limit, q.Offset)
		if e != nil {
			return e
		}
		for rows.Next() {
			item, e := scanCurrentTopic(rows)
			if e != nil {
				rows.Close()
				return e
			}
			out.Items = append(out.Items, item)
		}
		e = rows.Err()
		rows.Close()
		if e != nil {
			return e
		}
		if prefix == "" && q.Offset+len(out.Items) >= n && len(out.Items) < q.Limit {
			out.Items = append(out.Items, knowledgeadmin.CurrentTopic{TopicKey: "project:other", Title: "项目其他", TitleEn: "Project other", Kind: "project-other", Items: knowledgeadmin.Page[knowledgeadmin.PublicKnowledge]{Items: []knowledgeadmin.PublicKnowledge{}, Limit: 20}})
		}
		for i := range out.Items {
			out.Items[i].KnowledgeCount, e = currentTopicCount(ctx, tx, out.Items[i].TopicKey)
			if e != nil {
				return e
			}
		}
		return nil
	})
	return
}
func (s *Store) ReadCurrentTopic(ctx context.Context, key string, q knowledgeadmin.Query) (out knowledgeadmin.CurrentTopic, e error) {
	e = s.currentTx(ctx, func(ctx context.Context, tx *sql.Tx) error {
		if topicPrefix(key) == "project:other" {
			out = knowledgeadmin.CurrentTopic{TopicKey: "project:other", Title: "项目其他", TitleEn: "Project other", Kind: "project-other"}
		} else {
			var e error
			out, e = scanCurrentTopic(tx.QueryRowContext(ctx, currentTopicSelect+" WHERE n.kind<>'auxiliary' AND (n.id=$1 OR upper(n.code)=upper($1))", key))
			if errors.Is(e, sql.ErrNoRows) {
				return knowledgeadmin.ErrNotFound
			}
			if e != nil {
				return e
			}
		}
		q.TopicKey = out.TopicKey
		var e error
		out.Items, e = listCurrent(ctx, tx, q)
		if e != nil {
			return e
		}
		out.KnowledgeCount = out.Items.Total
		return nil
	})
	return
}
