package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"github.com/yyl1212/math_master/backend/internal/auth"
	k "github.com/yyl1212/math_master/backend/internal/knowledgeadmin"
)

type publishWork struct {
	ID, Token string
	Item      k.PublishItem
}

func (s *Store) StepPublishJob(ctx context.Context, a k.Access, id string, in k.PublishStepInput) (out k.PublishStepReceipt, e error) {
	if !k.ValidPublishJobID(id) {
		return out, k.ErrNotFound
	}
	size, e := in.Normalize()
	if e != nil {
		return out, e
	}
	digest := knowledgeFingerprint(struct {
		Sequence int64
		Size     int
	}{in.ExpectedSequence, size})
	e = s.publishTx(ctx, a, true, true, func(ctx context.Context, tx *sql.Tx, u auth.User) error {
		if found, e := publishReplay(ctx, tx, u.ID, "step", a.IdempotencyKey, id, digest, &out); e != nil || found {
			return e
		}
		j, e := publishJob(ctx, tx, u.ID, id, true)
		if e != nil {
			return e
		}
		if j.Sequence != in.ExpectedSequence {
			return k.ErrStale
		}
		if j.Pending == 0 {
			return k.ErrConflict
		}
		rows, e := tx.QueryContext(ctx, `SELECT knowledge_id,expected_token::text,external_id,title,title_zh FROM managed_knowledge_publish_items WHERE job_id=$1 AND state='pending' ORDER BY ordinal LIMIT $2 FOR UPDATE`, id, size)
		if e != nil {
			return e
		}
		work := []publishWork{}
		for rows.Next() {
			var w publishWork
			if e = rows.Scan(&w.ID, &w.Token, &w.Item.ExternalID, &w.Item.Title, &w.Item.TitleZH); e != nil {
				rows.Close()
				return e
			}
			w.Item.ID = w.ID
			work = append(work, w)
		}
		e = rows.Err()
		rows.Close()
		if e != nil {
			return e
		}
		if len(work) == 0 {
			return k.ErrNotConfigured
		}
		out = k.PublishStepReceipt{ActorID: u.ID, JobID: id, Sequence: j.Sequence + 1, Total: j.Total, Pending: j.Pending, Published: j.Published, Skipped: j.Skipped, Failed: j.Failed, Items: []k.PublishItem{}}
		for _, w := range work {
			var published, deleted bool
			var token string
			e = tx.QueryRowContext(ctx, `SELECT published,deleted_at IS NOT NULL,edit_token::text FROM managed_knowledge WHERE internal_id=$1 FOR UPDATE`, w.ID).Scan(&published, &deleted, &token)
			item := w.Item
			item.State = k.PublishPublished
			switch {
			case errors.Is(e, sql.ErrNoRows):
				item.State = k.PublishFailed
				item.ErrorCode = "NOT_FOUND"
			case e != nil:
				return e
			case deleted:
				item.State = k.PublishFailed
				item.ErrorCode = "DELETED"
			case published:
				item.State = k.PublishSkipped
			case token != w.Token:
				item.State = k.PublishFailed
				item.ErrorCode = "EDIT_TOKEN_STALE"
			default:
				_, e = publishManagedKnowledgeTx(ctx, tx, u, w.ID, w.Token, id)
				if e != nil {
					var invalid *k.DecodeError
					if errors.As(e, &invalid) {
						item.State = k.PublishFailed
						item.ErrorCode = "INVALID_FIELD"
						item.ErrorPath = publishPath(invalid.Path)
					} else if errors.Is(e, k.ErrInvalid) {
						item.State = k.PublishFailed
						item.ErrorCode = "INVALID_FIELD"
					} else {
						return e
					}
				}
			}
			if _, e = tx.ExecContext(ctx, `UPDATE managed_knowledge_publish_items SET state=$3,error_code=$4,error_path=$5 WHERE job_id=$1 AND knowledge_id=$2 AND state='pending'`, id, item.ID, item.State, item.ErrorCode, item.ErrorPath); e != nil {
				return e
			}
			out.Pending--
			switch item.State {
			case k.PublishPublished:
				out.Published++
			case k.PublishSkipped:
				out.Skipped++
			case k.PublishFailed:
				out.Failed++
			}
			out.Items = append(out.Items, item)
		}
		if _, e = tx.ExecContext(ctx, `UPDATE managed_knowledge_publish_jobs SET sequence=$3,pending=$4,published=$5,skipped=$6,failed=$7 WHERE job_id=$1 AND owner_user_id=$2`, id, u.ID, out.Sequence, out.Pending, out.Published, out.Skipped, out.Failed); e != nil {
			return e
		}
		return publishReceipt(ctx, tx, u.ID, "step", a.IdempotencyKey, id, digest, out)
	})
	return
}
func publishPath(path string) string {
	out := ""
	for _, r := range path {
		if len(out)+len(string(r)) > 512 {
			break
		}
		out += string(r)
	}
	return out
}
func (s *Store) ReadPublishJobReceipt(ctx context.Context, a k.Access, id, key string) (out *k.PublishStepReceipt, e error) {
	if !k.ValidPublishJobID(id) {
		return nil, k.ErrNotFound
	}
	if key == "" || len(key) > 128 {
		return nil, k.ErrInvalid
	}
	e = s.publishTx(ctx, a, false, false, func(ctx context.Context, tx *sql.Tx, u auth.User) error {
		if _, e := publishJob(ctx, tx, u.ID, id, false); e != nil {
			return e
		}
		var b []byte
		e := tx.QueryRowContext(ctx, `SELECT receipt FROM managed_knowledge_publish_receipts WHERE owner_user_id=$1 AND action='step' AND key=$2 AND job_id=$3`, u.ID, key, id).Scan(&b)
		if errors.Is(e, sql.ErrNoRows) {
			return nil
		}
		if e != nil {
			return e
		}
		var receipt k.PublishStepReceipt
		if e = json.Unmarshal(b, &receipt); e != nil {
			return k.ErrNotConfigured
		}
		if receipt.ActorID != u.ID || receipt.JobID != id {
			return k.ErrNotConfigured
		}
		out = &receipt
		return nil
	})
	return
}

var _ k.PublishRepository = (*Store)(nil)
