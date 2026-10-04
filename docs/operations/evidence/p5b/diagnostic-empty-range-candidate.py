from pathlib import Path
import json, runpy
r=Path('/Users/wiw/.codex/worktrees/p5-feedback-design/math_master')
runpy.run_path('/private/tmp/math-master-p5b-backfill-diagnostic.py')
runpy.run_path('/private/tmp/math-master-p5b-backfill-after-worker.py')
p=r/'backend/internal/store/correction_backfill.go';s=p.read_text()
line=next(v for v in s.splitlines() if 'items, e := tx.QueryContext' in v)
old=line.split('items, e := tx.QueryContext(ctx, ',1)[1].split(', caseID, limit-total)',1)[0]
helper='''
// 最早记录晚于截止时间时，该规则范围为空；原完整匹配仍决定所有非空范围。
func correctionBackfillTerminalSQL(kind correction.CaseKind) string {
 evidence:=correctionEvidenceRowsSQL
 if kind==correction.GradingRuleCase {
 evidence=`SELECT 'assessment'::text kind,id,owner_user_id owner,created_at,knowledge_id kid,knowledge_version kv,knowledge_sha256 kh,rule_version,state='submitted' terminal FROM assessment_attempts WHERE sealed
 AND c.cutoff >= (SELECT first.created_at FROM assessment_attempts first WHERE first.rule_version=c.rule_version ORDER BY first.created_at,first.id,first.owner_user_id LIMIT 1)
 UNION ALL SELECT 'practice',id,owner_user_id,created_at,knowledge_id,knowledge_version,knowledge_sha256,(seal#>>'{body,ruleVersion}')::integer,state='answered' FROM practice_attempts
 WHERE c.cutoff >= (SELECT first.created_at FROM practice_attempts first WHERE (first.seal#>>'{body,ruleVersion}')::integer=c.rule_version ORDER BY first.created_at,first.id,first.owner_user_id LIMIT 1)`
 }
 return `SELECT e.kind,e.id::text,e.owner::text FROM correction_cases c CROSS JOIN LATERAL (WITH evidence AS (`+evidence+`) SELECT e.kind,e.id,e.owner FROM evidence e WHERE e.terminal AND e.kind IN ('assessment','practice') AND `+correctionCasePredicateSQL(kind)+` AND NOT EXISTS(SELECT 1 FROM correction_jobs j WHERE j.source_key='terminal:'||c.id::text||':'||e.kind||':'||e.id::text) ORDER BY e.kind,e.id LIMIT $2 OFFSET 0) e WHERE c.id=$1 ORDER BY e.kind,e.id`
}
'''
assert s.count(old)==1
s=s.replace(old,'correctionBackfillTerminalSQL(c.Kind)')+helper
q=Path('/private/tmp/p5b_empty_range_backfill.go');q.write_text(s)
o=json.loads(Path('/private/tmp/math-master-p5b-backfill-after-worker-overlay.json').read_text())
o['Replace'][str(p)]=str(q)
a=Path('/private/tmp/p5b_capacity_diagnostic_internal_test.go');s=a.read_text();assert old in s
s=s.replace(old,'correctionBackfillTerminalSQL(c.Kind)');a.write_text(s)
Path('/private/tmp/math-master-p5b-empty-range-overlay.json').write_text(json.dumps(o))
print('Prepared disposable candidate: exact original full impact test and all guards; indexed earliest-created negative guard only, no product changes.')
