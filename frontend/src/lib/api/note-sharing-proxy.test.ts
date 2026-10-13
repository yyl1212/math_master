import {it,expect,vi} from 'vitest';
import {createNoteSharingProxy} from './note-sharing-proxy';
const options={publicOrigin:'https://example.com',production:false},cookie='mm_session_dev='+'A'.repeat(43);
it('public latest and health do not forward cookies',async()=>{
 const fetcher=vi.fn<typeof fetch>(async(url,_options)=>Response.json(String(url).endsWith('/health')?{capability:true,schemaReady:true}:{items:[]}));const proxy=createNoteSharingProxy('http://127.0.0.1:8080',options,fetcher);
 for(const path of ['/api/v3/shared-notes/latest','/api/v3/note-sharing/health']){const r=await proxy(new Request(options.publicOrigin+path,{headers:{Cookie:cookie}}));expect(r.status).toBe(200);expect(new Headers(fetcher.mock.calls.at(-1)?.[1]?.headers).has('Cookie')).toBe(false)}
});
it('requires and forwards a session for all-shared and admin reads',async()=>{
 const fetcher=vi.fn<typeof fetch>(async(_url,_options)=>Response.json({actorId:'51390b7d-a8e9-4e98-a3ae-22101c20b309',items:[],nextCursor:null}));const proxy=createNoteSharingProxy('http://127.0.0.1:8080',options,fetcher);
 expect((await proxy(new Request(options.publicOrigin+'/api/v3/shared-notes'))).status).toBe(401);expect(fetcher).not.toHaveBeenCalled();
 const r=await proxy(new Request(options.publicOrigin+'/api/v3/shared-notes',{headers:{Cookie:cookie}}));expect(r.status).toBe(200);expect(new Headers(fetcher.mock.calls[0]?.[1]?.headers).get('Cookie')).toBe(cookie);expect(r.headers.get('Cache-Control')).toBe('private, no-store');
});
it('keeps health two booleans including not-ready responses',async()=>{
 const fetcher=vi.fn(async()=>Response.json({capability:true,schemaReady:false},{status:503}));const proxy=createNoteSharingProxy('http://127.0.0.1:8080',options,fetcher);const r=await proxy(new Request(options.publicOrigin+'/api/v3/note-sharing/health'));expect(r.status).toBe(503);expect(await r.json()).toEqual({capability:true,schemaReady:false});
 const bad=createNoteSharingProxy('http://127.0.0.1:8080',options,async()=>Response.json({capability:'yes',schemaReady:true}));expect((await bad(new Request(options.publicOrigin+'/api/v3/note-sharing/health'))).status).toBe(503);
});
it('rejects unknown paths, methods and an oversized ordinary write',async()=>{
 const fetcher=vi.fn();const proxy=createNoteSharingProxy('http://127.0.0.1:8080',options,fetcher);
 expect((await proxy(new Request(options.publicOrigin+'/api/v3/shared-notes/unknown'))).status).toBe(404);
 expect((await proxy(new Request(options.publicOrigin+'/api/v3/shared-notes/latest',{method:'POST'}))).status).toBe(405);
 const r=await proxy(new Request(options.publicOrigin+'/api/v3/study/knowledge/k-'+'a'.repeat(56)+'/note-sharing/submit',{method:'POST',headers:{Cookie:cookie,Origin:options.publicOrigin,'Content-Type':'application/json','X-CSRF-Token':'A'.repeat(43),'Idempotency-Key':'one'},body:'x'.repeat(131073)}));expect(r.status).toBe(413);expect(fetcher).not.toHaveBeenCalled();
});
it('treats an oversized upstream response as unavailable',async()=>{
 const proxy=createNoteSharingProxy('http://127.0.0.1:8080',options,async()=>new Response('{}',{headers:{'Content-Length':String(1048577)}}));
 expect((await proxy(new Request(options.publicOrigin+'/api/v3/note-sharing/health'))).status).toBe(503);
});
