import { describe,it,expect } from 'vitest';
import { readFileSync } from 'node:fs';
import {metadata,json,id} from './test-fixtures';
import { resolveFeedbackRoute, metadataSchema, readFeedbackResponse } from './schemas';
import { validateFeedbackBytes, readFeedbackBytes } from './bytes';
const cases=JSON.parse(readFileSync('../api/feedback-boundary-cases.json','utf8')) as {name:string;route:string;rawBase64:string;expectedValid:boolean;expectedStatus:number|null;expectedCode:string|null}[];
function validateRaw(c:typeof cases[number]){try{const raw=new Uint8Array(Buffer.from(c.rawBase64,'base64'));const route=resolveFeedbackRoute(c.route,raw.length?'POST':'GET');if(raw.length)validateFeedbackBytes(raw,route.action);return {valid:true,status:null,code:null}}catch(e){return {valid:false,status:400,code:'INVALID_REQUEST'}}}
describe('shared Go/Node feedback bytes',()=>{for(const c of cases)it(c.name,()=>{const result=validateRaw(c);expect(result.valid).toBe(c.expectedValid);expect(result.status).toBe(c.expectedStatus);expect(result.code).toBe(c.expectedCode)})});
it('metadata rejects all private free-text and generated-label changes',()=>{expect(metadataSchema.safeParse({...metadata(),title:'answer-sentinel'}).success).toBe(false);expect(metadataSchema.safeParse({...metadata(),label:'answer-sentinel'}).success).toBe(false);expect(JSON.stringify(metadataSchema.parse(metadata()))).not.toContain('answer-sentinel')});
async function rejectsOversizeStream(n:number){try{await readFeedbackBytes(new Response(' '.repeat(n)),2097152,new AbortController().signal);return false}catch{return true}}
it('keeps exact complete byte limits and rejects truncated streams',async()=>{expect(await rejectsOversizeStream(2097152)).toBe(false);expect(await rejectsOversizeStream(2097153)).toBe(true);await expect(readFeedbackBytes(new Response('{}',{headers:{'Content-Length':'3'}}),2097152,new AbortController().signal)).rejects.toThrow()});
it('rejects unsafe upstream JSON without reflecting text',async()=>{await expect(readFeedbackResponse(json({...metadata(),title:'answer-sentinel'}),'/api/v1/feedback/tickets/'+id,new AbortController().signal)).rejects.toMatchObject({code:'SERVICE_UNAVAILABLE'})});

it.each([['FEEDBACK_CONFLICT',409],['FEEDBACK_ANSWER_OVERLAP',409],['RATE_LIMITED',429],['FEEDBACK_NOT_CONFIGURED',503]] as const)('preserves closed %s status without upstream text',async(code,status)=>{
 const response=new Response(JSON.stringify({error:{code,message:'answer-sentinel',requestId:'a'.repeat(32),...(code==='RATE_LIMITED'?{retryAt:'2026-10-03T04:00:00Z'}:{})}}),{status,headers:{'Content-Type':'application/json','Cache-Control':'private, no-store','X-Content-Type-Options':'nosniff','X-Request-ID':'a'.repeat(32)}});
 const error=await readFeedbackResponse(response,'/api/v1/feedback/tickets',new AbortController().signal).catch(e=>e);
 expect(error).toMatchObject({code,status});expect(error.message).not.toContain('answer-sentinel');
});
