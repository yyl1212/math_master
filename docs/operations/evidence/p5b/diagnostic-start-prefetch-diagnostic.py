from pathlib import Path
import json,runpy
runpy.run_path('/private/tmp/math-master-p5b-native-phase-diagnostic.py')
r=Path('/Users/wiw/.codex/worktrees/p5-feedback-design/math_master')
o=Path('/private/tmp/math-master-p5b-capacity-overlay.json');overlay=json.loads(o.read_text())
support=r'''package store
import("context";"database/sql";"time";"github.com/jackc/pgx/v5";"github.com/yyl1212/math_master/backend/internal/auth";"github.com/yyl1212/math_master/backend/internal/correction")
type correctionStartHintKey struct{}
type correctionStartHint struct{job string; ref correction.EvidenceRef}
type correctionStartFactsKey struct{}
type correctionStartFacts struct{tx *sql.Tx;hint correctionStartHint;j correctionJobRecord;meta correctionEvidenceMetadata;jobErr,metaErr error}
func correctionWithStartHint(ctx context.Context,l correction.Lease,ref correction.EvidenceRef)context.Context{return context.WithValue(ctx,correctionStartHintKey{},correctionStartHint{job:l.JobID,ref:ref})}
func correctionStartWorkerTx(ctx context.Context,tx *sql.Tx,locks bool)(bool,time.Time,*correctionStartFacts,error){
 var n,lc,qc int;var goose,marker bool
 e:=tx.QueryRowContext(ctx,`SELECT (SELECT count(*) FROM unnest($1::text[]) name WHERE to_regclass('public.'||name) IS NOT NULL),to_regclass('public.goose_db_version') IS NOT NULL,EXISTS(SELECT 1 FROM pg_attribute WHERE attrelid=to_regclass('public.goose_db_version') AND attname='correction_enabled' AND NOT attisdropped),(SELECT count(*) FROM unnest($2::text[]) name WHERE to_regclass('public.'||name) IS NOT NULL),(SELECT count(*) FROM unnest($3::text[]) name WHERE to_regclass('public.'||name) IS NOT NULL)`,correctionTables,learningTables,questionTables).Scan(&n,&goose,&marker,&lc,&qc)
 if e!=nil{return false,time.Time{},nil,e}
 if !goose||!marker||n!=len(correctionTables)||lc!=len(learningTables)||qc!=len(questionTables){on,e:=correctionSystemConfigured(ctx,tx);if e!=nil||!on{return on,time.Time{},nil,e};now,e:=correctionSystemEnter(ctx,tx,locks);return on,now,nil,e}
 query:=`WITH health AS MATERIALIZED (SELECT EXISTS(SELECT 1 FROM goose_db_version WHERE version_id=8 AND is_applied) AS version,EXISTS(SELECT 1 FROM goose_db_version g WHERE version_id=0 AND coalesce((to_jsonb(g)->>'correction_enabled')::boolean,false)) AS ever,(`+correctionSchemaIntegritySQL+`) AS intact),settings AS MATERIALIZED (SELECT set_config('lock_timeout','1s',true) FROM health WHERE version AND ever AND intact)`
 args:=correctionSchemaIntegrityArgs()
 if locks{query+=`,admin_fence AS MATERIALIZED(SELECT pg_advisory_xact_lock_shared($6) FROM settings),content_fence AS MATERIALIZED(SELECT pg_advisory_xact_lock_shared($7) FROM admin_fence),registration_fence AS MATERIALIZED(SELECT pg_advisory_xact_lock_shared($8) FROM content_fence)`;args=append(args,adminLockID,int64(1296127048),correctionRegistrationLock);query+=` SELECT version,ever,intact,(SELECT clock_timestamp() FROM registration_fence) FROM health`}else{query+=` SELECT version,ever,intact,(SELECT clock_timestamp() FROM settings) FROM health`}
 var version,ever,intact bool;var now *time.Time
 check:=func()error{if !version||!ever||!intact||now==nil{return correction.ErrNotConfigured};return nil}
 hint,hasHint:=ctx.Value(correctionStartHintKey{}).(correctionStartHint)
 if !hasHint{if e=tx.QueryRowContext(ctx,query,args...).Scan(&version,&ever,&intact,&now);e!=nil{return false,time.Time{},nil,e};if e=check();e!=nil{return false,time.Time{},nil,e};return true,*now,nil,nil}
 facts:=&correctionStartFacts{tx:tx,hint:hint,meta:correctionEvidenceMetadata{Ref:hint.ref}}
 batch:=&pgx.Batch{};batch.Queue(query,args...);batch.Queue(`SELECT `+correctionJobColumns+` FROM correction_jobs WHERE id=$1`,hint.job);batch.Queue(`WITH evidence AS (`+correctionEvidenceRowsSQL+`) SELECT owner::text,kid,kv,kh,terminal FROM evidence WHERE kind=$1 AND id=$2`,hint.ref.Kind,hint.ref.ID)
 supported,e:=correctionReadBatch(ctx,tx,batch,func(results pgx.BatchResults)error{
  if e:=results.QueryRow().Scan(&version,&ever,&intact,&now);e!=nil{return e};if e:=check();e!=nil{return e}
  facts.j,facts.jobErr=correctionScanJob(results.QueryRow());facts.jobErr=workflowRowError(facts.jobErr)
  var kid,kh *string;var kv *int
  facts.metaErr=results.QueryRow().Scan(&facts.meta.Owner,&kid,&kv,&kh,&facts.meta.Terminal);facts.metaErr=workflowRowError(facts.metaErr)
  if kid!=nil&&kv!=nil&&kh!=nil{facts.meta.Knowledge=&question.Identity{ID:*kid,Version:*kv,SHA256:*kh}}
  return nil
 });if !supported&&e==nil{if e=tx.QueryRowContext(ctx,query,args...).Scan(&version,&ever,&intact,&now);e!=nil{return false,time.Time{},nil,e};if e=check();e!=nil{return false,time.Time{},nil,e};return true,*now,nil,nil};if e!=nil{return false,time.Time{},nil,e};return true,*now,facts,nil
}
func correctionReadWorkerOwner(ctx context.Context,tx *sql.Tx,id string,lock bool)(accountRow,error){var a accountRow;q:=`SELECT id::text,must_change_password FROM auth_users WHERE id=$1`;if lock{q+=` FOR UPDATE`};e:=tx.QueryRowContext(ctx,q,id).Scan(&a.User.ID,&a.User.MustChangePassword);return a,workflowRowError(e)}
var _ = auth.ErrNotFound
'''
support=support.replace('"github.com/yyl1212/math_master/backend/internal/auth";','"github.com/yyl1212/math_master/backend/internal/auth";"github.com/yyl1212/math_master/backend/internal/question";')
q=Path('/private/tmp/p5b-start-prefetch.go');q.write_text(support);overlay['Replace'][str(r/'backend/internal/store/correction_start_prefetch.go')]=str(q)
p=Path(overlay['Replace'][str(r/'backend/internal/store/correction_jobs.go')]);s=p.read_text();a=s.index('\ton, e := correctionSystemConfigured(ctx, tx)');b=s.index('\tif e = fn(ctx, tx, now);',a)
s=s[:a]+'''\ton,now,facts,e:=correctionStartWorkerTx(ctx,tx,locks)
 if e!=nil{return correctionError(e)};if !on{return correction.ErrNeverEnabled};ctx=correctionWithConfig(ctx,true)
 if facts!=nil{ctx=context.WithValue(ctx,correctionStartFactsKey{},facts)}
'''+s[b:];p.write_text(s)
p=Path(overlay['Replace'][str(r/'backend/internal/store/correction_process.go')]);s=p.read_text()
a='\tj, e := correctionReadJob(ctx, tx, l.JobID, false)';assert s.count(a)==2
s=s.replace(a,'''\tvar j correctionJobRecord;var e error
 startFacts,hasStartFacts:=ctx.Value(correctionStartFactsKey{}).(*correctionStartFacts)
 hasStartFacts=hasStartFacts&&startFacts.tx==tx&&startFacts.hint.job==l.JobID&&startFacts.hint.ref==ref
 if hasStartFacts{j,e=startFacts.j,startFacts.jobErr}else{j,e=correctionReadJob(ctx,tx,l.JobID,false)}''',1)
a='\tmeta, e := correctionReadEvidenceMetadata(ctx, tx, ref)';assert s.count(a)==1
s=s.replace(a,'''\tvar meta correctionEvidenceMetadata
 if hasStartFacts{meta,e=startFacts.meta,startFacts.metaErr}else{meta,e=correctionReadEvidenceMetadata(ctx,tx,ref)}''')
s=s.replace('correctionReadAccount(ctx, tx, meta.Owner,','correctionReadWorkerOwner(ctx, tx, meta.Owner,')
a='\t\te = s.correctionSystemTx(ctx, true, func(ctx context.Context, tx *sql.Tx, now time.Time) error {\n\t\t\treturn correctionProcessEvidence';assert s.count(a)==1
s=s.replace(a,'\t\te = s.correctionSystemTx(correctionWithStartHint(ctx,l,ref), true, func(ctx context.Context, tx *sql.Tx, now time.Time) error {\n\t\t\treturn correctionProcessEvidence')
p.write_text(s);o.write_text(json.dumps(overlay))
print('Temporary candidate: fresh complete health -> gated original ordered locks/clock, original job and metadata statements, exact-tx consumed prefetch; worker ReadOwn reads only its ID/password-change policy fields; public role auth unchanged.')
