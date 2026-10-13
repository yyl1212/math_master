package store

import (
	"context"
	"database/sql"
	"github.com/yyl1212/math_master/backend/internal/auth"
	"github.com/yyl1212/math_master/backend/internal/knowledgeadmin"
	"github.com/yyl1212/math_master/backend/internal/notesharing"
	"github.com/yyl1212/math_master/backend/internal/study"
	"strings"
)

func (s *Store) Submit(ctx context.Context, a knowledgeadmin.Access, kid string, in notesharing.SubmitInput) (out notesharing.Receipt, e error) {
	if notesharing.ValidateSubmit(in) != nil {
		return out, notesharing.ErrInvalid
	}
	e = s.knowledgeTx(ctx, a, true, false, func(ctx context.Context, tx *sql.Tx, u auth.User) error {
		if e := sharingConfigured(ctx, tx); e != nil {
			return e
		}
		k, n, e := shareTarget(ctx, tx, u.ID, kid, true)
		if e != nil {
			return e
		}
		if got, replay, e := shareReplay(ctx, tx, u.ID, a, "submit", kid, in); e != nil {
			return e
		} else if replay {
			out = got
			return nil
		}
		if !k.Published || k.Deleted || n.Deleted || notesharing.ValidateBody(n.Body) != nil {
			return notesharing.ErrInvalid
		}
		if _, e = tx.ExecContext(ctx, `INSERT INTO note_share_heads(owner_user_id,knowledge_id) VALUES($1,$2) ON CONFLICT DO NOTHING`, u.ID, kid); e != nil {
			return e
		}
		h, e := readShareHead(ctx, tx, u.ID, kid, true)
		if e != nil {
			return e
		}
		if h.Status == notesharing.Pending || h.Status == notesharing.Shared {
			return notesharing.ErrLocked
		}
		if h.Sequence != in.ExpectedSequence || n.Revision != in.ExpectedRevision {
			return notesharing.ErrStale
		}
		id, e := shareID()
		if e != nil {
			return e
		}
		now, e := dbClock(ctx, tx)
		if e != nil {
			return e
		}
		_, e = tx.ExecContext(ctx, `INSERT INTO note_share_submissions(submission_id,owner_user_id,knowledge_id,note_revision,note_knowledge_ref,title,body,body_sha256,knowledge_title,knowledge_title_zh,topic_keys,submitted_at) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)`, id, u.ID, kid, n.Revision, knowledgeJSON(n.Knowledge), strings.TrimSpace(in.Title), n.Body, knowledgeadmin.DigestBytes([]byte(n.Body)), k.Point.Title, k.Point.TitleZH, knowledgeJSON(append([]string{}, k.TopicKeys...)), now)
		if e != nil {
			return e
		}
		h.Status = notesharing.Pending
		h.Sequence++
		h.SubmissionID = &id
		h.ApprovedAt = nil
		out, e = writeShareDecision(ctx, tx, u.ID, a, u.ID, kid, "submit", kid, "", in, h, now)
		return e
	})
	return
}
func (s *Store) Withdraw(ctx context.Context, a knowledgeadmin.Access, kid string, in notesharing.WithdrawInput) (out notesharing.Receipt, e error) {
	if notesharing.ValidateWithdraw(in) != nil {
		return out, notesharing.ErrInvalid
	}
	e = s.knowledgeTx(ctx, a, true, false, func(ctx context.Context, tx *sql.Tx, u auth.User) error {
		if e := sharingConfigured(ctx, tx); e != nil {
			return e
		}
		if _, _, e := shareTarget(ctx, tx, u.ID, kid, true); e != nil {
			return e
		}
		if got, replay, e := shareReplay(ctx, tx, u.ID, a, "withdraw", kid, in); e != nil {
			return e
		} else if replay {
			out = got
			return nil
		}
		h, e := readShareHead(ctx, tx, u.ID, kid, true)
		if e != nil {
			return e
		}
		if h.Sequence != in.ExpectedSequence || h.SubmissionID == nil || *h.SubmissionID != in.SubmissionID || (h.Status != notesharing.Pending && h.Status != notesharing.Shared) {
			return notesharing.ErrStale
		}
		now, e := dbClock(ctx, tx)
		if e != nil {
			return e
		}
		h.Sequence++
		h.Status = notesharing.Withdrawn
		h.ApprovedAt = nil
		out, e = writeShareDecision(ctx, tx, u.ID, a, u.ID, kid, "withdraw", kid, "", in, h, now)
		return e
	})
	return
}
func (s *Store) Decide(ctx context.Context, a knowledgeadmin.Access, id, action string, in notesharing.DecisionInput) (out notesharing.Receipt, e error) {
	if !study.ValidID(id) || notesharing.ValidateDecision(action, in) != nil {
		return out, notesharing.ErrInvalid
	}
	e = s.knowledgeTx(ctx, a, true, true, func(ctx context.Context, tx *sql.Tx, u auth.User) error {
		if e := sharingConfigured(ctx, tx); e != nil {
			return e
		}
		snap, e := readShareSnapshot(ctx, tx, id)
		if e != nil {
			return e
		}
		k, n, e := shareTarget(ctx, tx, snap.OwnerID, snap.KnowledgeID, true)
		if e != nil {
			return e
		}
		if got, replay, e := shareReplay(ctx, tx, u.ID, a, action, id, in); e != nil {
			return e
		} else if replay {
			out = got
			return nil
		}
		h, e := readShareHead(ctx, tx, snap.OwnerID, snap.KnowledgeID, true)
		if e != nil {
			return e
		}
		if h.Sequence != in.ExpectedSequence || h.SubmissionID == nil || *h.SubmissionID != id {
			return notesharing.ErrStale
		}
		if action == "hide" {
			if h.Status != notesharing.Shared {
				return notesharing.ErrStale
			}
			h.Status = notesharing.Hidden
		} else {
			if h.Status != notesharing.Pending {
				return notesharing.ErrStale
			}
			if action == "approve" {
				if !k.Published || k.Deleted || n.Deleted || n.Revision != snap.NoteRevision || n.Knowledge == nil || *n.Knowledge != snap.NoteKnowledgeRef || knowledgeadmin.DigestBytes([]byte(n.Body)) != snap.BodySHA {
					return notesharing.ErrStale
				}
				h.Status = notesharing.Shared
			} else {
				h.Status = notesharing.Returned
			}
		}
		now, e := dbClock(ctx, tx)
		if e != nil {
			return e
		}
		h.Sequence++
		h.ApprovedAt = nil
		if h.Status == notesharing.Shared {
			h.ApprovedAt = &now
		}
		out, e = writeShareDecision(ctx, tx, u.ID, a, snap.OwnerID, snap.KnowledgeID, action, id, strings.TrimSpace(in.Reason), in, h, now)
		return e
	})
	return
}
