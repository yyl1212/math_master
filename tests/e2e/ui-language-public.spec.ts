import {test,expect,fitsViewport} from "./fixtures";
test("Chinese SSR, manual switch, navigation and unchanged mathematical data",async({page,context,request,scene,runtime})=>{
 await scene("published");
 await context.addCookies([{name:"math_master_ui_locale",value:"zh-CN",url:"http://127.0.0.1:18080"}]);
 const html=await context.request.get("/knowledge");expect(await html.text()).toContain('<html lang="zh-CN"');
 await page.goto("/knowledge");await expect(page.getByRole("heading",{name:"知识地图",exact:true})).toBeVisible();await expect(page).toHaveTitle("知识地图 | Math Master");
 const raw=await (await request.get("/api/v1/knowledge/"+runtime.knowledgeId)).body();const svg=await (await request.get("/api/v1/assets/"+runtime.assetSha)).body();
 const writes:string[]=[];page.on("request",r=>{if(!["GET","HEAD"].includes(r.method()))writes.push(r.url());});
 const input=page.getByLabel("搜索学习板块");await input.fill("Save draft");
 await page.getByRole("button",{name:"English",exact:true}).click();await expect(page.locator("#domain-search")).toHaveValue("Save draft");await expect(page.locator("html")).toHaveAttribute("lang","en");expect(writes).toEqual([]);
 await page.getByRole("button",{name:"中文",exact:true}).click();
 await page.getByRole("link",{name:"Elementary Mathematics",exact:true}).click();await expect(page).toHaveTitle("学习板块 | Math Master");
 await page.goto("/knowledge/"+runtime.knowledgeId);await expect(page.getByRole("heading",{name:"Equivalent fractions",exact:true})).toBeVisible();await expect(page.getByRole("heading",{name:"来源与使用"})).toBeVisible();await expect(page.locator("h1")).toHaveAttribute("lang","en");
 expect(await (await request.get("/api/v1/knowledge/"+runtime.knowledgeId)).body()).toEqual(raw);expect(await (await request.get("/api/v1/assets/"+runtime.assetSha)).body()).toEqual(svg);
 await page.reload();await expect(page.locator("html")).toHaveAttribute("lang","zh-CN");
 const other=await context.newPage();await other.goto("/knowledge");await expect(other).toHaveTitle("知识地图 | Math Master");await other.close();await fitsViewport(page);
});
test("invalid and duplicate preference default to English",async({page,context,scene,request})=>{
 await scene("draft");await context.addCookies([{name:"math_master_ui_locale",value:"zh-HK",url:"http://127.0.0.1:18080"}]);await page.goto("/knowledge");await expect(page.getByRole("heading",{name:"Knowledge Map",exact:true})).toBeVisible();
 const response=await request.get("/knowledge",{headers:{cookie:"math_master_ui_locale=zh-CN; math_master_ui_locale=en"}});expect(await response.text()).toContain('<html lang="en"');
});
test("blocked preference writes keep the current navigation language and title",async({page,scene})=>{
 await scene("draft");await page.addInitScript(()=>{const descriptor=Object.getOwnPropertyDescriptor(Document.prototype,"cookie")!;Object.defineProperty(Document.prototype,"cookie",{configurable:true,get(){return descriptor.get!.call(this)},set(value:string){if(value.startsWith("math_master_ui_locale="))throw new DOMException("Preference blocked");descriptor.set!.call(this,value);}});});
 await page.goto("/knowledge");await page.getByRole("button",{name:"中文",exact:true}).click();await page.getByRole("link",{name:"Elementary Mathematics",exact:true}).click();await expect(page).toHaveTitle("学习板块 | Math Master");await expect(page.locator("html")).toHaveAttribute("lang","zh-CN");await page.reload();await expect(page.locator("html")).toHaveAttribute("lang","en");
});
