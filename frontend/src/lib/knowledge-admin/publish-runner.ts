import {readPublishJob,readPublishReceipt,stepPublishJob} from './publish-batch';
import {publishJobSchema,publishReceiptSchema} from './publish-batch-schemas';
import {KnowledgeError,type PublishJob,type PublishStepInput,type PublishStepReceipt} from './types';
export type PublishPending=Readonly<{actorId:string;jobId:string;input:Readonly<PublishStepInput>;key:string}>;
type Transport={read:typeof readPublishJob;receipt:typeof readPublishReceipt;step:typeof stepPublishJob};
type Options={actorId:string;jobId:string;signal:AbortSignal;onProgress:(job:PublishJob)=>void;onPending:(pending:PublishPending)=>void;onConfirmed:(receipt?:PublishStepReceipt)=>void;pending?:PublishPending|null;shouldPause?:()=>boolean;batchSize?:number;onBatchSize?:(size:number)=>void};
const api:Transport={read:readPublishJob,receipt:readPublishReceipt,step:stepPublishJob};
const authError=(e:unknown)=>e instanceof KnowledgeError&&['AUTHENTICATION_REQUIRED','FORBIDDEN','PASSWORD_CHANGE_REQUIRED','CSRF_FAILED'].includes(e.code);
export async function runPublishJob(o:Options,transport:Transport=api):Promise<PublishJob>{
 const check=()=>{if(o.signal.aborted)throw new KnowledgeError('REQUEST_INTERRUPTED')};
 async function current(){let last:unknown;for(let n=0;n<3;n++){check();try{const value=await transport.read(o.actorId,o.jobId,{},o.signal);check();const j=publishJobSchema.parse(value.job);if(j.actorId!==o.actorId||j.id!==o.jobId)throw new KnowledgeError('AUTHENTICATION_REQUIRED',401);o.onProgress(j);return j}catch(e){if(authError(e)||o.signal.aborted)throw e;last=e}}throw last}
 let job=await current(),pending=o.pending??null,size=o.batchSize??20,failures=0;
 if(pending&&(pending.actorId!==o.actorId||pending.jobId!==o.jobId))throw new KnowledgeError('AUTHENTICATION_REQUIRED',401);
 while(job.pending>0||pending){
  check();if(!pending&&o.shouldPause?.())return job;
  if(!pending){pending=Object.freeze({actorId:o.actorId,jobId:o.jobId,input:Object.freeze({expectedSequence:job.sequence,batchSize:size}),key:crypto.randomUUID()});o.onPending(pending)}
  let receipt:PublishStepReceipt|null=null;
  try{
   if(failures>0||o.pending===pending)receipt=await transport.receipt(o.actorId,o.jobId,pending.key,o.signal);
   check();if(!receipt)receipt=await transport.step(o.actorId,o.jobId,pending.input,pending.key,o.signal);check();
   receipt=publishReceiptSchema.parse(receipt);if(receipt.actorId!==o.actorId||receipt.jobId!==o.jobId||receipt.sequence!==pending.input.expectedSequence+1||receipt.items.length>(pending.input.batchSize??20))throw new KnowledgeError('REQUEST_INTERRUPTED');
  }catch(e){
   if(authError(e)||o.signal.aborted)throw e;
   if(e instanceof KnowledgeError&&e.code==='CONTENT_STALE'){pending=null;o.onConfirmed();return await current()}
   failures++;
   if(e instanceof KnowledgeError&&e.code==='PUBLISH_BATCH_ROLLED_BACK'){
    pending=null;o.onConfirmed();if(size===1)throw e;size=size>10?10:size>5?5:size>2?2:1;o.onBatchSize?.(size);
   }
   if(failures>=3)throw new KnowledgeError('PUBLISH_PAUSED');continue;
  }
  pending=null;o.onConfirmed(receipt);failures=0;job=await current();
 }
 return job;
}
