package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"github.com/yyl1212/math_master/backend/internal/knowledgeadmin"
	"github.com/yyl1212/math_master/backend/internal/publication"
	"github.com/yyl1212/math_master/backend/internal/study"
	"reflect"
	"regexp"
	"sort"
	"time"
)

// Readiness advertises the binary capability independently of permanent activation.
type ManagedSchemaHealth struct {
	Capability  bool `json:"capability"`
	SchemaReady bool `json:"schemaReady"`
	ManagedMode bool `json:"managedMode"`
}

var managedTables = []string{"knowledge_admin_state", "managed_knowledge", "managed_knowledge_topics", "managed_knowledge_sources", "managed_knowledge_imports", "managed_knowledge_events", "managed_study_records", "managed_study_notes", "managed_study_events", "managed_study_idempotency", "managed_study_content_changes"}

const managedSchemaDigestSQL = `SELECT md5(string_agg(value,E'\n' ORDER BY value COLLATE "C")) FROM (
SELECT 'C:'||c.conrelid::regclass::text||':'||c.conname||':'||c.contype::text||':'||c.convalidated::text||':'||pg_get_constraintdef(c.oid) value FROM pg_constraint c WHERE c.conrelid IN (SELECT oid FROM pg_class WHERE relnamespace='public'::regnamespace AND relname=ANY($1::text[])) OR (c.conrelid='feedback_tickets'::regclass AND c.conname='feedback_managed_target_shape')
UNION ALL SELECT 'T:'||t.tgrelid::regclass::text||':'||t.tgname||':'||t.tgtype::text||':'||t.tgenabled::text||':'||pg_get_triggerdef(t.oid)||':'||md5(pg_get_functiondef(t.tgfoid)) FROM pg_trigger t WHERE NOT t.tgisinternal AND (t.tgrelid IN (SELECT oid FROM pg_class WHERE relnamespace='public'::regnamespace AND relname=ANY($1::text[])) OR t.tgname='managed_knowledge_marker_guard')
UNION ALL SELECT 'A:'||a.attrelid::regclass::text||':'||a.attname||':'||format_type(a.atttypid,a.atttypmod)||':'||a.attnotnull::text||':'||coalesce(pg_get_expr(d.adbin,d.adrelid),'') FROM pg_attribute a LEFT JOIN pg_attrdef d ON d.adrelid=a.attrelid AND d.adnum=a.attnum WHERE a.attnum>0 AND NOT a.attisdropped AND a.attrelid IN (SELECT oid FROM pg_class WHERE relnamespace='public'::regnamespace AND relname=ANY($1::text[]))
UNION ALL SELECT 'F:'||p.oid::regprocedure::text||':'||md5(pg_get_functiondef(p.oid)) FROM pg_proc p WHERE p.pronamespace='public'::regnamespace AND p.proname IN ('managed_ref_valid','feedback_target_shape','feedback_target_proof','feedback_label')
) checks`
const managedSchemaDigest = "f81857f4af9a3be336d85f279578df8b"

func managedSchemaIntegrity(ctx context.Context, tx *sql.Tx) error {
	var markerColumn bool
	if e := tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM pg_attribute a WHERE a.attrelid=to_regclass('public.goose_db_version') AND a.attname='managed_knowledge_enabled' AND NOT a.attisdropped AND a.attnotnull AND a.atttypid='boolean'::regtype) AND EXISTS(SELECT 1 FROM goose_db_version WHERE version_id=13 AND is_applied)`).Scan(&markerColumn); e != nil {
		return e
	}
	if !markerColumn {
		return knowledgeadmin.ErrNotConfigured
	}
	var n int
	if e := tx.QueryRowContext(ctx, `SELECT count(*) FROM unnest($1::text[]) name WHERE to_regclass('public.'||name) IS NOT NULL`, managedTables).Scan(&n); e != nil {
		return e
	}
	if n != len(managedTables) {
		return knowledgeadmin.ErrNotConfigured
	}
	// Catalog deparsing must use a fixed resolution path. Restore the caller's
	// path before business clocks run (legacy tests deliberately shadow a clock).
	var previousPath string
	if e := tx.QueryRowContext(ctx, `SELECT current_setting('search_path')`).Scan(&previousPath); e != nil {
		return e
	}
	if _, e := tx.ExecContext(ctx, `SELECT set_config('search_path','pg_catalog,public',true)`); e != nil {
		return e
	}
	var digest string
	digestError := tx.QueryRowContext(ctx, managedSchemaDigestSQL, managedTables).Scan(&digest)
	_, restoreError := tx.ExecContext(ctx, `SELECT set_config('search_path',$1,true)`, previousPath)
	if digestError != nil {
		return digestError
	}
	if restoreError != nil {
		return restoreError
	}
	if digest != managedSchemaDigest {
		return knowledgeadmin.ErrNotConfigured
	}
	return nil
}
func (s *Store) ReadManagedSchemaHealth(ctx context.Context) (out ManagedSchemaHealth, e error) {
	out.Capability = true
	ctx, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()
	tx, e := s.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true, Isolation: sql.LevelRepeatableRead})
	if e != nil {
		return out, knowledgeError(e)
	}
	defer tx.Rollback()
	var markerColumn bool
	e = tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM pg_attribute WHERE attrelid=to_regclass('public.goose_db_version') AND attname='managed_knowledge_enabled' AND NOT attisdropped)`).Scan(&markerColumn)
	if e != nil {
		return out, e
	}
	if !markerColumn {
		var n int
		if e = tx.QueryRowContext(ctx, `SELECT count(*) FROM unnest($1::text[]) name WHERE to_regclass('public.'||name) IS NOT NULL`, managedTables).Scan(&n); e != nil {
			return out, e
		}
		var migration bool
		if e = tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM goose_db_version WHERE version_id=13 AND is_applied)`).Scan(&migration); e != nil {
			return out, e
		}
		out.SchemaReady = n == 0 && !migration
		return out, nil
	}
	if e = managedSchemaIntegrity(ctx, tx); e != nil {
		return out, nil
	}
	var once, marker bool
	var mode string
	e = tx.QueryRowContext(ctx, `SELECT content_mode,enabled_once,coalesce((SELECT managed_knowledge_enabled FROM goose_db_version WHERE version_id=0),false) FROM knowledge_admin_state WHERE singleton`).Scan(&mode, &once, &marker)
	if e != nil {
		return out, nil
	}
	out.ManagedMode = mode == "managed"
	out.SchemaReady = marker == once && once == out.ManagedMode
	return out, nil
}

var cleanupScopes = map[string]string{
	"study_legacy_event_links": "knowledge_id=ANY($1::text[])", "study_notes": "knowledge_id=ANY($1::text[])", "study_events": "knowledge_id=ANY($1::text[])", "study_idempotency": "knowledge_id=ANY($1::text[])", "study_records": "knowledge_id=ANY($1::text[])",
	"learning_records": "knowledge_id=ANY($1::text[])", "learning_events": "knowledge_id=ANY($1::text[])", "learning_unlocks": "knowledge_id=ANY($1::text[]) AND source_kind='learning-event'",
	"learning_evidence_dependencies": "evidence_kind='learning-event' AND evidence_id IN (SELECT id FROM learning_events WHERE knowledge_id=ANY($1::text[]))",
	"learning_idempotency":           "action IN ('startKnowledge','completeKnowledge') AND target=ANY($1::text[])"}
var maintenanceIgnored = map[string]bool{"knowledge_admin_state": true, "goose_db_version": true, "managed_knowledge_events": true, "auth_sessions": true, "auth_preauth": true, "auth_rate_limits": true, "auth_audit_events": true}

func maintenanceBackupValid(in knowledgeadmin.CutoverInput) bool {
	if !trustedCodePattern.MatchString(in.CodeSHA) || !backupRecordPattern.MatchString(in.BackupRecord) || !publication.ValidID(in.ActorID) || !in.RestoreVerified || !in.OffsiteVerified || in.BackupCreatedAt.IsZero() || in.BackupCreatedAt.After(time.Now().Add(time.Minute)) || time.Since(in.BackupCreatedAt) > 30*time.Minute {
		return false
	}
	return true
}
func maintenanceInputValid(in knowledgeadmin.CutoverInput) bool {
	if !maintenanceBackupValid(in) || len(in.OldIDs) != 30 {
		return false
	}
	seen := map[string]bool{}
	for _, id := range in.OldIDs {
		if !study.ValidKnowledgeID(id) || seen[id] {
			return false
		}
		seen[id] = true
	}
	return true
}
func maintenanceTables(ctx context.Context, tx *sql.Tx) ([]string, error) {
	rows, e := tx.QueryContext(ctx, `SELECT tablename FROM pg_tables WHERE schemaname='public' ORDER BY tablename COLLATE "C"`)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	out := []string{}
	for rows.Next() {
		var name string
		if e = rows.Scan(&name); e != nil {
			return nil, e
		}
		if !regexp.MustCompile(`^[a-z][a-z0-9_]*$`).MatchString(name) {
			return nil, knowledgeadmin.ErrInvalid
		}
		out = append(out, name)
	}
	return out, rows.Err()
}
func rowFingerprint(ctx context.Context, tx *sql.Tx, table, where string, ids []string) (int, string, error) {
	var n int
	var hash string
	q := `SELECT count(*),md5(coalesce(string_agg(to_jsonb(x)::text,E'\n' ORDER BY to_jsonb(x)::text COLLATE "C"),'')) FROM "` + table + `" x`
	var args []any
	if where != "" {
		q += " WHERE " + where
		args = []any{ids}
	}
	e := tx.QueryRowContext(ctx, q, args...).Scan(&n, &hash)
	return n, hash, e
}
func protectedFingerprint(ctx context.Context, tx *sql.Tx, ids []string) (string, error) {
	tables, e := maintenanceTables(ctx, tx)
	if e != nil {
		return "", e
	}
	values := map[string]string{}
	for _, table := range tables {
		if maintenanceIgnored[table] {
			continue
		}
		where := ""
		if scope, ok := cleanupScopes[table]; ok {
			where = "NOT (" + scope + ")"
		}
		_, hash, e := rowFingerprint(ctx, tx, table, where, ids)
		if e != nil {
			return "", e
		}
		values[table] = hash
	}
	return knowledgeFingerprint(values), nil
}
func maintenanceAdmin(ctx context.Context, tx *sql.Tx, id string) error {
	var valid bool
	e := tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM auth_users u JOIN auth_user_roles r ON r.user_id=u.id AND r.role='admin' WHERE u.id=$1 AND NOT u.must_change_password)`, id).Scan(&valid)
	if e != nil {
		return e
	}
	if !valid {
		return knowledgeadmin.ErrInvalid
	}
	return nil
}
func (s *Store) planKnowledgeCutoverTx(ctx context.Context, tx *sql.Tx, in knowledgeadmin.CutoverInput, at time.Time) (out knowledgeadmin.CutoverPlan, e error) {
	if !maintenanceInputValid(in) || s.codeSHA != in.CodeSHA {
		return out, knowledgeadmin.ErrInvalid
	}
	if e = maintenanceAdmin(ctx, tx, in.ActorID); e != nil {
		return out, e
	}
	if e = managedSchemaIntegrity(ctx, tx); e != nil {
		return out, e
	}
	in.OldIDs = append([]string{}, in.OldIDs...)
	sort.Strings(in.OldIDs)
	out = knowledgeadmin.CutoverPlan{Input: in, Counts: map[string]int{}, PlannedAt: at.UTC()}
	var ids []string
	var idsJSON []byte
	e = tx.QueryRowContext(ctx, `SELECT coalesce(jsonb_agg(m.id ORDER BY m.id),'[]'::jsonb) FROM publication_members m JOIN publication_heads h ON h.singleton AND h.snapshot_id=m.snapshot_id JOIN publication_snapshots s ON s.id=h.snapshot_id AND s.status='published' WHERE m.kind='knowledge' AND m.availability='active'`).Scan(&idsJSON)
	if e != nil {
		return out, e
	}
	if e = json.Unmarshal(idsJSON, &ids); e != nil {
		return out, e
	}
	if !reflect.DeepEqual(ids, in.OldIDs) {
		return out, knowledgeadmin.ErrConflict
	}
	out.Counts["published"] = len(ids)
	// These references are outside the already-authorized learning cleanup scope.
	var unrelated int
	e = tx.QueryRowContext(ctx, `SELECT (SELECT count(*) FROM practice_attempts WHERE knowledge_id=ANY($1::text[]))+(SELECT count(*) FROM assessment_attempts WHERE knowledge_id=ANY($1::text[]))+(SELECT count(*) FROM learning_path_nodes WHERE knowledge_id=ANY($1::text[]))+(SELECT count(*) FROM learning_unlocks WHERE knowledge_id=ANY($1::text[]) AND source_kind<>'learning-event')+(SELECT count(*) FROM feedback_tickets WHERE target#>>'{identity,id}'=ANY($1::text[]))+(SELECT count(*) FROM learning_evidence_dependencies WHERE kind='knowledge' AND id=ANY($1::text[]) AND evidence_kind<>'learning-event')`, ids).Scan(&unrelated)
	if e != nil {
		return out, e
	}
	if unrelated != 0 {
		return out, knowledgeadmin.ErrConflict
	}
	values := map[string]string{}
	for table, scope := range cleanupScopes {
		n, h, e := rowFingerprint(ctx, tx, table, scope, ids)
		if e != nil {
			return out, e
		}
		out.Counts[table] = n
		values[table] = h
	}
	n, h, e := rowFingerprint(ctx, tx, "knowledge_versions", "id=ANY($1::text[])", ids)
	if e != nil {
		return out, e
	}
	out.Counts["knowledgeVersions"] = n
	values["knowledge_versions"] = h
	out.OldFingerprint = knowledgeFingerprint(values)
	out.ProtectedFingerprint, e = protectedFingerprint(ctx, tx, ids)
	if e != nil {
		return out, e
	}
	out.PlanSHA = knowledgeFingerprint(out)
	return out, nil
}
func (s *Store) PlanKnowledgeCutover(ctx context.Context, in knowledgeadmin.CutoverInput) (out knowledgeadmin.CutoverPlan, e error) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	tx, e := s.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true, Isolation: sql.LevelRepeatableRead})
	if e != nil {
		return out, e
	}
	defer tx.Rollback()
	out, e = s.planKnowledgeCutoverTx(ctx, tx, in, time.Now())
	if e != nil {
		return out, knowledgeError(e)
	}
	return out, knowledgeError(tx.Commit())
}
func (s *Store) ApplyKnowledgeCutover(ctx context.Context, plan knowledgeadmin.CutoverPlan) (out knowledgeadmin.CutoverReceipt, e error) {
	ctx, cancel := context.WithTimeout(ctx, 4*time.Minute)
	defer cancel()
	tx, e := s.db.BeginTx(ctx, nil)
	if e != nil {
		return out, e
	}
	defer tx.Rollback()
	if _, e = tx.ExecContext(ctx, `SET LOCAL lock_timeout='2s'`); e != nil {
		return out, e
	}
	for _, lock := range []int64{adminLockID, 1296127048, knowledgeLockID} {
		if _, e = tx.ExecContext(ctx, `SELECT pg_advisory_xact_lock($1)`, lock); e != nil {
			return out, e
		}
	}
	if !maintenanceInputValid(plan.Input) || s.codeSHA != plan.Input.CodeSHA {
		return out, knowledgeadmin.ErrInvalid
	}
	if e = maintenanceAdmin(ctx, tx, plan.Input.ActorID); e != nil {
		return out, e
	}
	claimed := plan.PlanSHA
	unsigned := plan
	unsigned.PlanSHA = ""
	if !backupRecordPattern.MatchString(claimed) || knowledgeFingerprint(unsigned) != claimed {
		return out, knowledgeadmin.ErrStale
	}
	var saved []byte
	e = tx.QueryRowContext(ctx, `SELECT evidence->'receipt' FROM managed_knowledge_events WHERE action='clean-old' AND evidence->>'planSha'=$1`, plan.PlanSHA).Scan(&saved)
	if e == nil {
		if json.Unmarshal(saved, &out) != nil {
			return out, knowledgeadmin.ErrNotConfigured
		}
		return out, nil
	}
	if !errors.Is(e, sql.ErrNoRows) {
		return out, e
	}
	if time.Since(plan.PlannedAt) > 5*time.Minute || plan.PlannedAt.After(time.Now().Add(time.Minute)) {
		return out, knowledgeadmin.ErrStale
	}
	var mode string
	e = tx.QueryRowContext(ctx, `SELECT content_mode FROM knowledge_admin_state WHERE singleton FOR UPDATE`).Scan(&mode)
	if e != nil || mode != "managed" {
		return out, knowledgeadmin.ErrNotConfigured
	}
	var published int
	if e = tx.QueryRowContext(ctx, `SELECT count(*) FROM managed_knowledge WHERE published AND deleted_at IS NULL`).Scan(&published); e != nil {
		return out, e
	}
	tables, e := maintenanceTables(ctx, tx)
	if e != nil {
		return out, e
	}
	for _, table := range tables {
		if maintenanceIgnored[table] {
			continue
		}
		if _, e = tx.ExecContext(ctx, `LOCK TABLE "`+table+`" IN SHARE ROW EXCLUSIVE MODE`); e != nil {
			return out, e
		}
	}
	fresh, e := s.planKnowledgeCutoverTx(ctx, tx, plan.Input, plan.PlannedAt)
	if e != nil {
		return out, e
	}
	if fresh.PlanSHA != plan.PlanSHA || fresh.OldFingerprint != plan.OldFingerprint || fresh.ProtectedFingerprint != plan.ProtectedFingerprint || !reflect.DeepEqual(fresh.Counts, plan.Counts) {
		return out, knowledgeadmin.ErrStale
	}
	guards := [][2]string{{"study_legacy_event_links", "study_legacy_link_immutable"}, {"study_notes", "study_note_guard"}, {"study_events", "study_event_immutable"}, {"study_idempotency", "study_receipt_immutable"}, {"learning_records", "learning_record_guard"}, {"learning_events", "learning_event_immutable"}, {"learning_unlocks", "learning_unlock_immutable"}, {"learning_evidence_dependencies", "learning_dependency_immutable"}, {"learning_idempotency", "learning_idempotency_immutable"}}
	for _, g := range guards {
		if _, e = tx.ExecContext(ctx, `ALTER TABLE `+g[0]+` DISABLE TRIGGER `+g[1]); e != nil {
			return out, e
		}
	}
	order := []string{"study_legacy_event_links", "study_notes", "study_events", "study_idempotency", "study_records", "learning_unlocks", "learning_evidence_dependencies", "learning_records", "learning_idempotency", "learning_events"}
	for _, table := range order {
		result, e := tx.ExecContext(ctx, `DELETE FROM `+table+` WHERE `+cleanupScopes[table], plan.Input.OldIDs)
		if e != nil {
			return out, e
		}
		n, e := result.RowsAffected()
		if e != nil || n != int64(plan.Counts[table]) {
			return out, knowledgeadmin.ErrStale
		}
	}
	// Finish all deferred FK/constraint checks before ALTER restores guards.
	if _, e = tx.ExecContext(ctx, `SET CONSTRAINTS ALL IMMEDIATE`); e != nil {
		return out, e
	}
	for _, g := range guards {
		if _, e = tx.ExecContext(ctx, `ALTER TABLE `+g[0]+` ENABLE TRIGGER `+g[1]); e != nil {
			return out, e
		}
	}
	after, e := protectedFingerprint(ctx, tx, plan.Input.OldIDs)
	if e != nil {
		return out, e
	}
	if after != plan.ProtectedFingerprint {
		return out, knowledgeadmin.ErrConflict
	}
	out = knowledgeadmin.CutoverReceipt{PlanSHA: plan.PlanSHA, RemovedPublicKnowledge: 30, Counts: plan.Counts, ProtectedFingerprint: after}
	if e = tx.QueryRowContext(ctx, `SELECT gen_random_uuid()::text,clock_timestamp()`).Scan(&out.OperationID, &out.RecordedAt); e != nil {
		return out, e
	}
	evidence := map[string]any{"planSha": plan.PlanSHA, "plan": plan, "receipt": out}
	if _, e = tx.ExecContext(ctx, `INSERT INTO managed_knowledge_events(event_id,actor_user_id,action,evidence,recorded_at) VALUES($1,$2,'clean-old',$3,$4)`, out.OperationID, plan.Input.ActorID, knowledgeJSON(evidence), out.RecordedAt); e != nil {
		return out, e
	}
	return out, knowledgeError(tx.Commit())
}

func (s *Store) ActivateManagedKnowledge(ctx context.Context, in knowledgeadmin.CutoverInput) (out knowledgeadmin.CutoverReceipt, e error) {
	if !maintenanceBackupValid(in) || len(in.OldIDs) != 0 || in.CodeSHA != s.codeSHA {
		return out, knowledgeadmin.ErrInvalid
	}
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	tx, e := s.db.BeginTx(ctx, nil)
	if e != nil {
		return out, e
	}
	defer tx.Rollback()
	if _, e = tx.ExecContext(ctx, `SET LOCAL lock_timeout='2s'`); e != nil {
		return out, e
	}
	for _, lock := range []int64{adminLockID, 1296127048, knowledgeLockID} {
		if _, e = tx.ExecContext(ctx, `SELECT pg_advisory_xact_lock($1)`, lock); e != nil {
			return out, e
		}
	}
	if e = maintenanceAdmin(ctx, tx, in.ActorID); e != nil {
		return out, e
	}
	if e = managedSchemaIntegrity(ctx, tx); e != nil {
		return out, e
	}
	var mode string
	var marker bool
	if e = tx.QueryRowContext(ctx, `SELECT content_mode FROM knowledge_admin_state WHERE singleton FOR UPDATE`).Scan(&mode); e != nil {
		return out, e
	}
	if e = tx.QueryRowContext(ctx, `SELECT managed_knowledge_enabled FROM goose_db_version WHERE version_id=0`).Scan(&marker); e != nil {
		return out, e
	}
	if marker != (mode == "managed") {
		return out, knowledgeadmin.ErrNotConfigured
	}
	if marker {
		var b []byte
		if e = tx.QueryRowContext(ctx, `SELECT evidence->'receipt' FROM managed_knowledge_events WHERE action='activate' ORDER BY recorded_at LIMIT 1`).Scan(&b); e != nil {
			return out, knowledgeadmin.ErrNotConfigured
		}
		if e = json.Unmarshal(b, &out); e != nil {
			return out, e
		}
		return out, nil
	}
	var total, primary, secondary, specific, other, auxiliary int
	e = tx.QueryRowContext(ctx, `SELECT count(*),count(*) FILTER(WHERE n.kind='primary' AND n.level=1),count(*) FILTER(WHERE n.kind='primary' AND n.level=2),count(*) FILTER(WHERE n.kind='primary' AND n.level=3),count(*) FILTER(WHERE n.kind='other'),count(*) FILTER(WHERE n.kind='auxiliary') FROM taxonomy_nodes n JOIN taxonomy_heads h ON h.singleton JOIN taxonomy_releases r ON r.id=h.release_id AND r.status='published' AND r.taxonomy_version_id=n.taxonomy_version_id`).Scan(&total, &primary, &secondary, &specific, &other, &auxiliary)
	if e != nil {
		return out, e
	}
	if total != 6603 || primary != 63 || secondary != 534 || specific != 4969 || other != 534 || auxiliary != 503 {
		return out, knowledgeadmin.ErrNotConfigured
	}
	var n int
	if e = tx.QueryRowContext(ctx, `SELECT count(*) FROM managed_knowledge WHERE published AND deleted_at IS NULL`).Scan(&n); e != nil {
		return out, e
	}
	if _, e = tx.ExecContext(ctx, `UPDATE knowledge_admin_state SET content_mode='managed',enabled_once=true,activated_at=clock_timestamp() WHERE singleton`); e != nil {
		return out, e
	}
	if _, e = tx.ExecContext(ctx, `UPDATE goose_db_version SET managed_knowledge_enabled=true WHERE version_id=0`); e != nil {
		return out, e
	}
	if e = tx.QueryRowContext(ctx, `SELECT gen_random_uuid()::text,clock_timestamp()`).Scan(&out.OperationID, &out.RecordedAt); e != nil {
		return out, e
	}
	out.Counts = map[string]int{"published": n, "taxonomyNodes": total}
	out.PlanSHA = knowledgeFingerprint(in)
	if _, e = tx.ExecContext(ctx, `INSERT INTO managed_knowledge_events(event_id,actor_user_id,action,evidence,recorded_at) VALUES($1,$2,'activate',$3,$4)`, out.OperationID, in.ActorID, knowledgeJSON(map[string]any{"input": in, "receipt": out}), out.RecordedAt); e != nil {
		return out, e
	}
	return out, knowledgeError(tx.Commit())
}
