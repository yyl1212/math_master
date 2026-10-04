from pathlib import Path
import runpy
runpy.run_path('/private/tmp/math-master-p5b-source-bundle-diagnostic.py')
p=Path('/private/tmp/p5b_capacity_source_bundle.go');s=p.read_text()
a=s.index(' query:=');b=s.index('\n for rows.Next()',a)
s=s[:a]+''' query:=`WITH practice AS MATERIALIZED (SELECT * FROM practice_attempts WHERE $1='practice' AND id=$2 AND owner_user_id=$3 FOR UPDATE),items AS (`+itemSQL+`),pieces AS (
 SELECT 'practice'::text kind,0::bigint ordinal,jsonb_build_object('id',p.id,'knowledge',jsonb_build_object('id',p.knowledge_id,'version',p.knowledge_version,'sha256',p.knowledge_sha256),'state',p.state,'createdAt',p.created_at,'expiresAt',p.expires_at,'terminalAt',p.terminal_at,'answer',p.answer,'correct',p.correct) payload,p.seal_bytes raw,NULL::jsonb proof FROM practice p
 UNION ALL SELECT 'dependency',row_number() OVER(ORDER BY kind,id,version,sha256),jsonb_build_object('kind',kind,'id',id,'version',version,'sha256',sha256),NULL,NULL FROM learning_evidence_dependencies WHERE evidence_kind=$1 AND evidence_id=$2
 UNION ALL SELECT 'item',position,jsonb_build_object('position',position,'sha256',sha256),body_bytes,evidence FROM items
 ) SELECT kind,payload,raw,proof FROM pieces ORDER BY kind,ordinal`
 rows,e:=tx.QueryContext(ctx,query,meta.Ref.Kind,meta.Ref.ID,meta.Owner);if e!=nil{return nil,e};defer rows.Close()
''' +s[b:]
p.write_text(s)
p=Path('/private/tmp/p5b-phase-correction_results.go');s=p.read_text()
s=s.replace('if inputs!=nil{proofs=inputs.proofs;total=inputs.planBytes}', 'if false{proofs=inputs.proofs;total=inputs.planBytes}')
s=s.replace('if inputs!=nil {p:=inputs.planInputs[r.Ref];', 'if false {p:=inputs.planInputs[r.Ref];')
s=s.replace('if inputs!=nil{old=inputs.old;total+=inputs.parentBytes}', 'if false{old=inputs.old;total+=inputs.parentBytes}')
s=s.replace('if inputs!=nil{raw=inputs.parentRaw[r.id]}','if false{raw=inputs.parentRaw[r.id]}')
p.write_text(s)
print('Prepared temporary smaller original-practice/approved-items/dependency read; original approved-plan/parent SQL and staged bounds remain in place; exact tx-bound proof reused for positions.')
