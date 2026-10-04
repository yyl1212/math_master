from pathlib import Path
import json, re
r=Path('/Users/wiw/.codex/worktrees/p5-feedback-design/math_master')
p=r/'backend/internal/store/correction_backfill.go'
s=Path('/private/tmp/p5b_empty_range_backfill.go').read_text()
needle='\t\t\tcaseID := candidate.caseID.String\n';assert s.count(needle)==1
s=s.replace(needle,needle+'\t\t\tvar lockedKind correction.CaseKind\n\t\t\tvar lockedPlans []correction.PlanRef\n\t\t\tpreloaded := false\n',1)
start=s.index('\t\t\t\tvar locked string\n');end=s.index('\n\t\t\t\tif !exists {',start)
s=s[:start]+'''				var locked string
 var exists bool
 var rawPlans []byte
 e=tx.QueryRowContext(ctx, `WITH locked AS MATERIALIZED (
 SELECT id,kind FROM correction_cases WHERE id=$1 AND sealed FOR UPDATE SKIP LOCKED
 ),root AS MATERIALIZED (
 SELECT locked.*,EXISTS(SELECT 1 FROM correction_jobs j WHERE j.source_key='case:'||locked.id::text) present FROM locked
 ) SELECT root.id::text,root.kind,root.present,
 (SELECT coalesce(jsonb_agg(jsonb_build_object('id',p.id::text,'version',p.version) ORDER BY p.created_at,p.id,p.version),'[]'::jsonb) FROM (
 SELECT p.id,p.version,p.created_at FROM correction_plans p WHERE p.case_id=root.id AND p.status='approved' AND p.sealed
 AND NOT EXISTS(SELECT 1 FROM correction_jobs j WHERE j.source_key='plan:'||p.id::text||':'||p.version::text)
 ORDER BY p.created_at,p.id,p.version LIMIT greatest(0,$2::integer-CASE WHEN root.present THEN 0 ELSE 1 END)
 ) p) FROM root`,caseID,limit-total).Scan(&locked,&lockedKind,&exists,&rawPlans)
 if errors.Is(e,sql.ErrNoRows){continue}
 if e!=nil{return e}
 if e=json.Unmarshal(rawPlans,&lockedPlans);e!=nil{return auth.ErrUnavailable}
 preloaded=true
'''+s[end:]
start=s.index('\t\t\t\tplans, e := tx.QueryContext');end=s.index('\n\t\t\t\tfor _, p := range refs {',start)
old=s[start:end];old=old.replace('\t\t\t\trefs := []correction.PlanRef{}\n','')
s=s[:start]+'''				refs:=lockedPlans
 if !preloaded {
 refs=[]correction.PlanRef{}
'''+old+'''
 }
'''+s[end:]
start=s.index('\t\t\t\tc, e := correctionReadCase(ctx, tx, caseID)');end=s.index('\n\t\t\t\t// Bind the bounded',start)
s=s[:start]+'''				kind:=lockedKind
 if !preloaded {
 c,e:=correctionReadCase(ctx,tx,caseID)
 if e!=nil{return e}
 kind=c.Kind
 }
'''+s[end:]
s=s.replace('items, e := tx.QueryContext(ctx, correctionBackfillTerminalSQL(c.Kind),','items, e := tx.QueryContext(ctx, correctionBackfillTerminalSQL(kind),',1)
s=s.replace('"encoding/json"','"encoding/json"',1) if '"encoding/json"' in s else s.replace('"database/sql"','"database/sql"\n "encoding/json"',1)
q=Path('/private/tmp/p5b_backfill_metadata.go');q.write_text(s)
o=json.loads(Path('/private/tmp/math-master-p5b-empty-range-overlay.json').read_text());o['Replace'][str(p)]=str(q)
# Regenerate only the internal diagnostic after earlier stage profiling overwrote
# a temporary file referenced by the first full candidate overlay.
s=(r/'backend/internal/store/correction_results.go').read_text()
internal='''package store
import("context";"database/sql")
func CorrectionBackfillDiagnosticForTest(ctx context.Context,db *sql.DB,caseID string)([]string,error){
 tx,e:=db.BeginTx(ctx,nil);if e!=nil{return nil,e};defer tx.Rollback();c,e:=correctionReadCase(ctx,tx,caseID);if e!=nil{return nil,e};rows,e:=tx.QueryContext(ctx,"EXPLAIN (ANALYZE,BUFFERS) "+correctionBackfillTerminalSQL(c.Kind),caseID,48);if e!=nil{return nil,e};defer rows.Close();out:=[]string{};for rows.Next(){var v string;if e=rows.Scan(&v);e!=nil{return nil,e};out=append(out,v)};return out,rows.Err()
}
'''
a=Path('/private/tmp/p5b_backfill_metadata_internal_test.go');a.write_text(internal)
o['Replace'][str(r/'backend/internal/store/correction_capacity_diagnostic_internal_test.go')]=str(a)
# Remove the obsolete setup-only diagnostic; the unchanged original impact test
# remains, with timing and explain output only.
o.pop('unused',None)
o['Replace'].pop(str(r/'backend/internal/store/correction_capacity_diagnostic_test.go'),None)
Path('/private/tmp/math-master-p5b-backfill-metadata-overlay.json').write_text(json.dumps(o))
print('Prepared temporary candidate: locked-case root/plan/kind bounded metadata read; same mutations, error rollback, rotation, terminal scan and remaining global50 budget; original full impact assertions retained.')
