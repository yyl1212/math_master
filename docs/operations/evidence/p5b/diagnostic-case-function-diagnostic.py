from pathlib import Path
import runpy
runpy.run_path('/private/tmp/math-master-p5b-write-group-diagnostic.py')
r=Path('/Users/wiw/.codex/worktrees/p5-feedback-design/math_master')
sql=(r/'db/migrations/00008_correction_notifications.sql').read_text()
start=sql.index('CREATE FUNCTION correction_case_applies(');end=sql.index('$$;',sql.index('AS $$',start))+3
function=sql[start:end];body=function.split('AS $$',1)[1].rsplit('$$;',1)[0].strip()
prefix='WITH evidence AS (';boundary=') SELECT EXISTS(SELECT 1 FROM evidence e CROSS JOIN correction_cases c WHERE c.id=cid AND e.kind=ek AND e.id=eid AND e.owner=uid AND c.sealed AND '
assert body.startswith(prefix) and boundary in body
evidence,predicate=body[len(prefix):].split(boundary,1);assert predicate.endswith(')')
predicate=predicate[:-1]
candidate="WITH evidence AS MATERIALIZED (SELECT * FROM ("+evidence+") e WHERE e.kind=ek AND e.id=eid AND e.owner=uid), matched_case AS MATERIALIZED (SELECT * FROM correction_cases WHERE id=cid AND sealed) SELECT EXISTS(SELECT 1 FROM evidence e CROSS JOIN matched_case c WHERE "+predicate+")"
fn=function.split('AS $$',1)[0].replace('CREATE FUNCTION','CREATE OR REPLACE FUNCTION',1)+'AS $$\n'+candidate+'\n$$;'
Path('/private/tmp/math-master-p5b-case-function-candidate.sql').write_text(fn)
p=Path('/private/tmp/p5b_capacity_diagnostic_test.go');s=p.read_text();marker='t.Log("PHASE_PROFILE_JSON",store.CorrectionCapacityTimingReportForTest());'
assert marker in s
extra='''
f.db.SetMaxOpenConns(1);f.db.SetMaxIdleConns(1)
query:=`WITH chosen AS MATERIALIZED (SELECT id FROM practice_attempts ORDER BY id LIMIT 500) SELECT count(*) FILTER (WHERE correction_case_applies($1,'practice',chosen.id,$2)) FROM chosen`
counts:=[]int{};for _,caseID:=range []string{cid,c.ID}{for _,owner:=range []string{f.ids["learner_a"],f.ids["learner_b"],f.ids["author_a"]}{began:=time.Now();var n int;if e=f.db.QueryRow(query,caseID,owner).Scan(&n);e!=nil{t.Fatal(e)};counts=append(counts,n);t.Log("CASE_QUERY_OLD",caseID==cid,owner==f.ids["learner_a"],n,time.Since(began))}}
if counts[0]+counts[1]!=500||counts[2]!=0||counts[3]!=0||counts[4]!=0||counts[5]!=0{t.Fatal("fixture must prove owned withdrawal positives, foreign owner and old cutoff negatives",counts)}
if _,e=f.db.Exec(`CANDIDATE`);e!=nil{t.Fatal(e)}
idx:=0;for _,caseID:=range []string{cid,c.ID}{for _,owner:=range []string{f.ids["learner_a"],f.ids["learner_b"],f.ids["author_a"]}{began:=time.Now();var n int;if e=f.db.QueryRow(query,caseID,owner).Scan(&n);e!=nil{t.Fatal(e)};if n!=counts[idx]{t.Fatal("accurate source predicate changed",idx,n,counts[idx])};idx++;t.Log("CASE_QUERY_NEW",caseID==cid,owner==f.ids["learner_a"],n,time.Since(began))}}
store.CorrectionCapacityTimingResetForTest();began=time.Now()
for n:=0;n<10;n++ {l,e:=f.repo.ClaimCorrectionJob(f.ctx);if e!=nil||l==nil{t.Fatal(e)};v,e:=f.repo.ProcessCorrectionJob(f.ctx,*l,50);if e!=nil||v.Processed!=50{t.Fatal(v,e)}}
t.Log("DIAG exact-case500",time.Since(began));t.Log("CASE_PHASE_PROFILE_JSON",store.CorrectionCapacityTimingReportForTest());
'''.replace('CANDIDATE',fn)
s=s.replace(marker,marker+extra,1);p.write_text(s)
print('Prepared exact predicate factoring candidate in this disposable fixture only, old/new owner/cutoff equality and next 500 real transactions.')
