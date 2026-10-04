import test from 'node:test';
import assert from 'node:assert/strict';
import { existsSync } from 'node:fs';
const entry=(path,sha256,extra={})=>({local_relative_path:path,filename:path,sha256,...extra});
const index=(files,primary=[])=>Buffer.from(JSON.stringify({packages:[{json_files:files,primary_knowledge_local_relative_paths:primary}]}));
const h='a'.repeat(64),other='b'.repeat(64);
async function read(input) {
  assert.ok(existsSync(new URL('./source-index.mjs',import.meta.url)), 'three-index declaration reader is required');
  return (await import('./source-index.mjs')).readIndexDeclarations(input);
}
test('legacy one and two indices preserve primary selection and counts',async()=>{
 const first=index([entry('a.json',h,{recommended_knowledge_entry:true})]);
 const one=await read(new Map([['Knowledge_JSON_Index.json',first]]));
 assert.equal(one.packageCount,1);assert.deepEqual(one.primaryFiles,['a.json']);
 const two=await read(new Map([['Knowledge_JSON_Index.json',first],['Incremental_Knowledge_JSON_Index.json',index([entry('b.json',h)],['b.json'])]]));
 assert.equal(two.packageCount,2);assert.deepEqual(two.primaryFiles,['a.json','b.json']);
});
test('third index contributes primary entries while shared declarations retain all origins',async()=>{
 const result=await read(new Map([
 ['Knowledge_JSON_Index.json',index([entry('a.json',h)],['a.json'])],
 ['Incremental_Knowledge_JSON_Index.json',index([entry('a.json',h)])],
 ['MSC2020_Incremental_Knowledge_JSON_Index.json',index([entry('a.json',h),entry('c.json',h)],['c.json'])],
 ]));
 assert.deepEqual(result.indices,['Knowledge_JSON_Index.json','Incremental_Knowledge_JSON_Index.json','MSC2020_Incremental_Knowledge_JSON_Index.json']);
 assert.deepEqual(result.primaryFiles,['a.json','c.json']);assert.equal(result.packageCount,3);
 assert.equal(result.declarations.length,2);assert.equal(result.declarations[0].claims.length,3);assert.deepEqual(result.issues,[]);
});
test('conflicting declarations retain both hashes instead of last-wins',async()=>{
 const result=await read(new Map([['Knowledge_JSON_Index.json',index([entry('a.json',h)],['a.json'])],['MSC2020_Incremental_Knowledge_JSON_Index.json',index([entry('a.json',other)])]]));
 assert.deepEqual(result.declarations[0].claims,[{index:'Knowledge_JSON_Index.json',sha256:h},{index:'MSC2020_Incremental_Knowledge_JSON_Index.json',sha256:other}]);
 assert.deepEqual(result.issues,[{code:'INDEX_DIGEST_CONFLICT',path:'a.json'}]);
});
test('missing digest remains a report issue without invalidating legacy capture',async()=>{
 const result=await read(new Map([['Knowledge_JSON_Index.json',index([entry('a.json',undefined)],['a.json'])]]));
 assert.deepEqual(result.declarations[0].claims,[{index:'Knowledge_JSON_Index.json',sha256:null}]);
 assert.deepEqual(result.issues,[{code:'INDEX_DIGEST_MISSING',path:'a.json'}]);
});
test('rejects unsafe primary and declared paths and missing base index',async()=>{
 for(const p of ['../a.json','/a.json','a\\b.json','a/./b.json','a//b.json']) {
  await assert.rejects(()=>read(new Map([['Knowledge_JSON_Index.json',index([entry(p,h)])]])),/INDEX_PATH_ESCAPE/);
  await assert.rejects(()=>read(new Map([['Knowledge_JSON_Index.json',index([], [p])]])),/INDEX_PATH_ESCAPE/);
 }
 await assert.rejects(()=>read(new Map()),/INDEX_REQUIRED/);
});
