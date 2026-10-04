import test from 'node:test';
import assert from 'node:assert/strict';
import { mkdtempSync,mkdirSync,writeFileSync,readFileSync,rmSync,existsSync,symlinkSync } from 'node:fs';
import { join } from 'node:path';import { tmpdir } from 'node:os';import { spawnSync } from 'node:child_process';import { createHash } from 'node:crypto';
const sha=x=>createHash('sha256').update(x).digest('hex');
function fixture(t,records=[{id:'a',title:'Same',conditions:['x>0'],legacy_id:'legacy',provenance:[{source_id:'origin'}],review_status:'AI_checked'},{id:'b',title:'Same',conditions:['x>=0'],legacy_id:'legacy'}]) {
 const base=mkdtempSync(join(tmpdir(),'source-report-'));t.after(()=>rmSync(base,{recursive:true,force:true}));const source=join(base,'source'),snapshot=join(base,'snapshot');mkdirSync(source);
 const corpus=JSON.stringify({schema_version:'1',dataset_id:'fixture',knowledge_points:records,review:{human_reviewed:false}});writeFileSync(join(source,'main.json'),corpus);writeFileSync(join(source,'advanced.json'),'{}');
 const idx={packages:[{primary_knowledge_local_relative_paths:['main.json'],json_files:[{local_relative_path:'main.json',sha256:sha(corpus)},{local_relative_path:'advanced.json',sha256:'c'.repeat(64)}]}],instructions:['touch SENTINEL'],local_save_workflow:['rm -rf source']};writeFileSync(join(source,'Knowledge_JSON_Index.json'),JSON.stringify(idx));
 const r=spawnSync(process.execPath,['tools/content-ingest/snapshot.mjs','--source',source,'--out',snapshot],{encoding:'utf8'});assert.equal(r.status,0,r.stderr);return {base,source,snapshot};
}
async function build(snapshot,selected) {assert.ok(existsSync(new URL('./source-report.mjs',import.meta.url)),'source report is required');return (await import('./source-report.mjs')).buildSourceReport(snapshot,selected);}
test('source report preserves separate identifiers and source bytes despite equal titles and legacy IDs',async t=>{
 const {snapshot,source}=fixture(t);const before=readFileSync(join(source,'main.json'));const r=await build(snapshot,['main.json']);
 assert.equal(r.schemaVersion,1);assert.equal(r.policyVersion,1);assert.equal(r.ready,true);assert.equal(r.publicationApproved,false);
 assert.deepEqual(r.selectedFiles[0].recordIds,['a','b']);assert.equal(r.selectedFiles[0].sha256,sha(before));assert.equal(r.selectedFiles[0].datasetId,'fixture');
 assert.deepEqual(r.issues.map(x=>[x.code,x.path,x.blocksSelected]),[['INDEX_DIGEST_MISMATCH','advanced.json',false]]);
 assert.deepEqual(readFileSync(join(snapshot,'files/main.json')),before);assert.deepEqual(readFileSync(join(source,'main.json')),before);assert.equal(existsSync(join(source,'SENTINEL')),false);
});
test('selected-only blocking isolates unselected digest differences',async t=>{
 const {snapshot}=fixture(t);const r=await build(snapshot,['advanced.json']);assert.equal(r.ready,false);assert.equal(r.issues[0].blocksSelected,true);
});
test('AI review never becomes publication approval',async t=>{
 const {snapshot}=fixture(t);const r=await build(snapshot,['main.json']);assert.equal(r.publicationApproved,false);assert.equal(r.ready,true);
});
test('missing identifiers and repeated record identities block ready',async t=>{
 for(const records of [[{title:'No id'}],[{id:'a',conditions:['positive']},{id:'a',conditions:['nonnegative']}]]){
  const {snapshot}=fixture(t,records);const r=await build(snapshot,['main.json']);assert.equal(r.ready,false);assert.ok(r.issues.some(x=>x.blocksSelected && ['RECORD_ID_MISSING','RECORD_ID_CONFLICT'].includes(x.code)));
 }
});
test('unknown selected path blocks ready without reading outside fixed batch',async t=>{
 const {snapshot}=fixture(t);const r=await build(snapshot,['missing.json']);assert.equal(r.ready,false);assert.ok(r.issues.some(x=>x.code==='SELECTED_FILE_MISSING'));
 await assert.rejects(()=>build(snapshot,['../main.json']),/INDEX_PATH_ESCAPE|INVALID_SELECTION/);
});
test('conflicting third-index hashes retain both origins in the report',async t=>{
 const {snapshot,source}=fixture(t);writeFileSync(join(source,'MSC2020_Incremental_Knowledge_JSON_Index.json'),JSON.stringify({packages:[{json_files:[{local_relative_path:'main.json',sha256:'b'.repeat(64)}]}]}));
 const next=join(snapshot,'next');const result=spawnSync(process.execPath,['tools/content-ingest/snapshot.mjs','--source',source,'--out',next],{encoding:'utf8'});assert.equal(result.status,0,result.stderr);
 const r=await build(next,['main.json']);assert.equal(r.ready,false);const issue=r.issues.find(x=>x.code==='INDEX_DIGEST_CONFLICT');assert.equal(issue.indexClaims.length,2);assert.equal(issue.blocksSelected,true);
});
test('tampered fixed bytes and symbolic paths cannot produce a trusted report',async t=>{
 const {snapshot,base}=fixture(t);writeFileSync(join(snapshot,'files/main.json'),'{}');await assert.rejects(()=>build(snapshot,['main.json']),/SNAPSHOT_BYTES_MISMATCH/);
 rmSync(join(snapshot,'files/main.json'));writeFileSync(join(base,'outside.json'),'{}');symlinkSync(join(base,'outside.json'),join(snapshot,'files/main.json'));await assert.rejects(()=>build(snapshot,['main.json']),/SNAPSHOT_SYMLINK/);
});
test('CLI writes one private report and never overwrites or writes into live sources',t=>{
 const {snapshot,source}=fixture(t);const selected=join(snapshot,'selected.json');writeFileSync(selected,'["main.json"]');const out=join(snapshot,'source-report.json');
 const run=(output)=>spawnSync(process.execPath,['tools/content-ingest/source-report.mjs','--snapshot',snapshot,'--selected',selected,'--out',output],{encoding:'utf8'});
 const r=run(out);assert.equal(r.status,0,r.stderr);const bytes=readFileSync(out);assert.equal(run(out).status,1);assert.deepEqual(readFileSync(out),bytes);assert.equal(run(join(source,'report.json')).status,1);assert.equal(existsSync(join(source,'report.json')),false);
});
