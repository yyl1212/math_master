import {test,expect,TEST_PASSWORD,fitsViewport} from './fixtures';
import type {Page,Locator} from '../../frontend/node_modules/@playwright/test/index.js';
import {createHash} from 'node:crypto';
const id='k-'+createHash('sha256').update('demo-rational-fraction').digest('hex').slice(0,56);
const login=async(page:Page)=>{await page.goto('/login');await page.getByLabel('Username',{exact:true}).fill('auth_admin');await page.getByLabel('Password',{exact:true}).fill(TEST_PASSWORD);await page.getByRole('button',{name:'Sign in',exact:true}).click();await expect(page).toHaveURL(/\/learn$/)};
async function control(field:Locator){
 await expect(field).toBeVisible();
 const shape=await field.evaluate(node=>{const css=getComputedStyle(node);return {height:node.getBoundingClientRect().height,radius:css.borderRadius,color:css.color,border:css.borderTopColor,box:css.boxSizing,arrow:node.tagName==='SELECT'?css.backgroundImage:undefined}});
 expect(shape.height).toBe(48);expect(shape.radius).toBe('10px');expect(shape.color).toBe('rgb(32, 60, 46)');expect(shape.border).toBe('rgb(222, 223, 212)');expect(shape.box).toBe('border-box');
 if(shape.arrow!==undefined)expect(shape.arrow).not.toBe('none');
}
async function narrow(page:Page){await fitsViewport(page);await page.setViewportSize({width:320,height:844});await fitsViewport(page)};

test('history uses equal controls and explicit dependent-theme prompts without losing filters',async({page,scene},info)=>{
 await scene('managed-knowledge');await login(page);await page.goto('/learning-history');
 const primary=page.getByLabel('Main topics',{exact:true}),secondary=page.getByLabel('Subtopics',{exact:true}),leaf=page.getByLabel('Specific themes',{exact:true}),action=page.getByRole('combobox',{name:'Action',exact:true});
 await control(primary);await control(action);await expect(secondary).toBeDisabled();await expect(secondary.locator('option:checked')).toHaveText('Choose a main topic first');await expect(leaf.locator('option:checked')).toHaveText('Choose a secondary theme first');
 await action.focus();await expect(action).toBeFocused();await page.keyboard.press('s');await expect(action).toHaveValue('started');await page.keyboard.press('Tab');
 await expect(primary.locator('option[value="97-XX"]')).toHaveCount(1);await primary.selectOption('97-XX');await action.selectOption('completed');
 await page.getByRole('button',{name:'中文',exact:true}).click();await expect(page.getByRole('combobox',{name:'操作',exact:true})).toHaveValue('completed');await expect(page.getByLabel('一级主题',{exact:true})).toHaveValue('97-XX');
 await page.getByRole('button',{name:'检索知识点',exact:true}).click();await expect(page).toHaveURL(/kind=completed/);await expect(page).toHaveURL(/topicKey=97-XX/);await expect(page.getByRole('combobox',{name:'操作',exact:true})).toHaveValue('completed');if(!process.env.CI)await page.screenshot({path:info.outputPath('history.png'),fullPage:true});await narrow(page);
});

test('map and learning share controls and keep query and selection through localization',async({page,scene})=>{
 await scene('managed-knowledge');let release!:()=>void;const delayed=new Promise<void>(resolve=>{release=resolve});await page.route('**/api/v3/topics?*',async route=>{if(new URL(route.request().url()).searchParams.get('topicKey')==='97Fxx')await delayed;await route.continue()});
 try{await page.goto('/knowledge?mode=knowledge&topicKey=97F40');
 await expect(page.getByLabel('Keyword',{exact:true})).toBeEnabled();await control(page.getByRole('combobox',{name:'Search target',exact:true}));await control(page.getByLabel('Keyword',{exact:true}));await control(page.getByLabel('Specific themes',{exact:true}));
 await page.getByLabel('Keyword',{exact:true}).fill('有理');await page.getByRole('button',{name:'中文',exact:true}).click();await expect(page.getByLabel('关键词',{exact:true})).toHaveValue('有理');await expect(page.getByLabel('具体主题',{exact:true})).toHaveValue('97F40');await fitsViewport(page);
 }finally{release()}
 await page.getByRole('button',{name:'English',exact:true}).click();await login(page);await control(page.getByRole('combobox',{name:'Learning status',exact:true}));await control(page.getByLabel('Search knowledge',{exact:true}));await narrow(page);
});

test('upload gives native file selection and checkboxes the shared appearance',async({page,scene},info)=>{
 await scene('managed-knowledge');await login(page);await page.goto('/admin/knowledge');const file=page.locator('input[type=file]');await expect(file).toBeEnabled();
 expect(await file.evaluate(node=>getComputedStyle(node,'::file-selector-button').borderRadius)).toBe('10px');
 const publish=page.getByRole('checkbox',{name:'Upload and publish',exact:true});expect(await publish.evaluate(node=>getComputedStyle(node).accentColor)).toBe('rgb(49, 88, 68)');await expect(publish).toBeChecked();
 await file.setInputFiles({name:'测试知识文件-'+ 'long-name-'.repeat(20)+'.json',mimeType:'application/json',buffer:Buffer.from('{}')});
 await page.getByRole('button',{name:'中文',exact:true}).click();await expect(page.getByRole('checkbox',{name:'上传并发布',exact:true})).toBeChecked();if(!process.env.CI)await page.screenshot({path:info.outputPath('upload.png'),fullPage:true});await narrow(page);
});

test('feedback filters align with the action and account roles remain native checkboxes',async({page,scene},info)=>{
 await scene('managed-knowledge');await login(page);await page.goto('/review/feedback');
 const status=page.getByLabel('Status',{exact:true}),category=page.getByLabel('Category',{exact:true});await control(status);await control(category);
 const submit=page.getByRole('button',{name:'Filter queue',exact:true});if(page.viewportSize()!.width>=700){const boxes=await Promise.all([status.boundingBox(),submit.boundingBox()]);expect(Math.abs(boxes[0]!.y+boxes[0]!.height-boxes[1]!.y-boxes[1]!.height)).toBeLessThanOrEqual(1);}
 await status.selectOption('processing');await category.selectOption('typo');await submit.click();await expect(page).toHaveURL(/status=processing/);await expect(page).toHaveURL(/category=typo/);if(!process.env.CI)await page.screenshot({path:info.outputPath('feedback.png'),fullPage:true});await narrow(page);
 await page.goto('/admin/users');const admin=page.getByRole('checkbox',{name:'admin',exact:true});await expect(admin).toBeVisible();expect(await admin.evaluate(node=>getComputedStyle(node).accentColor)).toBe('rgb(49, 88, 68)');await expect(page.getByRole('checkbox',{name:'learner',exact:true})).toBeDisabled();await fitsViewport(page);
 await page.goto('/admin/knowledge/'+id);await control(page.getByLabel('Knowledge type',{exact:true}));await page.goto('/account');await control(page.getByLabel('Current password',{exact:true}));await fitsViewport(page);
});

test('keyboard focus and native invalid input remain visible with shared controls',async({page,scene})=>{
 await scene('managed-knowledge');await page.goto('/login');const username=page.getByLabel('Username',{exact:true}),password=page.getByLabel('Password',{exact:true});await control(username);
 await username.focus();await page.keyboard.press('Tab');await expect(password).toBeFocused();expect(await password.evaluate(node=>getComputedStyle(node).outlineColor)).toBe('rgb(181, 142, 62)');
 await username.fill('a');await password.fill(TEST_PASSWORD);await page.getByRole('button',{name:'Sign in',exact:true}).click();await expect(username).toBeFocused();expect(await username.evaluate(node=>getComputedStyle(node).borderTopColor)).toBe('rgb(148, 67, 46)');await narrow(page);
});
