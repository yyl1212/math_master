import type {Session,CommandReceipt,Schedule} from './types';
export const actorId='11111111-1111-4111-8111-111111111111',sessionId='22222222-2222-4222-8222-222222222222',reviewId='33333333-3333-4333-8333-333333333333';
export const knowledge={id:'k-'+'a'.repeat(56),contentSha256:'b'.repeat(64),sourceKind:'managed' as const};
export const date='2026-10-14T02:00:00Z';
export function schedule():Schedule{return{stage:1,dueAt:'2026-10-15T02:00:00Z',lastRef:knowledge,lastRating:'familiar',sequence:1,gradedAt:date}}
export function session(overrides:Partial<Session>={}):Session{return{actorId,sessionId,knowledgeId:knowledge.id,reviewId,knowledge,sequence:1,draftRevision:0,status:'working',mode:'recall',target:'statement',title:'Mathematics',titleZh:'数学',knowledgeType:'theorem',createdAt:date,comparedAt:null,gradedAt:null,abandonedAt:null,draftUpdatedAt:null,available:true,materialChanged:false,unanswered:false,prompt:[],answer:'',reference:null,schedule:null,grade:null,...overrides}}
export function receipt(overrides:Partial<CommandReceipt>={}):CommandReceipt{return{actorId,sessionId,knowledgeId:knowledge.id,reviewId,knowledge,sequence:1,draftRevision:0,status:'working',action:'start',appliedAt:date,grade:null,...overrides}}
