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
 const jobEnv=topic.slice(topic.indexOf('    env:'),topic.indexOf('    services:'));
 assert(!jobEnv.includes('${{ runner.'),'runner context belongs to step runtime');
 assert(topic.includes('TOPIC_CUTOVER_CAPACITY_RECEIPT=$RUNNER_TEMP/topic-cutover-capacity.json'));
 assert(topic.includes('>> \"$GITHUB_ENV\"'));
});

// Exact command inventory from approved pre-split f11a32e, not inferred from the current workflow.
const approvedFrontendCommands=[
  "node tools/verify/run.mjs --cwd frontend -- npm ci",
  "node tools/verify/run.mjs --cwd frontend -- npm run api:generate",
  "git diff --exit-code -- frontend/src/lib/api/generated.d.ts",
  "node tools/verify/run.mjs --cwd frontend -- npm run typecheck",
  "node tools/verify/run.mjs --cwd frontend -- npm test",
  "node tools/verify/run.mjs --cwd frontend -- npm run build",
  "node tools/verify/run.mjs --cwd frontend -- npm audit --omit=dev",
  "node tools/verify/run.mjs --cwd backend -- env CGO_ENABLED=0 GOTOOLCHAIN=go1.27.1 go test ./internal/e2etest ./internal/testutil -timeout 5m -count=1",
  "node tools/verify/run.mjs --cwd backend -- env CGO_ENABLED=0 GOTOOLCHAIN=go1.27.1 go build -o bin/e2e-harness ./cmd/e2e-harness",
  "node tools/verify/run.mjs --cwd frontend -- npm exec -- playwright install --with-deps chromium",
  "node tools/verify/run.mjs --cwd frontend -- env -u NO_COLOR npm run e2e -- catalogue.spec.ts reading.spec.ts",
  "node tools/verify/run.mjs --cwd frontend -- env -u NO_COLOR npm run e2e -- auth.spec.ts auth-security.spec.ts",
  "node tools/verify/run.mjs --cwd frontend -- env -u NO_COLOR npm run e2e -- content-authoring.spec.ts",
  "node tools/verify/run.mjs --cwd frontend -- env -u NO_COLOR npm run e2e -- content-review.spec.ts",
  "node tools/verify/run.mjs --cwd frontend -- env -u NO_COLOR npm run e2e -- content-release.spec.ts",
  "node tools/verify/run.mjs --cwd frontend -- env -u NO_COLOR npm run e2e -- content-security.spec.ts",
  "node tools/verify/run.mjs --cwd frontend -- env -u NO_COLOR npm run e2e -- question-authoring.spec.ts",
  "node tools/verify/run.mjs --cwd frontend -- env -u NO_COLOR npm run e2e -- question-review.spec.ts",
  "node tools/verify/run.mjs --cwd frontend -- env -u NO_COLOR npm run e2e -- question-release.spec.ts",
  "node tools/verify/run.mjs --cwd frontend -- env -u NO_COLOR npm run e2e -- question-security.spec.ts",
  "node tools/verify/run.mjs --cwd frontend -- env -u NO_COLOR npm run e2e -- learning-progress.spec.ts",
  "node tools/verify/run.mjs --cwd frontend -- env -u NO_COLOR npm run e2e -- learning-practice.spec.ts learning-assessment.spec.ts",
  "node tools/verify/run.mjs --cwd frontend -- env -u NO_COLOR npm run e2e -- learning-diagnostic.spec.ts learning-security.spec.ts",
  "node tools/verify/run.mjs --cwd frontend -- env -u NO_COLOR npm run e2e -- learning-review-regressions.spec.ts",
  "node tools/verify/run.mjs --cwd frontend -- env -u NO_COLOR npm run e2e -- feedback-user.spec.ts",
  "node tools/verify/run.mjs --cwd frontend -- env -u NO_COLOR npm run e2e -- feedback-review.spec.ts",
  "node tools/verify/run.mjs --cwd frontend -- env -u NO_COLOR npm run e2e -- feedback-security.spec.ts",
  "node tools/verify/run.mjs --cwd frontend -- env -u NO_COLOR npm run e2e -- correction-user.spec.ts",
  "node tools/verify/run.mjs --cwd frontend -- env -u NO_COLOR npm run e2e -- correction-review.spec.ts",
  "node tools/verify/run.mjs --cwd frontend -- env -u NO_COLOR npm run e2e -- notification-security.spec.ts",
  "node tools/verify/run.mjs --cwd frontend -- env -u NO_COLOR npm run e2e -- content-acceptance-reading.spec.ts",
  "node tools/verify/run.mjs --cwd frontend -- env -u NO_COLOR npm run e2e -- content-acceptance-learning.spec.ts",
  "node tools/verify/run.mjs --cwd frontend -- env -u NO_COLOR npm run e2e -- ui-language-public.spec.ts",
  "node tools/verify/run.mjs --cwd frontend -- env -u NO_COLOR npm run e2e -- ui-language-auth.spec.ts",
  "node tools/verify/run.mjs --cwd frontend -- env -u NO_COLOR npm run e2e -- ui-language-content.spec.ts",
  "node tools/verify/run.mjs --cwd frontend -- env -u NO_COLOR npm run e2e -- ui-language-question.spec.ts",
  "node tools/verify/run.mjs --cwd frontend -- env -u NO_COLOR npm run e2e -- ui-language-learning.spec.ts",
  "node tools/verify/run.mjs --cwd frontend -- env -u NO_COLOR npm run e2e -- ui-language-feedback.spec.ts",
  "node tools/verify/run.mjs --cwd frontend -- env -u NO_COLOR npm run e2e -- ui-language-correction.spec.ts",
  "node tools/verify/run.mjs --cwd frontend -- env -u NO_COLOR npm run e2e -- topic-navigation.spec.ts",
  "node tools/verify/run.mjs --cwd frontend -- env -u NO_COLOR npm run e2e -- ui-language-taxonomy.spec.ts",
  "node tools/verify/run.mjs --cwd frontend -- env -u NO_COLOR npm run e2e -- topic-study-navigation.spec.ts topic-study-progress.spec.ts topic-study-notes.spec.ts topic-study-security.spec.ts ui-language-study.spec.ts",
  "node tools/verify/run.mjs --cwd frontend -- env -u NO_COLOR npm run e2e -- topic-retirement.spec.ts topic-feedback.spec.ts topic-content-corrections.spec.ts",
  "node tools/verify/run.mjs --cwd frontend -- env -u NO_COLOR npm run e2e -- topic-learning-cutover.spec.ts",
  "node tools/verify/run.mjs --cwd frontend -- env -u NO_COLOR npm run e2e -- topic-learning-archive.spec.ts",
  "node tools/verify/run.mjs --cwd frontend -- env -u NO_COLOR npm run e2e -- ui-language-cutover.spec.ts",
  "node tools/verify/run.mjs --cwd frontend -- env -u NO_COLOR npm run e2e -- admin-knowledge-direct.spec.ts",
  "node tools/verify/run.mjs --cwd frontend -- env -u NO_COLOR npm run e2e -- managed-knowledge-study.spec.ts",
  "node tools/verify/run.mjs --cwd frontend -- env -u NO_COLOR npm run e2e -- ui-language-managed.spec.ts"
];
test('frontend browser batches have independent jobs and preserve every approved command',()=>{
 const f=read('.github/workflows/frontend.yml');assert(f.includes('  verify_later:'),'later browser batches need an independent job');
 const jobs=f.split(/^  (?=verify(?:_later)?:)/m).slice(1);assert.equal(jobs.length,2);
 for(const job of jobs){assert(job.includes('timeout-minutes: 30'));assert(job.includes('services:'));assert(job.includes('postgres:17.11'));assert(job.includes('go build -o bin/e2e-harness'));assert(job.includes('npm run build'));assert(job.includes('playwright install --with-deps chromium'));}
 const actual=f.split('\n').map(l=>l.trim().replace(/^run: /,'')).filter(l=>l.startsWith('node tools/verify/run.mjs')||l.startsWith('git diff --exit-code -- frontend/'));
 for(const command of approvedFrontendCommands)assert(actual.includes(command),'missing or changed original frontend command: '+command);
 for(const budget of f.matchAll(/timeout-minutes: (\d+)/g))assert.equal(budget[1],'30');
});
