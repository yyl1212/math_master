import assert from 'node:assert/strict';import {createHash} from 'node:crypto';
import {readPlaywrightPolicy} from './ui-language-ci.mjs';
const original={"backend": "78e6ebd236d1428910114ef7495f11eaa6e42e16e93cdd6b53dc0ed57deb05d9", "frontend": "6a05edbb9c6a96cada61ed1a78c5efa21c5e610246da3b638278d0f7924c7c7d"};
function job(source,name){const marker='  study_review_verify:\n';const at=source.indexOf(marker);assert(at>=0&&source.indexOf(marker,at+1)<0,'missing or duplicated new job');assert.equal(createHash('sha256').update(source.slice(0,at)).digest('hex'),original[name],'original workflow bytes changed');const value=source.slice(at);assert(!/^  [a-zA-Z_]+:/m.test(value.slice(marker.length)),'unexpected job');assert(value.includes('timeout-minutes: 30'),'new job budget');return value;}
export function validateStudyReviewCI({backend,frontend,playwright}){
 const b=job(backend,'backend'),f=job(frontend,'frontend'),policy=readPlaywrightPolicy(playwright);assert.deepEqual(policy,{workers:1,retries:0,globalTimeout:480000,viewports:[{width:1280,height:900},{width:390,height:844}]},'browser budget');
 const runs=s=>s.split('\n').map(l=>l.trim().replace(/^run: /,'')).filter(l=>l.startsWith('node tools/verify/run.mjs'));
 const br=runs(b),fr=runs(f);assert(br.includes('node tools/verify/run.mjs --cwd backend -- env CGO_ENABLED=0 GOTOOLCHAIN=go1.27.1 go test ./internal/studyreview -timeout 5m -count=1'),'missing full domain tests');
 for(const name of ['SchemaRestore','Read10000','Write10000'])assert(br.includes("node tools/verify/run.mjs --cwd backend -- env CGO_ENABLED=0 GOTOOLCHAIN=go1.27.1 go test ./internal/store -run '^TestReviewCapacity"+name+"$' -timeout 5m -count=1 -v"),'missing capacity');
 assert(br.some(l=>l.includes('./internal/studyreview ./internal/store ./internal/httpapi ./internal/e2etest')&&l.includes("-skip '^TestReviewCapacity' -timeout 5m -count=1")),'missing domain/real database coverage');
 assert(b.includes('STUDY_REVIEW_RESTORE_CONTAINER=${{ job.services.postgres.id }}'),'restore service missing');
 for(const name of ['flow','recall','security'])assert(fr.includes('node tools/verify/run.mjs --cwd frontend -- env -u NO_COLOR -u MATH_MASTER_E2E_FRONTEND npm run e2e -- study-review-'+name+'.spec.ts'),'missing current real browser');
 assert(!f.includes('MATH_MASTER_E2E_FRONTEND:'),'legacy frontend in new job');assert(fr.includes('node tools/verify/run.mjs --cwd frontend -- npm run build'));assert(fr.some(l=>l.includes('go build -o bin/e2e-harness')));
 return true;
}
