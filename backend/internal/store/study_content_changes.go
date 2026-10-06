package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"github.com/yyl1212/math_master/backend/internal/study"
	"github.com/yyl1212/math_master/backend/internal/taxonomy"
	"sort"
)

func optionalStudyConfigured(ctx context.Context, tx *sql.Tx) (bool, error) {
	var count int
	if e := tx.QueryRowContext(ctx, "SELECT count(*) FROM unnest($1::text[]) name WHERE to_regclass('public.'||name) IS NOT NULL", studyTables).Scan(&count); e != nil {
		return false, e
	}
	var hasMarker bool
	if e := tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM information_schema.columns WHERE table_schema='public' AND table_name='goose_db_version' AND column_name='topic_study_enabled')`).Scan(&hasMarker); e != nil {
		return false, e
	}
	ever := false
	if hasMarker {
		if e := tx.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM goose_db_version WHERE version_id=0 AND topic_study_enabled)").Scan(&ever); e != nil {
			return false, e
		}
	}
	var stateExists bool
	if e := tx.QueryRowContext(ctx, "SELECT to_regclass('public.topic_learning_state') IS NOT NULL").Scan(&stateExists); e != nil {
		return false, e
	}
	if stateExists {
		var current bool
		if e := tx.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM topic_learning_state WHERE singleton AND study_enabled)").Scan(&current); e != nil {
			return false, e
		}
		ever = ever || current
	}
	if count == 0 && !ever {
		return false, nil
	}
	if e := studyConfigured(ctx, tx); e != nil {
		return false, e
	}
	return true, nil
}
func appendStudyContentChangesTx(ctx context.Context, tx *sql.Tx, before, after taxonomy.KnowledgeSet, publicationID string) error {
	enabled, e := optionalStudyConfigured(ctx, tx)
	if e != nil || !enabled {
		return e
	}
	ids := map[string]bool{}
	for id := range before {
		ids[id] = true
	}
	for id := range after {
		ids[id] = true
	}
	ordered := []string{}
	for id := range ids {
		ordered = append(ordered, id)
	}
	sort.Strings(ordered)
	for _, id := range ordered {
		old, was := before[id]
		next, is := after[id]
		if was && is && old == next {
			continue
		}
		kind := "updated"
		if !was {
			kind = "added"
		} else if !is {
			kind = "withdrawn"
		}
		var oldRaw, newRaw any
		if was {
			raw, _ := json.Marshal(old)
			oldRaw = string(raw)
		}
		if is {
			raw, _ := json.Marshal(next)
			newRaw = string(raw)
		}
		if _, e = tx.ExecContext(ctx, `INSERT INTO study_content_changes(id,publication_id,knowledge_id,kind,before_ref,after_ref,recorded_at) VALUES(gen_random_uuid(),$1,$2,$3,$4,$5,clock_timestamp()) ON CONFLICT(publication_id,knowledge_id,kind) DO NOTHING`, publicationID, id, kind, oldRaw, newRaw); e != nil {
			return e
		}
	}
	return nil
}
func studyRemindersTx(ctx context.Context, tx *sql.Tx, actor string, scope studyScope, owned map[string]studyOwned) ([]study.ContentReminder, error) {
	out := []study.ContentReminder{}
	rows, e := tx.QueryContext(ctx, `SELECT id::text,knowledge_id,kind,recorded_at,after_ref FROM (SELECT DISTINCT ON (c.knowledge_id) c.* FROM study_content_changes c JOIN study_records r ON r.knowledge_id=c.knowledge_id AND r.owner_user_id=$1 WHERE c.kind IN ('updated','withdrawn') AND c.recorded_at>=COALESCE((r.body->>'firstStartedAt')::timestamptz,(r.body->>'firstCompletedAt')::timestamptz) ORDER BY c.knowledge_id,c.recorded_at DESC,c.id DESC) latest ORDER BY recorded_at DESC,id DESC LIMIT 1000`, actor)
	if e != nil {
		return out, e
	}
	defer rows.Close()
	for rows.Next() {
		var v study.ContentReminder
		var raw []byte
		if e = rows.Scan(&v.ChangeID, &v.KnowledgeID, &v.Kind, &v.RecordedAt, &raw); e != nil {
			return out, e
		}
		if len(raw) > 0 {
			var ref study.KnowledgeRef
			if json.Unmarshal(raw, &ref) != nil || !study.ValidRef(ref) {
				return out, study.ErrNotConfigured
			}
			v.CurrentRef = &ref
		}
		v.RecordedAt = v.RecordedAt.UTC()
		v.Reviewed = studyAcknowledged(owned[v.KnowledgeID].record, v.CurrentRef, v.RecordedAt)
		out = append(out, v)
	}
	return out, rows.Err()
}
