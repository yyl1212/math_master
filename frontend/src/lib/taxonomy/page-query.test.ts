import {describe,it,expect} from "vitest";
import {parseTopicDetailOffsets} from "./page-query";
describe("independent topic detail searches",()=>{
 it("preserves separate filters and offsets",()=>{
  expect(parseTopicDetailOffsets({childrenQ:"模块",knowledgeQ:"加法",offset:"20",knowledgeOffset:"40"})).toEqual({childrenQ:"模块",knowledgeQ:"加法",offset:20,knowledgeOffset:40});
  expect(parseTopicDetailOffsets({})).toEqual({childrenQ:"",knowledgeQ:"",offset:0,knowledgeOffset:0});
 });
 it("uses byte limits for each filter and rejects ambiguous URL inputs",()=>{
  expect(parseTopicDetailOffsets({childrenQ:"汉".repeat(170)+"aa",knowledgeQ:"k".repeat(512)})).not.toBeNull();
  for(const input of [{childrenQ:"汉".repeat(171)},{knowledgeQ:"k".repeat(513)},{childrenQ:["a","b"]},{knowledgeQ:["a","b"]},{offset:"100001"},{knowledgeOffset:"-1"},{unknown:"a"},{childrenQ:"a\u0000b"}])expect(parseTopicDetailOffsets(input)).toBeNull();
 });
});
