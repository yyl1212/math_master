import {defineConfig} from '../../frontend/node_modules/@playwright/test/index.js';
import base from './playwright.config';
if(process.env.MATH_MASTER_E2E_FRONTEND)throw new Error('Current frontend only');
export default defineConfig({
 ...base,
 testMatch:'**/page-beauty-cross-browser.spec.ts',
 projects:[
  {name:'firefox-desktop',use:{browserName:'firefox',viewport:{width:1280,height:900}}},
  {name:'firefox-mobile',use:{browserName:'firefox',viewport:{width:390,height:844}}},
  {name:'webkit-desktop',use:{browserName:'webkit',viewport:{width:1280,height:900}}},
  {name:'webkit-mobile',use:{browserName:'webkit',viewport:{width:390,height:844}}}
 ]
});
