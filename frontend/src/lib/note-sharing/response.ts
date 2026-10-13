import {NoteShareError} from './types';
export async function boundedBytes(value:Request|Response,max:number,signal:AbortSignal):Promise<Uint8Array>{
 if(Number(value.headers.get('Content-Length')??'0')>max)throw new NoteShareError('INPUT_TOO_LARGE',413);if(signal.aborted)throw new NoteShareError('REQUEST_INTERRUPTED');if(!value.body)return new Uint8Array();
 const reader=value.body.getReader(),parts:Uint8Array[]=[];let length=0,rejectAbort!:(error:Error)=>void;const stopped=new Promise<never>((_,reject)=>{rejectAbort=reject});
 const abort=()=>{rejectAbort(new NoteShareError('REQUEST_INTERRUPTED'));void reader.cancel().catch(()=>{})};signal.addEventListener('abort',abort,{once:true});
 try{if(signal.aborted)abort();while(true){const{done,value}=await Promise.race([reader.read(),stopped]);if(done)break;length+=value.length;if(length>max){await reader.cancel();throw new NoteShareError('INPUT_TOO_LARGE',413)}parts.push(value)}}finally{signal.removeEventListener('abort',abort);reader.releaseLock()}
 const out=new Uint8Array(length);let offset=0;for(const part of parts){out.set(part,offset);offset+=part.length}return out;
}
export async function parseNoteJson(response:Response,max:number,signal:AbortSignal):Promise<unknown>{const bytes=await boundedBytes(response,max,signal).catch(e=>{if(e instanceof NoteShareError&&e.status===413)throw new NoteShareError('NOTE_SHARING_NOT_READY');throw e});return JSON.parse(new TextDecoder('utf-8',{fatal:true}).decode(bytes)) as unknown}
export function withinSignal<T>(promise:Promise<T>,signal:AbortSignal):Promise<T>{return new Promise((resolve,reject)=>{const abort=()=>reject(new NoteShareError('REQUEST_INTERRUPTED'));if(signal.aborted){abort();return}signal.addEventListener('abort',abort,{once:true});promise.then(resolve,reject).finally(()=>signal.removeEventListener('abort',abort))})}
