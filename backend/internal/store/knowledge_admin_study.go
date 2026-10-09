package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"github.com/yyl1212/math_master/backend/internal/auth"
	"github.com/yyl1212/math_master/backend/internal/knowledgeadmin"
	"github.com/yyl1212/math_master/backend/internal/study"
)

const managedRecordColumns = `knowledge_id,state,sequence,first_started_at,first_completed_at,last_completed_at,last_read_at,last_reviewed_at,completed_ref,last_review_ref,last_review_id::text,active_review_id::text`

func managedStudyMode(ctx context.Context, tx *sql.Tx) error {
	var ready bool
	e := tx.QueryRowContext(ctx, `SELECT content_mode='managed' AND enabled_once AND (SELECT managed_knowledge_enabled FROM goose_db_version WHERE version_id=0) FROM knowledge_admin_state WHERE singleton`).Scan(&ready)
	if e != nil {
		return e
	}
	if !ready {
		return knowledgeadmin.ErrNotConfigured
	}
	return nil
}
func readManagedRecord(ctx context.Context, tx *sql.Tx, owner, id string, lock bool) (r knowledgeadmin.ManagedRecord, e error) {
	r = knowledgeadmin.ManagedRecord{KnowledgeID: id, State: study.Unlearned}
	q := "SELECT " + managedRecordColumns + " FROM managed_study_records WHERE owner_user_id=$1 AND knowledge_id=$2"
	if lock {
		q += " FOR UPDATE"
	}
	var completed, review []byte
	e = tx.QueryRowContext(ctx, q, owner, id).Scan(&r.KnowledgeID, &r.State, &r.Sequence, &r.FirstStartedAt, &r.FirstCompletedAt, &r.LastCompletedAt, &r.LastReadAt, &r.LastReviewedAt, &completed, &review, &r.LastReviewID, &r.ActiveReviewID)
	if errors.Is(e, sql.ErrNoRows) {
		return knowledgeadmin.ManagedRecord{KnowledgeID: id, State: study.Unlearned}, nil
	}
	if e != nil {
		return
	}
	if len(completed) > 0 {
		r.CompletedRef = &knowledgeadmin.Ref{}
		if e = json.Unmarshal(completed, r.CompletedRef); e != nil {
			return
		}
	}
	if len(review) > 0 {
		r.LastReviewRef = &knowledgeadmin.Ref{}
		e = json.Unmarshal(review, r.LastReviewRef)
	}
	return
}
func managedStudyCurrent(ctx context.Context, tx *sql.Tx, id string, lock bool) (knowledgeadmin.Knowledge, error) {
	q := "SELECT " + managedColumns + " FROM managed_knowledge WHERE internal_id=$1"
	if lock {
		q += " FOR SHARE"
	}
	k, e := scanManaged(tx.QueryRowContext(ctx, q, id))
	if e != nil {
		return k, e
	}
	k.TopicKeys, e = managedTopics(ctx, tx, id)
	return k, e
}
func managedDetail(ctx context.Context, tx *sql.Tx, owner string, k knowledgeadmin.Knowledge, r knowledgeadmin.ManagedRecord) (out knowledgeadmin.ManagedDetail, e error) {
	out = knowledgeadmin.ManagedDetail{ActorID: owner, Record: r, Available: k.Published && !k.Deleted}
	if !out.Available {
		return
	}
	pub := managedPublic(k)
	out.CurrentKnowledge = &pub
	var last []byte
	e = tx.QueryRowContext(ctx, "SELECT knowledge_ref FROM managed_study_events WHERE owner_user_id=$1 AND knowledge_id=$2 ORDER BY recorded_at DESC,event_id DESC LIMIT 1", owner, k.ID).Scan(&last)
	if errors.Is(e, sql.ErrNoRows) {
		e = nil
	}
	if e != nil {
		return
	}
	var ref knowledgeadmin.Ref
	if len(last) > 0 {
		if e = json.Unmarshal(last, &ref); e != nil {
			return
		}
		out.MaterialChanged = ref != k.Ref
	}
	if r.CompletedRef != nil && r.LastReviewRef == nil {
		out.MaterialChanged = out.MaterialChanged || *r.CompletedRef != k.Ref
	}
	if r.LastReviewRef != nil {
		out.MaterialChanged = *r.LastReviewRef != k.Ref
	}
	return
}
func saveManagedRecord(ctx context.Context, tx *sql.Tx, owner string, r knowledgeadmin.ManagedRecord) error {
	var completed, review any
	if r.CompletedRef != nil {
		completed = knowledgeJSON(r.CompletedRef)
	}
	if r.LastReviewRef != nil {
		review = knowledgeJSON(r.LastReviewRef)
	}
	_, e := tx.ExecContext(ctx, `INSERT INTO managed_study_records(owner_user_id,knowledge_id,state,sequence,first_started_at,first_completed_at,last_completed_at,last_read_at,last_reviewed_at,completed_ref,last_review_ref,last_review_id,active_review_id) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13) ON CONFLICT(owner_user_id,knowledge_id) DO UPDATE SET state=excluded.state,sequence=excluded.sequence,first_started_at=excluded.first_started_at,first_completed_at=excluded.first_completed_at,last_completed_at=excluded.last_completed_at,last_read_at=excluded.last_read_at,last_reviewed_at=excluded.last_reviewed_at,completed_ref=excluded.completed_ref,last_review_ref=excluded.last_review_ref,last_review_id=excluded.last_review_id,active_review_id=excluded.active_review_id`, owner, r.KnowledgeID, r.State, r.Sequence, r.FirstStartedAt, r.FirstCompletedAt, r.LastCompletedAt, r.LastReadAt, r.LastReviewedAt, completed, review, r.LastReviewID, r.ActiveReviewID)
	return e
}
func managedStudyReplay(ctx context.Context, tx *sql.Tx, owner string, a knowledgeadmin.Access, id, action, digest string) (bool, error) {
	var sha string
	e := tx.QueryRowContext(ctx, "SELECT input_sha256 FROM managed_study_idempotency WHERE owner_user_id=$1 AND knowledge_id=$2 AND action=$3 AND key=$4", owner, id, action, a.IdempotencyKey).Scan(&sha)
	if errors.Is(e, sql.ErrNoRows) {
		return false, nil
	}
	if e != nil {
		return false, e
	}
	if sha != digest {
		return false, knowledgeadmin.ErrIdempotency
	}
	return true, nil
}
func rememberManagedStudy(ctx context.Context, tx *sql.Tx, owner string, a knowledgeadmin.Access, id, action, digest string, receipt any) error {
	_, e := tx.ExecContext(ctx, `INSERT INTO managed_study_idempotency(owner_user_id,knowledge_id,action,key,input_sha256,receipt) VALUES($1,$2,$3,$4,$5,$6)`, owner, id, action, a.IdempotencyKey, digest, knowledgeJSON(receipt))
	return e
}
func (s *Store) ReadManagedStudy(ctx context.Context, a knowledgeadmin.Access, id string) (out knowledgeadmin.ManagedDetail, e error) {
	e = s.knowledgeTx(ctx, a, false, false, func(ctx context.Context, tx *sql.Tx, u auth.User) error {
		if e := managedStudyMode(ctx, tx); e != nil {
			return e
		}
		k, e := managedStudyCurrent(ctx, tx, id, false)
		if e != nil {
			return e
		}
		r, e := readManagedRecord(ctx, tx, u.ID, id, false)
		if e != nil {
			return e
		}
		if (!k.Published || k.Deleted) && r.Sequence == 0 && r.FirstStartedAt == nil {
			var own bool
			e = tx.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM managed_study_notes WHERE owner_user_id=$1 AND knowledge_id=$2)", u.ID, id).Scan(&own)
			if e != nil {
				return e
			}
			if !own {
				return knowledgeadmin.ErrNotFound
			}
		}
		out, e = managedDetail(ctx, tx, u.ID, k, r)
		return e
	})
	return
}
func (s *Store) ApplyManagedStudy(ctx context.Context, a knowledgeadmin.Access, id, action string, in knowledgeadmin.ManagedStudyInput) (out knowledgeadmin.ManagedDetail, e error) {
	if !knowledgeadmin.ValidManagedRef(in.Knowledge) || in.Knowledge.ID != id || in.ExpectedSequence < 0 || in.ExpectedSequence > study.MaxSequence {
		return out, knowledgeadmin.ErrInvalid
	}
	if action != "begin" && action != "complete" && action != "start-review" && action != "finish-review" {
		return out, knowledgeadmin.ErrInvalid
	}
	digest := knowledgeFingerprint(in)
	e = s.knowledgeTx(ctx, a, true, false, func(ctx context.Context, tx *sql.Tx, u auth.User) error {
		if e := managedStudyMode(ctx, tx); e != nil {
			return e
		}
		k, e := managedStudyCurrent(ctx, tx, id, true)
		if e != nil {
			return e
		}
		if !k.Published || k.Deleted {
			return knowledgeadmin.ErrNotFound
		}
		if k.Ref != in.Knowledge {
			return knowledgeadmin.ErrStale
		}
		_, e = tx.ExecContext(ctx, `INSERT INTO managed_study_records(owner_user_id,knowledge_id,state) VALUES($1,$2,'unlearned') ON CONFLICT DO NOTHING`, u.ID, id)
		if e != nil {
			return e
		}
		r, e := readManagedRecord(ctx, tx, u.ID, id, true)
		if e != nil {
			return e
		}
		replay, e := managedStudyReplay(ctx, tx, u.ID, a, id, action, digest)
		if e != nil {
			return e
		}
		if replay {
			out, e = managedDetail(ctx, tx, u.ID, k, r)
			return e
		}
		if r.Sequence != in.ExpectedSequence {
			return knowledgeadmin.ErrStale
		}
		op := study.Action(action)
		switch action {
		case "start-review":
			op = study.StartReview
		case "finish-review":
			op = study.FinishReview
		}
		if op == study.Complete && r.State == study.Reviewing && (r.CompletedRef == nil || *r.CompletedRef != k.Ref) {
			return knowledgeadmin.ErrConflict
		}
		next, e := study.TransitionState(r.State, op)
		if e != nil {
			return knowledgeadmin.ErrConflict
		}
		now, e := dbClock(ctx, tx)
		if e != nil {
			return e
		}
		changed := next != r.State
		kind := ""
		var reviewID *string
		switch op {
		case study.Begin:
			kind = "started"
			r.LastReadAt = &now
			if r.FirstStartedAt == nil {
				r.FirstStartedAt = &now
			}
		case study.Complete:
			kind = "completed"
			changed = !(r.CompletedRef != nil && *r.CompletedRef == k.Ref && (r.State == study.Completed || r.State == study.Reviewing))
			if changed {
				if r.FirstCompletedAt == nil {
					r.FirstCompletedAt = &now
				}
				r.LastCompletedAt = &now
				r.CompletedRef = &k.Ref
			}
		case study.StartReview:
			kind = "review-started"
			if changed {
				var rid string
				if e = tx.QueryRowContext(ctx, "SELECT gen_random_uuid()::text").Scan(&rid); e != nil {
					return e
				}
				r.ActiveReviewID = &rid
				reviewID = &rid
			}
		case study.FinishReview:
			kind = "review-finished"
			if in.ReviewID == nil || !study.ValidID(*in.ReviewID) {
				return knowledgeadmin.ErrInvalid
			}
			if r.State == study.Completed && r.LastReviewID != nil && *r.LastReviewID == *in.ReviewID {
				changed = false
			} else if r.ActiveReviewID == nil || *r.ActiveReviewID != *in.ReviewID {
				return knowledgeadmin.ErrConflict
			}
			if changed {
				r.ActiveReviewID = nil
				r.LastReviewID = in.ReviewID
				r.LastReviewRef = &k.Ref
				r.LastReviewedAt = &now
				reviewID = in.ReviewID
			}
		}
		if changed {
			if r.Sequence >= study.MaxSequence {
				return knowledgeadmin.ErrConflict
			}
			r.Sequence++
			r.State = next
		}
		if e = saveManagedRecord(ctx, tx, u.ID, r); e != nil {
			return e
		}
		if changed {
			_, e = tx.ExecContext(ctx, `INSERT INTO managed_study_events(owner_user_id,knowledge_id,knowledge_ref,topic_keys,kind,review_id,recorded_at) VALUES($1,$2,$3,$4,$5,$6,$7)`, u.ID, id, knowledgeJSON(k.Ref), knowledgeJSON(k.TopicKeys), kind, reviewID, now)
			if e != nil {
				return e
			}
		}
		out, e = managedDetail(ctx, tx, u.ID, k, r)
		if e != nil {
			return e
		}
		return rememberManagedStudy(ctx, tx, u.ID, a, id, action, digest, map[string]any{"sequence": r.Sequence, "state": r.State})
	})
	return
}

func (s *Store) ListManagedStudyKnowledge(ctx context.Context, a knowledgeadmin.Access, q knowledgeadmin.Query) (out knowledgeadmin.Page[knowledgeadmin.ManagedDetail], e error) {
	q, e = normalizeKnowledgeQuery(q)
	if e != nil {
		return
	}
	e = s.knowledgeTx(ctx, a, false, false, func(ctx context.Context, tx *sql.Tx, u auth.User) error {
		if e := managedStudyMode(ctx, tx); e != nil {
			return e
		}
		page, e := listCurrent(ctx, tx, q, u.ID)
		if e != nil {
			return e
		}
		out = knowledgeadmin.Page[knowledgeadmin.ManagedDetail]{Items: []knowledgeadmin.ManagedDetail{}, Total: page.Total, Limit: page.Limit, Offset: page.Offset}
		for _, pub := range page.Items {
			r, e := readManagedRecord(ctx, tx, u.ID, pub.ID, false)
			if e != nil {
				return e
			}
			var p knowledgeadmin.SourcePoint
			if json.Unmarshal(knowledgeJSON(pub.Point), &p) != nil {
				return knowledgeadmin.ErrNotConfigured
			}
			k := knowledgeadmin.Knowledge{ID: pub.ID, ExternalID: pub.ExternalID, TopicKeys: pub.TopicKeys, Point: p, Sources: pub.Sources, Ref: pub.Ref, Published: true, UpdatedAt: pub.UpdatedAt}
			d, e := managedDetail(ctx, tx, u.ID, k, r)
			if e != nil {
				return e
			}
			copy := pub
			d.CurrentKnowledge = &copy
			out.Items = append(out.Items, d)
		}
		return nil
	})
	return
}
func (s *Store) ReadManagedOverview(ctx context.Context, a knowledgeadmin.Access) (out knowledgeadmin.ManagedOverview, e error) {
	e = s.knowledgeTx(ctx, a, false, false, func(ctx context.Context, tx *sql.Tx, u auth.User) error {
		if e := managedStudyMode(ctx, tx); e != nil {
			return e
		}
		out = knowledgeadmin.ManagedOverview{ActorID: u.ID, Reminders: []knowledgeadmin.ManagedReminder{}}
		if e := tx.QueryRowContext(ctx, `SELECT count(*),count(*) FILTER(WHERE r.state='completed'),count(*) FILTER(WHERE r.state='learning'),count(*) FILTER(WHERE r.state='reviewing'),count(*) FILTER(WHERE NOT EXISTS(SELECT 1 FROM managed_knowledge_topics t WHERE t.internal_id=k.internal_id AND t.active)) FROM managed_knowledge k LEFT JOIN managed_study_records r ON r.knowledge_id=k.internal_id AND r.owner_user_id=$1 WHERE k.published AND k.deleted_at IS NULL`, u.ID).Scan(&out.Total, &out.Completed, &out.Learning, &out.Reviewing, &out.Unclassified); e != nil {
			return e
		}
		if e := tx.QueryRowContext(ctx, `SELECT count(*) FROM managed_study_records r JOIN managed_knowledge k ON k.internal_id=r.knowledge_id WHERE r.owner_user_id=$1 AND (NOT k.published OR k.deleted_at IS NOT NULL)`, u.ID).Scan(&out.Unavailable); e != nil {
			return e
		}
		rows, e := tx.QueryContext(ctx, `SELECT k.internal_id,c.kind,c.recorded_at FROM managed_knowledge k JOIN managed_study_records r ON r.knowledge_id=k.internal_id AND r.owner_user_id=$1 JOIN LATERAL (SELECT kind,recorded_at FROM managed_study_content_changes WHERE knowledge_id=k.internal_id ORDER BY recorded_at DESC LIMIT 1)c ON true JOIN LATERAL (SELECT knowledge_ref FROM managed_study_events WHERE knowledge_id=k.internal_id AND owner_user_id=$1 ORDER BY recorded_at DESC,event_id DESC LIMIT 1)e ON true WHERE k.published AND k.deleted_at IS NULL AND coalesce(r.last_review_ref,e.knowledge_ref)->>'contentSha256'<>k.content_sha256 ORDER BY c.recorded_at DESC,k.internal_id LIMIT 20`, u.ID)
		if e != nil {
			return e
		}
		for rows.Next() {
			var v knowledgeadmin.ManagedReminder
			if e = rows.Scan(&v.KnowledgeID, &v.Kind, &v.RecordedAt); e != nil {
				rows.Close()
				return e
			}
			out.Reminders = append(out.Reminders, v)
		}
		e = rows.Err()
		rows.Close()
		if e != nil {
			return e
		}
		return tx.QueryRowContext(ctx, `SELECT count(*) FROM managed_study_records r JOIN managed_knowledge k ON k.internal_id=r.knowledge_id JOIN LATERAL (SELECT knowledge_ref FROM managed_study_events WHERE owner_user_id=r.owner_user_id AND knowledge_id=r.knowledge_id ORDER BY recorded_at DESC,event_id DESC LIMIT 1)e ON true WHERE r.owner_user_id=$1 AND k.published AND k.deleted_at IS NULL AND coalesce(r.last_review_ref,e.knowledge_ref)->>'contentSha256'<>k.content_sha256`, u.ID).Scan(&out.MaterialChanged)
	})
	return
}
func (s *Store) ListManagedStudyTopics(ctx context.Context, a knowledgeadmin.Access, q knowledgeadmin.Query) (out knowledgeadmin.Page[knowledgeadmin.ManagedProgress], e error) {
	q, e = normalizeKnowledgeQuery(q)
	if e != nil {
		return
	}
	e = s.knowledgeTx(ctx, a, false, false, func(ctx context.Context, tx *sql.Tx, u auth.User) error {
		if e := managedStudyMode(ctx, tx); e != nil {
			return e
		}
		out = knowledgeadmin.Page[knowledgeadmin.ManagedProgress]{Items: []knowledgeadmin.ManagedProgress{}, Limit: q.Limit, Offset: q.Offset}
		key := topicPrefix(q.TopicKey)
		var directory bool
		if e := tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM taxonomy_heads h JOIN taxonomy_releases r ON r.id=h.release_id AND r.status='published')`).Scan(&directory); e != nil {
			return e
		}
		if directory {
			return managedDirectoryProgress(ctx, tx, u.ID, q, &out)
		}
		if e := tx.QueryRowContext(ctx, `SELECT count(DISTINCT t.topic_key) FROM managed_knowledge_topics t JOIN managed_knowledge k ON k.internal_id=t.internal_id WHERE t.active AND k.published AND k.deleted_at IS NULL AND ($1='' OR t.topic_key=$1 OR t.topic_key LIKE $1||'%')`, key).Scan(&out.Total); e != nil {
			return e
		}
		rows, e := tx.QueryContext(ctx, `SELECT t.topic_key,count(DISTINCT k.internal_id),count(DISTINCT k.internal_id) FILTER(WHERE r.state='completed'),count(DISTINCT k.internal_id) FILTER(WHERE r.state='learning'),count(DISTINCT k.internal_id) FILTER(WHERE r.state='reviewing') FROM managed_knowledge_topics t JOIN managed_knowledge k ON k.internal_id=t.internal_id LEFT JOIN managed_study_records r ON r.knowledge_id=k.internal_id AND r.owner_user_id=$1 WHERE t.active AND k.published AND k.deleted_at IS NULL AND ($2='' OR t.topic_key=$2 OR t.topic_key LIKE $2||'%') GROUP BY t.topic_key ORDER BY t.topic_key LIMIT $3 OFFSET $4`, u.ID, key, q.Limit, q.Offset)
		if e != nil {
			return e
		}
		defer rows.Close()
		for rows.Next() {
			var p knowledgeadmin.ManagedProgress
			if e = rows.Scan(&p.TopicKey, &p.Total, &p.Completed, &p.Learning, &p.Reviewing); e != nil {
				return e
			}
			if p.Total > 0 {
				ratio := float64(p.Completed+p.Reviewing) / float64(p.Total)
				p.CompletedRatio = &ratio
			}
			out.Items = append(out.Items, p)
		}
		return rows.Err()
	})
	return
}

func managedDirectoryProgress(ctx context.Context, tx *sql.Tx, owner string, q knowledgeadmin.Query, out *knowledgeadmin.Page[knowledgeadmin.ManagedProgress]) error {
	key := topicPrefix(q.TopicKey)
	// Current membership determines progress; historical topic context stays in immutable events.
	const topics = `WITH groups AS (SELECT n.code AS topic_key FROM taxonomy_nodes n JOIN taxonomy_heads h ON h.singleton JOIN taxonomy_releases r ON r.id=h.release_id AND r.status='published' AND r.taxonomy_version_id=n.taxonomy_version_id WHERE n.kind<>'auxiliary' AND (($1='' AND n.level=1) OR ($1<>'' AND (n.id=$2 OR upper(n.code)=upper($2)))) UNION ALL SELECT 'project:other' WHERE $1='' OR $1='project:other') `
	if e := tx.QueryRowContext(ctx, topics+`SELECT count(*) FROM groups`, key, q.TopicKey).Scan(&out.Total); e != nil {
		return e
	}
	rows, e := tx.QueryContext(ctx, topics+`SELECT g.topic_key,count(DISTINCT k.internal_id),count(DISTINCT k.internal_id) FILTER(WHERE sr.state='completed'),count(DISTINCT k.internal_id) FILTER(WHERE sr.state='learning'),count(DISTINCT k.internal_id) FILTER(WHERE sr.state='reviewing') FROM groups g LEFT JOIN managed_knowledge_topics t ON t.active AND (t.topic_key=g.topic_key OR t.topic_key LIKE regexp_replace(upper(g.topic_key),'[-]?XX$','')||'%') LEFT JOIN managed_knowledge k ON k.internal_id=t.internal_id AND k.published AND k.deleted_at IS NULL LEFT JOIN managed_study_records sr ON sr.knowledge_id=k.internal_id AND sr.owner_user_id=$3 GROUP BY g.topic_key ORDER BY g.topic_key LIMIT $4 OFFSET $5`, key, q.TopicKey, owner, q.Limit, q.Offset)
	if e != nil {
		return e
	}
	defer rows.Close()
	for rows.Next() {
		var p knowledgeadmin.ManagedProgress
		if e = rows.Scan(&p.TopicKey, &p.Total, &p.Completed, &p.Learning, &p.Reviewing); e != nil {
			return e
		}
		if p.Total > 0 {
			ratio := float64(p.Completed+p.Reviewing) / float64(p.Total)
			p.CompletedRatio = &ratio
		}
		out.Items = append(out.Items, p)
	}
	return rows.Err()
}
