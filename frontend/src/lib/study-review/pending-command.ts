import {reviewCommand,readSession} from './client';
import {receiptSchema,routeForReview} from './schemas';
import {ReviewError,type CommandReceipt,type Session,type Ref} from './types';
export type PendingReviewCommand={readonly actorId:string;readonly path:string;readonly method:string;readonly action:string;readonly resource:string;readonly key:string;readonly input:Readonly<Record<string,unknown>>;readonly knowledge?:Ref;receipt:CommandReceipt|null};
const equalRef=(a:Ref,b:Ref)=>a.id===b.id&&a.contentSha256===b.contentSha256&&a.sourceKind===b.sourceKind;
function freeze(value:unknown):void{if(value&&typeof value==='object'){for(const child of Object.values(value))freeze(child);Object.freeze(value)}}
export function makePendingReviewCommand(actorId:string,path:string,method:string,input:unknown,knowledge?:Ref):PendingReviewCommand{
 const route=routeForReview(path);if(!route||route.method!==method||method==='GET')throw new ReviewError('INVALID_REQUEST',400);
 let clone:Record<string,unknown>;try{clone=JSON.parse(JSON.stringify(input)) as Record<string,unknown>}catch{throw new ReviewError('INVALID_REQUEST',400)}
 if(!clone||typeof clone!=='object'||Array.isArray(clone))throw new ReviewError('INVALID_REQUEST',400);freeze(clone);
 const resource=route.kind==='start'?(clone.knowledge as Ref|undefined)?.id:route.id;if(!resource)throw new ReviewError('INVALID_REQUEST',400);
 const ref=knowledge??(route.kind==='start'?clone.knowledge as Ref:undefined);return {actorId,path,method,action:route.kind,resource,key:crypto.randomUUID(),input:clone,knowledge:ref?Object.freeze({...ref}):undefined,receipt:null};
}
export type ConfirmedReviewCommand={receipt:CommandReceipt;session:Session|null;readError?:string};
export async function confirmPendingReviewCommand(pending:PendingReviewCommand,send:typeof reviewCommand=reviewCommand,read:typeof readSession=readSession):Promise<ConfirmedReviewCommand>{
 let receipt=pending.receipt;
 if(!receipt){
  const raw=await send(pending.path,pending.method,pending.input,pending.key,pending.actorId);const parsed=receiptSchema.safeParse(raw);if(!parsed.success)throw new ReviewError('STUDY_REVIEW_NOT_READY');receipt=parsed.data;
  const expected=pending.action==='start'?1:Number(pending.input.expectedSequence)+1;
  const resultStatus=pending.action==='start'||pending.action==='draft'?'working':pending.action==='compare'?'compared':pending.action==='grade'?'completed':'abandoned';
  if(receipt.actorId!==pending.actorId||receipt.action!==pending.action||receipt.sequence!==expected||receipt.status!==resultStatus||(pending.action==='start'?receipt.knowledgeId!==pending.resource:receipt.sessionId!==pending.resource)||(pending.knowledge&&!equalRef(receipt.knowledge,pending.knowledge))||(pending.action==='draft'&&receipt.draftRevision!==Number(pending.input.expectedDraftRevision)+1)||(pending.action==='grade'&&receipt.grade?.rating!==pending.input.rating))throw new ReviewError('STUDY_REVIEW_NOT_READY');
  pending.receipt=receipt;
 }
 try{
  const session=await read(receipt.sessionId,pending.actorId);
  if(session.actorId!==pending.actorId||session.sessionId!==receipt.sessionId||session.sequence<receipt.sequence||!equalRef(session.knowledge,receipt.knowledge))throw new ReviewError('STUDY_REVIEW_NOT_READY');
  return {receipt,session};
 }catch(e){if(e instanceof ReviewError&&(e.status===401||e.status===403))throw e;return {receipt,session:null,readError:e instanceof ReviewError?e.code:'STUDY_REVIEW_NOT_READY'}}
}
