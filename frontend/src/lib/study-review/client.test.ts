import {it,expect,vi,afterEach} from 'vitest';
import {reviewRequest,reviewCommand,readSession} from './client';
import {getAuthContext} from '../auth/client';
import {actorId,sessionId,knowledge,session,receipt} from './test-fixtures';
vi.mock('../auth/client',()=>({getAuthContext:vi.fn()}));
const other='44444444-4444-4444-8444-444444444444';
const context=(id=actorId)=>({ok:true as const,data:{user:{id,username:'review_learner',roles:['learner' as const],mustChangePassword:false},csrfToken:'A'.repeat(43)}});
afterEach(()=>{vi.unstubAllGlobals();vi.useRealTimers();vi.resetAllMocks()});
it('rejects private metadata for another actor and late response after account switch',async()=>{
 vi.mocked(getAuthContext).mockResolvedValue(context());vi.stubGlobal('fetch',vi.fn(async()=>Response.json(session({actorId:other}))));await expect(readSession(sessionId,actorId)).rejects.toMatchObject({status:401});
 vi.mocked(getAuthContext).mockReset().mockResolvedValueOnce(context()).mockResolvedValueOnce(context(other));vi.stubGlobal('fetch',vi.fn(async()=>Response.json(session())));await expect(readSession(sessionId,actorId)).rejects.toMatchObject({status:401});
});
it('checks pending actor before mutation and sends exactly the original key/input',async()=>{
 vi.mocked(getAuthContext).mockResolvedValue(context(other));const fetcher=vi.fn<typeof fetch>(async()=>Response.json(receipt()));vi.stubGlobal('fetch',fetcher);const input={knowledge,mode:'recall',target:'statement'};
 await expect(reviewCommand('/api/v3/study/review/sessions','POST',input,'original-key',actorId)).rejects.toMatchObject({status:401});expect(fetcher).not.toHaveBeenCalled();
 vi.mocked(getAuthContext).mockResolvedValue(context());await reviewCommand('/api/v3/study/review/sessions','POST',input,'original-key',actorId);expect(fetcher).toHaveBeenCalledTimes(1);expect(fetcher.mock.calls[0][1]?.body).toBe(JSON.stringify(input));expect(new Headers(fetcher.mock.calls[0][1]?.headers).get('Idempotency-Key')).toBe('original-key');
});
it('never falls back to public and bounds fetch or body at10seconds',async()=>{
 vi.mocked(getAuthContext).mockResolvedValue({ok:false,status:401,code:'AUTHENTICATION_REQUIRED',message:'Sign in'});const fetcher=vi.fn();vi.stubGlobal('fetch',fetcher);await expect(readSession(sessionId,actorId)).rejects.toMatchObject({status:401});expect(fetcher).not.toHaveBeenCalled();
 vi.useFakeTimers();vi.stubGlobal('fetch',vi.fn(()=>new Promise(()=>{})));const pending=reviewRequest('/api/v3/study/review/health').catch(e=>e);await vi.advanceTimersByTimeAsync(10000);expect(await pending).toMatchObject({code:'REQUEST_INTERRUPTED'});
});
