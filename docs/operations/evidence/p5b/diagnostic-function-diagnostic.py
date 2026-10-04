from pathlib import Path
import runpy
runpy.run_path('/private/tmp/math-master-p5b-phase-diagnostic.py')
p=Path('/private/tmp/p5b_capacity_diagnostic_test.go');s=p.read_text()
old='store.CorrectionCapacityTimingResetForTest();began:=time.Now();'
new='''f.db.SetMaxOpenConns(1);f.db.SetMaxIdleConns(1);if _,e=f.db.Exec(`SET track_functions='all'`);e!=nil{t.Fatal(e)};store.CorrectionCapacityTimingResetForTest();began:=time.Now();'''
assert old in s;s=s.replace(old,new,1)
old='t.Log("PHASE_PROFILE_JSON",store.CorrectionCapacityTimingReportForTest());'
new='''t.Log("PHASE_PROFILE_JSON",store.CorrectionCapacityTimingReportForTest());
if _,e=f.db.Exec(`SELECT pg_stat_force_next_flush()`);e!=nil{t.Fatal(e)}
rows,e:=f.db.Query(`SELECT funcname,calls,total_time,self_time FROM pg_stat_user_functions WHERE schemaname='public' ORDER BY self_time DESC`);if e!=nil{t.Fatal(e)};defer rows.Close()
for rows.Next(){var name string;var calls int64;var total,self float64;if e=rows.Scan(&name,&calls,&total,&self);e!=nil{t.Fatal(e)};t.Log("FUNCTION_CPU",name,calls,total,self)};if e=rows.Err();e!=nil{t.Fatal(e)}
'''
assert old in s;s=s.replace(old,new,1);p.write_text(s)
print('Prepared session-only per-function times on 500 real atomic transactions; no product/schema/global settings changed.')
