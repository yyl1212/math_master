import test from'node:test';import assert from'node:assert/strict';import{readFileSync,statSync}from'node:fs';import{verifyTopicCompatibility,inverseTopicWorkflow}from'./topic-learning-compatibility.mjs';
const root=new URL('../../',import.meta.url).pathname,read=path=>readFileSync(new URL(path,new URL('../../',import.meta.url)),'utf8');
test('topic verification adds bounded batches and preserves the exact original workflows',()=>{
 const b=read('.github/workflows/backend.yml'),f=read('.github/workflows/frontend.yml');assert.equal(verifyTopicCompatibility({root,stage:process.env.TOPIC_COMPATIBILITY_STAGE??'taxonomy'}).ok,true);
 assert(b.includes("-run '^TestTaxonomyCapacity6603With1000Knowledge$' -timeout 5m -count=1 -v"));assert(b.includes('tools/topic-ingest/*.test.mjs'));assert(f.includes('e2e -- topic-navigation.spec.ts'));assert(f.includes('e2e -- ui-language-taxonomy.spec.ts'));
 for(const [path,source]of [['.github/workflows/backend.yml',b],['.github/workflows/frontend.yml',f]]){const previous=inverseTopicWorkflow(source,path);assert.notEqual(source,previous);assert(source.includes('timeout-minutes: 30'));for(const bad of [source.replace('timeout-minutes: 30','timeout-minutes: 31'),source.replace('5m -count=1','6m -count=1'),source.replace('topic-navigation.spec.ts','topic-navigation.spec.ts --workers 2')]){if(bad!==source)assert.equal(inverseTopicWorkflow(bad,path),bad)}}
});
test('pinned historical baseline is available in a fresh CI checkout',()=>{const b=read('.github/workflows/backend.yml');assert.equal(/actions\/checkout@[^\n]+\n\s+with:\n\s+fetch-depth: 0/.test(b),true,'pinned historical baseline checkout');});

test('topic capacity runs in its own bounded job without removing original checks',()=>{const b=read('.github/workflows/backend.yml');const original=b.slice(0,b.indexOf('  correction_verify:'));assert(!original.includes('- name: 个人主题学习最大存储读取'));const topic=b.slice(b.indexOf('  topic_verify:'));assert(topic.includes('timeout-minutes: 30'));assert(topic.includes('-run \'^TestTaxonomyCapacity6603With1000Knowledge$\''));assert(topic.includes('-run \'^TestStudyCapacityReadPages$\''));for(const name of ['最大合法内容工作流','最大合法题库工作流','题库候选容量','首批只读最大合法内容与题库'])assert(original.includes(name));});

test('trusted build revision and topic readiness unit tests are retained in CI',()=>{const b=read('.github/workflows/backend.yml');const runs=b.split('\n').filter(line=>line.trim().startsWith('run:')&&line.includes('go test '));assert(runs.some(line=>line.includes('./internal/buildmeta')&&line.includes('-timeout 5m -count=1')));assert(runs.some(line=>line.includes('./internal/httpapi')&&line.includes('-timeout 5m -count=1')));for(const line of runs){for(const path of line.match(/\.\/[a-zA-Z0-9_/-]+/g)??[]){assert(statSync(root+'backend/'+path).isDirectory(),'Go CI package directory exists: '+path)}}});

test('cutover capacity preparation and migration have independent five minute budgets and unconditional cleanup',()=>{
 const b=read('.github/workflows/backend.yml');
 const topic=b.slice(b.indexOf('  topic_verify:'));
 assert(topic.includes("-run '^TestTopicCutoverCapacityPrepare$' -timeout 5m -count=1 -v"));
 assert(topic.includes("-run '^TestTopicCutoverCapacityMigrate$' -timeout 5m -count=1 -v"));
 assert(/if: always\(\)[\s\S]*-run '\^TestTopicCutoverCapacityCleanup\$' -timeout 1m/.test(topic));
 assert(!topic.includes("-run '^TestTopicCutoverCapacity$'"));
});
