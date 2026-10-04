from pathlib import Path
import json
r=Path('/Users/wiw/.codex/worktrees/p5-feedback-design/math_master')
p=r/'backend/internal/store/correction_backfill.go';s=p.read_text()
line=next(v for v in s.splitlines() if 'items, e := tx.QueryContext' in v)
old=line.split('items, e := tx.QueryContext(ctx, ',1)[1].split(', caseID, limit-total)',1)[0]
new='''`WITH evidence AS (`+correctionEvidenceRowsSQL+`) SELECT e.kind,e.id::text,e.owner::text FROM correction_cases c CROSS JOIN LATERAL (SELECT e.kind,e.id,e.owner FROM evidence e WHERE e.terminal AND e.kind IN ('assessment','practice') AND `+correctionCasePredicateSQL(c.Kind)+` AND NOT EXISTS(SELECT 1 FROM correction_jobs j WHERE j.source_key='terminal:'||c.id::text||':'||e.kind||':'||e.id::text) ORDER BY e.kind,e.id LIMIT $2 OFFSET 0) e WHERE c.id=$1 ORDER BY e.kind,e.id`'''
assert s.count(old)==1
s=s.replace(old,new)
q=Path('/private/tmp/p5b_backfill_candidate.go');q.write_text(s)
internal=Path('/private/tmp/p5b_capacity_diagnostic_internal_test.go').read_text();assert old in internal
internal=internal.replace(old,new)
q2=Path('/private/tmp/p5b_backfill_candidate_diagnostic_test.go');q2.write_text(internal)
o=json.loads(Path('/private/tmp/math-master-p5b-backfill-after-worker-overlay.json').read_text());o['Replace'][str(p)]=str(q);o['Replace'][str(r/'backend/internal/store/correction_capacity_diagnostic_internal_test.go')]=str(q2)
Path('/private/tmp/math-master-p5b-backfill-candidate-overlay.json').write_text(json.dumps(o))
print('Prepared one-query temporary candidate: locked case feeds unchanged bounded evidence predicate through LATERAL; all original complete-set assertions remain.')
