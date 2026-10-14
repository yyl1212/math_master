import {test,expect,TEST_PASSWORD} from './fixtures';
import {auditPage,measurePage,assertPageFits,assertFocusVisible,reviewKid} from './page-beauty-helpers';
import {loginBeauty} from './page-beauty-helpers';
test('390页头高度、控件尺寸和键盘焦点符合阅读尺度',async({page,scene})=>{
 await scene('managed-knowledge');await page.setViewportSize({width:390,height:844});
 for(const role of ['anonymous','learner','admin']){
  if(role!=='anonymous'){await page.context().clearCookies();await loginBeauty(page,role==='admin')}
  await page.goto('/login');await page.waitForLoadState('networkidle');
  for(const language of ['中文','English']){await page.getByRole('button',{name:language,exact:true}).click();expect((await measurePage(page)).headerHeight,role+' '+language).toBeLessThanOrEqual(184);await assertPageFits(page)}
 }
 await page.context().clearCookies();await page.goto('/login');await page.waitForLoadState('networkidle');
 for(const selector of ['.language-switch button','.site-header nav a','.ui-form input']){const sizes=await page.locator(selector).evaluateAll(es=>es.map(e=>({height:e.getBoundingClientRect().height,size:parseFloat(getComputedStyle(e).fontSize)})));expect(sizes.length).toBeGreaterThan(0);for(const m of sizes){expect(m.height).toBeGreaterThanOrEqual(selector.includes('input')?48:44);expect(m.size).toBeGreaterThanOrEqual(13)}}
 await page.keyboard.press('Tab');await expect(page.getByRole('link',{name:/Skip/})).toBeFocused();await assertFocusVisible(page);
});
test('实际次要文字及控件边界达到对比要求',async({page})=>{
 await page.goto('/login');const colors=await page.evaluate(()=>{const body=getComputedStyle(document.body),root=getComputedStyle(document.documentElement),input=getComputedStyle(document.querySelector('input')!);return {muted:root.getPropertyValue('--muted'),paper:body.backgroundColor,border:input.borderTopColor,surface:input.backgroundColor}});
 function rgb(s:string){if(s.trim().startsWith('#')){const x=s.trim().slice(1);return x.match(/../g)!.map(v=>parseInt(v,16))}return s.match(/[\d.]+/g)!.slice(0,3).map(Number)}
 function luminance(s:string){const [r,g,b]=rgb(s).map(v=>{v/=255;return v<=.04045?v/12.92:((v+.055)/1.055)**2.4});return .2126*r+.7152*g+.0722*b}
 function contrast(a:string,b:string){const x=luminance(a),y=luminance(b);return (Math.max(x,y)+.05)/(Math.min(x,y)+.05)}
 expect(contrast(colors.muted,colors.paper)).toBeGreaterThanOrEqual(4.5);expect(contrast(colors.border,colors.surface)).toBeGreaterThanOrEqual(3);
});
test('长用户名和展开的管理员导航在320与768宽均可达',async({page,scene},info)=>{
 await scene('managed-knowledge');await page.goto('/register');await page.getByLabel('Username',{exact:true}).fill('original_long_username_123456789');await page.getByLabel('Password',{exact:true}).fill(TEST_PASSWORD);await page.getByRole('button',{name:'Create account',exact:true}).click();await expect(page).toHaveURL(/\/login$/);await page.getByLabel('Username',{exact:true}).fill('original_long_username_123456789');await page.getByLabel('Password',{exact:true}).fill(TEST_PASSWORD);await page.getByRole('button',{name:'Sign in',exact:true}).click();await expect(page).toHaveURL(/\/learn$/);
 for(const width of [320,768]){await page.setViewportSize({width,height:900});await assertPageFits(page);await expect(page.locator('header a[href="/account"]')).toContainText('original_long_username_123456789')}
 await page.context().clearCookies();await loginBeauty(page,true);await page.setViewportSize({width:320,height:900});await page.getByRole('button',{name:'English',exact:true}).click();await page.locator('header summary').click();for(const href of ['/admin/knowledge','/admin/note-shares','/review/feedback','/admin/users']){const link=page.locator('header a[href="'+href+'"]');await link.focus();await link.scrollIntoViewIfNeeded();await assertFocusVisible(page)}await assertPageFits(page);
 await auditPage(page,info,'G-header-expanded');
});
test('视觉工具检测横溢与焦点遮挡',async({page})=>{
 await page.goto('/login');await page.waitForLoadState('networkidle');const base=await measurePage(page);expect(base.pageWidth).toBeLessThanOrEqual(base.viewportWidth+1);
 const style=await page.addStyleTag({content:'body {min-width:2000px!important}'});await expect(assertPageFits(page)).rejects.toThrow('页面不得横向溢出');await style.evaluate(e=>e.parentNode?.removeChild(e));await assertPageFits(page);
 const input=page.getByLabel('Username',{exact:true});await input.click();await expect(input).toBeFocused();await input.scrollIntoViewIfNeeded();await assertFocusVisible(page);
 await page.evaluate(()=>{const e=document.createElement('div');e.id='focus-obstruction';e.style.cssText='position:fixed;inset:0;z-index:9999;background:white';document.body.append(e)});await expect(assertFocusVisible(page)).rejects.toThrow('焦点控件必须');await page.locator('#focus-obstruction').evaluate(e=>e.remove());await assertFocusVisible(page);
});
for(const [id,path] of [['A01','/'],['A02','/knowledge'],['A03','/topics/97-XX'],['A04','/knowledge/'+reviewKid],['A05','/notes'],['A07','/login'],['A08','/register']])test(id+' 公开页面双语基线',async({page,scene},info)=>{await scene('note-sharing');await page.goto(path);await expect(page.locator('main')).toBeVisible();await auditPage(page,info,id)});
test('A06 分享笔记正文基线',async({page,scene},info)=>{await scene('note-sharing');await page.goto('/notes');await page.locator('main a[href^="/notes/"]').first().click();await expect(page.locator('main h1')).toBeVisible();await auditPage(page,info,'A06')});
test('十二个退场入口保持退出语义',async({page,scene})=>{await scene('managed-knowledge');for(const path of ['/paths/demo','/practice/demo','/assessments/demo','/assessments/demo/result','/corrections/demo','/editor/questions','/editor/questions/drafts/demo','/review/questions','/review/questions/demo','/admin/question-publications','/admin/question-publications/demo','/admin/question-withdrawals']){await page.goto(path);await expect(page.locator('main')).toContainText(/This feature has been removed/);await assertPageFits(page)}});
test('公开旧入口保持跳转',async({page,scene})=>{await scene('managed-knowledge');await page.goto('/domains/demo');await expect(page).toHaveURL(/\/knowledge$/)});
