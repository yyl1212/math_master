from pathlib import Path
import runpy

runpy.run_path('/private/tmp/math-master-p5b-capacity-diagnostic.py')
p = Path('/private/tmp/p5b_capacity_diagnostic_test.go')
s = p.read_text()
old = 't.Log("DIAG setup",time.Since(start));lines,e:='
new = '''t.Log("DIAG setup",time.Since(start));
for n:=0;n<60;n++ { l,e:=f.repo.ClaimCorrectionJob(f.ctx);if e!=nil||l==nil{t.Fatal(e)};v,e:=f.repo.ProcessCorrectionJob(f.ctx,*l,50);if e!=nil||v.Processed!=50{t.Fatal(v,e)};if (n+1)%20==0{t.Log("DIAG completed",(n+1)*50,"elapsed",time.Since(start))} }
f.db.SetMaxOpenConns(1);if _,e=f.db.ExecContext(f.ctx,`LOAD 'auto_explain';SET auto_explain.log_min_duration='0.5ms';SET auto_explain.log_analyze=on;SET auto_explain.log_timing=off;SET auto_explain.log_nested_statements=on;SET auto_explain.log_parameter_max_length=0`);e!=nil{t.Fatal(e)};
var pid int;if e=f.db.QueryRow(`SELECT pg_backend_pid()`).Scan(&pid);e!=nil{t.Fatal(e)};t.Log("PG_AUTOEXPLAIN_PID",pid);
lines,e:='''
assert s.count(old) == 1
p.write_text(s.replace(old, new))
print('Prepared maximum-fixture diagnostic after 3000 real atomic results; SQL parameters remain unlogged.')
