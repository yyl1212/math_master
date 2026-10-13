import {it,expect,vi,afterEach} from 'vitest';
import {readReview} from './server-client';
import {readServerSession} from '../auth/server-client';
import {actorId,sessionId,session} from './test-fixtures';
vi.mock('../auth/server-client',()=>({readServerSession:vi.fn()}));vi.mock('../api/server-config',()=>({getGoOrigin:()=> 'http://127.0.0.1:8080'}));vi.mock('../auth/config',()=>({getAuthConfig:()=>({production:false,publicOrigin:'https://example.com'})}));
const user={id:actorId,username:'review_learner',roles:['learner' as const],mustChangePassword:false},cookie='mm_session_dev='+'A'.repeat(43);
afterEach(()=>{vi.unstubAllGlobals();vi.resetAllMocks()});
it('SSR private session uses selected cookie and retains no expected before compare',async()=>{
 vi.mocked(readServerSession).mockResolvedValue({ok:true,data:user});const fetcher=vi.fn<typeof fetch>(async()=>Response.json(session()));vi.stubGlobal('fetch',fetcher);
 const result=await readReview('/api/v3/study/review/sessions/'+sessionId,cookie+'; other=ignored');expect(result).toMatchObject({ok:true,data:{reference:null,prompt:[]}});expect(new Headers(fetcher.mock.calls[0][1]?.headers).get('Cookie')).toBe(cookie);expect(fetcher.mock.calls[0][1]?.cache).toBe('no-store');
});
it('SSR rejects missing/currently revoked identity and leaking working payload',async()=>{
 vi.mocked(readServerSession).mockResolvedValueOnce({ok:true,data:user}).mockResolvedValueOnce({ok:false,status:401,code:'AUTHENTICATION_REQUIRED',message:'Sign in'});vi.stubGlobal('fetch',vi.fn(async()=>Response.json(session())));expect(await readReview('/api/v3/study/review/sessions/'+sessionId,cookie)).toMatchObject({ok:false,status:401});
 vi.mocked(readServerSession).mockResolvedValue({ok:true,data:user});vi.stubGlobal('fetch',vi.fn(async()=>Response.json({...session(),expected:'forged answer'})));expect(await readReview('/api/v3/study/review/sessions/'+sessionId,cookie)).toMatchObject({ok:false,status:503});
});
