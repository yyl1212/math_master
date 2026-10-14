import {test,expect} from './fixtures';
import {auditPage,measurePage,assertPageFits,assertFocusVisible,reviewKid} from './page-beauty-helpers';
test('视觉工具检测横溢与焦点遮挡',async({page})=>{
 await page.goto('/login');await page.waitForLoadState('networkidle');const base=await measurePage(page);expect(base.pageWidth).toBeLessThanOrEqual(base.viewportWidth+1);
 const style=await page.addStyleTag({content:'body {min-width:2000px!important}'});await expect(assertPageFits(page)).rejects.toThrow('页面不得横向溢出');await style.evaluate(e=>e.remove());await assertPageFits(page);
 const input=page.getByLabel('Username',{exact:true});await input.click();await expect(input).toBeFocused();await input.scrollIntoViewIfNeeded();await assertFocusVisible(page);
 await page.evaluate(()=>{const e=document.createElement('div');e.id='focus-obstruction';e.style.cssText='position:fixed;inset:0;z-index:9999;background:white';document.body.append(e)});await expect(assertFocusVisible(page)).rejects.toThrow('焦点控件必须');await page.locator('#focus-obstruction').evaluate(e=>e.remove());await assertFocusVisible(page);
});
for(const [id,path] of [['A01','/'],['A02','/knowledge'],['A03','/topics/97-XX'],['A04','/knowledge/'+reviewKid],['A05','/notes'],['A07','/login'],['A08','/register']])test(id+' 公开页面双语基线',async({page,scene},info)=>{await scene('note-sharing');await page.goto(path);await expect(page.locator('main')).toBeVisible();await auditPage(page,info,id)});
test('A06 分享笔记正文基线',async({page,scene},info)=>{await scene('note-sharing');await page.goto('/notes');await page.locator('main a[href^="/notes/"]').first().click();await expect(page.locator('main h1')).toBeVisible();await auditPage(page,info,'A06')});
test('十二个退场入口保持退出语义',async({page,scene})=>{await scene('managed-knowledge');for(const path of ['/paths/demo','/practice/demo','/assessments/demo','/assessments/demo/result','/corrections/demo','/editor/questions','/editor/questions/drafts/demo','/review/questions','/review/questions/demo','/admin/question-publications','/admin/question-publications/demo','/admin/question-withdrawals']){await page.goto(path);await expect(page.locator('main')).toContainText(/This feature has been removed/);await assertPageFits(page)}});
test('公开旧入口保持跳转',async({page,scene})=>{await scene('managed-knowledge');await page.goto('/domains/demo');await expect(page).toHaveURL(/\/knowledge$/)});
