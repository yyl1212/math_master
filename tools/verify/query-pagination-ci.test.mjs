import test from 'node:test';import assert from 'node:assert/strict';import {readFileSync} from 'node:fs';
import {validateQueryPaginationCI} from './query-pagination-ci.mjs';
const read=p=>readFileSync(new URL('../../'+p,import.meta.url),'utf8');
test('query pagination has bounded dedicated SQL and browser coverage',()=>{const workflow=read('.github/workflows/query-pagination.yml'),playwright=read('tests/e2e/playwright.config.ts');assert(validateQueryPaginationCI(workflow,playwright));for(const value of ['Numbered','QueryPagination','query-pagination.spec.ts','npm run typecheck','npm run build'])assert.throws(()=>validateQueryPaginationCI(workflow.replaceAll(value,'removed'),playwright));});
