from pathlib import Path
import json, re, runpy
r=Path('/Users/wiw/.codex/worktrees/p5-feedback-design/math_master')
runpy.run_path('/private/tmp/math-master-p5b-native-phase-diagnostic.py')
o=Path('/private/tmp/math-master-p5b-capacity-overlay.json');overlay=json.loads(o.read_text())
p=r/'backend/internal/store/correction_results.go'
s=Path(overlay['Replace'][str(p)]).read_text()
start=s.index('\tvar id string\n\t// Rechecking an identical')
end=s.index('\n\ttype dependencyRow struct',start)
old=s[start:end]
dedup=re.search(r'`(SELECT id::text FROM correction_results.*?)`',old,re.S)[1]
insert=re.search(r'`(INSERT INTO correction_results.*?)`',old,re.S)[1]
# Keep all 13 original equality predicates. Remap only existing original insert
# argument positions; independent body comparisons remain separate parameters.
param={1:2,2:3,3:7,4:8,5:4,6:5,7:12,8:13,9:16,10:22,11:17,12:23,13:24}
dedup=re.sub(r'\$(\d+)',lambda m:'$'+str(param[int(m[1])]),dedup)
dedup=dedup.replace('SELECT id::text','SELECT id',1)
assert 'VALUES(' in insert
left,tail=insert.split(' VALUES(',1);values,conflict=tail.split(') ON CONFLICT',1)
insert=left+' SELECT '+values+' WHERE NOT EXISTS(SELECT 1 FROM identical) ON CONFLICT'+conflict
query='WITH identical AS MATERIALIZED ('+dedup+'),created AS ('+insert+') SELECT id::text,false FROM identical UNION ALL SELECT id::text,true FROM created'
new='''	// 先生成未使用的ID；幂等命中仍不写入任何新行或审计。
 id, e := workflowID()
 if e != nil { return "", e }
 var created bool
 e = tx.QueryRowContext(ctx, `QUERY`, id, meta.Owner, j.Meta.CaseID, pid, pv, b.ParentResultID, meta.Ref.Kind, meta.Ref.ID, kid, kv, kh, v.Status, v.Reason, v.Score, v.Passed, body(v.Correct), body(b.HandledCaseIDs), string(raw), raw, h, j.Source, body(b.EffectiveItems), body(b.PlanRefs), body(b.AuditDeps)).Scan(&id,&created)
 if errors.Is(e, sql.ErrNoRows) {
 e = tx.QueryRowContext(ctx, `SELECT id::text FROM correction_results WHERE source_key=$1 AND evidence_kind=$2 AND evidence_id=$3 AND basis_digest=$4 AND sealed`, j.Source, meta.Ref.Kind, meta.Ref.ID, h).Scan(&id)
 return id,e
 }
 if e != nil { return "", e }
 if !created { return id,nil }
'''.replace('QUERY',query)
s=s[:start]+new+s[end:]
q=Path('/private/tmp/p5b-worker-base-results.go');q.write_text(s);overlay['Replace'][str(p)]=str(q)
# Read owner and case in original order as separate statements. The later case
# query takes a fresh snapshot after owner waits and gates failed owner policy.
p=r/'backend/internal/store/correction_batch.go';s=Path(overlay['Replace'][str(p)]).read_text()
s=s.replace('"github.com/yyl1212/math_master/backend/internal/auth"','"github.com/yyl1212/math_master/backend/internal/auth"\n "github.com/yyl1212/math_master/backend/internal/correction"',1)
s+='''
func correctionWorkerOwnerAndCase(ctx context.Context,tx *sql.Tx,owner,caseID string)(correction.CaseKind,error){
 b:=&pgx.Batch{}
 b.Queue(`SELECT id::text,must_change_password FROM auth_users WHERE id=$1 FOR UPDATE`,owner)
 b.Queue(`SELECT kind FROM correction_cases WHERE id=$1 AND sealed AND EXISTS(SELECT 1 FROM auth_users WHERE id=$2 AND NOT must_change_password) FOR UPDATE`,caseID,owner)
 var a accountRow;var kind correction.CaseKind
 supported,e:=correctionReadBatch(ctx,tx,b,func(results pgx.BatchResults)error{
 if e:=results.QueryRow().Scan(&a.User.ID,&a.User.MustChangePassword);e!=nil{return workflowRowError(e)}
 if e:=correction.Authorize(a.User,correction.ReadOwnAction);e!=nil{return e}
 return workflowRowError(results.QueryRow().Scan(&kind))
 })
 if !supported && e==nil {
 a,e=correctionReadWorkerOwner(ctx,tx,owner,true);if e!=nil{return kind,e}
 if e=correction.Authorize(a.User,correction.ReadOwnAction);e!=nil{return kind,e}
 kind,e=correctionLockCaseKind(ctx,tx,caseID)
 }
 return kind,e
}
'''
q=Path('/private/tmp/p5b-worker-base-batch.go');q.write_text(s);overlay['Replace'][str(p)]=str(q)
p=r/'backend/internal/store/correction_process.go';s=Path(overlay['Replace'][str(p)]).read_text()
start=s.index('\taccount, e := correctionReadWorkerOwner(ctx, tx, meta.Owner, true)')
end=s.index('\n\tctx = context.WithValue(ctx, correctionProcessingKey{}',start)
s=s[:start]+'''	caseKind,e:=correctionWorkerOwnerAndCase(ctx,tx,meta.Owner,j.Meta.CaseID)
 if e!=nil{return e}
 ctx=learningWithProjection(ctx,tx,meta.Owner)
'''+s[end:]
q=Path('/private/tmp/p5b-worker-base-process.go');q.write_text(s);overlay['Replace'][str(p)]=str(q)
Path('/private/tmp/math-master-p5b-worker-base-overlay.json').write_text(json.dumps(overlay))
print('Prepared temporary worker candidate: original owner→case fresh statement order and all 13 identical-result predicates; guarded base INSERT remains prior to dependencies/seal; original conflict fallback unchanged.')
