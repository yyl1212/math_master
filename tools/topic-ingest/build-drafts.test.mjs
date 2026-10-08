import test from 'node:test';
import assert from 'node:assert/strict';
import {createHash} from 'node:crypto';
import {normalizePrimaryFile} from './adapters.mjs';
import {buildDraftInputs} from './build-drafts.mjs';
const sha=b=>createHash('sha256').update(b).digest('hex');
function setup(raw=[{id:'r1',title:'First definition',kind:'definition',statement:'Supplied statement',proof_idea:{text:'Only a sketch'},author_ids:['not-a-platform-author'],review_status:'approved'}]){
 const bytes=Buffer.from(JSON.stringify({records:raw})),digest=sha(bytes);
 const records=normalizePrimaryFile({bytes,packageId:'p',sourceId:'source-a',path:'Fixture/knowledge.json',sha256:digest}).records;
 const sourceFiles=[{path:'Knowledge_JSON/Fixture/knowledge.json',sizeBytes:bytes.length,sha256:digest}];
 const capture={accepted:true,snapshotId:sha(JSON.stringify(sourceFiles)),sourceFiles,nodes:[{id:'msc-13c60',kind:'primary',level:3}]};
 const resolutions=records.map((r,i)=>({sourceId:r.sourceId,originalId:r.originalId,websiteId:'knowledge-'+(i+1),version:1,type:'definition',topicIds:['msc-13c60'],workFamilyId:'work-a'}));
 return {capture,records,resolutions,legacyCatalogue:{version:1,domains:[]}};
}
test('builds saved drafts with exact provenance and no inherited approval or authors',()=>{
 const f=setup(),result=buildDraftInputs(f);
 assert.equal(result.packages.length,1);const e=result.packages[0];
 assert.equal(e.kind,'topic-draft');assert.equal(e.draft.catalogueVersion,1);assert.equal(e.draft.package.knowledge.length,1);
 assert.equal(e.draft.package.knowledge[0].statement,'Supplied statement');assert.equal(e.draft.package.knowledge[0].proof,'');
 assert.deepEqual(e.draft.package.units[0].angles,[]);assert.deepEqual(e.draft.package.paths,[]);
 assert.equal(e.draft.sourceMap[0].legacyId,'r1');assert.equal(e.draft.sourceMap[0].sha256,f.records[0].rawSHA);
 assert.deepEqual(e.assignments[0].topicIds,['msc-13c60']);
 assert.equal(JSON.stringify(e).includes('not-a-platform-author'),false);assert.equal('status' in e.draft,false);
});
test('same source ID and identical original record deduplicate but different works stay separate',()=>{
 const f=setup();f.records.push({...f.records[0]});f.records.push({...f.records[0],sourceId:'source-b'});
 f.resolutions.push({...f.resolutions[0],sourceId:'source-b',websiteId:'other-knowledge',workFamilyId:'other-work'});
 const result=buildDraftInputs(f);assert.equal(result.packages[0].draft.package.knowledge.length,2);
});
test('missing classification or type mapping reports issues instead of producing guessed content',()=>{
 const f=setup();f.resolutions=[];const r=buildDraftInputs(f);
 assert.equal(r.packages.length,1);assert.deepEqual(r.packages[0].assignments,[]);assert.equal(r.issues[0].code,'TOPIC_MAPPING_REQUIRED');
 const g=setup();g.resolutions[0].type='unsupported';assert.equal(buildDraftInputs(g).issues[0].code,'TYPE_MAPPING_REQUIRED');
 const h=setup();h.resolutions[0].topicIds=['msc-unknown'];assert.equal(buildDraftInputs(h).issues[0].code,'TOPIC_MAPPING_REQUIRED');
});
test('conflicting records and repeated website IDs are not silently merged',()=>{
 const f=setup();f.records.push({...f.records[0],statement:'Different statement',semanticSHA:'a'.repeat(64)});
 const r=buildDraftInputs(f);assert.equal(r.issues.some(i=>i.code==='RECORD_CONFLICT'),true);assert.equal(r.packages.length,0);
 const g=setup([{id:'a',title:'A',kind:'definition',statement:'A'},{id:'b',title:'B',kind:'definition',statement:'B'}]);g.resolutions[1].websiteId=g.resolutions[0].websiteId;
 assert.equal(buildDraftInputs(g).issues.some(i=>i.code==='WEBSITE_ID_CONFLICT'),true);
});
test('splits batches at one hundred and retains stable outputs for unchanged inputs',()=>{
 const f=setup(Array.from({length:101},(_,i)=>({id:'record-'+i,title:'Definition '+i,kind:'definition',statement:'Supplied '+i})));
 const a=buildDraftInputs(f),b=buildDraftInputs(f);assert.deepEqual(a,b);
 assert.deepEqual(a.packages.map(e=>e.draft.package.knowledge.length),[100,1]);
 for(const e of a.packages)assert.match(e.draft.package.id,/^[a-z][a-z0-9-]{0,63}$/);
});
test('does not package records outside accepted capture and leaves caller records intact',()=>{
 const f=setup(),before=JSON.stringify(f.records);f.capture.accepted=false;
 assert.throws(()=>buildDraftInputs(f),/BATCH_NOT_ACCEPTED/);assert.equal(JSON.stringify(f.records),before);
 const g=setup();g.capture.sourceFiles=[];assert.equal(buildDraftInputs(g).issues[0].code,'SOURCE_NOT_CAPTURED');
});

test('automatically binds a captured dot record mapping without per-record resolutions',()=>{
 const f=setup();f.resolutions=[];
 f.sourceMappings=[{source_id:'source-a',work_family_id:'work-a',msc_code:'13C60',knowledge_record_ids:['r1'],knowledge_local_primary_paths:['Fixture/knowledge.json']}];
 f.capture.sourceRecordIndex=[{sourceId:'source-a',workFamilyId:'work-a',recordId:'r1',path:'Fixture/knowledge.json',sha256:f.records[0].rawSHA}];
 const result=buildDraftInputs(f);assert.equal(result.packages.length,1);
 const draft=result.packages[0];assert.match(draft.draft.package.knowledge[0].id,/^k-[a-f0-9]{56}$/);
 assert.deepEqual(draft.assignments[0].topicIds,['msc-13c60']);assert.deepEqual(draft.assignments[0].sourceRefs,f.capture.sourceRecordIndex);
 assert.deepEqual(draft.draft.sourceMap[0].knowledge,draft.assignments[0].knowledge);
 assert.equal(draft.draft.package.knowledge[0].type,'definition');assert.equal('status' in draft.draft,false);
});
test('keeps an unmapped valid knowledge draft pending instead of discarding it',()=>{
 const f=setup();f.resolutions=[];f.sourceMappings=[];
 const result=buildDraftInputs(f);assert.equal(result.packages.length,1);assert.equal(result.packages[0].draft.package.knowledge.length,1);assert.deepEqual(result.packages[0].assignments,[]);
 assert.equal(result.issues.some(x=>x.code==='TOPIC_MAPPING_REQUIRED'),true);
});
test('conflicting direct and dot tags are pending and missing captured references cannot be fabricated',()=>{
 const f=setup([{id:'r1',title:'Original',kind:'definition',statement:'Original',msc_codes:['13C60']}]);f.resolutions=[];
 f.capture.nodes.push({id:'msc-13c10',kind:'primary',level:3});
 f.sourceMappings=[{source_id:'source-a',work_family_id:'work-a',msc_code:'13C10',knowledge_record_ids:['r1'],knowledge_local_primary_paths:['Fixture/knowledge.json']}];
 f.capture.sourceRecordIndex=[{sourceId:'source-a',workFamilyId:'work-a',recordId:'r1',path:'Fixture/knowledge.json',sha256:f.records[0].rawSHA}];
 let result=buildDraftInputs(f);assert.equal(result.packages.length,1);assert.deepEqual(result.packages[0].assignments,[]);assert.equal(result.issues.some(x=>x.code==='TOPIC_MAPPING_CONFLICT'),true);
 f.sourceMappings[0].msc_code='13C60';f.capture.sourceRecordIndex=[];result=buildDraftInputs(f);assert.deepEqual(result.packages[0].assignments,[]);assert.equal(result.issues.some(x=>x.code==='SOURCE_REFERENCE_REQUIRED'),true);
});

test('carries the installed taxonomy version for import without manual digest input',()=>{
 const f=setup();f.taxonomyVersionId='c'.repeat(64);
 assert.equal(buildDraftInputs(f).packages[0].taxonomyVersionId,'c'.repeat(64));
 f.taxonomyVersionId='invalid';assert.throws(()=>buildDraftInputs(f),/INVALID_TAXONOMY_VERSION/);
});
