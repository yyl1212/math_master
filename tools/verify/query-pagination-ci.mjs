import assert from 'node:assert/strict';import {activeRuns,readPlaywrightPolicy} from './ui-language-ci.mjs';
export function validateQueryPaginationCI(source,playwright){
 assert.deepEqual(readPlaywrightPolicy(playwright),{workers:1,retries:0,globalTimeout:480000,viewports:[{width:1280,height:900},{width:390,height:844}]});
 const runs=activeRuns(source).join('\n');for(const required of ['Numbered','QueryPagination','query-pagination.spec.ts','npm run typecheck','npm run build','npm test','./internal/pagination','go build -o bin/e2e-harness'])assert(runs.includes(required),'missing '+required);
 for(const line of runs.split('\n'))assert(line.startsWith('node tools/verify/run.mjs '),'unbounded command');return true;
}
