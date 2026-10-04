from pathlib import Path
import runpy, json
runpy.run_path('/private/tmp/math-master-p5b-capacity-diagnostic.py')
r=Path('/Users/wiw/.codex/worktrees/p5-feedback-design/math_master')
p=Path('/private/tmp/p5b_capacity_diagnostic_internal_test.go')
s=p.read_text()
line=next(v for v in (r/'backend/internal/store/correction_backfill.go').read_text().splitlines() if 'items, e := tx.QueryContext' in v)
query=line.split('items, e := tx.QueryContext(ctx, ',1)[1].split(', caseID, limit-total)',1)[0]
s+='''\nfunc CorrectionBackfillDiagnosticForTest(ctx context.Context,db *sql.DB,caseID string)([]string,error){
tx,e:=db.BeginTx(ctx,nil);if e!=nil{return nil,e};defer tx.Rollback();c,e:=correctionReadCase(ctx,tx,caseID);if e!=nil{return nil,e};rows,e:=tx.QueryContext(ctx,"EXPLAIN (ANALYZE,BUFFERS) "+QUERY,caseID,48);if e!=nil{return nil,e};defer rows.Close();out:=[]string{};for rows.Next(){var v string;if e=rows.Scan(&v);e!=nil{return nil,e};out=append(out,v)};return out,rows.Err()
}
'''.replace('QUERY',query)
p.write_text(s)
p=Path('/private/tmp/p5b_capacity_diagnostic_test.go');s=p.read_text();start=s.index('t.Log("DIAG setup",time.Since(start));')
s=s[:start]+'''t.Log("DIAG setup",time.Since(start));lines,e:=store.CorrectionBackfillDiagnosticForTest(f.ctx,f.db,c.ID);if e!=nil{t.Fatal(e)};for _,line:=range lines{t.Log("BACKFILL_PLAN",line)};began:=time.Now();for n:=0;n<100 && f.count(`SELECT count(*) FROM correction_cases WHERE last_backfill_at IS NULL`)>0;n++ {step:=time.Now();count,e:=f.repo.BackfillCorrections(f.ctx,50);if e!=nil{t.Fatal(e)};t.Logf("BACKFILL step=%d count=%d duration=%s remaining=%d",n,count,time.Since(step),f.count(`SELECT count(*) FROM correction_cases WHERE last_backfill_at IS NULL`))};t.Log("BACKFILL full1000",time.Since(began));
}
'''
s=s.replace('cid:=created.Data.Case.ID;f.cFinishRoots();f.cApproveAPI(cid);','cid:=created.Data.Case.ID;f.cFinishRoots();f.cApproveAPI(cid);')
p.write_text(s)
o=json.loads(Path('/private/tmp/math-master-p5b-capacity-overlay.json').read_text());Path('/private/tmp/math-master-p5b-backfill-overlay.json').write_text(json.dumps(o))
print('Prepared temporary-only exact backfill SQL explain and all 1000 cases on the unchanged 1000/1000/10000 fixture; product untouched.')
