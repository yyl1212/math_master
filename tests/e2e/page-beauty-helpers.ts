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
 const bounds=visible?null:await page.evaluate(()=>{const e=document.activeElement as HTMLElement,r=e.getBoundingClientRect();return {tag:e.tagName,top:r.top,bottom:r.bottom,height:innerHeight,scrollY,cover:document.elementFromPoint(Math.max(0,Math.min(innerWidth-1,r.left+r.width/2)),Math.max(0,Math.min(innerHeight-1,r.top+r.height/2)))?.tagName}});
 expect(visible,'焦点控件必须在视口内且不被遮挡 '+JSON.stringify(bounds)).toBe(true);
}
export async function captureAudit(page:Page,info:TestInfo,label:string):Promise<void>{
 await expect(page.locator('main h1')).toHaveCount(1);
 await expect(page.locator('main h1')).toBeVisible();
 await page.evaluate(()=>document.fonts.ready);
 const auth=await page.request.get('/api/v1/auth/context',{headers:{'X-Requested-With':'MathMaster'}});expect(auth.status()).toBe(200);const user=(await auth.json()).data.user;
 // Read dimensions, locale and content in one DOM snapshot after locale commits settle.
 const readPresentation=()=>page.evaluate(()=>({route:location.pathname+location.search,locale:document.documentElement.lang,viewportWidth:innerWidth,pageWidth:document.documentElement.scrollWidth,headerHeight:document.querySelector('header.site-header')?.getBoundingClientRect().height??0,headings:[...document.querySelectorAll('main h1')].map(e=>({text:e.textContent,size:parseFloat(getComputedStyle(e).fontSize)})),prose:[...document.querySelectorAll('main [class*="markdown"]')].map(e=>({size:parseFloat(getComputedStyle(e).fontSize),line:parseFloat(getComputedStyle(e).lineHeight)}))}));
 let presentation=await readPresentation();
 await expect.poll(async()=>{presentation=await readPresentation();return presentation.headings.length},{message:'采集必须是最终页面而不是加载态'}).toBe(1);
 const metric={role:user?.roles.includes('admin')?'admin':user?'learner':'anonymous',caseId:label,project:info.project.name,...presentation};
 const name=label+'-'+metric.locale+'-'+metric.viewportWidth;
 await info.attach(name,{body:JSON.stringify(metric,null,2),contentType:'application/json'});
 const png=await page.screenshot({fullPage:true,mask:[page.locator('input,textarea')]});
 await info.attach(name,{body:png,contentType:'image/png'});
 if(process.env.PAGE_BEAUTY_CAPTURE){
  const dir=process.env.PAGE_BEAUTY_CAPTURE;await mkdir(dir,{recursive:true});await writeFile(join(dir,name+'.json'),JSON.stringify(metric,null,2));
  await writeFile(join(dir,name+'.png'),png);
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

export function contrastRatio(first:string,second:string){
 const rgb=(s:string)=>{if(s.trim().startsWith('#')){let value=s.trim().slice(1);if(value.length===3)value=[...value].map(c=>c+c).join('');return value.match(/../g)!.map(v=>parseInt(v,16))}return s.match(/[\d.]+/g)!.slice(0,3).map(Number)};
 const luminance=(s:string)=>{const [r,g,b]=rgb(s).map(v=>{v/=255;return v<=.04045?v/12.92:((v+.055)/1.055)**2.4});return .2126*r+.7152*g+.0722*b};
 const a=luminance(first),b=luminance(second);return (Math.max(a,b)+.05)/(Math.min(a,b)+.05);
}
