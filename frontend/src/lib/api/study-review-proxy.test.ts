import {it,expect,vi,afterEach} from 'vitest';
import {createStudyReviewProxy} from './study-review-proxy';
import {actorId,date,sessionId,session} from '../study-review/test-fixtures';
const options={publicOrigin:'https://example.com',production:false},cookie='mm_session_dev='+'A'.repeat(43),base='/api/v3/study/review';
afterEach(()=>vi.useRealTimers());
it('health only forwards public metadata; private queue forwards selected session and drops Set-Cookie',async()=>{
 const fetcher=vi.fn<typeof fetch>(async(url,_o)=>String(url).endsWith('/health')?Response.json({capability:true,schemaReady:true}):Response.json({actorId,asOf:date,items:[],total:0,limit:20,offset:0},{headers:{'Set-Cookie':'forbidden=x'}}));const proxy=createStudyReviewProxy('http://127.0.0.1:8080',options,fetcher);
 const health=await proxy(new Request(options.publicOrigin+base+'/health',{headers:{Cookie:cookie}}));expect(health.status).toBe(200);expect(new Headers(fetcher.mock.calls[0][1]?.headers).has('Cookie')).toBe(false);
 expect((await proxy(new Request(options.publicOrigin+base+'/queue'))).status).toBe(401);
 const queue=await proxy(new Request(options.publicOrigin+base+'/queue',{headers:{Cookie:cookie+'; other=private'}}));expect(queue.status).toBe(200);expect(queue.headers.get('Cache-Control')).toBe('private, no-store');expect(queue.headers.has('Set-Cookie')).toBe(false);expect(new Headers(fetcher.mock.calls[1][1]?.headers).get('Cookie')).toBe(cookie);
});
it('rejects unknown namespace, methods, cross-origin GET/write and request over128KiB',async()=>{
 const fetcher=vi.fn();const proxy=createStudyReviewProxy('http://127.0.0.1:8080',options,fetcher);
 expect((await proxy(new Request(options.publicOrigin+'/api/v3/study/notes'))).status).toBe(404);
 expect((await proxy(new Request(options.publicOrigin+base+'/queue',{method:'POST'}))).status).toBe(405);
 expect((await proxy(new Request(options.publicOrigin+base+'/queue',{headers:{Cookie:cookie,Origin:'https://foreign.example'}}))).status).toBe(403);
 expect((await proxy(new Request(options.publicOrigin+base+'/sessions',{method:'POST',headers:{Cookie:cookie,Origin:'https://foreign.example'},body:'{}'}))).status).toBe(403);
 expect((await proxy(new Request(options.publicOrigin+base+'/sessions',{method:'POST',headers:{Cookie:cookie,Origin:options.publicOrigin},body:'x'.repeat(131073)}))).status).toBe(413);
 expect(fetcher).not.toHaveBeenCalled();
});
it('rejects malformed/leaking session and response beyond1MiB by length or stream',async()=>{
 const req=()=>new Request(options.publicOrigin+base+'/sessions/'+sessionId,{headers:{Cookie:cookie}});
 for(const response of [Response.json({...session(),reference:{sections:[{kind:'statement',items:['answer']}],sha256:'c'.repeat(64)}}),new Response('{}',{headers:{'Content-Length':'1048577'}}),new Response(new ReadableStream({start(c){c.enqueue(new Uint8Array(1048577));c.close()}}))]) {const proxy=createStudyReviewProxy('http://127.0.0.1:8080',options,async()=>response);const r=await proxy(req());expect(r.status).toBe(503);expect(await r.text()).not.toContain('answer')}
});
it('bounds even a fetcher that ignores its AbortSignal at10seconds',async()=>{
 vi.useFakeTimers();const proxy=createStudyReviewProxy('http://127.0.0.1:8080',options,()=>new Promise(()=>{}));const pending=proxy(new Request(options.publicOrigin+base+'/health'));await vi.advanceTimersByTimeAsync(10000);expect((await pending).status).toBe(503);
});
