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
 assert.equal(r.packages.length,0);assert.equal(r.issues[0].code,'MAPPING_REQUIRED');
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
