import {render,screen} from "@testing-library/react";
import {it,expect,vi} from "vitest";
import Page from "@/app/review/page";
import {fixtureID,otherID} from "@/lib/content/test-fixtures";
import type {Role} from "@/lib/auth/types";
const identity=vi.hoisted(()=>({roles:["learner","admin","reviewer"] as Role[]}));
vi.mock("@/features/content/page-access",()=>({contentPageAccess:async()=>({user:{id:fixtureID,username:"review_author",roles:identity.roles,mustChangePassword:false},cookie:"isolated-test-session"})}));
vi.mock("@/lib/content/server-client",()=>({readServerContent:async({query}:{query:{scope:string}})=>({ok:true,data:{items:query.scope==="review"?[]:[{id:fixtureID,workspaceId:otherID,ownerId:fixtureID,packageId:"own-submitted-package",packageVersion:1,catalogueVersion:1,status:"pending",revision:2,frozenDigest:"a".repeat(64),createdAt:"2026-10-06T00:00:00Z"}],total:query.scope==="review"?0:1,limit:20,offset:0}})}));
it("administrator reviewer sees their submitted version in the all-content list",async()=>{
 identity.roles=["learner","admin","reviewer"];
 render(await Page());
 expect(screen.queryByRole("link",{name:"own-submitted-package · Version 1"})).toBeInTheDocument();
});
it("ordinary reviewer does not see their own submission in the independent queue",async()=>{
 identity.roles=["learner","reviewer"];
 render(await Page());
 expect(screen.queryByRole("link",{name:"own-submitted-package · Version 1"})).not.toBeInTheDocument();
});
