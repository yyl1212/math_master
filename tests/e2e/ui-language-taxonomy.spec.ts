import {test,expect,fitsViewport} from "./fixtures";
import {createDraft,actor,wireContent} from "./content-helpers";
import type {SubmissionPage,SubmissionView} from "../../frontend/src/lib/content/types";
import type {DraftTopicView} from "../../frontend/src/lib/taxonomy/types";

test("topic search and unsaved assignment selections survive locale changes",async({page,scene})=>{
 await scene("topic-catalogue");await page.goto("/knowledge");await page.getByLabel("Search topics and knowledge").fill("13C60");await page.getByRole("button",{name:"中文",exact:true}).click();await expect(page.getByLabel("检索主题与知识点")).toHaveValue("13C60");await page.getByRole("button",{name:"搜索",exact:true}).click();await expect(page.locator(".domain-card")).toHaveCount(1);await expect(page).toHaveTitle("知识地图 | Math Master");await fitsViewport(page);
 await page.context().clearCookies();await createDraft(page);await page.getByLabel("Specific topic IDs").first().fill("msc-13c60");await page.getByRole("button",{name:"中文",exact:true}).click();await expect(page.getByLabel("具体主题编号").first()).toHaveValue("msc-13c60");await expect(page.getByRole("button",{name:"提交审核",exact:true})).toBeDisabled();await fitsViewport(page);
});

test("paired publication reason and history follow the interface language",async({page,scene})=>{
 await scene("topic-catalogue");await actor(page,"content_admin");await page.goto("/admin/publications");await page.getByLabel("Publication reason").fill("Preserve this original operator input while switching language.");await page.getByRole("button",{name:"中文",exact:true}).click();await expect(page.getByLabel("发布理由")).toHaveValue("Preserve this original operator input while switching language.");await expect(page.getByRole("heading",{name:"知识与主题成对发布",exact:true})).toBeVisible();await fitsViewport(page);
});
test("a prepared pair keeps its reason and activates only after a separate confirmed click",async({page,scene})=>{
 await scene("topic-catalogue");await actor(page,"content_admin");await page.goto("/admin/publications");const reason="Prepare and activate only this isolated catalogue replacement.";await page.getByLabel("Publication reason").fill(reason);await page.getByRole("button",{name:"Prepare paired publication",exact:true}).click();await expect(page.getByText("Pair prepared. Inspect topic changes before activation.",{exact:true})).toBeVisible();await expect(page.getByLabel("Publication reason")).toHaveValue(reason);await page.getByRole("button",{name:"Activate paired publication",exact:true}).click();await expect(page.getByRole("dialog")).toBeVisible();await page.getByLabel("Your password",{exact:true}).fill("Test-only 中文数学密码 with spaces");await page.getByRole("button",{name:"Verify password",exact:true}).click();await expect(page.getByRole("dialog")).toHaveCount(0);await expect(page.getByText("Knowledge and topic heads activated together.",{exact:true})).toHaveCount(0);await page.getByRole("button",{name:"Activate paired publication",exact:true}).click();await expect(page.getByText("Knowledge and topic heads activated together.",{exact:true})).toBeVisible();await fitsViewport(page);
});

test("imported topic assignments save with the body and uncertain writes retry without duplicate mutations",async({page,scene})=>{
 await scene("topic-catalogue");const {id,input}=await createDraft(page);
 const list=await wireContent<SubmissionPage>(page,"/api/v1/content/submissions?scope=mine&status=approved&limit=20");expect(list.status).toBe(200);
 const source=await wireContent<SubmissionView>(page,"/api/v1/content/submissions/"+list.data.items[0].id);expect(source.status).toBe(200);
 const batch=source.data.frozen.sourceMap[0].batchSha256;
 const initial=await wireContent<DraftTopicView>(page,"/api/v2/content/topic-assignments/drafts/"+id);expect(initial.status).toBe(200);
 const members=input.package.knowledge.map((k,i)=>({knowledge:{id:k.id,version:k.version},topicIds:[i===0?"msc-00a00":"msc-00a01"],sourceBatchSHA:batch,sourceRefs:[{sourceId:"original-browser-source",workFamilyId:"original-browser-work",recordId:i===0?"learning-root":"learning-middle",path:"Original/browser-fixture.json",sha256:"c".repeat(64)}]}));
 input.sourceMap=members.map(m=>({knowledge:m.knowledge,batchSha256:batch,relativePath:m.sourceRefs[0].path,sha256:m.sourceRefs[0].sha256,legacyId:m.sourceRefs[0].recordId,note:"Original isolated technical source binding; no mathematical approval."}));
 const inputs:unknown[]=[],keys:string[]=[];let uncertain=true;
 await page.route("**/api/v2/content/topic-assignments/drafts/"+id,async route=>{
  if(route.request().method()!=="PUT"){await route.continue();return}
  inputs.push(route.request().postDataJSON());keys.push(route.request().headers()["idempotency-key"]);
  if(uncertain){uncertain=false;const response=await route.fetch();expect(response.status()).toBe(200);await route.fulfill({status:503,contentType:"application/json",headers:{"X-Request-ID":"a".repeat(32)},body:JSON.stringify({error:{code:"SERVICE_UNAVAILABLE",message:"Original uncertain-response fixture."}})});return}
  await route.continue();
 });
 const envelope={kind:"topic-draft",schemaVersion:1,draft:input,assignments:members,sourceBatchSHA:batch,taxonomyVersionId:initial.data.taxonomyVersionId};
 await page.getByLabel("Import DraftInput JSON").setInputFiles({name:"auto-topic-draft.json",mimeType:"application/json",buffer:Buffer.from(JSON.stringify(envelope))});
 await expect(page.getByLabel("Specific topic IDs").first()).toHaveValue("msc-00a00");
 await page.getByRole("button",{name:"Save draft",exact:true}).click();
 await expect(page.getByText("Topic import: 0 assigned, 1 pending, 1 failed.",{exact:true})).toBeVisible();
 await page.getByRole("button",{name:"Retry unsaved assignments",exact:true}).click();
 await expect(page.getByText("Topic import: 2 assigned, 0 pending, 0 failed.",{exact:true})).toBeVisible();
 expect(keys[0]).toBeTruthy();expect(keys[0]).toBe(keys[1]);expect(inputs[0]).toEqual(inputs[1]);
 const saved=await wireContent<DraftTopicView>(page,"/api/v2/content/topic-assignments/drafts/"+id);expect(saved.status).toBe(200);expect(saved.data.members).toHaveLength(2);expect(saved.data.assignmentRevision).toBe(2);expect(saved.data.readyToSubmit).toBe(true);
 await page.getByRole("button",{name:"中文",exact:true}).click();await expect(page.getByText("主题导入：已归类 2 个，待确认或保存 0 个，保存失败 0 个。",{exact:true})).toBeVisible();await fitsViewport(page);
});
