import test from 'node:test';
import assert from 'node:assert/strict';
import {readFileSync} from 'node:fs';
import {validatePageBeautyCI} from './page-beauty-ci.mjs';
const read=path=>readFileSync(new URL('../../'+path,import.meta.url),'utf8');
const publicCommand='node tools/verify/run.mjs --cwd frontend -- env -u NO_COLOR -u MATH_MASTER_E2E_FRONTEND npm run e2e -- page-beauty-public.spec.ts';
const fixture=()=>({workflow:`timeout-minutes: 30
CGO_ENABLED: '0'
node-version: '24.17.0'
go-version: '1.27.1'
image: postgres:17.11
run: node tools/verify/run.mjs --cwd frontend -- npm ci
run: node tools/verify/run.mjs --cwd frontend -- npm run build
run: node tools/verify/run.mjs --cwd backend -- env CGO_ENABLED=0 GOTOOLCHAIN=go1.27.1 go build -o bin/e2e-harness ./cmd/e2e-harness
run: node tools/verify/run.mjs --cwd frontend -- npm exec -- playwright install --with-deps chromium firefox webkit
run: ${publicCommand}
run: ${publicCommand.replace('public','learning')}
run: ${publicCommand.replace('public','admin')}
run: node tools/verify/run.mjs --cwd frontend -- env -u NO_COLOR -u MATH_MASTER_E2E_FRONTEND npm exec -- playwright test --config ../tests/e2e/page-beauty.playwright.config.ts
run: node tools/verify/run.mjs -- node tools/verify/page-beauty-assets.mjs --capture "$PAGE_BEAUTY_CAPTURE"
run: node tools/verify/run.mjs -- node --test tools/verify/page-beauty-ci.test.mjs tools/verify/page-beauty-assets.test.mjs
`,config:`import base from './playwright.config';
if(process.env.MATH_MASTER_E2E_FRONTEND)throw new Error('Current frontend only');
export default defineConfig({...base,testMatch:'**/page-beauty-cross-browser.spec.ts',projects:[
{name:'firefox-desktop',use:{browserName:'firefox',viewport:{width:1280,height:900}}},
{name:'firefox-mobile',use:{browserName:'firefox',viewport:{width:390,height:844}}},
{name:'webkit-desktop',use:{browserName:'webkit',viewport:{width:1280,height:900}}},
{name:'webkit-mobile',use:{browserName:'webkit',viewport:{width:390,height:844}}}
]});`,base:read('tests/e2e/playwright.config.ts')});
test('独立视觉任务验证真实当前前端、三引擎安装及同口径资产门禁',()=>assert.equal(validatePageBeautyCI(fixture()),true));
test('移除任一浏览器批次、安装引擎或资产命令必须失败',()=>{for(const from of ['page-beauty-public.spec.ts','page-beauty-learning.spec.ts','page-beauty-admin.spec.ts','test --config ../tests/e2e/page-beauty.playwright.config.ts','chromium firefox webkit','page-beauty-assets.mjs --capture']){const s=fixture();assert.throws(()=>validatePageBeautyCI({...s,workflow:s.workflow.replace(from,'removed')}),from)}});
test('预算、重试、旧前端及缺少项目不能放行',()=>{for(const [key,from,to] of [['base','workers: 1','workers: 2'],['base','retries: 0','retries: 1'],['base','globalTimeout: 480000','globalTimeout: 540000'],['config','...base,','...base,workers:2,'],['config',"browserName:'webkit'","browserName:'chromium'"],['workflow','-u MATH_MASTER_E2E_FRONTEND',''],['workflow','timeout-minutes: 30','timeout-minutes: 31']]){const s=fixture();assert.throws(()=>validatePageBeautyCI({...s,[key]:s[key].replace(from,to)}),from)}const s=fixture();assert.throws(()=>validatePageBeautyCI({...s,workflow:s.workflow+'\nMATH_MASTER_E2E_FRONTEND: legacy\n'}))});
test('仓库实际新workflow与专用配置通过，原配置继续约束预算',()=>assert.equal(validatePageBeautyCI({workflow:read('.github/workflows/page-beauty.yml'),config:read('tests/e2e/page-beauty.playwright.config.ts'),base:read('tests/e2e/playwright.config.ts')}),true));
test('注释中的预算不能代替真实job预算',()=>{const s=fixture();assert.throws(()=>validatePageBeautyCI({...s,workflow:s.workflow.replace('timeout-minutes: 30','# timeout-minutes: 30\ntimeout-minutes: 90')}),/budget/)});
