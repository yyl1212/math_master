import test from 'node:test';
import assert from 'node:assert/strict';
import {createHash} from 'node:crypto';
import {normalizePrimaryFile} from './adapters.mjs';
const sha=b=>createHash('sha256').update(b).digest('hex');
const input=(data,extra={})=>{const bytes=Buffer.from(JSON.stringify(data));return {bytes,packageId:'fixture-package',sourceId:'fixture-source',path:'Fixture/knowledge.json',sha256:sha(bytes),...extra};};
test('normalizes both corpus forms without losing original IDs or unknown data',()=>{
 for(const field of ['knowledge_points','records']){
  const raw={id:'original.1',title:'An original definition',kind:'definition',statement:'Definition text',conditions:['x > 0'],proof:'A supplied proof',proof_scope:'full',provenance:[{section_ids:['1'],pdf_page_1_based:2}],extra_source_field:{note:'Keep in archive'}};
  const result=normalizePrimaryFile(input({[field]:[raw],license:{license_id:'fixture-license'}}));
  assert.equal(result.records.length,1);const r=result.records[0];
  assert.equal(r.originalId,'original.1');assert.equal(r.statement,'Definition text');assert.deepEqual(r.conditions,['x > 0']);
  assert.equal(r.proof,'A supplied proof');assert.equal(r.proofScope,'full');assert.deepEqual(r.raw.extra_source_field,{note:'Keep in archive'});
  assert.deepEqual(r.rightsMetadata,{license_id:'fixture-license'});
 }
});
test('software capabilities remain auxiliary and do not become mathematical knowledge',()=>{
 const result=normalizePrimaryFile(input({capabilities:Array.from({length:104},(_,i)=>({id:'software-capability-'+(i+1),title:'Capability '+i}))}));
 assert.equal(result.auxiliary.length,104);assert.equal(result.records.length,0);
});
test('unknown type and proof idea remain explicit and are not invented',()=>{
 const result=normalizePrimaryFile(input({records:[{id:'r',title:'A method',kind:'unclassified_source_kind',proof_idea:{text:'Only a sketch'}}]}));
 assert.equal(result.records[0].originalKind,'unclassified_source_kind');assert.equal(result.records[0].statement,'');
 assert.equal(result.records[0].proofScope,'sketch');assert.equal(result.records[0].proof,'Only a sketch');
 assert.equal(result.issues.some(i=>i.code==='STATEMENT_MISSING'),true);
});
test('rejects altered byte claims, unsafe paths and ambiguous corpus roots',()=>{
 assert.throws(()=>normalizePrimaryFile(input({records:[]},{sha256:'a'.repeat(64)})),/SOURCE_DIGEST_MISMATCH/);
 assert.throws(()=>normalizePrimaryFile(input({records:[]},{path:'../private.json'})),/PATH_ESCAPE/);
 assert.throws(()=>normalizePrimaryFile(input({records:[],knowledge_points:[]})),/AMBIGUOUS_CORPUS/);
});
