import {it,expect,vi,afterEach} from 'vitest';
import {noteShareRequest} from './client';
import {getAuthContext} from '../auth/client';
vi.mock('../auth/client',()=>({getAuthContext:vi.fn()}));afterEach(()=>vi.unstubAllGlobals());
const actor='51390b7d-a8e9-4e98-a3ae-22101c20b309',other='b1390b7d-a8e9-4e98-a3ae-22101c20b309';
const context=(id=actor)=>({ok:true,data:{user:{id,mustChangePassword:false,roles:['learner']},csrfToken:'A'.repeat(43)}});
it('rejects a private page for a different actor',async()=>{
 vi.mocked(getAuthContext).mockResolvedValue(context() as never);vi.stubGlobal('fetch',vi.fn(async()=>Response.json({actorId:other,items:[],nextCursor:null})));
 await expect(noteShareRequest('/api/v3/study/notes')).rejects.toMatchObject({status:401});
});
it('drops a late response after the session changes',async()=>{
 vi.mocked(getAuthContext).mockResolvedValueOnce(context() as never).mockResolvedValueOnce(context(other) as never);vi.stubGlobal('fetch',vi.fn(async()=>Response.json({actorId:actor,items:[],nextCursor:null})));
 await expect(noteShareRequest('/api/v3/study/notes')).rejects.toMatchObject({status:401});
});
it('never falls back to public after a revoked session',async()=>{
 vi.mocked(getAuthContext).mockResolvedValue({ok:false,status:401,code:'AUTHENTICATION_REQUIRED'} as never);const fetcher=vi.fn();vi.stubGlobal('fetch',fetcher);
 await expect(noteShareRequest('/api/v3/shared-notes')).rejects.toMatchObject({status:401});expect(fetcher).not.toHaveBeenCalled();
});
it('sends the exact caller input and original request key once',async()=>{
 vi.mocked(getAuthContext).mockResolvedValue(context() as never);const fetcher=vi.fn<typeof fetch>(async(_url,_options)=>Response.json({submissionId:actor,knowledgeId:'k-'+'a'.repeat(56),action:'submit',status:'pending',sequence:1}));vi.stubGlobal('fetch',fetcher);
 const input={expectedRevision:1,expectedSequence:0,title:'数学笔记'};await noteShareRequest('/api/v3/study/knowledge/k-'+'a'.repeat(56)+'/note-sharing/submit','POST',input,'original-key');expect(fetcher).toHaveBeenCalledTimes(1);expect(fetcher.mock.calls[0]?.[1]?.body).toBe(JSON.stringify(input));expect(new Headers(fetcher.mock.calls[0]?.[1]?.headers).get('Idempotency-Key')).toBe('original-key');
});
it('cancels a stalled body at the ten second deadline',async()=>{
 vi.useFakeTimers();let cancelled=false;const stream=new ReadableStream<Uint8Array>({cancel(){cancelled=true}});vi.stubGlobal('fetch',vi.fn(async()=>new Response(stream)));
 const result=noteShareRequest('/api/v3/shared-notes/latest').catch(error=>error);await vi.advanceTimersByTimeAsync(10001);expect(await result).toMatchObject({code:'REQUEST_INTERRUPTED',status:503});expect(cancelled).toBe(true);
});
