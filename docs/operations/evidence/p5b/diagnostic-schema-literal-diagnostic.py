from pathlib import Path
import json,runpy
runpy.run_path('/private/tmp/math-master-p5b-write-group-diagnostic.py')
r=Path('/Users/wiw/.codex/worktrees/p5-feedback-design/math_master');o=Path('/private/tmp/math-master-p5b-capacity-overlay.json');overlay=json.loads(o.read_text())
p=Path(overlay['Replace'][str(r/'backend/internal/store/correction_tx.go')]);s=p.read_text();s=s.replace('const correctionSchemaIntegritySQL = `','const correctionSchemaIntegrityTemplate = `',1)
old='return []any{body(correctionConstraintGuards), body(correctionTriggerGuards), correctionFunctionGuards, body(correctionUniqueIndexGuards), body(correctionColumns)}'
assert old in s;s=s.replace(old,'return nil',1);p.write_text(s)
q=Path('/private/tmp/p5b_capacity_schema_literal.go');q.write_text('''package store
import("strconv";"strings")
func correctionSchemaQuote(v string)string{return "'"+strings.ReplaceAll(v,"'","''")+"'"}
func correctionSchemaArray(v []string)string{parts:=[]string{};for _,s:=range v{parts=append(parts,correctionSchemaQuote(s))};return "ARRAY["+strings.Join(parts,",")+"]"}
func correctionSchemaCompile(q string,values []string)string{for idx,v:=range values{q=strings.ReplaceAll(q,"$"+strconv.Itoa(idx+1),v)};return q}
var correctionSchemaIntegritySQL=correctionSchemaCompile(correctionSchemaIntegrityTemplate,[]string{
 correctionSchemaQuote(body(correctionConstraintGuards)),correctionSchemaQuote(body(correctionTriggerGuards)),correctionSchemaArray(correctionFunctionGuards),correctionSchemaQuote(body(correctionUniqueIndexGuards)),correctionSchemaQuote(body(correctionColumns)),
})
''')
overlay['Replace'][str(r/'backend/internal/store/correction_capacity_schema_literal.go')]=str(q);o.write_text(json.dumps(overlay))
print('Prepared build-constant schema definitions as literal query constants; actual catalog predicates execute afresh in every transaction, no fact/result cache.')
