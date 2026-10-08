import test from 'node:test';
import assert from 'node:assert/strict';
import fs from 'node:fs/promises';
import {join} from 'node:path';
import {acceptedFixture} from './test-fixtures.mjs';
import {captureTopicBatch} from './capture.mjs';
import {buildCatalogueArchive} from './build-catalogue.mjs';
test('builds an offline catalogue with original codes and only captured record references',async t=>{
 const f=await acceptedFixture(t);await captureTopicBatch(f);
 const result=await buildCatalogueArchive(f.outDir);
 assert.deepEqual(result.nodes.map(n=>n.id),['msc-13','msc-13c','msc-13c60']);
 assert.equal(result.nodes[2].parentId,'msc-13c');
 assert.deepEqual(result.sourceRecordIndex,[{sourceId:'fixture-source',workFamilyId:'fixture-work',recordId:'record-1',path:f.primary,sha256:result.manifest.sourceFiles.find(x=>x.path==='Knowledge_JSON/'+f.primary).sha256}]);
 assert.equal(result.manifest.accepted,true);
 assert.equal('ownerId' in result,false);
});
test('rejects modified captured bytes without consulting or changing live source',async t=>{
 const f=await acceptedFixture(t);await captureTopicBatch(f);await fs.writeFile(join(f.outDir,'files/Knowledge_JSON',f.primary),'{}');
 await assert.rejects(()=>buildCatalogueArchive(f.outDir),/CAPTURE_BYTES_CHANGED/);
});

test('provides verified dot mappings to automatically classify the captured draft',async t=>{
 const f=await acceptedFixture(t);await captureTopicBatch(f);
 const {buildImportContext}=await import('./build-catalogue.mjs');
 const context=await buildImportContext(f.outDir);
 const {normalizePrimaryFile}=await import('./adapters.mjs');
 const {buildDraftInputs}=await import('./build-drafts.mjs');
 const bytes=await fs.readFile(join(f.outDir,'files/Knowledge_JSON',f.primary));
 const records=normalizePrimaryFile({bytes,packageId:'fixture',sourceId:'fixture-source',path:f.primary,sha256:context.capture.sourceRecordIndex[0].sha256}).records;
 const result=buildDraftInputs({...context,records,legacyCatalogue:{version:1}});
 assert.equal(result.packages.length,1);assert.deepEqual(result.packages[0].assignments[0].topicIds,['msc-13c60']);
 assert.equal(result.packages[0].assignments[0].sourceRefs[0].recordId,'record-1');
});
