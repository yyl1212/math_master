import {schemaForPath} from './schemas';
import {getAuthContext} from '../auth/client';
import {readContentBytes} from '../content/bytes';
import {parseContentJSON} from '../content/raw-json';
import {KnowledgeError,type PublishCreateInput,type PublishJob,type PublishJobPage,type PublishJobDetail,type PublishStepInput,type PublishStepReceipt,type PublishReceiptLookup} from './types';
import {publishCreateSchema,publishStepSchema,publishSchemaForPath} from './publish-batch-schemas';
const base='/api/v3/admin/knowledge/publish-jobs',uuid=/^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/;
export function publishAwait<T>(promise:Promise<T>,signal:AbortSignal):Promise<T>{return new Promise((resolve,reject)=>{const cancel=()=>reject(new KnowledgeError('REQUEST_INTERRUPTED'));if(signal.aborted){cancel();return}signal.addEventListener('abort',cancel,{once:true});promise.then(value=>{signal.removeEventListener('abort',cancel);if(signal.aborted)cancel();else resolve(value)},error=>{signal.removeEventListener('abort',cancel);reject(error)})})}
async function context(actor:string,signal:AbortSignal){const c=await publishAwait(getAuthContext(true),signal);if(!c.ok)throw new KnowledgeError(c.code,c.status);if(!c.data.user||c.data.user.id!==actor)throw new KnowledgeError('AUTHENTICATION_REQUIRED',401);if(c.data.user.mustChangePassword)throw new KnowledgeError('PASSWORD_CHANGE_REQUIRED',403);if(!c.data.user.roles.includes('admin'))throw new KnowledgeError('FORBIDDEN',403);return c.data}
async function request<T>(actor:string,path:string,method='GET',input?:unknown,key?:string,external?:AbortSignal):Promise<T>{
 if(!uuid.test(actor)||method!=='GET'&&(!key||new TextEncoder().encode(key).length>128))throw new KnowledgeError('INVALID_FIELD',422);
 const controller=new AbortController(),timer=setTimeout(()=>controller.abort(),10000),cancel=()=>controller.abort();external?.addEventListener('abort',cancel,{once:true});if(external?.aborted)cancel();const signal=controller.signal;
 try{const auth=await context(actor,signal),headers=new Headers({Accept:'application/json'});let body:string|undefined;
  if(method!=='GET'){body=JSON.stringify(input);if(new TextEncoder().encode(body).length>(path===base?262144:8192))throw new KnowledgeError('INPUT_TOO_LARGE',413);headers.set('Content-Type','application/json');headers.set('X-CSRF-Token',auth.csrfToken);headers.set('Idempotency-Key',key!)}
  const response=await publishAwait(fetch(path,{method,headers,body,credentials:'same-origin',cache:'no-store',redirect:'error',signal}),signal);
  if(response.headers.get('Cache-Control')!=='private, no-store'&&response.ok||!/^application\/json(?:;\s*charset=utf-8)?$/i.test(response.headers.get('Content-Type')??''))throw new KnowledgeError('REQUEST_INTERRUPTED');
  const raw=parseContentJSON(await readContentBytes(response,1048576,signal));
  if(!response.ok){const problem=raw as {error?:{code?:unknown}};const code=typeof problem?.error?.code==='string'?problem.error.code:'REQUEST_INTERRUPTED';throw new KnowledgeError(code,response.status)}
  if(response.status!==200)throw new KnowledgeError('REQUEST_INTERRUPTED');const isPublish=path.startsWith(base);const out=(isPublish?publishSchemaForPath(path,method):schemaForPath(path,method)).parse(raw) as T;const value=out as {actorId?:string;job?:PublishJob;jobId?:string;id?:string};
  if(isPublish&&(value.actorId??value.job?.actorId)!==actor)throw new KnowledgeError('AUTHENTICATION_REQUIRED',401);
  const id=isPublish?new URL(path,'http://localhost').pathname.slice(base.length+1).split('/')[0]:'';if(id&&((value.jobId??value.job?.id)!==id))throw new KnowledgeError('REQUEST_INTERRUPTED');
  await context(actor,signal);return out;
 }catch(e){if(e instanceof KnowledgeError)throw e;throw new KnowledgeError('REQUEST_INTERRUPTED')}finally{clearTimeout(timer);external?.removeEventListener('abort',cancel)}
}
function idPath(id:string){if(!uuid.test(id))throw new KnowledgeError('INVALID_FIELD',422);return base+'/'+id}
function queryPath(path:string,query:Record<string,string|number|undefined>){const params=new URLSearchParams();for(const[k,v]of Object.entries(query))if(v!==undefined)params.set(k,String(v));return path+(params.size?'?'+params:'')}
export function createPublishJob(actor:string,input:PublishCreateInput,key:string,signal?:AbortSignal){publishCreateSchema.parse(input);return request<PublishJob>(actor,base,'POST',input,key,signal)}
export function listPublishJobs(actor:string,query:{createKey?:string;limit?:number;offset?:number}={},signal?:AbortSignal){return request<PublishJobPage>(actor,queryPath(base,query),'GET',undefined,undefined,signal)}
export function readPublishJob(actor:string,id:string,query:{state?:'pending'|'published'|'skipped'|'failed';limit?:number;offset?:number}={},signal?:AbortSignal){return request<PublishJobDetail>(actor,queryPath(idPath(id),query),'GET',undefined,undefined,signal)}
export async function readPublishReceipt(actor:string,id:string,key:string,signal?:AbortSignal){const value=await request<PublishReceiptLookup>(actor,queryPath(idPath(id),{receiptKey:key}),'GET',undefined,undefined,signal);if(value.key!==key)throw new KnowledgeError('REQUEST_INTERRUPTED');return value.receipt}
export async function stepPublishJob(actor:string,id:string,input:PublishStepInput,key:string,signal?:AbortSignal){publishStepSchema.parse(input);const out=await request<PublishStepReceipt>(actor,idPath(id)+'/step','POST',input,key,signal);if(out.sequence!==input.expectedSequence+1||out.items.length>(input.batchSize??20))throw new KnowledgeError('REQUEST_INTERRUPTED');return out}
export async function readPublishRetryKnowledge(actor:string,id:string,signal?:AbortSignal){
 if(!/^k-[0-9a-f]{56}$/.test(id))throw new KnowledgeError('INVALID_FIELD',422);
 try{const out=await request<import('./types').Knowledge>(actor,'/api/v3/admin/knowledge/'+id,'GET',undefined,undefined,signal);return {id:out.id,editToken:out.editToken,title:out.point.title,titleZh:out.point.title_zh,published:out.published,deleted:out.deleted}}catch(e){if(e instanceof KnowledgeError&&e.code==='NOT_FOUND')return null;throw e}
}
