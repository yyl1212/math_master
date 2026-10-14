import assert from 'node:assert/strict';
import {activeRuns,readPlaywrightPolicy} from './ui-language-ci.mjs';
const currentCommand=file=>'node tools/verify/run.mjs --cwd frontend -- env -u NO_COLOR -u MATH_MASTER_E2E_FRONTEND npm run e2e -- page-beauty-'+file+'.spec.ts';
export function validatePageBeautyCI({workflow,config,base}){
 assert.deepEqual(readPlaywrightPolicy(base),{workers:1,retries:0,globalTimeout:480000,viewports:[{width:1280,height:900},{width:390,height:844}]},'original browser budget');
 assert(/import base from ['"]\.\/playwright\.config['"]/.test(config)&&config.includes('...base'),'inherit original server and safety config');
 assert(/testMatch:\s*['"]\*\*\/page-beauty-cross-browser\.spec\.ts['"]/.test(config),'explicit cross-browser scope');
 assert(!/\b(?:workers|retries|globalTimeout|timeout|webServer|baseURL)\s*:/.test(config),'no browser budget/server overrides');
 assert(config.includes('process.env.MATH_MASTER_E2E_FRONTEND')&&config.includes('throw new Error'),'current frontend only');
 const engines=[...config.matchAll(/browserName:\s*['"]([^'"]+)['"]/g)].map(m=>m[1]);assert.deepEqual(engines,['firefox','firefox','webkit','webkit'],'both engines and two widths');
 const viewports=[...config.matchAll(/viewport:\s*\{\s*width:\s*(\d+)\s*,\s*height:\s*(\d+)\s*\}/g)].map(m=>({width:Number(m[1]),height:Number(m[2])}));assert.deepEqual(viewports,[{width:1280,height:900},{width:390,height:844},{width:1280,height:900},{width:390,height:844}],'core viewports');
 const budgets=[...workflow.matchAll(/^\s*timeout-minutes:\s*(\d+)\s*$/gm)].map(m=>Number(m[1]));assert.deepEqual(budgets,[30],'job budget');
 assert(workflow.includes("CGO_ENABLED: '0'")&&workflow.includes("node-version: '24.17.0'")&&workflow.includes("go-version: '1.27.1'")&&workflow.includes('image: postgres:17.11'),'locked runtime');
 assert(!/^\s*MATH_MASTER_E2E_FRONTEND\s*:/m.test(workflow),'no legacy frontend environment');
 const commands=activeRuns(workflow).flatMap(run=>run.split('\n'));
 const required=[
 'node tools/verify/run.mjs --cwd frontend -- npm ci',
 'node tools/verify/run.mjs --cwd frontend -- npm run build',
 'node tools/verify/run.mjs --cwd backend -- env CGO_ENABLED=0 GOTOOLCHAIN=go1.27.1 go build -o bin/e2e-harness ./cmd/e2e-harness',
 'node tools/verify/run.mjs --cwd frontend -- npm exec -- playwright install --with-deps chromium firefox webkit',
 ...['public','learning','admin'].map(currentCommand),
 'node tools/verify/run.mjs --cwd frontend -- env -u NO_COLOR -u MATH_MASTER_E2E_FRONTEND npm exec -- playwright test --config ../tests/e2e/page-beauty.playwright.config.ts',
 'node tools/verify/run.mjs -- node tools/verify/page-beauty-assets.mjs --capture "$PAGE_BEAUTY_CAPTURE"',
 'node tools/verify/run.mjs -- node --test tools/verify/page-beauty-ci.test.mjs tools/verify/page-beauty-assets.test.mjs'
 ];
 for(const command of required)assert(commands.includes(command),'missing current real verification: '+command);
 return true;
}
