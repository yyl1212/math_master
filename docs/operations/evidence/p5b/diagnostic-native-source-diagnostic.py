from pathlib import Path
import json,runpy
runpy.run_path('/private/tmp/math-master-p5b-account-batch-diagnostic.py')
r=Path('/Users/wiw/.codex/worktrees/p5-feedback-design/math_master');o=Path('/private/tmp/math-master-p5b-capacity-overlay.json');overlay=json.loads(o.read_text())
p=Path('/private/tmp/p5b_capacity_source_bundle.go');s=p.read_text();s=s.replace('"strings";','"github.com/jackc/pgx/v5";"github.com/jackc/pgx/v5/stdlib";').replace(';raw []byte}', ';raw,inputRaw []byte}')
a=s.index(' itemSQL:=');b=s.index('\nfunc (x *correctionPracticeInputs) base()',a)
original=(r/'backend/internal/store/correction_results.go').read_text();line=next(v for v in original.splitlines() if 'SELECT p.id::text,p.version,p.case_id::text' in v);planExpr=line.split('rows, e := tx.QueryContext(ctx, ',1)[1].split(', ref.Kind, ref.ID, meta.Owner, caseID)',1)[0]
s=s[:a]+r'''
 link,ok:=ctx.Value(correctionConnectionKey{}).(correctionTransactionConnection);if !ok||link.tx!=tx{return nil,nil}
 handled:=false
 e:=link.conn.Raw(func(driverConn any)(err error){
 c,ok:=driverConn.(*stdlib.Conn);if !ok{return nil};handled=true
 batch:=&pgx.Batch{}
 batch.Queue(`SELECT id::text,knowledge_id,knowledge_version,knowledge_sha256,state,created_at,expires_at,terminal_at,seal_bytes,answer,correct FROM practice_attempts WHERE id=$1 AND owner_user_id=$2 FOR UPDATE`,meta.Ref.ID,meta.Owner)
 batch.Queue(`SELECT kind,id,version,sha256 FROM learning_evidence_dependencies WHERE evidence_kind=$1 AND evidence_id=$2 ORDER BY kind,id,version,sha256`,meta.Ref.Kind,meta.Ref.ID)
 if plan!=nil{
 batch.Queue(PLAN_EXPR,meta.Ref.Kind,meta.Ref.ID,meta.Owner,caseID)
 batch.Queue(`SELECT r.id::text,octet_length(r.basis_bytes) FROM correction_results r WHERE r.owner_user_id=$1 AND r.evidence_kind=$2 AND r.evidence_id=$3 AND r.source_key<>$4 AND `+correctionApprovedResultSQL("r")+` AND `+correctionLeafSQL("r")+` ORDER BY r.id LIMIT 101`,meta.Owner,meta.Ref.Kind,meta.Ref.ID,source)
 }
 results:=c.Conn().SendBatch(ctx,batch)
 var p practiceRecord;var raw,answer []byte;var terminal sql.NullTime
 err=results.QueryRow().Scan(&p.Summary.ID,&p.Summary.Knowledge.ID,&p.Summary.Knowledge.Version,&p.Summary.Knowledge.SHA256,&p.Summary.State,&p.Summary.CreatedAt,&p.Summary.ExpiresAt,&terminal,&raw,&answer,&p.Correct)
 if err!=nil{results.Close();return workflowRowError(err)}
 x.practiceRaw=raw
 m:=map[string]any{"id":p.Summary.ID,"knowledge":p.Summary.Knowledge,"state":p.Summary.State,"createdAt":p.Summary.CreatedAt,"expiresAt":p.Summary.ExpiresAt,"answer":json.RawMessage(answer),"correct":p.Correct};if terminal.Valid{m["terminalAt"]=terminal.Time};if len(answer)==0{m["answer"]=nil};x.practiceMeta,_=json.Marshal(m)
 rows,err:=results.Query();if err!=nil{results.Close();return err}
 for rows.Next(){var d correction.Dependency;if err=rows.Scan(&d.Kind,&d.ID,&d.Version,&d.SHA256);err!=nil{rows.Close();results.Close();return err};x.deps=append(x.deps,d)}
 err=rows.Err();rows.Close();if err!=nil{results.Close();return err}
 if plan!=nil{
 rows,err=results.Query();if err!=nil{results.Close();return err}
 for rows.Next(){var p correctionProofRecord;var id *string;var version *int;if err=rows.Scan(&p.Ref.ID,&p.Ref.Version,&p.CaseID,&id,&version,&p.Bytes);err!=nil{rows.Close();results.Close();return err};if id!=nil&&version!=nil{p.Parent=&correction.PlanRef{ID:*id,Version:*version}};x.proofs=append(x.proofs,p);x.planBytes+=p.Bytes};err=rows.Err();rows.Close();if err!=nil{results.Close();return err}
 rows,err=results.Query();if err!=nil{results.Close();return err}
 for rows.Next(){var p correctionPrior;if err=rows.Scan(&p.id,&p.n);err!=nil{rows.Close();results.Close();return err};x.old=append(x.old,p);x.parentBytes+=p.n};err=rows.Err();rows.Close();if err!=nil{results.Close();return err}
 }
 if err=results.Close();err!=nil{return err}
 p,err=learningDecodePracticeRecord(p,raw,answer,terminal);if err!=nil{return err}
 // Keep source-state and original-item validation before approval/parent bounds.
 if p.Summary.State!="answered"||p.Answer==nil{return nil}
 if len(p.Seal.Items)!=1&&len(p.Seal.Items)!=5{return nil}
 details:=&pgx.Batch{};details.Queue(learningItemsSQL,body(p.Seal.Items),p.Seal.QuestionPublicationID)
 planOK:=plan!=nil&&len(x.proofs)<=100&&x.planBytes<=correction.MaxResponseBytes
 if planOK{for _,p:=range x.proofs{details.Queue(`SELECT body,frozen_bytes FROM correction_plans WHERE id=$1 AND version=$2`,p.Ref.ID,p.Ref.Version)}}
 parentOK:=planOK&&len(x.old)<=100&&x.planBytes+x.parentBytes<=correction.MaxResponseBytes
 if parentOK{for _,p:=range x.old{details.Queue(`SELECT basis_bytes FROM correction_results WHERE id=$1`,p.id)}}
 results=c.Conn().SendBatch(ctx,details);defer func(){if e:=results.Close();err==nil{err=e}}()
 rows,err=results.Query();if err!=nil{return err};for rows.Next(){var i correctionInputItem;if err=rows.Scan(&i.Position,&i.SHA,&i.raw,&i.proof);err!=nil{rows.Close();return err};x.items=append(x.items,i)};err=rows.Err();rows.Close();if err!=nil{return err}
 if planOK{for _,p:=range x.proofs{var input,raw []byte;if err=results.QueryRow().Scan(&input,&raw);err!=nil{return workflowRowError(err)};var v correctionInputPlan;v.raw=raw;v.inputRaw=input;x.planInputs[p.Ref]=v}}
 if parentOK{for _,p:=range x.old{var raw []byte;if err=results.QueryRow().Scan(&raw);err!=nil{return err};x.parentRaw[p.id]=raw}}
 return nil
 });if !handled&&e==nil{return nil,nil};return x,e
}
''' .replace('PLAN_EXPR',planExpr)+s[b:]
p.write_text(s)
# Use native prefetches, retaining the original source-first and staged checks.
p=Path('/private/tmp/p5b-phase-correction_results.go');s=p.read_text()
s=s.replace('if false{proofs=inputs.proofs;total=inputs.planBytes}', 'if inputs!=nil{proofs=inputs.proofs;total=inputs.planBytes}')
s=s.replace('if false {p:=inputs.planInputs[r.Ref];','if inputs!=nil {p:=inputs.planInputs[r.Ref];')
s=s.replace('if false{old=inputs.old;total+=inputs.parentBytes}', 'if inputs!=nil{old=inputs.old;total+=inputs.parentBytes}')
s=s.replace('if false{raw=inputs.parentRaw[r.id]}','if inputs!=nil{raw=inputs.parentRaw[r.id]}')
s=s.replace('base,e=inputs.base()}else{','if inputs!=nil{base,e=inputs.base()}else{base,e=correctionOriginalBasis(ctx,tx,meta)}}else{')
s=s.replace('r.Proof,e=correctionDecodeFrozenPlan(p.raw,p.Input)', 'if json.Unmarshal(p.inputRaw,\u0026p.Input)!=nil{e=auth.ErrUnavailable}else if p.raw!=nil{r.Proof,e=correctionDecodeFrozenPlan(p.raw,p.Input)}')
p.write_text(s)
o.write_text(json.dumps(overlay))
print('Prepared temporary native practice prefetch in two ordered statement batches: exact source/deps/approved metadata/parent metadata, then decoded original items and bounded proof/parent bytes; no large combined SQL, per-statement snapshots preserved.')
