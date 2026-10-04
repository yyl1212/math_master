from pathlib import Path
import json,re,runpy
runpy.run_path('/private/tmp/math-master-p5b-phase-diagnostic.py')
r=Path('/Users/wiw/.codex/worktrees/p5-feedback-design/math_master')
o=Path('/private/tmp/math-master-p5b-capacity-overlay.json');overlay=json.loads(o.read_text())
names={'correctionReadBatch','correctionReadAccount','correctionReadPracticeInputs'}
for relative in ['correction_batch.go','correction_inputs.go']:
 p=r/'backend/internal/store'/relative;s=p.read_text();found=set()
 def change(m):
  name=m[1]
  if name not in names:return m[0]
  found.add(name)
  return m[0]+'\n capacityDiagnosticStarted:=time.Now();defer capacityDiagnosticRecord("'+name+'",capacityDiagnosticStarted);\n'
 s=re.sub(r'^func (?:\([^)]*\) )?(\w+)\([^\n]*\)[^\n{]*\{',change,s,flags=re.M)
 if '"time"' not in s:s=s.replace('import (\n','import (\n "time"\n',1)
 q=Path('/private/tmp/p5b-native-phase-'+relative);q.write_text(s);overlay['Replace'][str(p)]=str(q)
 names-=found
assert not names,names
o.write_text(json.dumps(overlay))
print('Fresh current-product native stage profile on original full 1000/1000/10000 fixture; 500 independently committed worker rows, diagnostic only.')
