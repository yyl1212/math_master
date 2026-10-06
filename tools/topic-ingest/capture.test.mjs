import test from 'node:test';
import assert from 'node:assert/strict';
import fs from 'node:fs/promises';
import {join} from 'node:path';
import {captureTopicBatch} from './capture.mjs';
import {acceptedFixture,stagedFixture,changingReadFixture,metadataNames,hash} from './test-fixtures.mjs';

test('capture_rejects_staged and unaccepted batches before writing',async t=>{
 const f=await stagedFixture(t);
 await assert.rejects(()=>captureTopicBatch(f),/BATCH_NOT_ACCEPTED/);
 await assert.rejects(()=>fs.stat(f.outDir),{code:'ENOENT'});
 const p=join(f.metadataDir,metadataNames[1]),m=JSON.parse(await fs.readFile(p));m.staged_only=false;m.formal_integration_performed=false;await fs.writeFile(p,JSON.stringify(m));
 await assert.rejects(()=>captureTopicBatch(f),/BATCH_NOT_ACCEPTED/);
});
test('capture_same_batch_is_stable and copies private original bytes',async t=>{
 const f=await acceptedFixture(t),before=await fs.readFile(join(f.knowledgeDir,f.primary));
 const first=await captureTopicBatch(f);
 assert.equal(first?.accepted,true);assert.equal(first?.batch,7);
 assert.match(first?.snapshotId??'',/^[0-9a-f]{64}$/);
 assert.equal(first.sourceFiles.some(e=>e.path==='Knowledge_JSON/'+f.primary&&e.sha256===hash(before)),true);
 assert.deepEqual(await fs.readFile(join(f.outDir,'files/Knowledge_JSON',f.primary)),before);
 assert.equal((await fs.stat(f.outDir)).mode&0o777,0o700);
 assert.equal((await fs.stat(join(f.outDir,'manifest.json'))).mode&0o777,0o600);
 const second=await captureTopicBatch({...f,outDir:join(f.base,'second'),previousManifestPath:join(f.outDir,'manifest.json')});
 assert.equal(first.snapshotId,second.snapshotId);assert.deepEqual(second.diff,{added:[],changed:[],missing:[]});
 await assert.rejects(()=>captureTopicBatch(f),/OUTPUT_EXISTS/);
 assert.deepEqual(await fs.readFile(join(f.knowledgeDir,f.primary)),before);
});
test('capture_rejects_path_escape and output in source',async t=>{
 const f=await acceptedFixture(t);
 for(const path of ['../private.json','/private.json','Fixture/../x.json','Fixture\\x.json','Fixture//x.json','Fixture/./x.json','a\u0000b.json'])await assert.rejects(()=>captureTopicBatch({...f,selectedPrimaryPaths:[path]}),/PATH_ESCAPE/);
 await assert.rejects(()=>captureTopicBatch({...f,outDir:join(f.sourceRoot,'generated')}),/OUTPUT_IN_SOURCE/);
});
test('capture_rejects symbolic input without reading outside files',async t=>{
 const f=await acceptedFixture(t),p=join(f.knowledgeDir,f.primary);
 await fs.unlink(p);await fs.symlink(join(f.metadataDir,metadataNames[0]),p);
 await assert.rejects(()=>captureTopicBatch(f),/SOURCE_SYMLINK/);
});
test('capture_changes_during_read rejects mutation and removes partial output',async t=>{
 const f=await changingReadFixture(t),target=await fs.realpath(join(f.knowledgeDir,f.primary));
 const open=fs.open.bind(fs);let changed=false;
 t.mock.method(fs,'open',async(...args)=>{
  const handle=await open(...args);
  if(String(args[0])===target){
   const read=handle.read.bind(handle);
   t.mock.method(handle,'read',async(...readArgs)=>{
    const result=await read(...readArgs);
    if(!changed){changed=true;await fs.writeFile(target,Buffer.from('A changed source during capture'));}
    return result;
   });
  }
  return handle;
 });
 await assert.rejects(()=>captureTopicBatch(f),/SOURCE_CHANGED_RETRY/);
 await assert.rejects(()=>fs.stat(f.outDir),{code:'ENOENT'});
});
test('capture rejects oversized metadata without loading it and honors cancellation',async t=>{
 const f=await acceptedFixture(t),p=join(f.metadataDir,metadataNames[1]);
 await fs.truncate(p,(8<<20)+1);await assert.rejects(()=>captureTopicBatch(f),/SOURCE_LIMIT_EXCEEDED/);
 const abort=new AbortController();abort.abort();
 await assert.rejects(()=>captureTopicBatch({...f,signal:abort.signal}),/CAPTURE_ABORTED/);
});
test('capture_quarantines an index mismatch without altering source index',async t=>{
 const f=await acceptedFixture(t),p=join(f.knowledgeDir,'Knowledge_JSON_Index.json');
 const index=JSON.parse(await fs.readFile(p));index.packages[0].json_files[0].sha256='a'.repeat(64);await fs.writeFile(p,JSON.stringify(index));
 const before=await fs.readFile(p),m=await captureTopicBatch(f);
 assert.equal(m?.accepted,true);assert.deepEqual(m.issues,[{code:'INDEX_DIGEST_MISMATCH',path:f.primary}]);
 assert.equal(m.sourceFiles.some(x=>x.path==='Knowledge_JSON/'+f.primary),false);
 assert.deepEqual(await fs.readFile(p),before);
});
test('capture_rejects duplicate keys, NUL and unknown metadata envelope fields',async t=>{
 const f=await acceptedFixture(t),p=join(f.metadataDir,metadataNames[1]),original=await fs.readFile(p);
 for(const bytes of [
  original.toString().replace('"candidate_batch":7','"candidate_batch":7,"candidate_batch":8'),
  original.toString().replace('"schema_version":"1.0"','"schema_version":"\\u0000"'),
  original.toString().replace('{','{"run_shell":"untrusted text",')
 ]){
  await fs.writeFile(p,bytes);await assert.rejects(()=>captureTopicBatch(f),/INVALID_SOURCE_JSON|UNKNOWN_METADATA_FIELD/);
 }
});
test('capture_rejects mismatched accepted batches and classification hashes',async t=>{
 const f=await acceptedFixture(t),p=join(f.metadataDir,metadataNames[2]),m=JSON.parse(await fs.readFile(p));m.candidate_batch=8;
 await fs.writeFile(p,JSON.stringify(m));await assert.rejects(()=>captureTopicBatch(f),/BATCH_MISMATCH/);
 m.candidate_batch=7;m.classification_raw_sha256='b'.repeat(64);await fs.writeFile(p,JSON.stringify(m));
 await assert.rejects(()=>captureTopicBatch(f),/CLASSIFICATION_DIGEST_MISMATCH/);
});
test('capture accepts the recorded source8 reconciliation pin but not arbitrary keys',async t=>{
 const f=await acceptedFixture(t),p=join(f.metadataDir,metadataNames[3]);
 const registry=JSON.parse(await fs.readFile(p));registry.previous_catalog_snapshot_before_source8_reconciliation={sha256:'d'.repeat(64),book_id_count:1};
 await fs.writeFile(p,JSON.stringify(registry));const m=await captureTopicBatch(f);assert.equal(m.accepted,true);
 registry.previous_catalog_snapshot_before_source8_reconciliation={sha256:'d'.repeat(64),book_id_count:-1};await fs.writeFile(p,JSON.stringify(registry));
 await assert.rejects(()=>captureTopicBatch({...f,outDir:join(f.base,'invalid-pin')}),/INVALID_SOURCE_JSON/);
});
