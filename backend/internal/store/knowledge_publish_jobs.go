package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"github.com/yyl1212/math_master/backend/internal/auth"
	k "github.com/yyl1212/math_master/backend/internal/knowledgeadmin"
	"github.com/yyl1212/math_master/backend/internal/pagination"
	"sort"
	"time"
)

// Publish transactions preserve the existing identity -> knowledge lock order.
// Only a positively acknowledged rollback before COMMIT may be retried with smaller input.
func (s *Store) publishTx(ctx context.Context, a k.Access, write, step bool, fn func(context.Context, *sql.Tx, auth.User) error) (err error) {
	ctx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()
	opts := &sql.TxOptions{}
	if !write {
		opts.ReadOnly = true
		opts.Isolation = sql.LevelRepeatableRead
	}
	tx, e := s.db.BeginTx(ctx, opts)
	if e != nil {
		return knowledgeError(e)
	}
	committing := false
	defer func() {
		rolled := tx.Rollback()
		if err != nil && step && !committing && rolled == nil && !publishDomainError(err) {
			err = k.ErrPublishRolledBack
		}
	}()
	if _, err = tx.ExecContext(ctx, "SET LOCAL lock_timeout='1s'"); err != nil {
		return
	}
	if write {
		for _, id := range []int64{adminLockID, 1296127048} {
			if _, err = tx.ExecContext(ctx, "SELECT pg_advisory_xact_lock_shared($1)", id); err != nil {
				return
			}
		}
	}
	u, e := knowledgeIdentity(ctx, tx, a, write, write, true)
	if e != nil {
		return knowledgeError(e)
	}
	if err = knowledgeConfigured(ctx, tx); err != nil {
		return knowledgeError(err)
	}
	if err = publishSchemaIntegrity(ctx, tx); err != nil {
		return knowledgeError(err)
	}
	if write {
		if _, err = tx.ExecContext(ctx, "SELECT pg_advisory_xact_lock($1)", knowledgeLockID); err != nil {
			return
		}
	}
	if err = fn(ctx, tx, u); err != nil {
		return
	}
	if _, err = knowledgeIdentity(ctx, tx, a, write, false, true); err != nil {
		return knowledgeError(err)
	}
	committing = true
	if err = tx.Commit(); err != nil {
		return k.ErrPublishInterrupted
	}
	return nil
}
func publishDomainError(e error) bool {
	for _, v := range []error{k.ErrInvalid, k.ErrConflict, k.ErrStale, k.ErrIdempotency, k.ErrNotConfigured, k.ErrNotFound, auth.ErrForbidden, auth.ErrAuthenticationRequired, auth.ErrPasswordChangeRequired, auth.ErrCSRF} {
		if errors.Is(e, v) {
			return true
		}
	}
	return false
}

const publishJobColumns = "owner_user_id::text,job_id::text,mode,sequence,total,pending,published,skipped,failed,created_at"

func scanPublishJob(row knowledgeScanner) (j k.PublishJob, e error) {
	e = row.Scan(&j.ActorID, &j.ID, &j.Mode, &j.Sequence, &j.Total, &j.Pending, &j.Published, &j.Skipped, &j.Failed, &j.CreatedAt)
	j.CreatedAt = j.CreatedAt.UTC()
	return
}
func publishJob(ctx context.Context, tx *sql.Tx, owner, id string, lock bool) (k.PublishJob, error) {
	q := "SELECT " + publishJobColumns + " FROM managed_knowledge_publish_jobs WHERE owner_user_id=$1 AND job_id=$2"
	if lock {
		q += " FOR UPDATE"
	}
	j, e := scanPublishJob(tx.QueryRowContext(ctx, q, owner, id))
	return j, knowledgeError(e)
}
func publishReplay(ctx context.Context, tx *sql.Tx, owner, action, key, id, digest string, out any) (bool, error) {
	var storedID, sha string
	var b []byte
	e := tx.QueryRowContext(ctx, "SELECT job_id::text,input_sha256,receipt FROM managed_knowledge_publish_receipts WHERE owner_user_id=$1 AND action=$2 AND key=$3", owner, action, key).Scan(&storedID, &sha, &b)
	if errors.Is(e, sql.ErrNoRows) {
		return false, nil
	}
	if e != nil {
		return false, e
	}
	if sha != digest || id != "" && storedID != id {
		return false, k.ErrIdempotency
	}
	if e = json.Unmarshal(b, out); e != nil {
		return false, k.ErrNotConfigured
	}
	return true, nil
}
func publishReceipt(ctx context.Context, tx *sql.Tx, owner, action, key, id, digest string, out any) error {
	_, e := tx.ExecContext(ctx, `INSERT INTO managed_knowledge_publish_receipts(owner_user_id,action,key,job_id,input_sha256,receipt) VALUES($1,$2,$3,$4,$5,$6)`, owner, action, key, id, digest, knowledgeJSON(out))
	return e
}
func (s *Store) CreatePublishJob(ctx context.Context, a k.Access, in k.PublishCreateInput) (out k.PublishJob, e error) {
	if e = in.Validate(); e != nil {
		return
	}
	in.Items = append([]k.PublishSelection(nil), in.Items...)
	sort.Slice(in.Items, func(i, j int) bool { return in.Items[i].ID < in.Items[j].ID })
	digest := knowledgeFingerprint(in)
	e = s.publishTx(ctx, a, true, false, func(ctx context.Context, tx *sql.Tx, u auth.User) error {
		if found, e := publishReplay(ctx, tx, u.ID, "create", a.IdempotencyKey, "", digest, &out); e != nil || found {
			return e
		}
		var total, failed int64
		if in.Mode == k.PublishAll {
			if e := tx.QueryRowContext(ctx, "SELECT count(*) FROM managed_knowledge WHERE NOT published AND deleted_at IS NULL").Scan(&total); e != nil {
				return e
			}
		} else {
			total = int64(len(in.Items))
			if e := tx.QueryRowContext(ctx, `SELECT count(*) FROM jsonb_to_recordset($1) AS x(id text,"editToken" text) LEFT JOIN managed_knowledge m ON m.internal_id=x.id WHERE m.internal_id IS NULL`, knowledgeJSON(in.Items)).Scan(&failed); e != nil {
				return e
			}
		}
		var e error
		out, e = scanPublishJob(tx.QueryRowContext(ctx, `INSERT INTO managed_knowledge_publish_jobs(owner_user_id,mode,key,input_sha256,total,pending,failed) VALUES($1,$2,$3,$4,$5,$5::bigint-$6::bigint,$6) RETURNING `+publishJobColumns, u.ID, in.Mode, a.IdempotencyKey, digest, total, failed))
		if e != nil {
			return e
		}
		if in.Mode == k.PublishAll {
			_, e = tx.ExecContext(ctx, `INSERT INTO managed_knowledge_publish_items(job_id,knowledge_id,expected_token,ordinal,external_id,title,title_zh) SELECT $1,internal_id,edit_token,row_number() OVER(ORDER BY internal_id)-1,external_id,left(coalesce(point->>'title',''),128),left(coalesce(point->>'title_zh',''),128) FROM managed_knowledge WHERE NOT published AND deleted_at IS NULL`, out.ID)
		} else {
			_, e = tx.ExecContext(ctx, `INSERT INTO managed_knowledge_publish_items(job_id,knowledge_id,expected_token,ordinal,external_id,title,title_zh,state,error_code) SELECT $1,x.id,x."editToken"::uuid,row_number() OVER(ORDER BY x.id)-1,coalesce(m.external_id,''),left(coalesce(m.point->>'title',''),128),left(coalesce(m.point->>'title_zh',''),128),CASE WHEN m.internal_id IS NULL THEN 'failed' ELSE 'pending' END,CASE WHEN m.internal_id IS NULL THEN 'NOT_FOUND' ELSE '' END FROM jsonb_to_recordset($2) AS x(id text,"editToken" text) LEFT JOIN managed_knowledge m ON m.internal_id=x.id`, out.ID, knowledgeJSON(in.Items))
		}
		if e != nil {
			return e
		}
		return publishReceipt(ctx, tx, u.ID, "create", a.IdempotencyKey, out.ID, digest, out)
	})
	return
}
func publishPage(limit int, offset int64) (int, error) {
	if limit == 0 {
		limit = 20
	}
	if limit < 1 || limit > 100 || offset < 0 || offset > k.MaxPublishInteger {
		return 0, k.ErrInvalid
	}
	return limit, nil
}
func (s *Store) ListPublishJobs(ctx context.Context, a k.Access, q k.PublishJobQuery) (out pagination.ActorPage[k.PublishJob], e error) {
	q.Limit, e = publishPage(q.Limit, q.Offset)
	if e != nil || len(q.CreateKey) > 128 {
		if e == nil {
			e = k.ErrInvalid
		}
		return
	}
	out.Items = []k.PublishJob{}
	out.Limit = q.Limit
	e = s.publishTx(ctx, a, false, false, func(ctx context.Context, tx *sql.Tx, u auth.User) error {
		out.ActorID = u.ID
		if e := tx.QueryRowContext(ctx, `SELECT count(*) FROM managed_knowledge_publish_jobs WHERE owner_user_id=$1 AND ($2='' OR key=$2)`, u.ID, q.CreateKey).Scan(&out.Total); e != nil {
			return e
		}
		meta, _ := pagination.Window(out.Total, q.Offset/int64(q.Limit)+1, q.Limit)
		out.Meta = meta
		rows, e := tx.QueryContext(ctx, "SELECT "+publishJobColumns+" FROM managed_knowledge_publish_jobs WHERE owner_user_id=$1 AND ($2='' OR key=$2) ORDER BY created_at DESC,job_id LIMIT $3 OFFSET $4", u.ID, q.CreateKey, q.Limit, out.Offset)
		if e != nil {
			return e
		}
		defer rows.Close()
		for rows.Next() {
			j, e := scanPublishJob(rows)
			if e != nil {
				return e
			}
			out.Items = append(out.Items, j)
		}
		return rows.Err()
	})
	return
}
func (s *Store) ReadPublishJob(ctx context.Context, a k.Access, id string, q k.PublishItemQuery) (out k.PublishJobDetail, e error) {
	if !k.ValidPublishJobID(id) {
		return out, k.ErrNotFound
	}
	q.Limit, e = publishPage(q.Limit, q.Offset)
	if e != nil {
		return
	}
	if q.State != "" && q.State != k.PublishPending && q.State != k.PublishPublished && q.State != k.PublishSkipped && q.State != k.PublishFailed {
		return out, k.ErrInvalid
	}
	out.Items = []k.PublishItem{}
	out.Limit = q.Limit
	e = s.publishTx(ctx, a, false, false, func(ctx context.Context, tx *sql.Tx, u auth.User) error {
		var e error
		out.Job, e = publishJob(ctx, tx, u.ID, id, false)
		if e != nil {
			return e
		}
		if e = tx.QueryRowContext(ctx, `SELECT count(*) FROM managed_knowledge_publish_items WHERE job_id=$1 AND ($2='' OR state=$2)`, id, q.State).Scan(&out.Total); e != nil {
			return e
		}
		meta, _ := pagination.Window(out.Total, q.Offset/int64(q.Limit)+1, q.Limit)
		out.Meta = meta
		rows, e := tx.QueryContext(ctx, `SELECT knowledge_id,external_id,title,title_zh,state,error_code,error_path FROM managed_knowledge_publish_items WHERE job_id=$1 AND ($2='' OR state=$2) ORDER BY ordinal LIMIT $3 OFFSET $4`, id, q.State, q.Limit, out.Offset)
		if e != nil {
			return e
		}
		defer rows.Close()
		for rows.Next() {
			var v k.PublishItem
			if e = rows.Scan(&v.ID, &v.ExternalID, &v.Title, &v.TitleZH, &v.State, &v.ErrorCode, &v.ErrorPath); e != nil {
				return e
			}
			out.Items = append(out.Items, v)
		}
		return rows.Err()
	})
	return
}
