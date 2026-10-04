from pathlib import Path
import json,runpy
runpy.run_path('/private/tmp/math-master-p5b-write-group-diagnostic.py')
r=Path('/Users/wiw/.codex/worktrees/p5-feedback-design/math_master');o=Path('/private/tmp/math-master-p5b-capacity-overlay.json');overlay=json.loads(o.read_text())
def temp(name):
 source=r/'backend/internal/store'/name
 if str(source) not in overlay['Replace']:
  q=Path('/private/tmp/p5b-bundle-'+name);q.write_text(source.read_text());overlay['Replace'][str(source)]=str(q)
 return Path(overlay['Replace'][str(source)])
# Share the unchanged decoders with the existing readers.
p=temp('practice_read.go');s=p.read_text();start=s.index('\tp.Summary.Kind = "practice"',s.index('func learningReadPractice('));end=s.index('\n}\nfunc learningSafeQuestions',start)
original=s[start:end];s=s[:start]+'\treturn learningDecodePracticeRecord(p, raw, answer, terminal)\n}\nfunc learningDecodePracticeRecord(p practiceRecord,raw,answer []byte,terminal sql.NullTime)(practiceRecord,error){\n var e error\n'+original+s[end:];p.write_text(s)
p=temp('assessment_evidence.go');s=p.read_text();start=s.index('\t\tvar env struct {',s.index('func learningLoadItems('));end=s.index('\n\t\tout = append(out, env.Body)',start)
original=s[start:end];original=original.replace('return out,','return question.Instance{},');original+='\n\treturn env.Body,nil\n'
s=s[:start]+'''\t\ti,e:=learningDecodeBoundItem(seal,pos,sha,raw,proof)
\t\tif e!=nil{return out,e}'''+s[end:];s=s.replace('out = append(out, env.Body)','out = append(out, i)',1)
pos=s.index('\n// Fixed evidence');s=s[:pos]+'\nfunc learningDecodeBoundItem(seal assessment.Seal,pos int,sha string,raw,proof []byte)(question.Instance,error){\n'+original+'}\n'+s[pos:];p.write_text(s)
p=temp('correction_plans.go');s=p.read_text();start=s.index('\tif p.Frozen != nil {',s.index('func correctionReadPlan('));end=s.index('\n\treturn p, nil',start)
old=s[start:end];inside=old[len('\tif p.Frozen != nil {\n'):-len('\n\t}')]
inside=inside.replace('json.Unmarshal(p.Frozen, &env)','json.Unmarshal(frozen, &env)').replace('body(p.Input)','body(input)').replace('return p, auth.ErrUnavailable','return correction.PlanProof{}, auth.ErrUnavailable').replace('\t\tp.Proof = env.Body.Proof','\t\treturn env.Body.Proof,nil')
s=s[:start]+'''\tif p.Frozen !=nil {p.Proof,e=correctionDecodeFrozenPlan(p.Frozen,p.Input);if e!=nil{return p,e}}'''+s[end:]
pos=s.index('\nfunc (s *Store) CreateCorrectionPlan');s=s[:pos]+'\nfunc correctionDecodeFrozenPlan(frozen []byte,input correction.PlanInput)(correction.PlanProof,error){\n'+inside+'\n}\n'+s[pos:];p.write_text(s)
# Current approved-plan selection expression, unchanged predicates/limits.
source=(r/'backend/internal/store/correction_results.go').read_text();line=next(v for v in source.splitlines() if 'SELECT p.id::text,p.version,p.case_id::text' in v)
planExpr=line.split('rows, e := tx.QueryContext(ctx, ',1)[1].split(', ref.Kind, ref.ID, meta.Owner, caseID)',1)[0]
code=r'''package store
import("context";"database/sql";"encoding/json";"strings";"time";"github.com/yyl1212/math_master/backend/internal/assessment";"github.com/yyl1212/math_master/backend/internal/auth";"github.com/yyl1212/math_master/backend/internal/correction";"github.com/yyl1212/math_master/backend/internal/question")
type correctionBasisFactsKey struct{}
type correctionBasisFacts struct{tx *sql.Tx;proofs map[correction.PlanRef]correction.PlanProof}
func correctionBasisFactsFor(ctx context.Context,tx *sql.Tx)*correctionBasisFacts{f,_:=ctx.Value(correctionBasisFactsKey{}).(*correctionBasisFacts);if f==nil||f.tx!=tx{return nil};return f}
type correctionPrior struct{id string;n int}
type correctionPracticeInputs struct{practiceRaw,practiceMeta []byte;items []correctionInputItem;deps []correction.Dependency;proofs []correctionProofRecord;planInputs map[correction.PlanRef]correctionInputPlan;old []correctionPrior;parentRaw map[string][]byte;planBytes,parentBytes int}
type correctionInputItem struct{Position int `json:"position"`;SHA string `json:"sha256"`;raw,proof []byte}
type correctionInputPlan struct{Ref correction.PlanRef `json:"ref"`;CaseID string `json:"caseId"`;Parent *correction.PlanRef `json:"parent"`;Bytes int `json:"bytes"`;Input correction.PlanInput `json:"input"`;raw []byte}
func correctionReadPracticeInputs(ctx context.Context,tx *sql.Tx,meta correctionEvidenceMetadata,caseID string,plan *correction.PlanRef,source string)(*correctionPracticeInputs,error){
 x:=&correctionPracticeInputs{deps:[]correction.Dependency{},proofs:[]correctionProofRecord{},planInputs:map[correction.PlanRef]correctionInputPlan{},old:[]correctionPrior{},parentRaw:map[string][]byte{}}
 itemSQL:=strings.ReplaceAll(learningItemsSQL,"$1::jsonb","(SELECT seal#>'{body,items}' FROM practice)");itemSQL=strings.ReplaceAll(itemSQL,"$2","(SELECT question_publication_id FROM practice)")
 query:=`WITH practice AS MATERIALIZED (SELECT * FROM practice_attempts WHERE $1='practice' AND id=$2 AND owner_user_id=$3 FOR UPDATE),
 items AS (`+itemSQL+`),
 plan_candidates(id,version,case_id,parent_id,parent_version,bytes) AS MATERIALIZED (`+PLAN_EXPR+`),
 plan_meta AS MATERIALIZED (SELECT row_number() OVER(ORDER BY id::uuid,version) ordinal,* FROM plan_candidates WHERE $5),
 parent_meta AS MATERIALIZED (SELECT row_number() OVER(ORDER BY r.id) ordinal,r.id,octet_length(r.basis_bytes) bytes FROM correction_results r WHERE $5 AND r.owner_user_id=$3 AND r.evidence_kind=$1 AND r.evidence_id=$2 AND r.source_key<>$6 AND `+correctionApprovedResultSQL("r")+` AND `+correctionLeafSQL("r")+` ORDER BY r.id LIMIT 101),
 limits AS MATERIALIZED (SELECT (SELECT count(*) FROM plan_meta) plans,coalesce((SELECT sum(bytes) FROM plan_meta),0) plan_bytes,(SELECT count(*) FROM parent_meta) parents,coalesce((SELECT sum(bytes) FROM parent_meta),0) parent_bytes),
 pieces AS (
 SELECT 'practice'::text kind,0::bigint ordinal,jsonb_build_object('id',p.id,'knowledge',jsonb_build_object('id',p.knowledge_id,'version',p.knowledge_version,'sha256',p.knowledge_sha256),'state',p.state,'createdAt',p.created_at,'expiresAt',p.expires_at,'terminalAt',p.terminal_at,'answer',p.answer,'correct',p.correct) payload,p.seal_bytes raw,NULL::jsonb proof FROM practice p
 UNION ALL SELECT 'dependency',row_number() OVER(ORDER BY kind,id,version,sha256),jsonb_build_object('kind',kind,'id',id,'version',version,'sha256',sha256),NULL,NULL FROM learning_evidence_dependencies WHERE evidence_kind=$1 AND evidence_id=$2
 UNION ALL SELECT 'item',position,jsonb_build_object('position',position,'sha256',sha256),body_bytes,evidence FROM items
 UNION ALL SELECT 'limit',0,jsonb_build_object('planBytes',plan_bytes,'parentBytes',parent_bytes),NULL,NULL FROM limits
 UNION ALL SELECT 'plan',m.ordinal,jsonb_build_object('ref',jsonb_build_object('id',m.id,'version',m.version),'caseId',m.case_id,'parent',CASE WHEN m.parent_id IS NULL THEN NULL ELSE jsonb_build_object('id',m.parent_id,'version',m.parent_version) END,'bytes',m.bytes,'input',CASE WHEN l.plans<=100 AND l.plan_bytes<=$7 THEN p.body ELSE NULL END),CASE WHEN l.plans<=100 AND l.plan_bytes<=$7 THEN p.frozen_bytes ELSE NULL END,NULL FROM plan_meta m JOIN correction_plans p ON p.id=m.id::uuid AND p.version=m.version CROSS JOIN limits l
 UNION ALL SELECT 'parent',m.ordinal,jsonb_build_object('id',m.id,'bytes',m.bytes),CASE WHEN l.parents<=100 AND l.parent_bytes+l.plan_bytes<=$7 THEN r.basis_bytes ELSE NULL END,NULL FROM parent_meta m JOIN correction_results r ON r.id=m.id CROSS JOIN limits l
 ) SELECT kind,payload,raw,proof FROM pieces ORDER BY kind,ordinal`
 rows,e:=tx.QueryContext(ctx,query,meta.Ref.Kind,meta.Ref.ID,meta.Owner,caseID,plan!=nil,source,correction.MaxResponseBytes);if e!=nil{return nil,e};defer rows.Close()
 for rows.Next(){var kind string;var payload,raw,proof []byte;if e=rows.Scan(&kind,&payload,&raw,&proof);e!=nil{return nil,e}
 switch kind{case "practice":x.practiceRaw=raw;x.practiceMeta=payload
 case "dependency":var d correction.Dependency;if json.Unmarshal(payload,&d)!=nil{return nil,auth.ErrUnavailable};x.deps=append(x.deps,d)
 case "item":var i correctionInputItem;if json.Unmarshal(payload,&i)!=nil{return nil,auth.ErrUnavailable};i.raw=raw;i.proof=proof;x.items=append(x.items,i)
 case "limit":var l struct{PlanBytes int `json:"planBytes"`;ParentBytes int `json:"parentBytes"`};if json.Unmarshal(payload,&l)!=nil{return nil,auth.ErrUnavailable};x.planBytes=l.PlanBytes;x.parentBytes=l.ParentBytes
 case "plan":var p correctionInputPlan;if json.Unmarshal(payload,&p)!=nil{return nil,auth.ErrUnavailable};p.raw=raw;x.planInputs[p.Ref]=p;x.proofs=append(x.proofs,correctionProofRecord{Ref:p.Ref,CaseID:p.CaseID,Parent:p.Parent,Bytes:p.Bytes})
 case "parent":var p struct{ID string `json:"id"`;Bytes int `json:"bytes"`};if json.Unmarshal(payload,&p)!=nil{return nil,auth.ErrUnavailable};x.old=append(x.old,correctionPrior{p.ID,p.Bytes});x.parentRaw[p.ID]=raw
 default:return nil,auth.ErrUnavailable}}
 if e=rows.Err();e!=nil{return nil,e};return x,nil
}
func (x *correctionPracticeInputs) base()(correction.Basis,error){
 b:=correctionEmptyBasis();b.AuditDeps=append(b.AuditDeps,x.deps...);b.EffectiveDeps=append(b.EffectiveDeps,b.AuditDeps...)
 if len(x.practiceRaw)==0{return b,auth.ErrNotFound}
 var m struct{ID string `json:"id"`;Knowledge question.Identity `json:"knowledge"`;State string `json:"state"`;CreatedAt time.Time `json:"createdAt"`;ExpiresAt time.Time `json:"expiresAt"`;TerminalAt *time.Time `json:"terminalAt"`;Answer json.RawMessage `json:"answer"`;Correct *bool `json:"correct"`}
 if json.Unmarshal(x.practiceMeta,&m)!=nil{return b,auth.ErrUnavailable};p:=practiceRecord{Summary:assessment.AttemptSummary{ID:m.ID,Knowledge:m.Knowledge,State:m.State,CreatedAt:m.CreatedAt,ExpiresAt:m.ExpiresAt},Correct:m.Correct}
 terminal:=sql.NullTime{};if m.TerminalAt!=nil{terminal=sql.NullTime{Time:*m.TerminalAt,Valid:true}}
 p,e:=learningDecodePracticeRecord(p,x.practiceRaw,m.Answer,terminal);if e!=nil{return b,e};if p.Summary.State!="answered"||p.Answer==nil{return b,correction.ErrSourceStale};b.OriginalSeal=p.Seal;b.OriginalAnswers=append(b.OriginalAnswers,*p.Answer)
 if len(p.Seal.Items)!=1&&len(p.Seal.Items)!=5{return b,auth.ErrInvalidInput}
 for _,i:=range x.items{q,e:=learningDecodeBoundItem(p.Seal,i.Position,i.SHA,i.raw,i.proof);if e!=nil{return b,e};b.OriginalItems=append(b.OriginalItems,q)}
 if len(b.OriginalItems)!=len(p.Seal.Items){return b,auth.ErrNotFound};b.EffectiveItems=append(b.EffectiveItems,b.OriginalItems...);return b,nil
}
'''.replace('PLAN_EXPR',planExpr)
q=Path('/private/tmp/p5b_capacity_source_bundle.go');q.write_text(code);overlay['Replace'][str(r/'backend/internal/store/correction_capacity_source_bundle.go')]=str(q)
p=temp('correction_process.go');s=p.read_text();needle='ctx = context.WithValue(ctx, correctionProcessingKey{}, j)';assert needle in s;s=s.replace(needle,needle+'\n ctx=context.WithValue(ctx,correctionBasisFactsKey{},&correctionBasisFacts{tx:tx,proofs:map[correction.PlanRef]correction.PlanProof{}})',1);p.write_text(s)
p=temp('correction_results.go');s=p.read_text();start=s.index('func correctionBuildBasisMetadata(');old='\tbase, e := correctionOriginalBasis(ctx, tx, meta)';pos=s.index(old,start)
s=s[:pos]+''' var inputs *correctionPracticeInputs
 var base correction.Basis
 var e error
 if j,ok:=ctx.Value(correctionProcessingKey{}).(correctionJobRecord);ok&&meta.Ref.Kind==correction.PracticeEvidence{inputs,e=correctionReadPracticeInputs(ctx,tx,meta,caseID,plan,j.Source);if e!=nil{return correctionEmptyBasis(),e};base,e=inputs.base()}else{base,e=correctionOriginalBasis(ctx,tx,meta)}'''+s[pos+len(old):]
pos=s.index('\trows, e := tx.QueryContext(ctx, `WITH evidence AS MATERIALIZED',start);end=s.index('\n\tif len(proofs) > 100',pos);block=s[pos:end];block=block.replace('rows, e :=','rows, e =',1).replace('\tproofs := []correctionProofRecord{}\n\ttotal := 0\n','')
s=s[:pos]+''' var rows *sql.Rows
 proofs:=[]correctionProofRecord{};total:=0
 if inputs!=nil{proofs=inputs.proofs;total=inputs.planBytes}else{
'''+block+'\n }'+s[end:]
old='''		p, e := correctionReadPlan(ctx, tx, r.Ref, false)
		if e != nil {
			return base, e
		}
		r.Proof = p.Proof'''
new=''' if inputs!=nil {p:=inputs.planInputs[r.Ref];r.Proof,e=correctionDecodeFrozenPlan(p.raw,p.Input)} else{var p correctionPlanRecord;p,e=correctionReadPlan(ctx,tx,r.Ref,false);r.Proof=p.Proof}
 if e!=nil{return base,e}
 if facts:=correctionBasisFactsFor(ctx,tx);facts!=nil{facts.proofs[r.Ref]=r.Proof}'''
assert old in s;s=s.replace(old,new,1)
pos=s.index('\trows, e = tx.QueryContext(ctx, `SELECT r.id::text,octet_length(r.basis_bytes)',start);end=s.index('\n\tif len(old) > 100',pos);block=s[pos:end];a=block.index('\ttype prior struct {');b=block.index('\tfor rows.Next()',a);block=block[:a]+block[b:];block=block.replace('var r prior','var r correctionPrior')
s=s[:pos]+''' old:=[]correctionPrior{}
 if inputs!=nil{old=inputs.old;total+=inputs.parentBytes}else{
'''+block+'\n }'+s[end:]
old='''		if e = tx.QueryRowContext(ctx, `SELECT basis_bytes FROM correction_results WHERE id=$1`, r.id).Scan(&raw); e != nil {
			return base, e
		}'''
new=''' if inputs!=nil{raw=inputs.parentRaw[r.id]}else{if e=tx.QueryRowContext(ctx,`SELECT basis_bytes FROM correction_results WHERE id=$1`,r.id).Scan(&raw);e!=nil{return base,e}}'''
assert old in s;s=s.replace(old,new,1)
pos=s.index('\t\trows, e := tx.QueryContext(ctx, `SELECT m#>',s.index('func correctionPositionMap('));end=s.index('\t\tfor depth :=',pos);block=s[pos:end]
prefix=''' known:=false
 if facts:=correctionBasisFactsFor(ctx,tx);facts!=nil{known=true;for _,ref:=range b.PlanRefs{p,ok:=facts.proofs[ref];if !ok{known=false;break};for _,m:=range p.Mappings{links=append(links,link{o:m.Original.Identity,r:m.Replacement.Identity,ot:m.Original.Template,rt:m.Replacement.Template})}}}
 if !known{links=[]link{}
'''
s=s[:pos]+prefix+block+'\n }\n'+s[end:];p.write_text(s)
o.write_text(json.dumps(overlay))
print('Prepared temporary practice source/approved-proof/parent bundle with original staged bounds, shared decoders and tx-bound exact proof facts; cumulative algorithm unchanged.')
