package store

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/json"
	"errors"
	"github.com/yyl1212/math_master/backend/internal/auth"
	"github.com/yyl1212/math_master/backend/internal/feedback"
	"github.com/yyl1212/math_master/backend/internal/question"
	"time"
)

type feedbackTicket struct {
	Metadata   feedback.Metadata
	Owner      string
	Source     feedback.Source
	Resolution *feedback.Resolution
}

func feedbackLoadTicket(ctx context.Context, tx *sql.Tx, id string, lock bool) (feedbackTicket, error) {
	t := feedbackTicket{}
	if !question.ValidID(id) {
		return t, auth.ErrInvalidInput
	}
	var target, source, res []byte
	suffix := ""
	if lock {
		suffix = " FOR UPDATE"
	}
	e := tx.QueryRowContext(ctx, `SELECT id,owner_user_id,target,source,category,status,sequence,created_at,updated_at,current_resolution FROM feedback_tickets WHERE id=$1`+suffix, id).Scan(&t.Metadata.ID, &t.Owner, &target, &source, &t.Metadata.Category, &t.Metadata.Status, &t.Metadata.Sequence, &t.Metadata.CreatedAt, &t.Metadata.UpdatedAt, &res)
	if errors.Is(e, sql.ErrNoRows) {
		return t, auth.ErrNotFound
	}
	if e != nil {
		return t, e
	}
	if json.Unmarshal(target, &t.Metadata.Target) != nil || json.Unmarshal(source, &t.Source) != nil {
		return t, auth.ErrUnavailable
	}
	if len(res) > 0 {
		var r feedback.Resolution
		if json.Unmarshal(res, &r) != nil {
			return t, auth.ErrUnavailable
		}
		t.Resolution = &r
		t.Metadata.ResolutionKind = &r.Kind
	}
	t.Metadata.CreatedAt = t.Metadata.CreatedAt.UTC()
	t.Metadata.UpdatedAt = t.Metadata.UpdatedAt.UTC()
	return t, nil
}
func feedbackTargetValidity(ctx context.Context, tx *sql.Tx, b feedback.Binding) (string, error) {
	if b.Target.Kind == "site" {
		return "current", nil
	}
	refs := []map[string]any{}
	add := func(kind string, i question.Identity) {
		refs = append(refs, map[string]any{"kind": kind, "id": i.ID, "version": i.Version, "sha256": i.SHA256})
	}
	if b.Knowledge != nil {
		add("knowledge", *b.Knowledge)
	}
	if b.Target.Kind == "path" {
		add("path", *b.Target.Identity)
	}
	if b.Instance != nil {
		add("instance", *b.Instance)
	}
	if b.Template != nil {
		add("template", *b.Template)
	}
	for _, i := range b.Units {
		add("unit", i)
	}
	for _, a := range b.Assets {
		refs = append(refs, map[string]any{"kind": "asset", "id": a.ID, "version": nil, "sha256": a.SHA256})
	}
	var withdrawn bool
	e := tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM jsonb_to_recordset($1::jsonb) r(kind text,id text,version integer,sha256 text) WHERE EXISTS(SELECT 1 FROM content_withdrawals w WHERE w.kind=r.kind AND w.sha256=r.sha256 AND (r.kind='asset' OR w.target_id=r.id AND w.target_version=r.version)) OR EXISTS(SELECT 1 FROM question_withdrawals w WHERE w.kind=r.kind AND w.target_id=r.id AND w.target_version=r.version AND w.sha256=r.sha256))`, body(refs)).Scan(&withdrawn)
	if e != nil {
		return "", e
	}
	if withdrawn {
		return "withdrawn", nil
	}
	var current bool
	if b.Instance != nil {
		e = tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM question_heads h JOIN question_publication_members m ON m.publication_id=h.publication_id AND m.kind='instance' WHERE m.id=$1 AND m.version=$2 AND m.sha256=$3)`, b.Instance.ID, b.Instance.Version, b.Instance.SHA256).Scan(&current)
	} else {
		e = tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM publication_heads h JOIN publication_members m ON m.snapshot_id=h.snapshot_id WHERE m.kind=$1 AND m.id=$2 AND m.version=$3 AND m.availability='active')`, b.Target.Kind, b.Target.Identity.ID, b.Target.Identity.Version).Scan(&current)
	}
	if e != nil {
		return "", e
	}
	if current {
		return "current", nil
	}
	return "historical", nil
}
func feedbackProjectMetadata(ctx context.Context, tx *sql.Tx, u auth.User, t feedbackTicket) (feedback.Metadata, error) {
	m := t.Metadata
	m.Label = feedbackLabel(m.Target)
	m.CanHandle = feedback.CanHandle(u, t.Owner)
	b, e := feedbackResolveTarget(ctx, tx, t.Owner, m.Target, t.Source, false)
	if e != nil {
		return m, e
	}
	m.TargetValidity, e = feedbackTargetValidity(ctx, tx, b)
	return m, e
}
func feedbackAppend(ctx context.Context, tx *sql.Tx, u auth.User, t feedbackTicket, to feedback.Status, message string, resolution, effective *feedback.Resolution, requestID string, now time.Time) error {
	if t.Metadata.Sequence >= feedback.MaxSequence {
		return feedback.ErrConflict
	}
	kind := "replied"
	if to != t.Metadata.Status {
		kind = "transitioned"
	}
	if u.ID == t.Owner {
		kind = "replied"
	}
	var r, eff any
	if resolution != nil {
		r = body(resolution)
	}
	if effective != nil {
		eff = body(effective)
	}
	if _, e := tx.ExecContext(ctx, `INSERT INTO feedback_events(ticket_id,sequence,actor_user_id,kind,from_status,to_status,message,resolution,effective_resolution,recorded_at,request_id) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)`, t.Metadata.ID, t.Metadata.Sequence+1, u.ID, kind, t.Metadata.Status, to, message, r, eff, now, requestID); e != nil {
		return e
	}
	_, e := tx.ExecContext(ctx, `UPDATE feedback_tickets SET status=$2,sequence=sequence+1,current_resolution=$3,updated_at=$4 WHERE id=$1`, t.Metadata.ID, to, eff, now)
	return e
}
func (s *Store) CreateFeedback(ctx context.Context, a question.Access, in feedback.CreateInput) (feedback.Envelope[feedback.Receipt], error) {
	var out feedback.Envelope[feedback.Receipt]
	e := s.feedbackTx(ctx, a, feedback.CreateAction, nil, func(ctx context.Context, tx *sql.Tx, u auth.User, _ time.Time) error {
		_, digest, e := feedback.CanonicalCommand(feedback.CreateAction, "tickets", in)
		if e != nil {
			return e
		}
		r, found, e := feedbackReplay(ctx, tx, u.ID, "create", "tickets", a.IdempotencyKey, digest)
		if e != nil {
			return e
		}
		if found {
			out = feedback.Envelope[feedback.Receipt]{ActorID: u.ID, Data: r}
			return nil
		}
		if _, e = feedbackResolveTarget(ctx, tx, u.ID, in.Target, in.Source, true); e != nil {
			return e
		}
		now, e := dbClock(ctx, tx)
		if e != nil {
			return e
		}
		if e = feedbackConsumeRates(ctx, tx, u.ID, feedback.CreateAction, now); e != nil {
			return e
		}
		id, e := auth.NewID(rand.Reader)
		if e != nil {
			return e
		}
		_, e = tx.ExecContext(ctx, `INSERT INTO feedback_tickets(id,owner_user_id,target,source,original_title,original_location,category,status,sequence,created_at,updated_at) VALUES($1,$2,$3,$4,$5,$6,$7,'new',1,$8,$8)`, id, u.ID, body(in.Target), body(in.Source), in.Title, in.Location, in.Category, now)
		if e != nil {
			return e
		}
		_, e = tx.ExecContext(ctx, `INSERT INTO feedback_events(ticket_id,sequence,actor_user_id,kind,to_status,message,recorded_at,request_id) VALUES($1,1,$2,'created','new',$3,$4,$5)`, id, u.ID, in.Message, now, a.RequestID)
		if e != nil {
			return e
		}
		t, e := feedbackLoadTicket(ctx, tx, id, false)
		if e != nil {
			return e
		}
		m, e := feedbackProjectMetadata(ctx, tx, u, t)
		if e != nil {
			return e
		}
		r = feedback.Receipt{Status: 201, Ticket: m}
		if e = feedbackRemember(ctx, tx, u.ID, "create", "tickets", a.IdempotencyKey, digest, r); e != nil {
			return e
		}
		out = feedback.Envelope[feedback.Receipt]{ActorID: u.ID, Data: r}
		return nil
	})
	if e != nil {
		return feedback.Envelope[feedback.Receipt]{}, e
	}
	return out, nil
}
func (s *Store) ReplyFeedback(ctx context.Context, a question.Access, id string, in feedback.ReplyInput) (feedback.Envelope[feedback.Receipt], error) {
	var out feedback.Envelope[feedback.Receipt]
	e := s.feedbackTx(ctx, a, feedback.ReplyAction, nil, func(ctx context.Context, tx *sql.Tx, u auth.User, _ time.Time) error {
		t, e := feedbackLoadTicket(ctx, tx, id, true)
		if e != nil {
			return e
		}
		if t.Owner != u.ID {
			return auth.ErrNotFound
		}
		_, digest, e := feedback.CanonicalCommand(feedback.ReplyAction, id, in)
		if e != nil {
			return e
		}
		r, found, e := feedbackReplay(ctx, tx, u.ID, "reply", id, a.IdempotencyKey, digest)
		if e != nil {
			return e
		}
		if found {
			out = feedback.Envelope[feedback.Receipt]{ActorID: u.ID, Data: r}
			return nil
		}
		if t.Metadata.Sequence != in.ExpectedSequence {
			return feedback.ErrConflict
		}
		to, e := feedback.OwnerReplyState(t.Metadata.Status)
		if e != nil {
			return e
		}
		now, e := dbClock(ctx, tx)
		if e != nil {
			return e
		}
		if e = feedbackConsumeRates(ctx, tx, u.ID, feedback.ReplyAction, now); e != nil {
			return e
		}
		if e = feedbackAppend(ctx, tx, u, t, to, in.Message, nil, nil, a.RequestID, now); e != nil {
			return e
		}
		t, e = feedbackLoadTicket(ctx, tx, id, false)
		if e != nil {
			return e
		}
		m, e := feedbackProjectMetadata(ctx, tx, u, t)
		if e != nil {
			return e
		}
		r = feedback.Receipt{Status: 200, Ticket: m}
		if e = feedbackRemember(ctx, tx, u.ID, "reply", id, a.IdempotencyKey, digest, r); e != nil {
			return e
		}
		out = feedback.Envelope[feedback.Receipt]{ActorID: u.ID, Data: r}
		return nil
	})
	if e != nil {
		return feedback.Envelope[feedback.Receipt]{}, e
	}
	return out, nil
}
