from pathlib import Path
import json
r=Path('/Users/wiw/.codex/worktrees/p5-feedback-design/math_master')
s=(r/'backend/internal/store/correction_results.go').read_text()
if 'var correctionApprovedPlanMetadataSQL =' in s:
 expression='correctionApprovedPlanMetadataSQL'
else:
 line=next(v for v in s.splitlines() if 'SELECT p.id::text,p.version,p.case_id::text' in v)
 expression=line.split('rows, e := tx.QueryContext(ctx, ',1)[1].split(', ref.Kind, ref.ID, meta.Owner, caseID)',1)[0]
internal='''package store
import("context";"database/sql";"fmt";"time";"github.com/yyl1212/math_master/backend/internal/correction")
func CorrectionCapacityDiagnosticForTest(ctx context.Context,db *sql.DB,ref correction.EvidenceRef,owner,caseID string)([]string,error){
 tx,e:=db.BeginTx(ctx,nil);if e!=nil{return nil,e};defer tx.Rollback();out:=[]string{}
 started:=time.Now();for n:=0;n<10;n++{if _,e=correctionConfigured(ctx,tx);e!=nil{return nil,e};if _,e=learningConfigured(ctx,tx);e!=nil{return nil,e};if e=questionConfigured(ctx,tx);e!=nil{return nil,e}};out=append(out,fmt.Sprintf("DIAG health 10=%s",time.Since(started)))
 meta,e:=correctionReadEvidenceMetadata(ctx,tx,ref);if e!=nil{return nil,e};started=time.Now();for n:=0;n<10;n++{if _,e=correctionOriginalBasis(ctx,tx,meta);e!=nil{return nil,e}};out=append(out,fmt.Sprintf("DIAG original basis 10=%s",time.Since(started)))
 query:=EXPRESSION
 started=time.Now();for n:=0;n<10;n++{rows,e:=tx.QueryContext(ctx,query,ref.Kind,ref.ID,owner,caseID);if e!=nil{return nil,e};for rows.Next(){};e=rows.Err();rows.Close();if e!=nil{return nil,e}};out=append(out,fmt.Sprintf("DIAG applicable plans 10=%s",time.Since(started)))
 rows,e:=tx.QueryContext(ctx,"EXPLAIN (ANALYZE, BUFFERS) "+query,ref.Kind,ref.ID,owner,caseID);if e!=nil{return nil,e};defer rows.Close();for rows.Next(){var v string;if e=rows.Scan(&v);e!=nil{return nil,e};out=append(out,v)};return out,rows.Err()
}
'''.replace('EXPRESSION',expression)
external='''package store_test
import("testing";"time";"github.com/yyl1212/math_master/backend/internal/correction";"github.com/yyl1212/math_master/backend/internal/question";"github.com/yyl1212/math_master/backend/internal/store")
func TestCorrectionCapacityDiagnostic(t *testing.T){
 start:=time.Now();f:=newCorrectionFixture(t);correctionCapacityContext(t,f);c:=f.registerRule(nil,1);seed:=f.cApproveAPI(c.ID);f.cFinishRoots();f.cCapacityClonePlans(seed,998);first:=f.cCapacityPractices();var i question.Identity;if e:=f.db.QueryRow(`SELECT seal#>>'{body,items,0,instance,id}',(seal#>>'{body,items,0,instance,version}')::int,seal#>>'{body,items,0,instance,sha256}' FROM practice_attempts WHERE id=$1`,first).Scan(&i.ID,&i.Version,&i.SHA256);e!=nil{t.Fatal(e)};wid:=f.legacyCorrectionWithdrawal(i);created,e:=f.repo.CreateCorrectionCase(f.ctx,f.Access("admin_a",false),correction.CaseInput{Kind:correction.WithdrawalCase,Withdrawal:&correction.WithdrawalRef{Space:"question",ID:wid}});if e!=nil{t.Fatal(e)};cid:=created.Data.Case.ID;f.cFinishRoots();f.cApproveAPI(cid);t.Log("DIAG setup",time.Since(start));lines,e:=store.CorrectionCapacityDiagnosticForTest(f.ctx,f.db,correction.EvidenceRef{Kind:correction.PracticeEvidence,ID:first},f.ids["learner_a"],cid);if e!=nil{t.Fatal(e)};for _,line:=range lines{t.Log(line)};l,e:=f.repo.ClaimCorrectionJob(f.ctx);if e!=nil||l==nil{t.Fatal(e)};start=time.Now();v,e:=f.repo.ProcessCorrectionJob(f.ctx,*l,50);if e!=nil||v.Processed!=50{t.Fatal(v,e)};t.Log("DIAG real batch50",time.Since(start));
}
'''
a=Path('/private/tmp/p5b_capacity_diagnostic_internal_test.go');a.write_text(internal)
b=Path('/private/tmp/p5b_capacity_diagnostic_test.go');b.write_text(external)
overlay={'Replace':{str(r/'backend/internal/store/correction_capacity_diagnostic_internal_test.go'):str(a),str(r/'backend/internal/store/correction_capacity_diagnostic_test.go'):str(b)}}
Path('/private/tmp/math-master-p5b-capacity-overlay.json').write_text(json.dumps(overlay))
print('Prepared temporary overlay: exact production plan SQL and maximum approved fixture diagnostic.')
