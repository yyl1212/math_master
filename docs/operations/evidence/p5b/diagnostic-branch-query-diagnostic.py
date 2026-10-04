from pathlib import Path
import runpy

runpy.run_path('/private/tmp/math-master-p5b-late-capacity-diagnostic.py')
p=Path('/private/tmp/p5b_capacity_diagnostic_internal_test.go')
s=p.read_text()
old='matched_cases AS MATERIALIZED (SELECT c.id FROM correction_cases c CROSS JOIN evidence e WHERE (c.id=$4 OR `+correctionCaseAffectsSQL+`))'
new='matched_cases AS MATERIALIZED (SELECT c.id FROM correction_cases c CROSS JOIN evidence e WHERE c.id=$4 UNION SELECT c.id FROM correction_cases c CROSS JOIN evidence e WHERE `+correctionGradingCaseAffectsSQL+` UNION SELECT c.id FROM correction_cases c CROSS JOIN evidence e WHERE `+correctionWithdrawalCaseAffectsSQL+`)'
assert s.count(old)==1
s=s.replace(old,new)
# This diagnostic reads only a candidate query; the 3000 result writes still use
# the unmodified production query, then share the same fixture/statistics.
p.write_text(s)
print('Prepared same-fixture candidate UNION query; product source and PG settings unchanged.')
