import {test,expect,fitsViewport} from './fixtures';
test('query pagination across current pages and mobile keyboard jump',async({page,scene})=>{
 await scene('query-pagination');await page.goto('/knowledge?mode=knowledge');
 await expect(page.getByText('Page 1 / 3 · 41 results',{exact:true})).toBeVisible();
 await page.getByRole('link',{name:'Last page',exact:true}).click();await expect(page).toHaveURL(/page=3/);
 await expect(page.getByText('Page 3 / 3 · 41 results',{exact:true})).toBeVisible();
 await expect(page.locator('main article')).toHaveCount(1);
 await page.goBack();await expect(page.getByText('Page 1 / 3 · 41 results',{exact:true})).toBeVisible();
 const input=page.getByRole('textbox',{name:'Page number',exact:true});await input.fill('2');await input.press('Enter');
 await expect(page.getByText('Page 2 / 3 · 41 results',{exact:true})).toBeVisible();await page.reload();await expect(page.locator('main article')).toHaveCount(20);
 await page.setViewportSize({width:320,height:844});await fitsViewport(page);await expect(page.getByRole('link',{name:'First page',exact:true})).toBeVisible();
 await input.fill('99');await input.press('Enter');await expect(page.locator('main').getByRole('alert')).toBeVisible();await expect(page).toHaveURL(/page=2/);
 await page.goto('/knowledge?mode=knowledge&q=no-such-unique-fixture');await expect(page.getByText('Page 0 / 0 · 0 results',{exact:true})).toBeVisible();
});
test('guest boundary and retired routes remain unchanged',async({page,scene})=>{
 await scene('query-pagination');await page.goto('/notes');await expect(page.getByRole('textbox',{name:'Page number',exact:true})).toHaveCount(0);await expect(page.getByRole('link',{name:'Sign in',exact:true}).first()).toBeVisible();
 await page.goto('/notifications');await expect(page).toHaveURL(/\/learn$/);
});
