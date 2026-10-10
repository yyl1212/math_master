import {test,expect,TEST_PASSWORD} from './fixtures';import {join} from 'node:path';
test('downloaded 100 point source imports through the administrator UI and replay skips every point',async({page,scene})=>{
 const root=process.env.MANAGED_IMPORT_TEST_DIR;if(!root)throw Error('MANAGED_IMPORT_TEST_DIR is required for this separate real source acceptance');
 await scene('managed-knowledge');await page.goto('/login');
 await page.getByLabel('Username',{exact:true}).fill('auth_admin');await page.getByLabel('Password',{exact:true}).fill(TEST_PASSWORD);
 await page.getByRole('button',{name:'Sign in',exact:true}).click();await expect(page).toHaveURL(/\/learn$/);
 await page.goto('/admin/knowledge');await expect(page.locator('input[type="file"]')).toBeEnabled();await page.locator('input[type="file"]').setInputFiles(join(root!,'hefferon.part-0001.json'));
 const previews:{status:number;data:{counts:Record<string,number>;items:unknown[]}}[]=[],receipts:typeof previews=[];
 // route.fetch forwards the actual request unchanged and reads the real API body before Chromium evicts it.
 await page.route('**/api/v3/admin/knowledge/imports**',async route=>{
  const response=await route.fetch();if(route.request().method()==='POST'){
   const data=await response.json();(route.request().url().endsWith('/apply')?receipts:previews).push({status:response.status(),data});
  };await route.fulfill({response});
 });
 await page.getByRole('button',{name:'Upload and publish',exact:true}).click();
 await expect(page.getByRole('heading',{name:'Import preview',exact:true})).toBeVisible();
 expect(previews[0].status).toBe(200);expect(previews[0].data.counts).toMatchObject({createdKnowledge:100,linkedTopics:134,conflicts:0,invalidItems:0});expect(previews[0].data.items).toHaveLength(100);
 await page.getByRole('button',{name:'Apply selected knowledge',exact:true}).click();
 await expect(page.getByRole('status').filter({hasText:'New knowledge: 100'})).toBeVisible();
 expect(receipts[0].status).toBe(200);expect(receipts[0].data.counts).toMatchObject({createdKnowledge:100,linkedTopics:134,conflicts:0,invalidItems:0});expect(receipts[0].data.items).toHaveLength(100);
 await page.getByRole('button',{name:'Upload and publish',exact:true}).click();
 await expect(page.getByRole('heading',{name:'Import preview',exact:true})).toBeVisible();
 expect(previews[1].status).toBe(200);expect(previews[1].data.counts).toMatchObject({createdKnowledge:0,skippedItems:100,conflicts:0,invalidItems:0});
 await page.getByRole('button',{name:'Apply selected knowledge',exact:true}).click();
 await expect(page.getByRole('status').filter({hasText:'Duplicates skipped: 100'})).toBeVisible();
 expect(receipts[1].status).toBe(200);expect(receipts[1].data.counts.skippedItems).toBe(100);
 await page.goto('/topics/15A06?q=hefferon4e.ch01.linear-system');await expect(page.getByRole('link',{name:'Linear equations, systems, and solutions',exact:true})).toBeVisible();
 await page.getByRole('button',{name:'中文',exact:true}).click();await expect(page.getByRole('link',{name:'线性方程、方程组与解',exact:true})).toBeVisible();
 await page.getByRole('link',{name:'线性方程、方程组与解',exact:true}).click();
 await expect(page.locator('article')).toContainText('A real linear equation');
});
