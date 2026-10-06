import {inverseTopicWorkflow} from "./topic-learning-compatibility.mjs";
import {languageBatches} from "./ui-language-ci.mjs";
import test from 'node:test';
import assert from 'node:assert/strict';
import {readFileSync,readdirSync} from 'node:fs';
const root=new URL('../../',import.meta.url),read=p=>{const value=readFileSync(new URL(p,root),'utf8');return p.startsWith('.github/workflows/')?inverseTopicWorkflow(value,p):value};
const backend=read('.github/workflows/backend.yml'),frontend=read('.github/workflows/frontend.yml'),playwright=read('tests/e2e/playwright.config.ts');
const batches=[['catalogue','reading'],['auth','auth-security'],['content-authoring'],['content-review'],['content-release'],['content-security'],['question-authoring'],['question-review'],['question-release'],['question-security'],['learning-progress'],['learning-practice','learning-assessment'],['learning-diagnostic','learning-security'],['learning-review-regressions'],['feedback-user'],['feedback-review'],['feedback-security'],['correction-user'],['correction-review'],['notification-security']].map(v=>v.map(f=>f+'.spec.ts'));
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
 const browser=fc.filter(c=>c.includes('e2e')).map(c=>{const child=wrapped(c);assert.deepEqual(child.slice(0,7),['env','-u','NO_COLOR','npm','run','e2e','--']);return child.slice(7)});
 assert.deepEqual(browser.slice(0,20),batches,'old 20 browser batches');
 assert.deepEqual(browser.slice(20),[['content-acceptance-reading.spec.ts'],['content-acceptance-learning.spec.ts'],...languageBatches],'new browser batches');
 const tests=bc.filter(c=>c.includes('go')&&c[c.indexOf('go')+1]==='test');
 for(const c of tests){const child=wrapped(c);assert.deepEqual(child.slice(0,5),['env','CGO_ENABLED=0','GOTOOLCHAIN=go1.27.1','go','test']);assert.equal(option(c,'-timeout'),'5m');assert.equal(option(c,'-count'),'1');assert(!c.includes('--timeout-ms'))}
 for(const name of ['^TestFeedbackCapacity$','^TestLearningCapacitySourceVolume$','^TestLearningCapacityMaxPool$','^TestCorrectionCapacityImpact$','^TestCorrectionCapacityNotifications$'])assert.equal(tests.filter(c=>option(c,'-run')===name).length,1,'old capacity '+name);
 assert.equal(tests.filter(c=>c.includes('./internal/contentaudit')).length,1,'pure contentaudit');
 const regular=tests.find(c=>option(c,'-run')==='^TestContentAudit');assert(regular,'audit regular');assert(regular.includes('./internal/store')&&regular.includes('./internal/cli'));assert.equal(option(regular,'-skip'),'^TestContentAuditCapacity$');
 const capacity=tests.filter(c=>option(c,'-run')==='^TestContentAuditCapacity$');assert.equal(capacity.length,1,'audit capacity');assert(!capacity[0].includes('-skip'));
 const wide=tests.find(c=>c.includes('./internal/cli')&&!c.includes('-run'));assert.equal(option(wide,'-skip'),'^Test(Learning|Assessment|Feedback|Correction|Notification|ContentAudit)','no duplicate audit capacity');
 const node=bc.find(c=>c.includes('--test'));for(const file of ['tools/content-ingest/source-index.test.mjs','tools/content-ingest/source-report.test.mjs','tools/verify/content-acceptance.test.mjs','tools/verify/content-acceptance-ci.test.mjs'])assert(node.includes(file),'new node '+file);
 assert.deepEqual([...b.matchAll(/^\s+timeout-minutes:\s*(\d+)/gm)].map(m=>+m[1]),[30,30]);assert.deepEqual([...f.matchAll(/^\s+timeout-minutes:\s*(\d+)/gm)].map(m=>+m[1]),[30]);
 for(const[key,value]of [['workers',1],['retries',0],['globalTimeout',480000]])assert.equal(Number(p.match(new RegExp('\\b'+key+'\\s*:\\s*(\\d+)'))?.[1]),value);
 assert.match(read('tools/verify/run.mjs'),/timeout = 540000/);
 for(const s of [b,f])for(const step of s.split(/^      - /m))if(commands('      - '+step).some(c=>c.includes('go')||c.includes('e2e')||c.includes('--test')))assert.doesNotMatch(step,/^\s+if:/m,'conditional verification');
}
test('new audit batches and all old budgets are wired',()=>validate(backend,frontend,playwright));
test('missing new browser or audit capacity is detected',()=>{for(const needle of ['content-acceptance-reading.spec.ts','content-acceptance-learning.spec.ts'])assert.throws(()=>validate(backend,frontend.split('\n').filter(l=>!l.includes('run:')||!l.includes(needle)).join('\n'),playwright));assert.throws(()=>validate(backend.split('\n').filter(l=>!l.includes("-run '^TestContentAuditCapacity$'")).join('\n'),frontend,playwright))});
test('old browser deletion and budget changes are detected',()=>{assert.throws(()=>validate(backend,frontend.replace('catalogue.spec.ts reading.spec.ts','reading.spec.ts'),playwright),/old 20/);assert.throws(()=>validate(backend.replace('-timeout 5m','-timeout 6m'),frontend,playwright));assert.throws(()=>validate(backend,frontend,playwright.replace('retries: 0','retries: 1')))});

// Test the scene transport independently of browser dependencies so the backend
// workflow keeps its existing Node-only verification entry point.
async function sceneControl() { return import('../../tests/e2e/scene-control.ts'); }
const sceneRuntime={controlURL:'http://127.0.0.1:1',token:'test-only-scene-token'};
test('complete content scene accepts one setup slower than the old five-second limit', {timeout:15000}, async t=>{
 const {createServer}=await import('node:http');
 const {changeScene}=await sceneControl();
 let requests=0;
 const server=createServer((req,res)=>{
  requests++;
  assert.equal(req.url,'/scene/content-acceptance');
  assert.equal(req.headers.authorization,'Bearer '+sceneRuntime.token);
  setTimeout(()=>{res.writeHead(204);res.end();},5500);
 });
 await new Promise(resolve=>server.listen(0,'127.0.0.1',resolve));
 t.after(()=>new Promise(resolve=>{server.closeAllConnections();server.close(resolve);}));
 const runtime={...sceneRuntime,controlURL:'http://127.0.0.1:'+server.address().port};
 const request={post:async(url,options)=>{
  const response=await fetch(url,{method:'POST',headers:options.headers,signal:AbortSignal.timeout(options.timeout)});
  return {status:()=>response.status};
 }};
 await changeScene(request,runtime,'content-acceptance');
 assert.equal(requests,1,'no automatic retry of fixture writes');
});
test('all legacy and unknown scenes retain the five-second request limit',async()=>{
 const {changeScene}=await sceneControl();
 for(const name of ['draft','question','learning','feedback','correction','content-acceptance-unknown']){
  let calls=0;
  await changeScene({post:async(url,options)=>{
   calls++;assert.equal(url,sceneRuntime.controlURL+'/scene/'+name);assert.equal(options.timeout,5000);
   return {status:()=>204};
  }},sceneRuntime,name);
  assert.equal(calls,1);
 }
});
test('content setup has a ten-second limit within the unchanged individual test budget',async()=>{
 const {changeScene}=await sceneControl();
 await changeScene({post:async(_url,options)=>{assert.equal(options.timeout,10000);return {status:()=>204};}},sceneRuntime,'content-acceptance');
 assert.match(playwright,/\btimeout:\s*30000\b/);
});
test('scene failures are single-attempt and redact both transport and timeout errors',async()=>{
 const {changeScene}=await sceneControl();
 for(const failure of [401,400,503,'network request leaked test-only-scene-token http://127.0.0.1:1','apiRequestContext.post: Timeout 10000ms exceeded; test-only-scene-token']){
  let calls=0;
  await assert.rejects(changeScene({post:async()=>{
   calls++;
   if(typeof failure==='number')return {status:()=>failure};
   throw new Error(failure);
  }},sceneRuntime,'content-acceptance'),error=>{
   assert.equal(error.message,typeof failure==='string'&&failure.includes('Timeout')?'Test scene change timed out':'Test scene change failed');
   assert.equal(error.cause,undefined);assert.doesNotMatch(error.stack,/test-only-scene-token|http:\/\/127/);return true;
  });
  assert.equal(calls,1);
 }
});
