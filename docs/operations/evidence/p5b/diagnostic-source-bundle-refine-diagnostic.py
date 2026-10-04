from pathlib import Path
import json,runpy
runpy.run_path('/private/tmp/math-master-p5b-source-bundle-diagnostic.py')
p=Path('/private/tmp/p5b_capacity_source_bundle.go');s=p.read_text()
s=s.replace('parent_version,bytes) AS MATERIALIZED','parent_version,bytes,input,frozen) AS MATERIALIZED')
s=s.replace('p.parent_version,p.bytes FROM matched_cases','p.parent_version,p.bytes,p.input,p.frozen FROM matched_cases')
s=s.replace('octet_length(p.frozen_bytes) bytes FROM correction_plans','octet_length(p.frozen_bytes) bytes,p.body input,p.frozen_bytes frozen FROM correction_plans')
s=s.replace('octet_length(r.basis_bytes) bytes FROM correction_results','octet_length(r.basis_bytes) bytes,r.basis_bytes raw FROM correction_results')
s=s.replace('THEN p.body ELSE NULL','THEN m.input ELSE NULL').replace('THEN p.frozen_bytes ELSE NULL','THEN m.frozen ELSE NULL')
s=s.replace(' FROM plan_meta m JOIN correction_plans p ON p.id=m.id::uuid AND p.version=m.version CROSS JOIN limits l',' FROM plan_meta m CROSS JOIN limits l')
s=s.replace('THEN r.basis_bytes ELSE NULL','THEN m.raw ELSE NULL')
s=s.replace(' FROM parent_meta m JOIN correction_results r ON r.id=m.id CROSS JOIN limits l',' FROM parent_meta m CROSS JOIN limits l')
p.write_text(s)
print('Prepared temporary bundle refinement: return exact selected immutable proof/parent bytes without joining their tables a second time.')
