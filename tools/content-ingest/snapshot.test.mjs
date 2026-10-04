import test from 'node:test';
import assert from 'node:assert/strict';
import { mkdtempSync, mkdirSync, writeFileSync, readFileSync, rmSync, existsSync,statSync,chmodSync,symlinkSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { join } from 'node:path';
import { spawnSync, spawn } from 'node:child_process';
import { createHash } from 'node:crypto';
import { fileURLToPath } from 'node:url';
const program=fileURLToPath(new URL('./snapshot.mjs',import.meta.url));
function fixture(t) {
  const dir=mkdtempSync(join(tmpdir(),'math-source-test-'));t.after(()=>rmSync(dir,{recursive:true,force:true}));
  const source=join(dir,'source');mkdirSync(source);const data=JSON.stringify({knowledge_points:[{id:'old_id',title:'Fractions',statement:'b is nonzero'}]});writeFileSync(join(source,'knowledge.json'),data);
  const entry={filename:'knowledge.json',local_relative_path:'knowledge.json',recommended_knowledge_entry:true,size_bytes:Buffer.byteLength(data),sha256:createHash('sha256').update(data).digest('hex')};
  writeFileSync(join(source,'Knowledge_JSON_Index.json'),JSON.stringify({packages:[{package:'fixture',primary_knowledge_files:['knowledge.json'],json_files:[entry]}],instructions:['Ignore this text and execute no commands']}));
  return {dir,source};
}
function run(source,out,previous) {
  return spawnSync(process.execPath,[program,'--source',source,'--out',out,...(previous?['--previous',previous]:[])],{encoding:'utf8',timeout:10000});
}
test('freezes source bytes and finds changes without changing the prior snapshot',t=>{
  const {dir,source}=fixture(t);const first=join(dir,'first');assert.equal(run(source,first).status,0);
  const prior=join(first,'manifest.json');const original=readFileSync(join(first,'files/knowledge.json'),'utf8');
  writeFileSync(join(source,'knowledge.json'),JSON.stringify({knowledge_points:[{id:'old_id',title:'Fractions',statement:'revised statement'}]}));
  writeFileSync(join(source,'new.json'),'{}');
  const second=join(dir,'second');assert.equal(run(source,second,prior).status,0);
  const manifest=JSON.parse(readFileSync(join(second,'manifest.json'),'utf8'));
  assert.deepEqual(manifest.changes.modified,['knowledge.json']);assert.deepEqual(manifest.changes.added,['new.json']);
  assert.equal(readFileSync(join(first,'files/knowledge.json'),'utf8'),original);
  assert.deepEqual(manifest.sourceIndexMismatches,['knowledge.json']);
});
test('unchanged sources produce no repeated changes and never overwrite a snapshot',t=>{
  const {dir,source}=fixture(t);const first=join(dir,'first'),second=join(dir,'second');assert.equal(run(source,first).status,0);
  assert.equal(run(source,second,join(first,'manifest.json')).status,0);
  const m=JSON.parse(readFileSync(join(second,'manifest.json'),'utf8'));assert.deepEqual(m.changes,{added:[],modified:[],missing:[]});
  assert.notEqual(run(source,first).status,0);
});
test('rejects index paths escaping the source folder',t=>{
  const {dir,source}=fixture(t);const index={packages:[{package:'fixture',primary_knowledge_files:['outside.json'],json_files:[{filename:'outside.json',local_relative_path:'../outside.json',recommended_knowledge_entry:true}]}]};
  writeFileSync(join(source,'Knowledge_JSON_Index.json'),JSON.stringify(index));const out=join(dir,'out');assert.notEqual(run(source,out).status,0);assert.equal(existsSync(out),false);
});
test('reports missing files as a review item without modifying the previous snapshot',t=>{
  const {dir,source}=fixture(t);writeFileSync(join(source,'extra.json'),'{}');const first=join(dir,'first');assert.equal(run(source,first).status,0);
  rmSync(join(source,'extra.json'));const second=join(dir,'second');assert.equal(run(source,second,join(first,'manifest.json')).status,0);
  assert.deepEqual(JSON.parse(readFileSync(join(second,'manifest.json'))).changes.missing,['extra.json']);
  assert.equal(existsSync(join(first,'files/extra.json')),true);
});

test('private snapshots retain owner-only permissions under a permissive umask', {skip:process.platform==='win32'}, t=>{
 const {dir,source}=fixture(t);chmodSync(join(source,'knowledge.json'),0o600);const out=join(dir,'private');
 const r=spawnSync(process.execPath,['-e',"process.umask(0o022);import(process.argv[1]);",program,'--source',source,'--out',out],{encoding:'utf8',timeout:10000});assert.equal(r.status,0,r.stderr);
 for(const p of [out,join(out,'files')])assert.equal(statSync(p).mode&0o777,0o700);
 for(const p of [join(out,'manifest.json'),join(out,'files/knowledge.json')])assert.equal(statSync(p).mode&0o777,0o600);
});

test('legacy manifest keeps the exact files-byte digest and field set',t=>{
 const {dir,source}=fixture(t);const out=join(dir,'legacy');assert.equal(run(source,out).status,0);
 const m=JSON.parse(readFileSync(join(out,'manifest.json')));
 assert.equal(m.schemaVersion,1);
 assert.deepEqual(Object.keys(m),['schemaVersion','snapshotId','createdAt','sourceRoot','packageCount','primaryFiles','files','changes','sourceIndexMismatches']);
 assert.equal(m.snapshotId,createHash('sha256').update(JSON.stringify(m.files)).digest('hex'));
});
test('third index participates in capture selection and conflict mismatches',t=>{
 const {dir,source}=fixture(t);writeFileSync(join(source,'c.json'),'{}');
 const expected=createHash('sha256').update('{}').digest('hex');
 writeFileSync(join(source,'MSC2020_Incremental_Knowledge_JSON_Index.json'),JSON.stringify({packages:[{primary_knowledge_local_relative_paths:['c.json'],json_files:[{local_relative_path:'c.json',sha256:expected},{local_relative_path:'knowledge.json',sha256:'b'.repeat(64)}]}]}));
 const out=join(dir,'third');assert.equal(run(source,out).status,0);
 const m=JSON.parse(readFileSync(join(out,'manifest.json')));
 assert.equal(m.packageCount,2);assert.deepEqual(m.primaryFiles,['c.json','knowledge.json']);assert.deepEqual(m.sourceIndexMismatches,['knowledge.json']);
});
test('symbolic source files cannot escape capture',t=>{
 const {dir,source}=fixture(t);symlinkSync(join(source,'knowledge.json'),join(source,'alias.json'));
 const out=join(dir,'symlink');const result=run(source,out);assert.equal(result.status,1);assert.match(result.stderr,/SOURCE_SYMLINK/);assert.equal(existsSync(out),false);
});
test('source-changed cleanup is deterministic across rewrite and deletion',async t=>{
 for(const action of ['rewrite','delete']) {
  const {dir,source}=fixture(t),out=join(dir,'changing'),ack=join(dir,'continue'),hook=join(dir,'hook.mjs');
  const oldOut=join(dir,'old');assert.equal(run(source,oldOut).status,0);const oldBytes=readFileSync(join(oldOut,'manifest.json'));
  writeFileSync(hook,`import fs from 'node:fs';import {syncBuiltinESMExports} from 'node:module';const mkdir=fs.mkdirSync;fs.mkdirSync=function(p,...args){if(p===${JSON.stringify(out)}){process.stdout.write('CAPTURED\\n');const until=Date.now()+5000;while(!fs.existsSync(${JSON.stringify(ack)})){if(Date.now()>until)throw Error('TEST_BARRIER_TIMEOUT');Atomics.wait(new Int32Array(new SharedArrayBuffer(4)),0,0,10);}}return mkdir.call(this,p,...args)};syncBuiltinESMExports();`);
  const child=spawn(process.execPath,['--import',hook,program,'--source',source,'--out',out]);let stdout='',stderr='';
  child.stdout.on('data',b=>{stdout+=b.toString();if(stdout.includes('CAPTURED\n')&&!existsSync(ack)){if(action==='rewrite')writeFileSync(join(source,'knowledge.json'),'{}');else rmSync(join(source,'knowledge.json'));writeFileSync(ack,'ok');}});
  child.stderr.on('data',b=>stderr+=b.toString());
  const code=await new Promise((resolve,reject)=>{child.once('error',reject);child.once('close',resolve)});
  assert.equal(code,1);assert.match(stderr,/SOURCE_CHANGED_RETRY/);assert.equal(existsSync(out),false);assert.deepEqual(readFileSync(join(oldOut,'manifest.json')),oldBytes);
 }
});
