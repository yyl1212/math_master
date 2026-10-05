import test from 'node:test';
import assert from 'node:assert/strict';
import {readFileSync,readdirSync} from 'node:fs';
import {compareApprovedBytes,exceptionPath} from './admin-review-compatibility.mjs';
const root=new URL('../../',import.meta.url),read=p=>readFileSync(new URL(p,root));
const baseline=JSON.parse(read('docs/operations/evidence/p6b/compatibility-baseline.json'));
function compare(reader=read){compareApprovedBytes(baseline,reader)}
function goFiles(path){return readdirSync(new URL(path+'/',root),{withFileTypes:true}).flatMap(e=>e.isDirectory()?goFiles(path+'/'+e.name):e.name.endsWith('.go')&&!e.name.endsWith('_test.go')?[path+'/'+e.name]:[])}
test('historical public contracts and mathematics retain exact bytes with approved administrator-review exceptions',()=>{
 assert.equal(baseline.baseCommit,'b78108d3dd363eae237dae59f76a7be774b9de85');
 assert.equal(Object.keys(baseline.files).filter(p=>p.startsWith('db/migrations/')).length,8);
 assert(Object.keys(baseline.files).length>150);compare();
});
test('API, migration, generator, grading, dual-head and legacy evidence tampering is rejected',()=>{
 for(const path of ['api/openapi.yaml','frontend/src/lib/api/generated.d.ts','db/migrations/00008_correction_notifications.sql','backend/internal/question/generate.go','backend/internal/question/grade.go','backend/internal/question/model.go','backend/internal/contentaudit/model.go','backend/internal/cli/content_audit.go']){
  assert(baseline.files[path]);assert.throws(()=>compare(p=>p===path?Buffer.concat([read(p),Buffer.from(' changed')]):read(p)),new RegExp(path.replaceAll('.','\\.')));
 }
});
test('new review entry and pure layer have no database, network, runtime configuration or publication side effects',()=>{
 for(const path of ['backend/cmd/content-review/main.go','backend/internal/cli/content_review.go',...goFiles('backend/internal/contentreview')]){
  const source=read(path).toString();
  assert.doesNotMatch(source,/"(?:database\/sql|net(?:\/[^"\n]+)?|os\/exec|github\.com\/jackc\/pgx[^"\n]*|github\.com\/yyl1212\/math_master\/backend\/internal\/(?:store|config|auth))"/,path);
  assert.doesNotMatch(source,/os\.(?:Getenv|LookupEnv)|\.OpenDatabase\(|\.ActivateRelease\(|\.PrepareRelease\(/,path);
 }
 const runtime=[read('backend/cmd/server/main.go').toString(),...goFiles('backend/internal/httpapi').map(p=>read(p).toString())].join('\n');
 assert.doesNotMatch(runtime,/contentreview|content-review|RunContentReview|DecodeOperationalJSON/);
 assert.doesNotMatch(read('api/openapi.yaml').toString(),/\/content-review|\/verify-evidence/);
});
test('eight MiB operational decoder is used only by new offline paths and original strict boundary remains',()=>{
 const allowed=new Set(['backend/internal/question/archive.go','backend/internal/cli/content_review.go','backend/internal/contentaudit/draft_selection.go']);
 for(const p of goFiles('backend'))if(read(p).toString().includes('DecodeOperationalJSON'))assert(allowed.has(p)||p.startsWith('backend/internal/contentreview/'),p);
 assert.match(read('backend/internal/question/archive.go').toString(),/func DecodeStrictJSON\([\s\S]*?limit > MaxEnvelopeBytes[\s\S]*?return decodeStrictJSON\(r, limit, out\)/);
 assert.match(read('backend/internal/question/decode.go').toString(),/MaxEnvelopeBytes\s*=\s*4\s*\*\s*1024\s*\*\s*1024/);
 assert.match(read('backend/internal/contentaudit/draft.go').toString(),/func LoadDraft\(/);
});
test('new technical tests stay in the old integration entry without skips or formal approval claims',()=>{
 const store=read('backend/internal/store/content_review_import_test.go').toString(),cli=read('backend/internal/cli/content_review_test.go').toString();
 assert.match(store,/func TestContentReviewWorkflowRoundTrip\(/);assert.match(store,/func TestContentReviewExistingRouteConflict\(/);
 assert.doesNotMatch(store+'\n'+cli,/t\.Skip(?:f|Now)?\(/);
 assert.match(store,/FixtureOnly: true/);assert.match(store,/awaiting_review/);
 assert.match(read('docs/operations/evidence/p6b/preparation.json').toString(),/"acceptanceEvidenceCreated": false/);
});

test('approved administrator-review files and new migration still reject tampering',()=>{
 const exception=JSON.parse(read(exceptionPath));
 for(const path of [...Object.keys(exception.files),...Object.keys(exception.newFiles)]) {
  assert.throws(()=>compare(p=>p===path?Buffer.concat([read(p),Buffer.from(' changed')]):read(p)),new RegExp(path.replaceAll('.','\\.')));
 }
});
test('compatibility manifest cannot excuse unrelated contracts or rewrite original evidence',()=>{
 const exception=JSON.parse(read(exceptionPath));
 for(const mutate of [
  x=>{x.files['api/openapi.yaml']={previousSha256:baseline.files['api/openapi.yaml'],sha256:baseline.files['api/openapi.yaml']}},
  x=>{x.files['backend/internal/question/model.go'].previousSha256='0'.repeat(64)},
  x=>{x.newFiles['db/migrations/00008_correction_notifications.sql']='0'.repeat(64)},
 ]) {
  const changed=structuredClone(exception);mutate(changed);
  assert.throws(()=>compare(p=>p===exceptionPath?Buffer.from(JSON.stringify(changed)):read(p)));
 }
});
