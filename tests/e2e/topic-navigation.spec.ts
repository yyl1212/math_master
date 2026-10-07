import {test,expect,fitsViewport} from "./fixtures";

test("published topic catalogue has accurate paged counts and private management",async({page,request,scene})=>{
 await scene("topic-catalogue");
 const first=await request.get("/api/v2/topics?level=1&limit=100");expect(first.status()).toBe(200);const firstPage=await first.json();expect(firstPage.items).toHaveLength(63);expect(firstPage.total).toBe(63);expect(JSON.stringify(firstPage)).not.toContain("sourceRefs");
 for(const [query,total]of [["level=2",534],["level=3",4969],["kind=auxiliary",503],["kind=other",534]]as const){const response=await request.get("/api/v2/topics?"+query+"&limit=20");expect(response.status()).toBe(200);const data=await response.json();expect(data.total).toBe(total);expect(data.items.length).toBeLessThanOrEqual(20)}
 expect((await request.get("/api/v2/topics?q=a&q=b")).status()).toBe(400);
 expect((await request.get("/api/v2/content/topic-assignments/drafts/11111111-1111-4111-8111-111111111111")).status()).toBe(401);
 expect((await request.get("/api/v2/admin/publications")).status()).toBe(401);
 const html=await request.get("/knowledge");const bytes=await html.body();expect(bytes.byteLength).toBeLessThan(2<<20);expect(bytes.toString()).not.toContain("Original fixture specific 13C60");
 await page.goto("/knowledge");await expect(page.locator(".domain-card")).toHaveCount(63);await fitsViewport(page);
 await page.getByLabel("Topic level").selectOption("2");await page.getByRole("button",{name:"Search",exact:true}).click();await expect(page.locator(".domain-card")).toHaveCount(100);await expect(page.getByRole("link",{name:"Next topics"})).toBeVisible();await fitsViewport(page);
});

test("topic deep links, Chinese search and legacy domain navigation",async({page,scene})=>{
 await scene("topic-catalogue");await page.goto("/topics/msc-13c60");await expect(page.getByText("13C60",{exact:true})).toHaveCount(1);await expect(page.getByRole("link",{name:"Original fixture theme 13-XX",exact:true})).toHaveAttribute("href","/topics/msc-13");await expect(page.getByText("This topic has no published knowledge yet.")).toBeVisible();await fitsViewport(page);
 await page.getByRole("button",{name:"中文",exact:true}).click();await expect(page.getByText("此主题暂无已发布知识点。")).toBeVisible();await expect(page).toHaveTitle("主题详情 | Math Master");
 await page.goto("/knowledge");await page.getByLabel("检索主题与知识点").fill("原创主题 13-XX");await page.getByRole("button",{name:"搜索",exact:true}).click();await expect(page.locator(".domain-card")).toHaveCount(1);await expect(page.getByRole("link",{name:"原创主题 13-XX",exact:true})).toHaveAttribute("href","/topics/msc-13");
 await page.goto("/domains/linear-algebra");await expect(page.getByRole("link",{name:"原创主题 15-XX",exact:true})).toHaveAttribute("href","/topics/msc-15");await fitsViewport(page);
 await page.goto("/topics/msc-00a00");await expect(page.getByRole("link",{name:"加法练习 v1",exact:true})).toHaveAttribute("href","/knowledge/learning-root");
});

test("map ancestry and independent detail searches use the actual published catalogue",async({page,scene})=>{
 await scene("topic-catalogue");
 await page.goto("/knowledge?q=13C60");
 const path=page.getByRole("navigation",{name:"Topic hierarchy"});
 await expect(path.getByRole("link",{name:"13-XX Original fixture theme 13-XX",exact:true})).toHaveAttribute("href","/topics/msc-13");
 await expect(path.getByRole("link",{name:"13Cxx Original fixture subtheme 13Cxx",exact:true})).toHaveAttribute("href","/topics/msc-13c");
 await fitsViewport(page);
 await page.goto("/topics/msc-00a");
 await expect(page.getByRole("link",{name:"Addition basics v1",exact:true})).toBeVisible();
 await page.getByRole("search",{name:"Search subtopics"}).getByRole("searchbox").fill("no-such-topic");
 await page.getByRole("search",{name:"Search subtopics"}).getByRole("button").click();
 await expect(page.getByText("No topics match your search.")).toBeVisible();
 await expect(page.getByRole("link",{name:"Addition basics v1",exact:true})).toBeVisible();
 await page.getByRole("search",{name:"Search knowledge"}).getByRole("searchbox").fill("no-such-knowledge");
 await page.getByRole("search",{name:"Search knowledge"}).getByRole("button").click();
 await expect(page.getByText("No published knowledge matches your search.")).toBeVisible();
 await expect(page.getByRole("search",{name:"Search subtopics"}).getByRole("searchbox")).toHaveValue("no-such-topic");
 await expect(page).toHaveURL(/childrenQ=no-such-topic/);
 await page.getByRole("button",{name:"中文",exact:true}).click();
 await expect(page.getByRole("search",{name:"检索知识内容"}).getByRole("searchbox")).toHaveValue("no-such-knowledge");
 await fitsViewport(page);
});
