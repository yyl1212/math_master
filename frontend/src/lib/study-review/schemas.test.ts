import {describe,it,expect} from 'vitest';
import {schemaForReviewPath,routeForReview} from './schemas';
import {session,receipt,knowledge,actorId,date,sessionId} from './test-fixtures';
const path='/api/v3/study/review/sessions/'+sessionId;
describe('私人复习严格契约',()=>{
 it('初始默写无目标，reject任何泄漏或额外字段',()=>{
 const schema=schemaForReviewPath(path);expect(schema.safeParse(session()).success).toBe(true);
 const reference={sections:[{kind:'statement' as const,items:['正确内容']}],sha256:'c'.repeat(64)};
 for(const value of [session({reference}),{...session(),expected:'正确内容'},session({prompt:[{kind:'statement',items:['正确内容']}]}),session({actorId:'other'}),session({answer:'字'.repeat(16001)})])expect(schema.safeParse(value).success).toBe(false);
 });
 it('只有当前可用且已比对会话才可返回目标，下架题干一并拒绝',()=>{
 const reference={sections:[{kind:'proof' as const,items:['证明']}],sha256:'c'.repeat(64)};
 expect(schemaForReviewPath(path).safeParse(session({target:'proof',answer:'本人证明',status:'compared',sequence:2,draftRevision:1,comparedAt:date,reference})).success).toBe(true);
 expect(schemaForReviewPath(path).safeParse(session({available:false,target:'proof',prompt:[{kind:'statement',items:['失效题干']}]})).success).toBe(false);
 expect(schemaForReviewPath(path).safeParse(session({materialChanged:true,target:'proof',answer:'本人证明',status:'compared',comparedAt:date,reference})).success).toBe(false);
 });
 it('无正文回执严格绑定Ref/ID和grade动作',()=>{
 const schema=schemaForReviewPath('/api/v3/study/review/sessions','POST');expect(schema.safeParse(receipt()).success).toBe(true);
 for(const value of [{...receipt(),body:'private'},receipt({knowledgeId:'k-'+'c'.repeat(56)}),receipt({sequence:0}),receipt({grade:{rating:'familiar',gradedAt:date,schedule:{stage:6,dueAt:date,lastRef:knowledge,lastRating:'familiar',sequence:1,gradedAt:date}}})])expect(schema.safeParse(value).success).toBe(false);
 });
 it('分页默认上限50和最大安全序列、严格health',()=>{
 const schema=schemaForReviewPath('/api/v3/study/review/queue');const item={knowledge,title:'Title',titleZh:'标题',knowledgeType:'theorem',topicKeys:['97F40'],bucket:'unplanned',materialChanged:false,proofAvailable:false,activeSessionId:null,schedule:null};
 expect(schema.safeParse({actorId,asOf:date,items:[item],total:1,limit:20,offset:0}).success).toBe(true);
 expect(schema.safeParse({actorId,asOf:date,items:Array(51).fill(item),total:51,limit:50,offset:0}).success).toBe(false);
 expect(schemaForReviewPath('/api/v3/study/review/health').safeParse({capability:true,schemaReady:true,actorId}).success).toBe(false);
 expect(routeForReview('/api/v3/study/knowledge/'+knowledge.id+'/note')).toBeNull();expect(routeForReview(path+'/compare')?.method).toBe('POST');
 });
});
it('Go Unicode whitespace and BOM produce the same unanswered projection in the browser',()=>{
 const reference={sections:[{kind:'statement' as const,items:['参考']}],sha256:'c'.repeat(64)};
 for(const [answer,unanswered] of [['\u0085',true],['\ufeff',false]] as const){
  expect(schemaForReviewPath(path).safeParse(session({status:'compared',sequence:2,draftRevision:1,comparedAt:date,answer,unanswered,reference})).success).toBe(true);
 }
});
