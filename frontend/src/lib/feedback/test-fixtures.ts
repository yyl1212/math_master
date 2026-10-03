import type {Metadata} from './types';
export const id='11111111-1111-4111-8111-111111111111',otherId='22222222-2222-4222-8222-222222222222';
export function metadata():Metadata{return {id,target:{kind:'knowledge',identity:{id:'math-root',version:1,sha256:'a'.repeat(64)},area:null,part:null},label:'knowledge math-root · v1',category:'math_error',status:'new',sequence:1,createdAt:'2026-10-03T00:00:00Z',updatedAt:'2026-10-03T00:00:00Z',resolutionKind:null,targetValidity:'current',canHandle:false}}
export function json(data:unknown,status=200){return Response.json({actorId:id,data},{status,headers:{'Cache-Control':'private, no-store','X-Content-Type-Options':'nosniff','X-Request-ID':'a'.repeat(32)}})}
