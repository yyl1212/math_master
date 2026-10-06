import {inverseTopicWorkflow} from "./topic-learning-compatibility.mjs";
import {removeApprovedLanguageSteps} from "./ui-language-ci.mjs";
import test from 'node:test';
import assert from 'node:assert/strict';
import {readFileSync} from 'node:fs';
import {createHash} from 'node:crypto';
const root=new URL('../../',import.meta.url),read=p=>{const value=readFileSync(new URL(p,root),'utf8');return p.startsWith('.github/workflows/')?inverseTopicWorkflow(value,p):value};
const baseline=JSON.parse(read('docs/operations/evidence/p6b/compatibility-baseline.json'));
const backend=read('.github/workflows/backend.yml'),frontend=read('.github/workflows/frontend.yml');
const addition=' tools/verify/content-review.test.mjs tools/verify/content-review-ci.test.mjs';
const batch="      - name: 全量复核准备与证据纯层\n        run: node tools/verify/run.mjs --cwd backend -- env CGO_ENABLED=0 GOTOOLCHAIN=go1.27.1 go test ./internal/contentreview -timeout 5m -count=1\n";
const candidateBase=backend.replace(batch,'').replace(addition,'');
const candidate=candidateBase.replace('tools/verify/content-acceptance-ci.test.mjs\n','tools/verify/content-acceptance-ci.test.mjs'+addition+'\n').replace('      - name: 构建\n',batch+'      - name: 构建\n');
const sha=s=>createHash('sha256').update(s).digest('hex');
function validate(b,f){
 assert.equal(b.split(batch).length,2,'exact mandatory pure-layer batch');
 assert.equal(b.split(addition).length,2,'both new Node checks appended exactly once');
 const node=b.split('\n').find(l=>l.includes('run: node tools/verify/run.mjs -- node --test'));
 assert(node?.endsWith(addition),'new Node checks at old command tail');
 assert.equal(sha(b.replace(batch,'').replace(addition,'')),baseline.backendWorkflowSHA256,'all old backend batches, names, arguments and budgets unchanged');
 assert.equal(sha(removeApprovedLanguageSteps(f)),baseline.frontendWorkflowSHA256,'all 22 original browser batches and budgets unchanged');
 for(const name of ['FeedbackCapacity','LearningCapacitySourceVolume','LearningCapacityMaxPool','CorrectionCapacityImpact','CorrectionCapacityNotifications','ContentAuditCapacity'])assert.equal(b.split("-run '^Test"+name+"$'").length,2,name+' not omitted or duplicated');
 const wide=b.split('\n').find(l=>l.includes('go test ./internal/store ./internal/cli -skip'));
 assert(wide?.includes('-skip "^Test(Learning|Assessment|Feedback|Correction|Notification|ContentAudit)"'),'unchanged broad integration selection includes TestContentReview');
 assert.match(read('tools/verify/run.mjs'),/timeout = 540000/);
 const p=read('tests/e2e/playwright.config.ts');
 for(const[key,value]of [['workers',1],['retries',0],['globalTimeout',480000]])assert.match(p,new RegExp('\\b'+key+'\\s*:\\s*'+value+'\\b'));
}
test('new mandatory pure-layer and Node batches are appended to all old verification gates',()=>validate(backend,frontend));
test('removing either new check or new pure-layer batch is detected',()=>{
 validate(candidate,frontend);
 for(const needle of [' tools/verify/content-review.test.mjs',' tools/verify/content-review-ci.test.mjs',batch])assert.throws(()=>validate(candidate.replace(needle,''),frontend));
});
test('old backend step deletion, renaming, omission and budget widening are rejected',()=>{
 validate(candidate,frontend);
 for(const change of [
  b=>b.replace('内容与 HTTP 校验','renamed'),
  b=>b.split('\n').filter(l=>!l.includes('go test ./cmd/server ')).join('\n'),
  b=>b.replace('CGO_ENABLED=0 ','').replace('CGO_ENABLED=0 ',''),
  b=>b.replace('-timeout 5m','-timeout 6m'),
  b=>b.replace('timeout-minutes: 30','timeout-minutes: 31'),
  b=>b.replace('CGO_ENABLED=0 GOTOOLCHAIN=go1.27.1 go test ./internal/contentreview','GOTOOLCHAIN=go1.27.1 go test ./internal/contentreview'),
  b=>b.replace(batch,'      - name: 全量复核准备与证据纯层\n        if: false\n'+batch.slice(batch.indexOf('        run:'))),
  b=>b.replace('Notification|ContentAudit)','Notification|ContentAudit|ContentReview)'),
 ])assert.throws(()=>validate(change(candidate),frontend));
});
test('each old capacity gate and original browser batch is still mandatory',()=>{
 validate(candidate,frontend);
 for(const name of ['FeedbackCapacity','LearningCapacitySourceVolume','LearningCapacityMaxPool','CorrectionCapacityImpact','CorrectionCapacityNotifications','ContentAuditCapacity'])assert.throws(()=>validate(candidate.split('\n').filter(l=>!l.includes("-run '^Test"+name+"$'")).join('\n'),frontend));
 for(const row of JSON.parse(read('docs/operations/evidence/p6a/matrix-browser.json'))){
  const command=row.command;assert(frontend.includes('run: '+command));
  assert.throws(()=>validate(candidate,frontend.split('\n').filter(l=>!l.includes('run: '+command)).join('\n')));
 }
 assert.throws(()=>validate(candidate,frontend.replace('真实公共目录与阅读浏览器回归','renamed')));
});
