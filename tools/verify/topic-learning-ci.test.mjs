import test from'node:test';import assert from'node:assert/strict';import{readFileSync}from'node:fs';import{verifyTopicCompatibility,inverseTopicWorkflow}from'./topic-learning-compatibility.mjs';
const root=new URL('../../',import.meta.url).pathname,read=path=>readFileSync(new URL(path,new URL('../../',import.meta.url)),'utf8');
test('topic verification adds bounded batches and preserves the exact original workflows',()=>{
 const b=read('.github/workflows/backend.yml'),f=read('.github/workflows/frontend.yml');assert.equal(verifyTopicCompatibility({root,stage:'taxonomy'}).ok,true);
 assert(b.includes("-run '^TestTaxonomyCapacity6603With1000Knowledge$' -timeout 5m -count=1 -v"));assert(b.includes('tools/topic-ingest/*.test.mjs'));assert(f.includes('e2e -- topic-navigation.spec.ts'));assert(f.includes('e2e -- ui-language-taxonomy.spec.ts'));
 for(const [path,source]of [['.github/workflows/backend.yml',b],['.github/workflows/frontend.yml',f]]){const previous=inverseTopicWorkflow(source,path);assert.notEqual(source,previous);assert(source.includes('timeout-minutes: 30'));for(const bad of [source.replace('timeout-minutes: 30','timeout-minutes: 31'),source.replace('5m -count=1','6m -count=1'),source.replace('topic-navigation.spec.ts','topic-navigation.spec.ts --workers 2')]){if(bad!==source)assert.equal(inverseTopicWorkflow(bad,path),bad)}}
});
