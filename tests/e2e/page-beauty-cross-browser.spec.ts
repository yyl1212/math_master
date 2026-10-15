import {test,expect} from './fixtures';
import {reviewKid,loginBeauty,assertPageFits,captureAudit,assertFocusVisible} from './page-beauty-helpers';
import {startReviewAPI} from './study-review-helpers';
test('核心首页与数学正文在真实引擎中可阅读',async({page,scene},info)=>{
 await scene('managed-knowledge');await page.goto('/');await expect(page.locator('main h1')).toBeVisible();await assertPageFits(page);await captureAudit(page,info,'cross-'+info.project.name+'-home');
 await page.goto('/knowledge/'+reviewKid);for(const width of [page.viewportSize()!.width,320]){await page.setViewportSize({width,height:900});const body=page.locator('article [class*="markdown"]').first();await expect(body).toBeVisible();expect(await body.evaluate(e=>parseFloat(getComputedStyle(e).fontSize))).toBe(width<=600?16:17);await assertPageFits(page)}
});
test('笔记输入与长公式滚动保留安全和键盘入口',async({page,scene},info)=>{
 await scene('managed-knowledge');await loginBeauty(page);await page.goto('/knowledge/'+reviewKid+'?mode=notes');const note=page.getByLabel('Private note',{exact:true});await note.fill('原创笔记\n\n$$\n'+Array.from({length:50},(_,i)=>'x_{'+i+'}').join('+')+'\n$$');await page.getByRole('button',{name:'中文',exact:true}).click();await page.getByRole('button',{name:'English',exact:true}).click();await expect(note).toHaveValue(/原创笔记/);await page.getByRole('button',{name:'Preview',exact:true}).click();const formula=page.getByRole('region',{name:'Scrollable formula',exact:true});await expect(formula).toHaveAttribute('tabindex','0');await formula.focus();await formula.scrollIntoViewIfNeeded();await formula.press('ArrowRight');await expect.poll(()=>formula.evaluate(e=>e.scrollLeft)).toBeGreaterThan(0);await assertFocusVisible(page);await assertPageFits(page);await captureAudit(page,info,'cross-'+info.project.name+'-notes');
});
test('默写比对按原揭示条件冻结并保留自评',async({page,scene},info)=>{
 await scene('study-review');await loginBeauty(page);const r=await startReviewAPI(page);await page.goto('/learn/review/'+r.sessionId);await expect(page.getByRole('region',{name:'Reference content',exact:true})).toHaveCount(0);const answer=page.getByRole('textbox',{name:'My answer',exact:true});await answer.fill('原创作答 $b\\ne0$');await page.getByRole('button',{name:'Answer finished; show reference',exact:true}).click();await expect(page.getByRole('button',{name:'Familiar',exact:true})).toBeVisible();await expect(answer).toHaveCount(0);if(page.viewportSize()!.width<=800)await page.getByRole('button',{name:'Reference content',exact:true}).click();await expect(page.getByRole('region',{name:'Reference content',exact:true})).toBeVisible();await assertPageFits(page);await captureAudit(page,info,'cross-'+info.project.name+'-compare');
});
test('管理表格、表单与对话框焦点在真实引擎中可用',async({page,scene},info)=>{
 await scene('managed-knowledge');await loginBeauty(page,true);await page.setViewportSize({width:320,height:844});await page.goto('/admin/knowledge');const region=page.getByRole('region',{name:'Knowledge management',exact:true});await expect(page.getByRole('columnheader')).toHaveCount(7);await region.focus();await region.scrollIntoViewIfNeeded();await region.press('ArrowRight');await expect.poll(()=>region.evaluate(e=>e.scrollLeft)).toBeGreaterThan(0);await assertPageFits(page);
 await page.goto('/admin/knowledge/'+reviewKid);await expect(page.getByLabel('Knowledge title',{exact:true})).toBeEnabled();expect(await page.locator('.managed-fields textarea').count()).toBeGreaterThan(10);await assertPageFits(page);
 await page.goto('/admin/users');await page.getByRole('button',{name:/^auth_learner\s/}).click();await page.getByLabel('admin',{exact:true}).check();await page.getByLabel('Reason',{exact:true}).fill('Original cross-browser role request. 原创测试理由。');const submit=page.getByRole('button',{name:'Save roles',exact:true});await submit.evaluate(e=>(e as HTMLButtonElement).form!.requestSubmit(e as HTMLButtonElement));const dialog=page.getByRole('dialog');await expect(dialog.getByLabel('Your password',{exact:true})).toBeFocused();await captureAudit(page,info,'cross-'+info.project.name+'-dialog');await dialog.getByRole('button',{name:'Cancel',exact:true}).click();await expect(submit).toBeFocused();await assertFocusVisible(page);
});

test('窄屏宽字体下原生主题选框保留完整选项和值且不横溢',async({page,scene})=>{await scene('managed-knowledge');await loginBeauty(page,true);await page.setViewportSize({width:320,height:844});await page.goto('/admin/knowledge');await page.waitForLoadState('networkidle');await page.evaluate(()=>document.body.style.fontFamily='monospace');const primary=page.getByRole('combobox',{name:'Main topics',exact:true}),secondary=page.getByRole('combobox',{name:'Subtopics',exact:true}),specific=page.getByRole('combobox',{name:'Specific themes',exact:true});await expect(specific.locator('option:checked')).toHaveText('Choose a secondary theme first');await assertPageFits(page);await primary.selectOption('97-XX');await expect(primary).toHaveValue('97-XX');await expect(secondary).toBeEnabled();await expect(specific).toBeDisabled();for(const select of [primary,secondary,specific]){const rect=await select.boundingBox();expect(rect).not.toBeNull();expect(rect!.x+rect!.width).toBeLessThanOrEqual(320);expect(rect!.height).toBeGreaterThanOrEqual(48)}await primary.focus();await assertFocusVisible(page);await assertPageFits(page)});

for(const locale of ['zh-CN','en'])test('真实引擎顶部导航跨页稳定 '+locale,async({page,scene})=>{
 await scene('study-review');await loginBeauty(page);await page.getByRole('button',{name:locale==='en'?'English':'中文',exact:true}).click();
 const hrefs=['/learn','/learn?view=notes','/learn/review','/learn/review/history','/learn?view=history'];
 for(const width of [page.viewportSize()!.width,320]){
  await page.setViewportSize({width,height:900});let reference:number[][]|undefined;
  for(const href of hrefs){
   const nav=page.getByRole('navigation',{name:locale==='en'?'My learning sections':'我的学习栏目',exact:true});
   await nav.locator(`a[href="${href}"]`).click();await expect(page).toHaveURL('http://127.0.0.1:18080'+href);
   await expect(nav.locator('[aria-current="page"]')).toHaveCount(1);await expect(nav.locator('[aria-current="page"]')).toHaveAttribute('href',href);
   expect(await nav.locator('a').evaluateAll(es=>es.map(e=>e.getAttribute('href')))).toEqual(hrefs);await page.evaluate(()=>document.fonts.ready);
   const rects=await nav.locator('a').evaluateAll(es=>es.map(e=>{const r=e.getBoundingClientRect(),p=e.parentElement!.getBoundingClientRect();return [r.x,r.y-p.y,r.width,r.height]}));
   for(const rect of rects)expect(rect[3]).toBeGreaterThanOrEqual(44);
   if(reference)rects.forEach((rect,i)=>rect.forEach((n,j)=>expect(Math.abs(n-reference![i][j])).toBeLessThanOrEqual(1)));else reference=rects;
   await assertPageFits(page);
  }
 }
});
