import {test,expect,TEST_PASSWORD,fitsViewport} from './fixtures';
import {createHash} from 'node:crypto';
const id='k-'+createHash('sha256').update('demo-rational-fraction').digest('hex').slice(0,56);

test('map owns knowledge search and empty legacy topic queries remain readable',async({page,scene})=>{
 await scene('managed-knowledge');await page.goto('/topics/97-XX?q=');
 await expect(page.getByRole('heading',{name:'Original fixture theme 97-XX',exact:true})).toBeVisible();await expect(page.getByRole('searchbox')).toHaveCount(0);
 await page.getByRole('link',{name:'Search knowledge in this theme',exact:true}).click();
 await expect(page).toHaveURL(/mode=knowledge&topicKey=97-XX/);await expect(page.getByRole('combobox',{name:'Search target',exact:true})).toHaveValue('knowledge');
 await expect(page.getByLabel('Keyword',{exact:true})).toBeEnabled();await page.getByLabel('Keyword',{exact:true}).fill('   ');await page.getByRole('button',{name:'Search',exact:true}).click();
 await expect(page.getByRole('link',{name:'Rational fraction',exact:true})).toBeVisible();await expect(page.getByLabel('Keyword',{exact:true})).toHaveValue('');
 await page.goto('/knowledge?mode=knowledge&offset=20&q=');await expect(page.getByRole('link',{name:'Rational fraction',exact:true})).toBeVisible();await fitsViewport(page);
});

test('map keeps empty results in place for knowledge and themes and clears the filter',async({page,scene})=>{
 await scene('managed-knowledge');await page.goto('/knowledge?mode=knowledge&q=no-match-xyz');
 await expect(page.getByText('No published knowledge matches your search.',{exact:true})).toBeVisible();await expect(page.getByLabel('Keyword',{exact:true})).toHaveValue('no-match-xyz');
 await page.getByRole('combobox',{name:'Search target',exact:true}).selectOption('topics');await page.getByRole('button',{name:'Search',exact:true}).click();
 await expect(page.getByText('No themes match your search.',{exact:true})).toBeVisible();await expect(page.getByRole('link',{name:/Project other/})).toHaveCount(0);
 await page.getByRole('link',{name:'Clear filters',exact:true}).click();await expect(page.getByRole('link',{name:/Original fixture theme 97-XX/})).toBeVisible();await fitsViewport(page);
});

test('map search keeps keywords and theme selection through both languages at narrow widths',async({page,scene})=>{
 await scene('managed-knowledge');await page.goto('/knowledge?mode=knowledge&topicKey=97F40');
 await expect(page.getByLabel('Keyword',{exact:true})).toBeEnabled();await page.getByLabel('Keyword',{exact:true}).fill('有理');
 await page.getByRole('button',{name:'中文',exact:true}).click();await expect(page.getByLabel('关键词',{exact:true})).toHaveValue('有理');
 await expect(page.getByRole('combobox',{name:'检索对象',exact:true})).toHaveValue('knowledge');await expect(page.getByLabel('具体主题',{exact:true})).toHaveValue('97F40');
 await page.getByRole('button',{name:'检索',exact:true}).click();await expect(page.getByRole('link',{name:'有理分数',exact:true})).toBeVisible();
 await page.getByRole('button',{name:'English',exact:true}).click();await expect(page.getByLabel('Keyword',{exact:true})).toHaveValue('有理');await expect(page.getByLabel('Specific themes',{exact:true})).toHaveValue('97F40');
 await fitsViewport(page);await page.setViewportSize({width:320,height:844});await fitsViewport(page);
});

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
