import {isReviewAnswerBlank} from './answer';
import {z} from 'zod';
const uuid=z.string().regex(/^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/),kid=z.string().regex(/^k-[0-9a-f]{56}$/),sha=z.string().regex(/^[a-f0-9]{64}$/),time=z.iso.datetime({offset:true}),seq=z.number().int().min(0).max(Number.MAX_SAFE_INTEGER);
const text=(max:number,bytes?:number)=>z.string().refine(v=>v.isWellFormed()&&!v.includes('\0')&&Array.from(v).length<=max&&(bytes===undefined||new TextEncoder().encode(v).length<=bytes));
const body=text(16000,65536),title=text(1024,4096),short=text(128),type=z.enum(['concept','definition','axiom','theorem','corollary','method','mathematical-thinking','other']);
const mode=z.enum(['read','recall']),target=z.enum(['statement','proof']),rating=z.enum(['forgotten','uncertain','familiar']),status=z.enum(['working','compared','completed','abandoned']),action=z.enum(['start','draft','compare','grade','abandon']);
export const refSchema=z.object({id:kid,contentSha256:sha,sourceKind:z.literal('managed')}).strict();
const equalRef=(a:z.infer<typeof refSchema>,b:z.infer<typeof refSchema>)=>a.id===b.id&&a.contentSha256===b.contentSha256&&a.sourceKind===b.sourceKind;
const section=z.object({kind:z.enum(['statement','conditions','scope','system','equations','proof','proof-scope']),items:z.array(text(524288,524288)).min(1)}).strict(),sections=z.array(section).max(7);
const reference=z.object({sections,sha256:sha}).strict();
export const scheduleSchema=z.object({stage:z.number().int().min(0).max(5),dueAt:time,lastRef:refSchema,lastRating:rating,sequence:seq.min(1),gradedAt:time}).strict();
const grade=z.object({rating,gradedAt:time,schedule:scheduleSchema}).strict().refine(v=>v.rating===v.schedule.lastRating&&Date.parse(v.gradedAt)===Date.parse(v.schedule.gradedAt));
const common={actorId:uuid,sessionId:uuid,knowledgeId:kid,reviewId:uuid,knowledge:refSchema,sequence:seq.min(1),draftRevision:seq,status};
export const receiptSchema=z.object({...common,action,appliedAt:time,grade:grade.nullable()}).strict().refine(v=>v.knowledgeId===v.knowledge.id&&(v.action==='grade')===(v.grade!==null)&&(!v.grade||equalRef(v.grade.schedule.lastRef,v.knowledge)));
export const sessionSchema=z.object({...common,mode,target,title,titleZh:title,knowledgeType:type,createdAt:time,comparedAt:time.nullable(),gradedAt:time.nullable(),abandonedAt:time.nullable(),draftUpdatedAt:time.nullable(),available:z.boolean(),materialChanged:z.boolean(),unanswered:z.boolean(),prompt:sections,answer:body.nullable(),reference:reference.nullable(),schedule:scheduleSchema.nullable(),grade:grade.nullable()}).strict().superRefine((v,ctx)=>{
 const invalid=()=>ctx.addIssue({code:'custom',message:'Invalid review projection'});
 if(v.knowledgeId!==v.knowledge.id||(v.mode==='read'&&(v.answer!==null||v.unanswered||v.target!=='statement'))||(v.mode==='recall'&&v.answer===null))invalid();
 if((!v.available||v.materialChanged)&&(v.prompt.length>0||v.reference!==null))invalid();
 if(v.mode==='recall'&&v.target==='statement'&&v.prompt.length>0)invalid();
 if(v.mode==='read'&&v.prompt.length>0)invalid();
 if(v.mode==='recall'&&v.status!=='compared'&&v.status!=='completed'&&v.reference!==null)invalid();
 if(v.mode==='recall'&&v.target==='proof'&&v.reference?.sections.some(s=>s.kind!=='proof'&&s.kind!=='proof-scope'))invalid();
 if(v.mode==='recall'&&v.target==='proof'&&v.prompt.some(s=>s.kind==='proof'||s.kind==='proof-scope'))invalid();
 if(v.mode==='recall'&&v.target==='statement'&&v.reference?.sections.some(s=>s.kind==='proof'||s.kind==='proof-scope'))invalid();
 if(v.status==='working'&&(v.comparedAt!==null||v.gradedAt!==null||v.abandonedAt!==null||v.grade!==null||v.unanswered))invalid();
 if(v.status==='compared'&&(v.comparedAt===null||v.gradedAt!==null||v.abandonedAt!==null||v.grade!==null))invalid();
 if(v.status==='completed'&&(v.comparedAt===null||v.gradedAt===null||v.abandonedAt!==null||v.grade===null))invalid();
 if(v.status==='abandoned'&&(v.abandonedAt===null||v.gradedAt!==null||v.grade!==null))invalid();
 if(v.unanswered&&v.grade&&v.grade.rating!=='forgotten')invalid();
 if(v.grade&&(!equalRef(v.grade.schedule.lastRef,v.knowledge)||Date.parse(v.gradedAt??'')!==Date.parse(v.grade.gradedAt)))invalid();
 if(v.mode==='recall'&&v.comparedAt!==null&&(isReviewAnswerBlank(v.answer??'')!==v.unanswered))invalid();
});
const count=seq;
const overview=z.object({actorId:uuid,asOf:time,total:count,unfinished:count,changed:count,due:count,unplanned:count,future:count}).strict().refine(v=>v.total===v.unfinished+v.changed+v.due+v.unplanned+v.future);
const queueItem=z.object({knowledge:refSchema,title:short,titleZh:short,knowledgeType:type,topicKeys:z.array(z.string()),bucket:z.enum(['unfinished','changed','due','unplanned','future']),materialChanged:z.boolean(),proofAvailable:z.boolean(),activeSessionId:uuid.nullable(),schedule:scheduleSchema.nullable()}).strict();
const queuePage=z.object({actorId:uuid,asOf:time,items:z.array(queueItem).max(50),total:count,limit:z.number().int().min(1).max(50),offset:z.number().int().min(0).max(100000)}).strict();
const historyItem=z.object({sessionId:uuid,knowledge:refSchema,title:short,titleZh:short,knowledgeType:type,mode,target,grade,available:z.boolean(),materialChanged:z.boolean()}).strict().refine(v=>equalRef(v.grade.schedule.lastRef,v.knowledge));
const historyPage=z.object({actorId:uuid,asOf:time,items:z.array(historyItem).max(50),nextCursor:z.string().max(512).nullable()}).strict();
const health=z.object({capability:z.boolean(),schemaReady:z.boolean()}).strict();
export const reviewProblem=z.object({error:z.object({code:z.string(),message:z.string(),requestId:z.string(),sessionId:uuid.optional()}).strict()}).strict();
export type ReviewRoute={kind:string;private:boolean;method:'GET'|'POST'|'PUT';id?:string};
const uuidPath='[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}';
export function routeForReview(path:string):ReviewRoute|null{
 const base='/api/v3/study/review/';if(!path.startsWith(base))return null;
 const rest=path.slice(base.length);if(rest==='health')return {kind:'health',private:false,method:'GET'};
 if(['overview','queue','history'].includes(rest))return {kind:rest,private:true,method:'GET'};
 if(rest==='sessions')return {kind:'start',private:true,method:'POST'};
 const match=new RegExp('^sessions/('+uuidPath+')(?:/(draft|compare|grade|abandon))?$').exec(rest);if(!match)return null;
 const kind=match[2]??'session';return {kind,private:true,method:kind==='session'?'GET':kind==='draft'?'PUT':'POST',id:match[1]};
}
export function schemaForReviewPath(path:string,method='GET'){
 const route=routeForReview(path);if(!route||route.method!==method)return z.never();
 switch(route.kind){case 'health':return health;case 'overview':return overview;case 'queue':return queuePage;case 'history':return historyPage;case 'session':return sessionSchema;default:return receiptSchema}
}
export function isStudyReviewPath(path:string){return path==='/api/v3/study/review'||path.startsWith('/api/v3/study/review/')}
