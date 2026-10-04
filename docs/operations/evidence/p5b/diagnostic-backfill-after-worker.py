from pathlib import Path
import json
r=Path('/Users/wiw/.codex/worktrees/p5-feedback-design/math_master')
p=r/'backend/internal/store/correction_capacity_test.go';s=p.read_text()
old='t.Logf("capacity metadata complete elapsed=%s", time.Since(started))'
assert s.count(old)==1
s=s.replace(old,old+'''
lines, explainErr := store.CorrectionBackfillDiagnosticForTest(f.ctx,f.db,c.ID);if explainErr!=nil{t.Fatal(explainErr)};for _,line:=range lines{t.Log("BACKFILL_POST_WORKER_PLAN",line)}
backfillStarted:=time.Now()
''')
s=s.replace('n, e := f.repo.BackfillCorrections(f.ctx, 50)','backfillStepStarted:=time.Now();n, e := f.repo.BackfillCorrections(f.ctx, 50);t.Logf("BACKFILL_POST_WORKER step=%d count=%d duration=%s total=%s",i,n,time.Since(backfillStepStarted),time.Since(backfillStarted))',1)
q=Path('/private/tmp/p5b_capacity_after_worker_test.go');q.write_text(s)
o=json.loads(Path('/private/tmp/math-master-p5b-backfill-overlay.json').read_text());o['Replace'][str(p)]=str(q)
Path('/private/tmp/math-master-p5b-backfill-after-worker-overlay.json').write_text(json.dumps(o))
print('Prepared unchanged full impact test with post-worker backfill timing and exact SQL plan diagnostics only.')
