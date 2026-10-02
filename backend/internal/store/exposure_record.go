package store

import (
	"context"
	"database/sql"
	"github.com/yyl1212/math_master/backend/internal/assessment"
	"github.com/yyl1212/math_master/backend/internal/auth"
	"github.com/yyl1212/math_master/backend/internal/learning"
	"github.com/yyl1212/math_master/backend/internal/question"
	"time"
)

func learningRecordExposure(ctx context.Context, tx *sql.Tx, actor string, refs []learning.ExposureRef, now time.Time) (int64, error) {
	enabled, e := learningConfigured(ctx, tx)
	if e != nil || !enabled {
		return 0, e
	}
	if !question.ValidID(actor) {
		return 0, auth.ErrInvalidInput
	}
	if len(refs) == 0 {
		var n int64
		e = tx.QueryRowContext(ctx, `SELECT coalesce((SELECT sequence FROM learner_exposure_state WHERE owner_user_id=$1),0)`, actor).Scan(&n)
		return n, e
	}
	unique := map[learning.ExposureRef]bool{}
	input := []map[string]any{}
	for _, r := range refs {
		if r.Kind != "instance" && r.Kind != "template" || r.Identity.Version < 1 || !question.ValidSHA(r.Identity.SHA256) || r.Kind == "instance" && !question.ValidInstanceID(r.Identity.ID) || r.Kind == "template" && !question.ValidMathID(r.Identity.ID) {
			return 0, auth.ErrInvalidInput
		}
		if !unique[r] {
			unique[r] = true
			input = append(input, map[string]any{"kind": r.Kind, "id": r.Identity.ID, "version": r.Identity.Version, "sha256": r.Identity.SHA256})
		}
	}
	var n int64
	e = tx.QueryRowContext(ctx, `INSERT INTO learner_exposure_state(owner_user_id,sequence,updated_at) VALUES($1,1,$2) ON CONFLICT(owner_user_id) DO UPDATE SET sequence=learner_exposure_state.sequence+1,updated_at=greatest(learner_exposure_state.updated_at,EXCLUDED.updated_at) RETURNING sequence`, actor, now).Scan(&n)
	if e != nil {
		return 0, e
	}
	_, e = tx.ExecContext(ctx, `INSERT INTO learner_answer_exposures(owner_user_id,kind,id,version,sha256,sequence,exposed_at) SELECT $1,kind,id,version,sha256,$2,$3 FROM jsonb_to_recordset($4::jsonb) r(kind text,id text,version integer,sha256 text) ORDER BY kind,id,version,sha256 ON CONFLICT(owner_user_id,kind,id,version,sha256) DO UPDATE SET sequence=EXCLUDED.sequence,exposed_at=greatest(learner_answer_exposures.exposed_at,EXCLUDED.exposed_at)`, actor, n, now, body(input))
	return n, e
}
func learningExposureSince(ctx context.Context, tx *sql.Tx, actor string, watermark int64, items []assessment.ItemBinding) (bool, error) {
	var exposed bool
	e := tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM learner_answer_exposures e CROSS JOIN jsonb_to_recordset($3::jsonb) r(instance jsonb,template jsonb) WHERE e.owner_user_id=$1 AND e.sequence>$2 AND ((e.kind='instance' AND e.id=r.instance->>'id' AND e.version=(r.instance->>'version')::integer AND e.sha256=r.instance->>'sha256') OR (e.kind='template' AND e.id=r.template->>'id' AND e.version=(r.template->>'version')::integer AND e.sha256=r.template->>'sha256')))`, actor, watermark, body(items)).Scan(&exposed)
	return exposed, e
}
func learningRecordViews(ctx context.Context, tx *sql.Tx, actor string, items []assessment.ItemBinding, now time.Time) error {
	_, e := tx.ExecContext(ctx, `INSERT INTO learner_question_views(owner_user_id,instance_id,instance_version,instance_sha256,first_seen_at,last_seen_at) SELECT $1,instance->>'id',(instance->>'version')::integer,instance->>'sha256',$2,$2 FROM jsonb_to_recordset($3::jsonb) r(instance jsonb) ORDER BY instance->>'id' ON CONFLICT(owner_user_id,instance_id,instance_version,instance_sha256) DO UPDATE SET last_seen_at=greatest(learner_question_views.last_seen_at,EXCLUDED.last_seen_at)`, actor, now, body(items))
	return e
}
