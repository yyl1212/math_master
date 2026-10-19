import {getAuthContext} from '../auth/client';
import {noteProblem,routeForNote,schemaForNotePath} from './schemas';
import {parseNoteJson,withinSignal} from './response';
import {NoteShareError} from './types';
export async function noteShareRequest<T>(path:string,method='GET',input?:unknown,key?:string):Promise<T>{
 const route=routeForNote(path.split('?')[0]);if(!route||route.method!==method)throw new NoteShareError('INVALID_REQUEST',400);
 const controller=new AbortController(),timer=setTimeout(()=>controller.abort(),10000);
 try{
  const headers=new Headers({Accept:'application/json'});let actor:string|undefined;
  if(route.private){const context=await withinSignal(getAuthContext(true),controller.signal);if(!context.ok)throw new NoteShareError(context.code,context.status);if(!context.data.user)throw new NoteShareError('AUTHENTICATION_REQUIRED',401);if(context.data.user.mustChangePassword)throw new NoteShareError('PASSWORD_CHANGE_REQUIRED',403);actor=context.data.user.id;if(method!=='GET'){headers.set('Content-Type','application/json');headers.set('X-CSRF-Token',context.data.csrfToken);headers.set('Idempotency-Key',key??crypto.randomUUID())}}
  const response=await fetch(path,{method,headers,...(method!=='GET'?{body:JSON.stringify(input)}:{}),credentials:route.private?'same-origin':'omit',cache:'no-store',redirect:'error',signal:controller.signal});
  const raw=await parseNoteJson(response,route.list?4<<20:1<<20,controller.signal);if(!response.ok){const parsed=noteProblem.safeParse(raw);throw new NoteShareError(parsed.success?parsed.data.error.code:'NOTE_SHARING_NOT_READY',response.status)}
  const value=schemaForNotePath(path,method).parse(raw);
  if(route.private){if('actorId' in value&&value.actorId!==actor)throw new NoteShareError('AUTHENTICATION_REQUIRED',401);const current=await withinSignal(getAuthContext(true),controller.signal);if(!current.ok||!current.data.user||current.data.user.id!==actor||current.data.user.mustChangePassword)throw new NoteShareError('AUTHENTICATION_REQUIRED',401)}
  return value as T;
 }catch(e){if(e instanceof NoteShareError)throw e;throw new NoteShareError(controller.signal.aborted?'REQUEST_INTERRUPTED':'NOTE_SHARING_NOT_READY')}finally{clearTimeout(timer)}
}
