import test from 'node:test';
import assert from 'node:assert/strict';
import { mkdtempSync, mkdirSync, writeFileSync, readFileSync, rmSync, existsSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { join } from 'node:path';
import { spawnSync } from 'node:child_process';
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
