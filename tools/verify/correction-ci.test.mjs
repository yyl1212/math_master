import test from 'node:test';
import assert from 'node:assert/strict';
import {readFileSync,readdirSync} from 'node:fs';
const root=new URL('../../',import.meta.url),read=p=>readFileSync(new URL(p,root),'utf8');
const backend=read('.github/workflows/backend.yml'),frontend=read('.github/workflows/frontend.yml'),playwright=read('tests/e2e/playwright.config.ts');
const batches=[['catalogue','reading'],['auth','auth-security'],['content-authoring'],['content-review'],['content-release'],['content-security'],['question-authoring'],['question-review'],['question-release'],['question-security'],['learning-progress'],['learning-practice','learning-assessment'],['learning-diagnostic','learning-security'],['learning-review-regressions'],['feedback-user'],['feedback-review'],['feedback-security'],['correction-user'],['correction-review'],['notification-security']].map(v=>v.map(f=>f+'.spec.ts'));
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

function jobBodies(source) {
 const lines=source.split('\n'),jobs=new Map();let current=null,inside=false;
 for(const line of lines){if(line==='jobs:'){inside=true;continue}if(!inside)continue;const m=line.match(/^  ([a-z_]+):\s*$/);if(m){current=m[1];jobs.set(current,[])}else if(current)jobs.get(current).push(line)}
 return new Map([...jobs].map(([k,v])=>[k,v.join('\n')]));
}
function singleOption(cmd,flag) {assert(cmd.filter(v=>v===flag||v.startsWith(flag+'=')).length<=1,'duplicate '+flag);return option(cmd,flag)}
function goTests(source) {
 return commands(source).filter(c=>c.includes('go')&&c[c.indexOf('go')+1]==='test').map(c=>{
  const child=wrapped(c);assert.deepEqual(child.slice(0,5),['env','CGO_ENABLED=0','GOTOOLCHAIN=go1.27.1','go','test'],'executable Go prefix');
  assert.equal(singleOption(child,'-timeout'),'5m','Go timeout');assert.equal(singleOption(child,'-count'),'1','Go count');singleOption(child,'-run');singleOption(child,'-skip');
  assert(!c.includes('--timeout-ms'),'wrapper deadline override');return child;
 });
}
function validate(b,f,p) {
 const jobs=jobBodies(b);assert.deepEqual([...jobs.keys()],['verify','correction_verify'],'independent correction job');
 for(const[_,body]of jobs){assert.match(body,/^    timeout-minutes: 30$/m,'job deadline');assert.match(body,/^        image: postgres:17\.11$/m,'isolated same PostgreSQL');assert.match(body,/^      CGO_ENABLED: '0'$/m);assert.match(body,/^      GOTOOLCHAIN: go1\.27\.1$/m);
  assert.match(body,/actions\/checkout@[0-9a-f]{40}/);assert.match(body,/actions\/setup-go@[0-9a-f]{40}/);assert.match(body,/actions\/setup-node@[0-9a-f]{40}/);
  // Required verification run steps are unconditional, so a nonexecuting if
  // cannot disguise a declared capacity or browser command.
  assert.doesNotMatch(body,/^\s+if:/m,'conditional required Go verification');
 }
 assert.deepEqual([...f.matchAll(/^\s+timeout-minutes:\s*(\d+)/gm)].map(m=>Number(m[1])),[30],'frontend job deadline');
 const browser=commands(f).filter(c=>c.includes('e2e')).map(c=>{const child=wrapped(c);assert.deepEqual(child.slice(0,7),['env','-u','NO_COLOR','npm','run','e2e','--'],'executable browser prefix');const files=child.slice(7);assert(files.every(v=>/^[a-z-]+\.spec\.ts$/.test(v)),'browser selection override');assert(!c.includes('--timeout-ms'),'wrapper deadline override');return files});
 assert.deepEqual(browser,batches,'all 20 original-prefix browser batches');
 for(const step of f.split(/^      - /m))if(commands('      - '+step).some(c=>c.includes('e2e')))assert.doesNotMatch(step,/^\s+if:/m,'conditional required browser verification');
 for(const[key,value]of [['workers',1],['retries',0],['globalTimeout',480000]])assert.equal(Number(p.match(new RegExp('\\b'+key+'\\s*:\\s*(\\d+)'))?.[1]),value,'Playwright '+key);
 const old=goTests(jobs.get('verify')),fresh=goTests(jobs.get('correction_verify'));goTests(f);
 const foundation=old.find(c=>c.includes('./internal/httpapi'));assert(foundation,'foundation');assert(!foundation.includes('-run')&&!foundation.includes('-skip'),'foundation selection');
 for(const pkg of ['correction','notification','feedback','assessment','learning','question','auth','content','publication','config','httpapi','e2etest','testutil'])assert(foundation.includes('./internal/'+pkg),'foundation '+pkg);assert(foundation.includes('./cmd/server'),'HTTP drain coverage');
 const oldStore=old.find(c=>c.includes('./internal/cli'));assert(oldStore,'original store');assert.equal(option(oldStore,'-skip'),'^Test(Learning|Assessment|Feedback|Correction|Notification)','old skip');assert(!oldStore.includes('-run'),'old selection');
 const expectedOld=['^TestFeedbackCapacity$','^TestLearningCapacitySourceVolume$','^TestLearningCapacityMaxPool$'];
 assert.deepEqual(old.filter(c=>expectedOld.includes(option(c,'-run'))).map(c=>option(c,'-run')),expectedOld,'old three capacities');
 for(const name of expectedOld)assert(!old.find(c=>option(c,'-run')===name).includes('-skip'),'capacity skip');
 assert.equal(fresh.length,3,'three new Go batches');
 const non=fresh.filter(c=>option(c,'-run')==='^Test(Correction|Notification)');assert.equal(non.length,1,'correction noncapacity');assert(non[0].includes('./internal/store')&&non[0].includes('./internal/cli'),'new CLI coverage');assert.equal(option(non[0],'-skip'),'^TestCorrectionCapacity','new capacity exclusion');
 for(const name of ['^TestCorrectionCapacityImpact$','^TestCorrectionCapacityNotifications$']){const rows=fresh.filter(c=>option(c,'-run')===name);assert.equal(rows.length,1,'capacity '+name);assert.deepEqual(rows[0].filter(v=>v.startsWith('./')),['./internal/store'],'capacity packages');assert(!rows[0].includes('-skip'),'new capacity skip')}
 const all=[...old,...fresh];let count=0;
 for(const pkg of ['store','cli'])for(const filename of readdirSync(new URL('backend/internal/'+pkg+'/',root)).filter(v=>v.endsWith('_test.go'))){const source=read('backend/internal/'+pkg+'/'+filename);for(const m of source.matchAll(/^func (Test(?:Correction|Notification)\w*)\(/gm)){
  const name=m[1],matching=all.filter(c=>c.includes('./internal/'+pkg)&&(!option(c,'-run')||new RegExp(option(c,'-run')).test(name))&&(!option(c,'-skip')||!new RegExp(option(c,'-skip')).test(name)));
  assert.equal(matching.length,1,'classification '+pkg+'/'+name);count++;
 }}assert(count>0,'actual new tests classified');
 const node=commands(jobs.get('verify')).find(c=>c.includes('--test'));wrapped(node);for(const path of ['tools/verify/run.test.mjs','tools/verify/learning-compatibility.test.mjs','tools/verify/feedback-compatibility.test.mjs','tools/verify/correction-compatibility.test.mjs','tools/verify/feedback-ci.test.mjs','tools/verify/correction-ci.test.mjs','tools/content-ingest/snapshot.test.mjs'])assert(node.includes(path),'Node '+path);
 const runner=read('tools/verify/run.mjs');assert.match(runner,/let timeout = 540000/);assert.match(runner,/timeout > 540000/);
 return {classified:count,browserBatches:browser.length};
}
function changed(source,from,to){const result=source.replace(from,()=>to);assert.notEqual(result,source,'mutation must apply');return result}
test('new correction CI is executable, classified once and keeps every original batch and deadline',()=>validate(backend,frontend,playwright));
test('deleting any of the 20 browser batches is rejected',()=>{for(const files of batches){const line=frontend.split('\n').find(v=>v.includes('run:')&&v.includes('e2e -- '+files.join(' ')));assert(line);assert.throws(()=>validate(backend,changed(frontend,line,''),playwright),/browser batches/)}});
test('step labels and echo commands cannot stand in for actual browser execution',()=>{const from='run: node tools/verify/run.mjs --cwd frontend -- env -u NO_COLOR npm run e2e -- correction-user.spec.ts';for(const to of ['name: npm run e2e -- correction-user.spec.ts','run: echo npm run e2e -- correction-user.spec.ts','run: node tools/verify/run.mjs --cwd frontend -- echo env -u NO_COLOR npm run e2e -- correction-user.spec.ts'])assert.throws(()=>validate(backend,changed(frontend,from,to),playwright),/browser|wrapper/)});
test('all five capacity commands must remain executable and separate',()=>{for(const name of ['FeedbackCapacity','LearningCapacitySourceVolume','LearningCapacityMaxPool','CorrectionCapacityImpact','CorrectionCapacityNotifications']){const line=backend.split('\n').find(v=>v.includes('run:')&&v.includes("-run '^Test"+name+"$'"));assert(line);assert.throws(()=>validate(changed(backend,line,''),frontend,playwright),/capacities|Go batches|capacity/);assert.throws(()=>validate(changed(backend,line,line.replace('run:','name:')),frontend,playwright),/capacities|Go batches|capacity/)}});
test('extra skips, duplicate classification and omitted notification/CLI tests are rejected',()=>{
 for(const[from,to]of [['^Test(Learning|Assessment|Feedback|Correction|Notification)','^Test(Learning|Assessment|Feedback|Correction|Notification|Question)'],['-skip \'^TestCorrectionCapacity\'','-skip \'^Test(CorrectionCapacity|Notification)\''],['-run \'^Test(Correction|Notification)\'','-run \'^TestCorrection\''],['./internal/store ./internal/cli -run \'^Test(Correction|Notification)\'','./internal/store -run \'^Test(Correction|Notification)\''],['-skip \'^TestCorrectionCapacity\'','-skip \'^TestUnusedPrefix\'']])assert.throws(()=>validate(changed(backend,from,to),frontend,playwright),/skip|classification|CLI|exclusion|noncapacity/);
});
test('Go, job, browser and wrapper time limits cannot be expanded or disguised with duplicate flags',()=>{
 assert.throws(()=>validate(changed(backend,'-timeout 5m','-timeout 6m'),frontend,playwright),/Go timeout/);
 assert.throws(()=>validate(changed(backend,'-timeout 5m','-timeout 5m -timeout 6m'),frontend,playwright),/duplicate/);
 assert.throws(()=>validate(changed(backend,'timeout-minutes: 30','timeout-minutes: 31'),frontend,playwright),/job deadline/);
 assert.throws(()=>validate(backend,frontend,changed(playwright,'globalTimeout: 480000','globalTimeout: 480001')),/Playwright globalTimeout/);
 assert.throws(()=>validate(backend,changed(frontend,'e2e -- correction-user.spec.ts','e2e -- correction-user.spec.ts --retries=1'),playwright),/selection override/);
 assert.throws(()=>validate(backend,changed(frontend,'node tools/verify/run.mjs --cwd frontend -- env -u NO_COLOR npm run e2e -- correction-user.spec.ts','node tools/verify/run.mjs --timeout-ms 540001 --cwd frontend -- env -u NO_COLOR npm run e2e -- correction-user.spec.ts'),playwright),/wrapper/);
});
test('new capacity names inside an echo are rejected',()=>{const from='-- env CGO_ENABLED=0 GOTOOLCHAIN=go1.27.1 go test ./internal/store -run \'^TestCorrectionCapacityImpact$\'';assert.throws(()=>validate(changed(backend,from,from.replace('-- env','-- echo env')),frontend,playwright),/executable Go prefix/)});
test('a second actual new test classification in the original job is rejected',()=>{const extra="      - name: Duplicate real classification\n        run: node tools/verify/run.mjs --cwd backend -- env CGO_ENABLED=0 GOTOOLCHAIN=go1.27.1 go test ./internal/store ./internal/cli -run '^Test(Correction|Notification)' -skip '^TestCorrectionCapacity' -timeout 5m -count=1\n";assert.throws(()=>validate(changed(backend,'  correction_verify:',extra+'  correction_verify:'),frontend,playwright),/classification/)});

test('required correction Go and browser steps cannot be declared but conditionally disabled',()=>{for(const from of ['      - name: 千案件万证据完整影响处理','      - name: 真实纠错通过重测与通知回归']){const target=from.includes('千案件')?backend:frontend;const modified=changed(target,from,from+'\n        if: false');assert.throws(()=>validate(target===backend?modified:backend,target===frontend?modified:frontend,playwright),/conditional required/)}});
