import {test,expect,TEST_PASSWORD,fitsViewport} from './fixtures';
import {createHash} from 'node:crypto';
const id='k-'+createHash('sha256').update('demo-rational-fraction').digest('hex').slice(0,56);

test('production homepage, structured editing and retired entry translate without changing input',async({page,scene})=>{
 await scene('managed-knowledge');await page.goto('/');
 await expect(page.getByRole('heading',{name:'Understand mathematics, one idea at a time'})).toBeVisible();
 await page.getByRole('button',{name:'中文',exact:true}).click();
 await expect(page.getByRole('link',{name:'开始学习',exact:true})).toHaveAttribute('href','/login');
 await fitsViewport(page);await page.getByRole('button',{name:'English',exact:true}).click();
 await page.goto('/login');await page.getByLabel('Username',{exact:true}).fill('auth_admin');
 await page.getByLabel('Password',{exact:true}).fill(TEST_PASSWORD);await page.getByRole('button',{name:'Sign in',exact:true}).click();
 await expect(page).toHaveURL(/\/learn$/);await page.goto('/admin/knowledge/'+id);
 await page.getByLabel('Source title',{exact:true}).first().fill('原始来源保持不变');
 await page.getByLabel('Explanation text',{exact:true}).first().fill('数学解释 x + y 保持不变');
 await expect(page.getByLabel('Main topics',{exact:true})).toBeVisible();
 await page.getByRole('button',{name:'中文',exact:true}).click();
 await expect(page.getByLabel('来源标题',{exact:true}).first()).toHaveValue('原始来源保持不变');
 await expect(page.getByLabel('解释内容',{exact:true}).first()).toHaveValue('数学解释 x + y 保持不变');
 await expect(page.getByLabel('一级主题',{exact:true})).toBeVisible();await fitsViewport(page);
 await page.goto('/paths/removed');await expect(page.getByRole('heading',{name:'此功能已移除'})).toBeVisible();
 await page.getByRole('button',{name:'English',exact:true}).click();
 await expect(page.getByRole('heading',{name:'This feature has been removed'})).toBeVisible();await fitsViewport(page);
});
