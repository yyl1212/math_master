import {it,expect} from "vitest";
import {commandSchema,detailSchema,noteInputSchema,noteReceiptSchema,historyEntrySchema} from "./schemas";
const ref={id:"fractions",version:1,sha256:"a".repeat(64)},actor="11111111-1111-4111-8111-111111111111",command={knowledge:ref,expectedKnowledgeHead:actor,expectedSequence:0};
const record={knowledgeId:ref.id,state:"unlearned",sequence:0,firstStartedAt:null,firstCompletedAt:null,lastCompletedAt:null,lastReadAt:null,lastReviewedAt:null,completedRef:null,lastReviewRef:null,lastReviewId:null,activeReviewId:null};
const detail={actorId:actor,record,currentKnowledge:{...ref,title:"Fractions",titleZh:"分数",topicIds:["msc-00a00"]},pair:{knowledgeHead:actor,taxonomyHead:actor,taxonomyVersionId:"b".repeat(64)},available:true,materialChanged:false};
it("rejects unknown states, actor identifiers, fields and unsafe command numbers",()=>{
 expect(commandSchema.safeParse(command).success).toBe(true);expect(detailSchema.safeParse(detail).success).toBe(true);
 for(const value of [{...command,actorId:actor},{...command,expectedSequence:-1},{...command,expectedSequence:Number.MAX_SAFE_INTEGER+1},{...command,knowledge:{...ref,version:0}}])expect(commandSchema.safeParse(value).success).toBe(false);
 for(const value of [{...detail,actorId:"other"},{...detail,score:100},{...detail,record:{...record,state:"mastered"}}])expect(detailSchema.safeParse(value).success).toBe(false);
});
it("shares exact scalar and UTF8 note limits and returns receipts without text",()=>{
 const input={knowledge:ref,expectedRevision:0,body:"😀".repeat(16000)};expect(noteInputSchema.safeParse(input).success).toBe(true);
 for(const body of ["😀".repeat(16001),"a".repeat(16001),"bad\0text","\ud800"])expect(noteInputSchema.safeParse({...input,body}).success).toBe(false);
 const receipt={actorId:actor,knowledgeId:ref.id,revision:1,deleted:true,updatedAt:"2026-10-06T12:00:00Z"};expect(noteReceiptSchema.safeParse(receipt).success).toBe(true);expect(noteReceiptSchema.safeParse({...receipt,body:"deleted text"}).success).toBe(false);
});

it("keeps unknown legacy classification explicit without accepting missing native evidence",()=>{
 const event={id:actor,knowledge:ref,taxonomyVersionId:null,taxonomyHead:null,kind:"completed",occurredAt:"2026-10-06T12:00:00Z",noteRevision:null,reviewId:null,sourceKind:"legacy",originEventId:actor};
 expect(historyEntrySchema.safeParse(event).success).toBe(true);
 expect(historyEntrySchema.safeParse({...event,sourceKind:"native",originEventId:null}).success).toBe(false);
});
import {topicsPageSchema} from "./schemas";
it("accepts exact one quarter completed progress while reviewing",()=>{const page={actorId:actor,items:[{topicId:"msc-00",total:4,completed:1,learning:0,reviewing:1,added:0,removed:0,completedRatio:0.25}],total:1,limit:100,offset:0};expect(topicsPageSchema.safeParse(page).success).toBe(true)});
