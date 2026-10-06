import {inverseTopicWorkflow} from "./topic-learning-compatibility.mjs";
import test from 'node:test';
import assert from 'node:assert/strict';
import {readFileSync, readdirSync} from 'node:fs';
const root=new URL('../../',import.meta.url);
const read=p=>{const value=readFileSync(new URL(p,root),'utf8');return p.startsWith('.github/workflows/')?inverseTopicWorkflow(value,p):value};
const backend=read('.github/workflows/backend.yml');
const frontend=read('.github/workflows/frontend.yml');
const playwright=read('tests/e2e/playwright.config.ts');
const oldBrowserBatches=[
 ['catalogue.spec.ts','reading.spec.ts'],['auth.spec.ts','auth-security.spec.ts'],
 ['content-authoring.spec.ts'],['content-review.spec.ts'],['content-release.spec.ts'],['content-security.spec.ts'],
 ['question-authoring.spec.ts'],['question-review.spec.ts'],['question-release.spec.ts'],['question-security.spec.ts'],
 ['learning-progress.spec.ts'],['learning-practice.spec.ts','learning-assessment.spec.ts'],
 ['learning-diagnostic.spec.ts','learning-security.spec.ts'],['learning-review-regressions.spec.ts'],
];
const newFeedbackBatches=[['feedback-user.spec.ts'],['feedback-review.spec.ts'],['feedback-security.spec.ts']];
// Read active YAML run scalars; names, comments and unrelated strings cannot count as commands.
function runs(source){
 const lines=source.split('\n'),result=[];
 for(let n=0;n<lines.length;n++){
  const m=lines[n].match(/^(\s*)run:\s*(.*)$/);if(!m)continue;
  if(m[2]!=='|'){result.push(m[2]);continue;}
  const body=[];
  while(n+1<lines.length&&(!lines[n+1].trim()||lines[n+1].match(/^\s*/)[0].length>m[1].length))body.push(lines[++n].slice(m[1].length+2));
  result.push(body.join('\n'));
 }
 return result;
}
function shellCommands(source){
 const commands=[];let args=[],word='',quote=null,escaped=false,started=false;
 const push=()=>{if(started){args.push(word);word='';started=false;}};
 for(const c of source+'\n'){
  if(escaped){word+=c;started=true;escaped=false;continue;}
  if(c==='\\'&&quote!=="'"){escaped=true;continue;}
  if(quote){if(c===quote)quote=null;else word+=c;started=true;continue;}
  if(c==='"'||c==="'"){quote=c;started=true;continue;}
  if(c==='\n'){push();if(args.length)commands.push(args);args=[];continue;}
  if(/\s/.test(c)){push();continue;}
  word+=c;started=true;
 }
 assert.equal(quote,null,'unclosed shell quote');assert.equal(escaped,false,'unfinished shell escape');return commands;
}
const commands=source=>runs(source).flatMap(shellCommands);
const option=(cmd,flag)=>{
 const n=cmd.findIndex(v=>v===flag||v.startsWith(flag+'='));
 if(n<0)return undefined;
 return cmd[n]===flag?cmd[n+1]:cmd[n].slice(flag.length+1);
};
function wrapped(cmd){
 assert.deepEqual(cmd.slice(0,2),['node','tools/verify/run.mjs'],'verification wrapper');
 const split=cmd.indexOf('--');assert(split>1,'wrapper separator');
 if(cmd.includes('--timeout-ms'))assert(Number(option(cmd,'--timeout-ms'))<=540000,'wrapper time limit');
 return cmd.slice(split+1);
}
function validate(b,f,p){
 const bc=commands(b),fc=commands(f);
 for(const [source,want] of [[b,[30,30]],[f,[30]]])assert.deepEqual([...source.matchAll(/^\s+timeout-minutes:\s*(\d+)/gm)].map(m=>Number(m[1])),want,'job timeout');
 const browser=fc.filter(c=>c.includes('e2e')).map(c=>{
  const child=wrapped(c);const npm=child.indexOf('npm');
  assert.deepEqual(child.slice(npm,npm+4),['npm','run','e2e','--'],'browser command');
  const files=child.slice(npm+4);assert(files.every(p=>/^[a-z-]+\.spec\.ts$/.test(p)),'browser skip/retry/selection override');return files;
 });
 assert.equal(oldBrowserBatches.length,14);assert.equal(newFeedbackBatches.length,3);
 assert.deepEqual(browser.slice(0,17),[...oldBrowserBatches,...newFeedbackBatches],'browser batches');
 for(const [key,want]of [['workers',1],['retries',0],['globalTimeout',480000]])assert.equal(Number(p.match(new RegExp('\\b'+key+'\\s*:\\s*(\\d+)'))?.[1]),want,'Playwright '+key);
 assert.deepEqual([...p.matchAll(/name:\s*"(desktop|mobile)"/g)].map(m=>m[1]),['desktop','mobile'],'both viewports');
 const tests=[...bc,...fc].filter(c=>c.includes('go')&&c[c.indexOf('go')+1]==='test');
 for(const c of tests){wrapped(c);assert(c.includes('CGO_ENABLED=0'));assert(c.includes('GOTOOLCHAIN=go1.27.1'));assert.equal(option(c,'-timeout'),'5m','Go test timeout');assert.equal(option(c,'-count'),'1','uncached tests');}
 const store=bc.filter(c=>c.includes('./internal/store'));
 const capacities=store.filter(c=>/^\^Test(?:LearningCapacity(?:SourceVolume|MaxPool)|FeedbackCapacity)\$$/.test(option(c,'-run')));
 assert.equal(capacities.length,3,'three separate capacity batches');
 assert.deepEqual(capacities.map(c=>option(c,'-run')).sort(),['^TestLearningCapacitySourceVolume$','^TestLearningCapacityMaxPool$','^TestFeedbackCapacity$'].sort());
 for(const c of capacities)assert.equal(c.includes('-skip'),false,'capacity skip');
 const old=store.find(c=>c.includes('./internal/cli'));assert(old,'old store batch');
 assert.equal(option(old,'-skip'),'^Test(Learning|Assessment|Feedback|Correction|Notification|ContentAudit)','old store skip');assert.equal(old.includes('-run'),false,'old store restricted selection');
 const learning=store.find(c=>option(c,'-run')==='^Test(Learning|Assessment)');assert(learning,'learning batch');assert.equal(option(learning,'-skip'),'^TestLearningCapacity','learning skip');
 const feedback=store.find(c=>option(c,'-run')==='^TestFeedback');assert(feedback,'feedback noncapacity batch');assert.equal(option(feedback,'-skip'),'^TestFeedbackCapacity','feedback skip');
 const foundation=bc.find(c=>c.includes('./internal/httpapi'));assert(foundation?.includes('./internal/feedback'),'pure feedback');
 for(const p of ['assessment','learning','question','auth','content','publication','config','httpapi','e2etest','testutil'])assert(foundation.includes('./internal/'+p),'old pure package '+p);
 const node=bc.find(c=>c.includes('--test'));assert(node,'old node batch');wrapped(node);
 for(const p of ['tools/verify/run.test.mjs','tools/verify/learning-compatibility.test.mjs','tools/verify/feedback-compatibility.test.mjs','tools/verify/feedback-ci.test.mjs','tools/content-ingest/snapshot.test.mjs'])assert(node.includes(p),'node coverage '+p);
 return {oldBrowserBatches:browser.slice(0,14),newFeedbackBatches:browser.slice(14,17),capacityCommands:capacities};
}
test('actual CI preserves old batches and executes every new feedback batch within original limits',()=>validate(backend,frontend,playwright));
test('deleting an original browser batch is detected',()=>{
 const modified=frontend.replace(/^.*run:.*npm run e2e -- catalogue\.spec\.ts reading\.spec\.ts\n/m,'');
 assert.notEqual(modified,frontend);assert.throws(()=>validate(backend,modified,playwright),/browser batches/);
});
test('adding browser skip selection or retries is detected',()=>{
 assert.throws(()=>validate(backend,frontend.replace('e2e -- feedback-user.spec.ts','e2e -- feedback-user.spec.ts --grep-invert private'),playwright),/browser skip/);
 assert.throws(()=>validate(backend,frontend,playwright.replace('retries: 0','retries: 1')),/Playwright retries/);
});
test('deleting a capacity batch or skipping extra old store tests is detected',()=>{
 const modified=backend.replace(/^.*run:.*-run '\^TestLearningCapacityMaxPool\$'.*\n/m,'');assert.notEqual(modified,backend);
 assert.throws(()=>validate(modified,frontend,playwright),/capacity batches/);
 assert.throws(()=>validate(backend.replace('^Test(Learning|Assessment|Feedback|Correction|Notification|ContentAudit)','^Test(Learning|Assessment|Feedback|Correction|Notification|ContentAudit|Question)'),frontend,playwright),/old store skip/);
});
test('no browser suite or case is silently skipped, focused or retried',()=>{
 for(const name of readdirSync(new URL('tests/e2e/',root)).filter(p=>p.endsWith('.spec.ts'))){
  const source=read('tests/e2e/'+name);
  assert.doesNotMatch(source,/\btest(?:\.describe)?\.(?:skip|fixme|only)\s*\(/,name);
  for(const m of source.matchAll(/test\.setTimeout\((\d+)\)/g))assert(Number(m[1])<=480000,name+' single-case deadline');
 }
 const runner=read('tools/verify/run.mjs');assert.match(runner,/let timeout = 540000/);assert.match(runner,/timeout > 540000/);
});

test('active command parser handles split and equals options without counting step labels',()=>{
 assert.equal(option(['go','test','-count=1'],'-count'),'1');
 assert.equal(option(['go','test','-count','1'],'-count'),'1');
 assert.equal(option(['go','test'],'-count'),undefined);
 assert.deepEqual(commands("      - name: npm run e2e -- ignored.spec.ts\n        run: node tools/verify/run.mjs --cwd backend -- go test -run '^Test(Learning|Assessment)' -count=1\n"),[['node','tools/verify/run.mjs','--cwd','backend','--','go','test','-run','^Test(Learning|Assessment)','-count=1']]);
});
