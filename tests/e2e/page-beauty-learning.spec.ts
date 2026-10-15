import {test,expect} from './fixtures';
import {auditPage,loginBeauty,discussionBeauty,reviewKid,assertPageFits,assertFocusVisible,captureAudit,contrastRatio} from './page-beauty-helpers';
import {startReviewAPI,proofKid} from './study-review-helpers';
for(const [id,path] of [['A09','/learn'],['A10','/learn?view=notes'],['A11','/learn?view=history'],['A12','/learn/review'],['A14','/learn/review/history'],['A15','/account'],['A16','/feedback'],['A17','/feedback/new?kind=managed-knowledge&id='+reviewKid]])test(id+' 本人工作区双语基线',async({page,scene},info)=>{await scene(id==='A10'?'note-sharing':'study-review');await loginBeauty(page);await page.goto(path);await expect(page.locator('main h1')).toBeVisible();await auditPage(page,info,id)});
test('A13 作答、冻结比对与自评基线',async({page,scene},info)=>{await scene('study-review');await loginBeauty(page);const receipt=await startReviewAPI(page);await page.goto('/learn/review/'+receipt.sessionId);await expect(page.getByRole('textbox',{name:'My answer',exact:true})).toBeVisible();await auditPage(page,info,'A13-edit');await page.getByRole('textbox',{name:'My answer',exact:true}).fill('原创回忆：$b\\ne0$ 时，$\\frac{a}{b}$ 是有理分式。');await page.getByRole('button',{name:'Answer finished; show reference',exact:true}).click();await expect(page.getByRole('button',{name:'Familiar',exact:true})).toBeVisible();await auditPage(page,info,'A13-compare')});
test('A18 本人反馈真实讨论基线',async({page,scene},info)=>{await scene('managed-knowledge');await discussionBeauty(page);await auditPage(page,info,'A18')});
test('两个本人旧入口保留跳转',async({page,scene})=>{await scene('study-review');await loginBeauty(page);await page.goto('/learning-history');await expect(page).toHaveURL(/\/learn\?view=history/);await page.goto('/notifications');await expect(page).toHaveURL(/\/learn$/)});
for(const mode of ['read','proof'])test('A13 '+mode+' 展示基线',async({page,scene},info)=>{await scene('study-review');await loginBeauty(page);const receipt=await startReviewAPI(page,mode==='proof'?proofKid:reviewKid,mode==='read'?'read':'recall',mode==='proof'?'proof':'statement');await page.goto('/learn/review/'+receipt.sessionId);await expect(page.locator('main h1')).toBeVisible();await auditPage(page,info,'A13-'+mode)});
test('当前正文在桌面17px、手机16px且保持舒适行距',async({page,scene})=>{await scene('managed-knowledge');await page.goto('/knowledge/'+reviewKid);for(const width of [1280,390]){await page.setViewportSize({width,height:900});const prose=page.locator('article [class*="markdown"]').first();const m=await prose.evaluate(e=>({size:parseFloat(getComputedStyle(e).fontSize),line:parseFloat(getComputedStyle(e).lineHeight)}));expect(m.size).toBe(width===1280?17:16);expect(m.line/m.size).toBeCloseTo(width===1280?1.9:1.85,2);await assertPageFits(page)}});
test('笔记长公式局部滚动可命名聚焦，短公式不额外占Tab',async({page,scene},info)=>{
 await scene('managed-knowledge');await loginBeauty(page);await page.goto('/knowledge/'+reviewKid+'?mode=notes');await page.setViewportSize({width:390,height:844});const input=page.getByLabel('Private note',{exact:true});await input.fill('$$\nx+1\n$$\n\n$$\n'+Array.from({length:60},(_,i)=>'x_{'+i+'}').join('+')+'\n$$');await page.getByRole('button',{name:'Preview',exact:true}).click();
 const formula=page.locator('[class*="preview"] .katex-display');await expect(formula).toHaveCount(2);await expect(formula.first()).not.toHaveAttribute('tabindex','0');await expect(formula.last()).toHaveAttribute('tabindex','0');await expect(formula.last()).toHaveAccessibleName('Scrollable formula');await formula.last().focus();await formula.last().scrollIntoViewIfNeeded();await assertFocusVisible(page);await formula.last().press('ArrowRight');await expect.poll(()=>formula.last().evaluate(e=>e.scrollLeft)).toBeGreaterThan(0);await assertPageFits(page);await captureAudit(page,info,'G-long-formula');
});
test('笔记草稿跨语言和1099/1100断点保留同一输入与焦点',async({page,scene})=>{
 await scene('managed-knowledge');await loginBeauty(page);await page.goto('/knowledge/'+reviewKid+'?mode=notes');const input=page.getByLabel('Private note',{exact:true});await input.fill('原创未保存笔记 $x^2$');await input.evaluate(e=>e.setAttribute('data-retained','yes'));await page.getByRole('button',{name:'中文',exact:true}).click();await expect(page.getByRole('button',{name:'中文',exact:true})).toBeFocused();await page.getByRole('button',{name:'English',exact:true}).click();await expect(input).toHaveValue('原创未保存笔记 $x^2$');
 for(const width of [1099,1100,320]){await input.focus();await page.setViewportSize({width,height:900});await expect(input).toHaveAttribute('data-retained','yes');await expect(input).toBeFocused();await expect(input).toHaveValue('原创未保存笔记 $x^2$');await assertPageFits(page)}page.once('dialog',dialog=>dialog.dismiss());await page.getByRole('link',{name:'Math Master home',exact:true}).click();await expect(input).toHaveValue('原创未保存笔记 $x^2$');
});
test('长行内公式也在自身横滚而不撑开笔记页面',async({page,scene})=>{await scene('managed-knowledge');await loginBeauty(page);await page.goto('/knowledge/'+reviewKid+'?mode=notes');await page.setViewportSize({width:320,height:844});await page.getByLabel('Private note',{exact:true}).fill('Original $'+Array.from({length:60},(_,i)=>'a_{'+i+'}').join('+')+'$');await page.getByRole('button',{name:'Preview',exact:true}).click();await assertPageFits(page);await expect(page.locator('[class*="preview"] .katex')).toHaveAttribute('tabindex','0')});
test('复习工作区本地导航目标44px，五入口和四学习状态保持',async({page,scene})=>{await scene('study-review');await loginBeauty(page);await page.goto('/learn/review');const nav=page.locator('main nav').first();await expect(nav.locator('a')).toHaveCount(5);for(const height of await nav.locator('a').evaluateAll(es=>es.map(e=>e.getBoundingClientRect().height)))expect(height).toBeGreaterThanOrEqual(44);const hrefs=await page.locator('main nav').nth(1).locator('a').evaluateAll(es=>es.map(e=>new URL((e as HTMLAnchorElement).href).searchParams.get('bucket')));expect(hrefs).toEqual(['due','unplanned','unfinished','all','future']);for(const width of [1280,320]){await page.setViewportSize({width,height:900});expect(await page.locator('main h1').evaluate(e=>parseFloat(getComputedStyle(e).fontSize))).toBeLessThanOrEqual(width===320?32:40);await assertPageFits(page)}await page.goto('/learn');await expect(page.locator('main dl > [data-state]')).toHaveCount(4);await assertPageFits(page)});
test('比对800与801断点保留合法揭示和原自评条件',async({page,scene},info)=>{await scene('study-review');await loginBeauty(page);const r=await startReviewAPI(page);await page.goto('/learn/review/'+r.sessionId);await expect(page.getByRole('region',{name:'Reference content',exact:true})).toHaveCount(0);await page.getByRole('textbox',{name:'My answer',exact:true}).fill('自己的答案 $b\\ne0$');await page.getByRole('button',{name:'Answer finished; show reference',exact:true}).click();await expect(page.getByRole('button',{name:'Familiar',exact:true})).toBeVisible();await expect(page.getByRole('textbox',{name:'My answer',exact:true})).toHaveCount(0);for(const width of [801,800]){await page.setViewportSize({width,height:900});const reference=page.getByRole('region',{name:'Reference content',exact:true});if(width===801)await expect(reference).toBeVisible();else{await expect(reference).not.toBeVisible();await page.getByRole('button',{name:'Reference content',exact:true}).click();await expect(reference).toBeVisible()}await assertPageFits(page);await captureAudit(page,info,'G-compare-'+width)}});
test('未选阅读标签的实际控件边界仍达到3:1',async({page,scene})=>{await scene('managed-knowledge');await loginBeauty(page);await page.setViewportSize({width:390,height:844});await page.goto('/knowledge/'+reviewKid);const tab=page.getByRole('button',{name:'My notes',exact:true});await expect(tab).toBeVisible();const c=await tab.evaluate(e=>{const s=getComputedStyle(e);return {border:s.borderTopColor,background:s.backgroundColor}});expect(contrastRatio(c.border,c.background)).toBeGreaterThanOrEqual(3)});
test('当前阅读与私人笔记双栏及窄屏状态留存证据',async({page,scene},info)=>{await scene('managed-knowledge');await loginBeauty(page);await page.goto('/knowledge/'+reviewKid+'?mode=notes');for(const width of [1280,390,768,1440]){await page.setViewportSize({width,height:900});await expect(page.getByLabel('Private note',{exact:true})).toBeVisible();await assertPageFits(page);await captureAudit(page,info,'G-reading-notes')} });

for(const locale of ['zh-CN','en'])test('顶部学习导航跨五页固定顺序和位置 '+locale,async({page,scene},info)=>{
 await scene('study-review');await loginBeauty(page);
 await page.getByRole('button',{name:locale==='en'?'English':'中文',exact:true}).click();
 const hrefs=['/learn','/learn?view=notes','/learn/review','/learn/review/history','/learn?view=history'];
 const labels=locale==='en'?['Knowledge learning','My notes','Review workspace','Self-rating history','Learning history']:['知识学习','我的笔记','复习工作区','自评历史','学习历史'];
 for(const width of info.project.name==='desktop'?[1280,320]:[390]){
  await page.setViewportSize({width,height:900});
  let reference:number[][]|undefined;
  for(const [index,href] of hrefs.entries()){
   const nav=page.getByRole('navigation',{name:locale==='en'?'My learning sections':'我的学习栏目',exact:true});
   await nav.getByRole('link',{name:labels[index],exact:true}).click();
   await expect(page).toHaveURL('http://127.0.0.1:18080'+href);
   const links=nav.getByRole('link');await expect(links).toHaveCount(5);
   await expect(nav.locator('[aria-current="page"]')).toHaveCount(1);
   await expect(nav.locator('[aria-current="page"]')).toHaveAttribute('href',href);
   expect(await links.evaluateAll(es=>es.map(e=>e.getAttribute('href')))).toEqual(hrefs);
   await expect(links).toHaveText(labels);await page.evaluate(()=>document.fonts.ready);
   const rects=await links.evaluateAll(es=>es.map(e=>{const r=e.getBoundingClientRect(),parent=e.parentElement!.getBoundingClientRect();return [r.x,r.y-parent.y,r.width,r.height]}));
   for(const rect of rects){expect(rect[3]).toBeGreaterThanOrEqual(44);expect(rect[0]).toBeGreaterThanOrEqual(0);expect(rect[0]+rect[2]).toBeLessThanOrEqual(width)}
   if(reference)rects.forEach((rect,i)=>rect.forEach((n,j)=>expect(Math.abs(n-reference![i][j]),'切换栏目后按钮位置和尺寸保持').toBeLessThanOrEqual(1)));else reference=rects;
   await links.first().focus();for(let i=1;i<5;i++){await page.keyboard.press('Tab');await expect(links.nth(i)).toBeFocused()}
   await assertPageFits(page);
   if(width!==320&&(index===1||index===2))await captureAudit(page,info,'navigation-fixed-'+index);
  }
 }
});

test('强制颜色模式保留当前栏目非颜色提示且切页不移位',async({page,scene})=>{
 await scene('study-review');await loginBeauty(page);await page.emulateMedia({forcedColors:'active'});
 expect(await page.evaluate(()=>matchMedia('(forced-colors: active)').matches)).toBe(true);
 let reference:number[][]|undefined;
 for(const href of ['/learn','/learn?view=notes','/learn/review','/learn/review/history','/learn?view=history']){
  const nav=page.getByRole('navigation',{name:'My learning sections',exact:true});
  await nav.locator(`a[href="${href}"]`).click();await expect(page).toHaveURL('http://127.0.0.1:18080'+href);
  const current=nav.locator('[aria-current="page"]');await expect(current).toHaveCount(1);
  await expect(current).toHaveAttribute('href',href);await expect(current).toBeVisible();
  expect(await current.evaluate(e=>getComputedStyle(e).textDecorationLine)).toContain('underline');
  for(const marker of await nav.locator('a:not([aria-current])').evaluateAll(es=>es.map(e=>getComputedStyle(e).textDecorationLine)))expect(marker).not.toContain('underline');
  const rects=await nav.locator('a').evaluateAll(es=>es.map(e=>{const r=e.getBoundingClientRect(),p=e.parentElement!.getBoundingClientRect();return [r.x,r.y-p.y,r.width,r.height]}));
  if(reference)rects.forEach((rect,i)=>rect.forEach((n,j)=>expect(Math.abs(n-reference![i][j])).toBeLessThanOrEqual(1)));else reference=rects;
  await assertPageFits(page);
 }
});
