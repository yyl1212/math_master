import {test,expect,TEST_PASSWORD,fitsViewport} from './fixtures';import {createHash} from 'node:crypto';
const id='k-'+createHash('sha256').update('demo-rational-fraction').digest('hex').slice(0,56);
test('editing waits for hydration and preserves replacement text through a locale change',async({page,scene})=>{
 await scene('managed-knowledge');await page.goto('/login');
 await page.getByLabel('Username',{exact:true}).fill('auth_admin');await page.getByLabel('Password',{exact:true}).fill(TEST_PASSWORD);
 await page.getByRole('button',{name:'Sign in',exact:true}).click();await expect(page).toHaveURL(/\/learn$/);
 let release!:()=>void;const blocked=new Promise<void>(resolve=>{release=resolve});const scripts=/\/_next\/static\/.*\.js(?:\?.*)?$/;
 await page.route(scripts,async route=>{await blocked;await route.continue()});
 try{
  await page.goto('/admin/knowledge/'+id,{waitUntil:'commit'});
  await expect(page.getByLabel('Statement',{exact:true})).toBeVisible();
  await expect(page.getByLabel('Statement',{exact:true})).toBeDisabled();
  release();await expect(page.getByLabel('Statement',{exact:true})).toBeEnabled();
  await page.getByLabel('Statement',{exact:true}).fill('水合完成后替换的完整数学正文。');
  await page.getByRole('button',{name:'中文',exact:true}).click();
  await expect(page.getByLabel('知识正文',{exact:true})).toHaveValue('水合完成后替换的完整数学正文。');
 }finally{release();await page.unroute(scripts)}
});
test('current management and private learning translate while retaining mathematical input',async({page,scene})=>{await scene('managed-knowledge');await page.goto('/login');await page.getByLabel('Username',{exact:true}).fill('auth_admin');await page.getByLabel('Password',{exact:true}).fill(TEST_PASSWORD);await page.getByRole('button',{name:'Sign in',exact:true}).click();await expect(page).toHaveURL(/\/learn$/);await page.goto('/admin/knowledge/'+id);await page.getByLabel('Statement',{exact:true}).fill('保留数学正文 x + y = y + x。');await page.getByRole('button',{name:'中文',exact:true}).click();await expect(page.getByLabel('知识正文',{exact:true})).toHaveValue('保留数学正文 x + y = y + x。');await page.getByRole('button',{name:'English',exact:true}).click();await expect(page.getByLabel('Statement',{exact:true})).toHaveValue('保留数学正文 x + y = y + x。');await page.goto('/knowledge/'+id);await page.getByLabel('Private note',{exact:true}).fill('私人数学笔记保持不变。');await page.getByRole('button',{name:'中文',exact:true}).click();await expect(page.getByLabel('私人笔记',{exact:true})).toHaveValue('私人数学笔记保持不变。');await page.getByRole('button',{name:'保存笔记',exact:true}).click();await expect(page.getByText('笔记已保存。',{exact:true})).toBeVisible();await page.goto('/learning-history');await expect(page.getByRole('heading',{name:'学习历史',exact:true})).toBeVisible();await page.getByRole('button',{name:'English',exact:true}).click();await expect(page.getByRole('heading',{name:'Learning history',exact:true})).toBeVisible();await page.goto('/knowledge');await expect(page.getByRole('heading',{name:'Knowledge map',exact:true})).toBeVisible();await fitsViewport(page)});
