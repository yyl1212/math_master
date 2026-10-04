from pathlib import Path
import json,runpy
runpy.run_path('/private/tmp/math-master-p5b-write-group-diagnostic.py')
r=Path('/Users/wiw/.codex/worktrees/p5-feedback-design/math_master');o=Path('/private/tmp/math-master-p5b-capacity-overlay.json');overlay=json.loads(o.read_text())
p=Path(overlay['Replace'][str(r/'backend/internal/store/correction_tx.go')]);s=p.read_text()
start=s.index('const correctionSchemaIntegritySQL = `');end=s.index('`\n\nfunc correctionSchemaIntegrityArgs()',start)
sql='''(WITH relations AS MATERIALIZED (SELECT name,to_regclass('public.'||name) oid FROM unnest($6::text[]) name)
 SELECT NOT EXISTS(SELECT 1 FROM jsonb_each_text($1::jsonb) g LEFT JOIN relations rel ON rel.name=split_part(g.key,'/',1) WHERE NOT EXISTS(
 SELECT 1 FROM pg_constraint c WHERE c.conrelid=rel.oid AND c.conname=split_part(g.key,'/',2) AND c.convalidated AND c.contype::text||':'||md5(pg_get_constraintdef(c.oid))=g.value OFFSET 0))
 AND NOT EXISTS(SELECT 1 FROM jsonb_each_text($2::jsonb) g LEFT JOIN relations rel ON rel.name=split_part(g.value,':',1) WHERE NOT EXISTS(
 SELECT 1 FROM pg_trigger t WHERE t.tgrelid=rel.oid AND t.tgfoid=to_regprocedure('public.'||split_part(g.value,':',2)||'()') AND NOT t.tgisinternal AND t.tgenabled='O' AND t.tgname=g.key AND t.tgtype::text||':'||t.tgdeferrable::text||':'||t.tginitdeferred::text=split_part(g.value,':',3)||':'||split_part(g.value,':',4)||':'||split_part(g.value,':',5)))
 AND NOT EXISTS(SELECT 1 FROM unnest($3::text[]) g WHERE to_regprocedure('public.'||g) IS NULL)
 AND NOT EXISTS(SELECT 1 FROM jsonb_each_text($4::jsonb) g WHERE NOT EXISTS(
 SELECT 1 FROM pg_index i JOIN pg_class r ON r.oid=i.indexrelid JOIN pg_namespace n ON n.oid=r.relnamespace WHERE n.nspname='public' AND r.relname=g.key AND i.indisvalid AND i.indisready AND i.indisunique AND md5(pg_get_indexdef(i.indexrelid))=g.value))
 AND NOT EXISTS(SELECT 1 FROM jsonb_each_text($5::jsonb) r LEFT JOIN relations rel ON rel.name=r.key CROSS JOIN LATERAL unnest(string_to_array(r.value,',')) c WHERE NOT EXISTS(
 SELECT 1 FROM pg_attribute i WHERE i.attrelid=rel.oid AND i.attname=c AND NOT i.attisdropped OFFSET 0)))'''
s=s[:start]+'const correctionSchemaIntegritySQL = `'+sql+s[end:]
old='body(correctionUniqueIndexGuards), body(correctionColumns)}'
assert old in s;s=s.replace(old,'body(correctionUniqueIndexGuards), body(correctionColumns), append(append([]string{},correctionTables...),"goose_db_version")}',1);p.write_text(s)
o.write_text(json.dumps(overlay))
print('Prepared temporary per-query table OID factoring; every 125/132/17/19/index guard remains; no product/migration/cache changed.')
