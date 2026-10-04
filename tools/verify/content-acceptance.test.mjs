import test from 'node:test';
import assert from 'node:assert/strict';
import {readFileSync,readdirSync} from 'node:fs';
import {createHash} from 'node:crypto';
const root=new URL('../../',import.meta.url),read=p=>readFileSync(new URL(p,root));
const baseline=JSON.parse(read('docs/operations/evidence/p6a/compatibility-baseline.json'));
function compare(reader=read){for(const[p,sha]of Object.entries(baseline.files))assert.equal(createHash('sha256').update(reader(p)).digest('hex'),sha,p)}
test('eight migrations, public contracts, old data and mathematical bytes remain unchanged',()=>{assert.equal(Object.keys(baseline.files).filter(p=>p.startsWith('db/migrations/')).length,8);compare()});
test('tampered migration and old mathematics fail byte protection',()=>{for(const p of ['db/migrations/00008_correction.sql','content/packages/elementary-fractions.v1.json']){const actual=Object.keys(baseline.files).find(n=>n===p||p.includes('00008')&&n.startsWith('db/migrations/00008'));assert(actual);assert.throws(()=>compare(n=>n===actual?Buffer.concat([read(n),Buffer.from(' changed')]):read(n)))}});
test('production endpoints never expose the audit or test harness',()=>{assert.doesNotMatch(read('backend/cmd/server/main.go').toString(),/e2etest|content-audit|RunContentAudit/);const handlers=readdirSync(new URL('backend/internal/httpapi/',root)).filter(p=>p.endsWith('.go')&&!p.endsWith('_test.go')).map(p=>read('backend/internal/httpapi/'+p).toString()).join('\n');assert.doesNotMatch(handlers,/ReadContentAudit|RunContentAudit|content-acceptance/);assert.doesNotMatch(read('api/openapi.yaml').toString(),/\/content-audit|\/content-acceptance/)});
test('legacy snapshot identity and CLI explicit secret boundary persist',()=>{assert.match(read('tools/content-ingest/snapshot.mjs').toString(),/schemaVersion:\s*1/);assert.match(read('backend/internal/cli/content_audit.go').toString(),/os.Getenv\(databaseEnv\)/);assert.doesNotMatch(read('backend/internal/cli/content_audit.go').toString(),/os.Getenv\("DATABASE_URL"\)/)});
