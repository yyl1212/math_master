import {describe,it,expect} from "vitest";
import {topicNodeSchema,assignmentInputSchema,topicPageSchema} from "./schemas";
const node={id:"msc-13c60",code:"13C60",name:"Module categories",nameZh:"",kind:"primary",level:3,parentId:"msc-13c"};
const pair={knowledgeHead:null,taxonomyHead:null,taxonomyVersionId:"a".repeat(64)};
describe("topic contract boundaries",()=>{
 it("accepts a specific theme and rejects forged hierarchy or extra fields",()=>{
  expect(topicNodeSchema.safeParse(node).success).toBe(true);
  expect(topicNodeSchema.safeParse({...node,level:2}).success).toBe(false);
  expect(topicNodeSchema.safeParse({...node,actorId:"forged"}).success).toBe(false);
 });
 it("rejects invalid assignment identities, duplicates and unsafe paths",()=>{
  const v={knowledge:{id:"knowledge-a",version:1},topicIds:["msc-13c60"],sourceRefs:[{sourceId:"source",workFamilyId:"work",recordId:"original.1",path:"Algebra/data.json",sha256:"a".repeat(64)}],sourceBatchSHA:"b".repeat(64)};
  expect(assignmentInputSchema.safeParse(v).success).toBe(true);
  expect(assignmentInputSchema.safeParse({...v,topicIds:["msc-13c60","msc-13c60"]}).success).toBe(false);
  expect(assignmentInputSchema.safeParse({...v,sourceRefs:[{...v.sourceRefs[0],path:"../private"}]}).success).toBe(false);
  expect(assignmentInputSchema.safeParse({...v,sourceRefs:[{...v.sourceRefs[0],sourceId:"汉".repeat(43)}]}).success).toBe(false);
 });
 it("a public page never accepts private source references or invalid pagination",()=>{
  const summary={...node,ancestors:[],publishedKnowledgeCount:0,hasChildren:false};
  expect(topicPageSchema.safeParse({items:[summary],total:1,limit:20,offset:0,pair}).success).toBe(true);
  expect(topicPageSchema.safeParse({items:[{...summary,sourceRefs:[]}],total:1,limit:20,offset:0,pair}).success).toBe(false);
  expect(topicPageSchema.safeParse({items:[summary],total:1,limit:101,offset:0,pair}).success).toBe(false);
 });
});
