import{it,expect}from"vitest";import{parseStudyResponseJSON}from"./response-json";
const bytes=(v:string)=>new TextEncoder().encode(v);
it("allows fractional completion ratios without relaxing integer counters or lexical guards",()=>{expect(parseStudyResponseJSON(bytes('{"completedRatio":0.25,"total":4}'))).toEqual({completedRatio:0.25,total:4});for(const value of ['{"total":1.0}','{"total":9007199254740992}','{"completedRatio":0.25,"CompletedRatio":0.5}','{"body":"\\ud800"}','{"body":"\\u0000"}','{"completedRatio":1e400}'])expect(()=>parseStudyResponseJSON(bytes(value))).toThrow()});
