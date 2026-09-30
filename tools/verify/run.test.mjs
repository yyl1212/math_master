import test from 'node:test';
import assert from 'node:assert/strict';
import * as requireFS from 'node:fs';
import {join as tmpJoin} from 'node:path';
import {tmpdir as tmpRoot} from 'node:os';
import { spawnSync } from 'node:child_process';
import { fileURLToPath } from 'node:url';
const runner = fileURLToPath(new URL('./run.mjs', import.meta.url));
const run = (...args) => spawnSync(process.execPath,[runner,...args],{encoding:'utf8',timeout:10000});
test('keeps success and failure exit codes', () => {
  assert.equal(run('--',process.execPath,'-e','process.exit(0)').status,0);
  assert.equal(run('--',process.execPath,'-e','process.exit(7)').status,7);
});
test('terminates commands on timeout', () => {
  assert.equal(run('--timeout-ms','50','--',process.execPath,'-e','setInterval(()=>{},1000)').status,124);
});
test('rejects timeouts above the test ceiling', () => {
  assert.equal(run('--timeout-ms','540001','--',process.execPath,'-e','process.exit(0)').status,2);
});

test('kills remaining process group after its leader exits on timeout', {skip:process.platform==='win32'}, t=>{
  const {mkdtempSync,readFileSync,rmSync,existsSync}=requireFS;
  const dir=mkdtempSync(tmpJoin(tmpRoot(),'math-timeout-test-'));const pidFile=tmpJoin(dir,'pid');
  let pid;t.after(()=>{if(pid)try{process.kill(pid,'SIGKILL')}catch{};rmSync(dir,{recursive:true,force:true});});
  const descendant=`require('node:fs').writeFileSync(${JSON.stringify(pidFile)},String(process.pid));process.on('SIGTERM',()=>{});setInterval(()=>{},1000);`;
  const leader=`require('node:child_process').spawn(process.execPath,['-e',${JSON.stringify(descendant)}],{stdio:'ignore'});setInterval(()=>{},1000);`;
  const result=run('--timeout-ms','300','--',process.execPath,'-e',leader);assert.equal(result.status,124);assert.equal(existsSync(pidFile),true);pid=Number(readFileSync(pidFile));
  let running=true;try{process.kill(pid,0)}catch(e){if(e.code==='ESRCH')running=false;else throw e}
  // Linux can briefly keep the killed orphan as a zombie; it cannot execute work.
  if(running&&process.platform==='linux'){try {const state=readFileSync(`/proc/${pid}/stat`,'utf8').split(') ')[1]?.[0];running=state!=='Z';} catch(e) {if(e.code==='ENOENT')running=false;else throw e;}}
  assert.equal(running,false,'descendant survived timeout cleanup');
});
