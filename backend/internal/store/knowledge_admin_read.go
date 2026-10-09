package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"github.com/yyl1212/math_master/backend/internal/auth"
	"github.com/yyl1212/math_master/backend/internal/knowledgeadmin"
)

const managedColumns = `internal_id,external_id,point,public_sources,content_sha256,published,(deleted_at IS NOT NULL),edit_token::text,updated_at`

type knowledgeScanner interface{ Scan(...any) error }

func scanManaged(row knowledgeScanner) (k knowledgeadmin.Knowledge, e error) {
	var point, sources []byte
	var sha string
	e = row.Scan(&k.ID, &k.ExternalID, &point, &sources, &sha, &k.Published, &k.Deleted, &k.EditToken, &k.UpdatedAt)
	if e != nil {
		return
	}
	if e = json.Unmarshal(point, &k.Point); e != nil {
		return
	}
	e = json.Unmarshal(sources, &k.Sources)
	k.Ref = knowledgeadmin.Ref{ID: k.ID, ContentSHA256: sha, SourceKind: "managed"}
	return
}
func readManaged(ctx context.Context, tx *sql.Tx, id string, lock bool) (knowledgeadmin.Knowledge, error) {
	q := "SELECT " + managedColumns + " FROM managed_knowledge WHERE internal_id=$1"
	if lock {
		q += " FOR UPDATE"
	}
	k, e := scanManaged(tx.QueryRowContext(ctx, q, id))
	if e == nil {
		k.TopicKeys, e = managedTopics(ctx, tx, id)
	}
	return k, e
}
func (s *Store) ReadManagedKnowledge(ctx context.Context, a knowledgeadmin.Access, id string) (out knowledgeadmin.Knowledge, e error) {
	e = s.knowledgeTx(ctx, a, false, true, func(ctx context.Context, tx *sql.Tx, _ auth.User) error {
		var e error
		out, e = readManaged(ctx, tx, id, false)
		return e
	})
	return
}
func normalizeKnowledgeQuery(q knowledgeadmin.Query) (knowledgeadmin.Query, error) {
	if q.Limit == 0 {
		q.Limit = 20
	}
	if q.Limit < 1 || q.Limit > 100 || q.Offset < 0 || q.Offset > 1000000 || len(q.Q) > 256 || q.Difficulty < 0 || q.Difficulty > 5 {
		return q, knowledgeadmin.ErrInvalid
	}
	if q.Status != "" && q.Status != "published" && q.Status != "unpublished" && q.Status != "deleted" {
		return q, knowledgeadmin.ErrInvalid
	}
	return q, nil
}

const managedFilter = `($1='' OR EXISTS(SELECT 1 FROM managed_knowledge_topics t WHERE t.internal_id=k.internal_id AND t.active AND (t.topic_key=$1 OR t.topic_key LIKE $1||'%'))) AND ($2='' OR k.point->>'title' ILIKE '%'||$2||'%' OR k.point->>'title_zh' ILIKE '%'||$2||'%' OR k.external_id ILIKE '%'||$2||'%') AND ($3='' OR k.point->>'type'=$3) AND ($4=0 OR (k.point#>>'{learning_difficulty,difficulty_level}')::int=$4)`

func (s *Store) ListManagedKnowledge(ctx context.Context, a knowledgeadmin.Access, q knowledgeadmin.Query) (out knowledgeadmin.Page[knowledgeadmin.Knowledge], e error) {
	q, e = normalizeKnowledgeQuery(q)
	if e != nil {
		return
	}
	out = knowledgeadmin.Page[knowledgeadmin.Knowledge]{Items: []knowledgeadmin.Knowledge{}, Limit: q.Limit, Offset: q.Offset}
	e = s.knowledgeTx(ctx, a, false, true, func(ctx context.Context, tx *sql.Tx, _ auth.User) error {
		filter := managedFilter + ` AND (($5='' AND k.deleted_at IS NULL) OR ($5='published' AND k.published AND k.deleted_at IS NULL) OR ($5='unpublished' AND NOT k.published AND k.deleted_at IS NULL) OR ($5='deleted' AND k.deleted_at IS NOT NULL))`
		args := []any{q.TopicKey, q.Q, q.Type, q.Difficulty, q.Status}
		if e := tx.QueryRowContext(ctx, "SELECT count(*) FROM managed_knowledge k WHERE "+filter, args...).Scan(&out.Total); e != nil {
			return e
		}
		rows, e := tx.QueryContext(ctx, "SELECT "+managedColumns+" FROM managed_knowledge k WHERE "+filter+" ORDER BY updated_at DESC,internal_id LIMIT $6 OFFSET $7", append(args, q.Limit, q.Offset)...)
		if e != nil {
			return e
		}
		defer rows.Close()
		for rows.Next() {
			k, e := scanManaged(rows)
			if e != nil {
				return e
			}
			k.Point.Statement = ""
			k.Point.Proof = ""
			k.Point.Examples = []string{}
			k.Point.Explanations = []knowledgeadmin.Explanation{}
			k.Point.OriginalBinding = nil
			k.Point.Extensions = map[string]any{}
			out.Items = append(out.Items, k)
		}
		if e = rows.Err(); e != nil {
			return e
		}
		if e = rows.Close(); e != nil {
			return e
		}
		for n := range out.Items {
			out.Items[n].TopicKeys, e = managedTopics(ctx, tx, out.Items[n].ID)
			if e != nil {
				return e
			}
		}
		return nil
	})
	return
}
