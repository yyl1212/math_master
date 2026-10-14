import {expect, type Page, type TestInfo} from '../../frontend/node_modules/@playwright/test/index.js';
import {mkdir, readFile, writeFile} from 'node:fs/promises';
import {join} from 'node:path';
import {gzipSync} from 'node:zlib';
import {reviewLogin, reviewKid} from './study-review-helpers';
import {report, handleReport, actor} from './feedback-helpers';

export {reviewKid};
export async function measurePage(page:Page):Promise<{viewportWidth:number;pageWidth:number;headerHeight:number}> {
 return page.evaluate(()=>({viewportWidth:innerWidth,pageWidth:document.documentElement.scrollWidth,headerHeight:document.querySelector('header.site-header')?.getBoundingClientRect().height??0}));
}
export async function assertPageFits(page:Page){const m=await measurePage(page);expect(m.pageWidth,'页面不得横向溢出').toBeLessThanOrEqual(m.viewportWidth+1)}
export async function assertFocusVisible(page:Page){
 const visible=await page.evaluate(()=>{const e=document.activeElement;if(!(e instanceof HTMLElement)||e===document.body||e===document.documentElement)return false;const r=e.getBoundingClientRect(),x=Math.max(0,Math.min(innerWidth-1,r.left+r.width/2)),y=Math.max(0,Math.min(innerHeight-1,r.top+r.height/2));const top=document.elementFromPoint(x,y);return r.width>0&&r.height>0&&r.top>=0&&r.bottom<=innerHeight&&!!top&&(e===top||e.contains(top));});
 expect(visible,'焦点控件必须在视口内且不被遮挡').toBe(true);
}
export async function captureAudit(page:Page,info:TestInfo,label:string):Promise<void>{
 await page.evaluate(()=>document.fonts.ready);
 const metric={caseId:label,route:new URL(page.url()).pathname+new URL(page.url()).search,locale:await page.locator('html').getAttribute('lang'),project:info.project.name,...await measurePage(page),...await page.evaluate(()=>({headings:[...document.querySelectorAll('main h1')].map(e=>({text:e.textContent,size:parseFloat(getComputedStyle(e).fontSize)})),prose:[...document.querySelectorAll('main [class*="markdown"]')].map(e=>({size:parseFloat(getComputedStyle(e).fontSize),line:parseFloat(getComputedStyle(e).lineHeight)}))}))};
 const name=label+'-'+metric.locale+'-'+metric.viewportWidth;
 await info.attach(name,{body:JSON.stringify(metric,null,2),contentType:'application/json'});
 const png=await page.screenshot({fullPage:true,mask:[page.locator('input,textarea')]});
 await info.attach(name,{body:png,contentType:'image/png'});
 if(process.env.PAGE_BEAUTY_CAPTURE){
  const dir=process.env.PAGE_BEAUTY_CAPTURE;await mkdir(dir,{recursive:true});await writeFile(join(dir,name+'.json'),JSON.stringify(metric,null,2));
  if(['A01','A04','A13-compare','A18','A20'].includes(label))await writeFile(join(dir,name+'.png'),png);
  const paths=[...new Set([...((await page.content()).matchAll(/<script[^>]+src="([^"?]+)(?:\?[^" ]*)?"/g))].map(m=>m[1]).filter(p=>p.startsWith('/_next/static/')))];
  const assets=await Promise.all(paths.map(async path=>({path,gzipBytes:gzipSync(await readFile(join(process.cwd(),'.next',path.slice('/_next/'.length))),{level:9}).length})));
  await writeFile(join(dir,name+'-assets.json'),JSON.stringify({route:metric.route,assets,gzipBytes:assets.reduce((n,a)=>n+a.gzipBytes,0)},null,2));
 }
}
export async function auditPage(page:Page,info:TestInfo,label:string){
 const original=page.viewportSize()!;
 for(const locale of ['zh-CN','en']){await page.getByRole('button',{name:locale==='en'?'English':'中文',exact:true}).click();for(const width of (original.width===1280?[1280,320]:[390])){await page.setViewportSize({width,height:original.height});await assertPageFits(page);await captureAudit(page,info,label)}}
 await page.setViewportSize(original);await page.getByRole('button',{name:'English',exact:true}).click();
}
export async function loginBeauty(page:Page,admin=false){await reviewLogin(page,admin?'auth_admin':'auth_learner')}
export async function discussionBeauty(page:Page,review=false){
 await loginBeauty(page);const id=await report(page,'/feedback/new?kind=managed-knowledge&id='+reviewKid,'原创数学反馈 Original mathematical report','请补充假设条件与反例。 Original discussion.');
 await actor(page,'auth_admin');await handleReport(page,id,'waiting_details','原创回复：请说明需要展开的数学条件。 Original independent reply.');
 if(!review)await actor(page,'auth_learner');await page.goto((review?'/review/feedback/':'/feedback/')+id);await page.getByRole('button',{name:'Read discussion',exact:true}).click();await expect(page.locator('article')).toHaveCount(2);
 return id;
}
