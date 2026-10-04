from pathlib import Path
import json,re,runpy
r=Path('/Users/wiw/.codex/worktrees/p5-feedback-design/math_master')
runpy.run_path('/private/tmp/math-master-p5b-capacity-diagnostic.py')
overlay=json.loads(Path('/private/tmp/math-master-p5b-capacity-overlay.json').read_text())
functions={'body','correctionConfigured','learningConfigured','questionConfigured','learningLocks','correctionRegistrationFence','dbClock','correctionReadJob','readAccount','learningEvidenceDependencies','learningReadPractice','learningLoadItems','correctionReadEvidenceMetadata','correctionOriginalBasis','correctionBuildBasis','correctionWriteResult','correctionPositionMap','correctionEvent','notificationAppend','correctionAdvanceCursor','correctionProcessEvidence','correctionSystemTx'}
if (r/'backend/internal/store/correction_system.go').exists():functions|={'correctionSystemConfigured','correctionSystemEnter','correctionReadJobTime','correctionLockCaseKind','correctionInstanceDeps','correctionBuildBasisMetadata'}
found=set()
for p in (r/'backend/internal/store').glob('*.go'):
 if p.name.endswith('_test.go'):continue
 s=p.read_text();changed=False
 def instrument(m):
  global changed
  name=m[1]
  if name not in functions:return m[0]
  found.add(name);changed=True
  return m[0]+'\n capacityDiagnosticStarted:=time.Now();defer capacityDiagnosticRecord("'+name+'",capacityDiagnosticStarted);\n'
 s=re.sub(r'^func (?:\([^)]*\) )?(\w+)\([^\n]*\)[^\n{]*\{',instrument,s,flags=re.M)
 if changed:
  if '"time"' not in s:
   assert 'import (\n' in s,p.name;s=s.replace('import (\n','import (\n "time"\n',1)
  q=Path('/private/tmp/p5b-phase-'+p.name);q.write_text(s);overlay['Replace'][str(p)]=str(q)
assert found==functions,(functions-found)
support='''package store
import("encoding/json";"sync";"time")
type capacityDiagnosticStat struct{Calls int;Nanoseconds int64;MaxNanoseconds int64}
var capacityDiagnosticMu sync.Mutex
var capacityDiagnosticActive bool
var capacityDiagnosticStats map[string]capacityDiagnosticStat
func capacityDiagnosticRecord(name string,start time.Time){d:=time.Since(start).Nanoseconds();capacityDiagnosticMu.Lock();defer capacityDiagnosticMu.Unlock();if !capacityDiagnosticActive{return};v:=capacityDiagnosticStats[name];v.Calls++;v.Nanoseconds+=d;if d>v.MaxNanoseconds{v.MaxNanoseconds=d};capacityDiagnosticStats[name]=v}
func CorrectionCapacityTimingResetForTest(){capacityDiagnosticMu.Lock();defer capacityDiagnosticMu.Unlock();capacityDiagnosticStats=map[string]capacityDiagnosticStat{};capacityDiagnosticActive=true}
func CorrectionCapacityTimingReportForTest()string{capacityDiagnosticMu.Lock();defer capacityDiagnosticMu.Unlock();capacityDiagnosticActive=false;b,_:=json.Marshal(capacityDiagnosticStats);return string(b)}
'''
q=Path('/private/tmp/p5b_capacity_timing.go');q.write_text(support);overlay['Replace'][str(r/'backend/internal/store/correction_capacity_timing.go')]=str(q)
p=Path('/private/tmp/p5b_capacity_diagnostic_test.go');s=p.read_text()
start=s.index('t.Log("DIAG setup",time.Since(start));')
s=s[:start]+'''t.Log("DIAG setup",time.Since(start));store.CorrectionCapacityTimingResetForTest();began:=time.Now();
for n:=0;n<10;n++ { l,e:=f.repo.ClaimCorrectionJob(f.ctx);if e!=nil||l==nil{t.Fatal(e)};v,e:=f.repo.ProcessCorrectionJob(f.ctx,*l,50);if e!=nil||v.Processed!=50{t.Fatal(v,e)} }
t.Log("DIAG real500",time.Since(began));t.Log("PHASE_PROFILE_JSON",store.CorrectionCapacityTimingReportForTest());
}
'''
p.write_text(s)
Path('/private/tmp/math-master-p5b-capacity-overlay.json').write_text(json.dumps(overlay))
print('Prepared temporary-only exact-method timings for 500 actual atomic results on the full fixture.')
