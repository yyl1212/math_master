from pathlib import Path
import runpy
runpy.run_path('/private/tmp/math-master-p5b-capacity-diagnostic.py')
p=Path('/private/tmp/p5b_capacity_diagnostic_test.go');s=p.read_text()
old='f.cFinishRoots();f.cCapacityClonePlans(seed,998);first:=f.cCapacityPractices();'
new='f.cFinishRoots();phase:=time.Now();f.cCapacityClonePlans(seed,998);t.Log("DIAG setup plans",time.Since(phase));phase=time.Now();first:=f.cCapacityPractices();t.Log("DIAG setup practices",time.Since(phase));'
assert s.count(old)==1;p.write_text(s.replace(old,new))
print('Prepared temporary-only constructor phase timings; all fixture guards and quantities unchanged.')
