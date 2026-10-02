import { parseContentJSON } from '../content/raw-json';
import { readContentBytes } from '../content/bytes';
import { feedbackInputSchemas } from './schemas';
import { FeedbackRequestError,type FeedbackAction } from './types';
export function feedbackAwait<T>(pending:Promise<T>,signal:AbortSignal):Promise<T>{return new Promise((resolve,reject)=>{const abort=()=>reject(new FeedbackRequestError());if(signal.aborted){abort();return}signal.addEventListener('abort',abort,{once:true});void pending.then(resolve,reject).finally(()=>signal.removeEventListener('abort',abort))})}
export async function readFeedbackBytes(response:Response,max:number,signal:AbortSignal):Promise<Uint8Array>{return feedbackAwait(readContentBytes(response,max,signal),signal)}
export function validateFeedbackBytes(raw:Uint8Array,action:FeedbackAction):unknown{if(raw.byteLength>65536)throw new FeedbackRequestError('INVALID_REQUEST');try{const value=parseContentJSON(raw);const schema=feedbackInputSchemas[action as keyof typeof feedbackInputSchemas];if(!schema)throw new Error();return schema.parse(value)}catch{throw new FeedbackRequestError('INVALID_REQUEST')}}

export async function withFeedbackDeadline<T>(signal: AbortSignal | undefined, work: (active: AbortSignal) => Promise<T>): Promise<T> {
  const controller = new AbortController(), abort = () => controller.abort();
  const timer = setTimeout(abort, 10000);
  signal?.addEventListener('abort', abort, { once: true });
  if (signal?.aborted) abort();
  try { return await feedbackAwait(work(controller.signal), controller.signal); }
  catch (e) { throw e instanceof FeedbackRequestError ? e : new FeedbackRequestError(); }
  finally { clearTimeout(timer); signal?.removeEventListener('abort', abort); }
}
