import {it,expect,vi,afterEach} from 'vitest';
import {readFeedbackServer} from './server-client';
import {json,metadata,id} from './test-fixtures';
vi.mock('server-only',()=>({}));const jar=vi.hoisted(()=>({cookies:vi.fn()}));vi.mock('next/headers',()=>({cookies:jar.cookies}));
afterEach(()=>{vi.unstubAllGlobals();vi.unstubAllEnvs();vi.resetAllMocks()});
it('reads current cookies per request with no private response cache',async()=>{vi.stubEnv('GO_API_INTERNAL_URL','http://127.0.0.1:18080');vi.stubEnv('AUTH_PUBLIC_ORIGIN','http://127.0.0.1:3000');vi.stubEnv('APP_ENV','development');const fetcher=vi.fn<typeof fetch>(async()=>json(metadata()));vi.stubGlobal('fetch',fetcher);for(const cookie of ['A','B']){jar.cookies.mockResolvedValue({toString:()=>`mm_session_dev=${cookie.repeat(43)}`});await readFeedbackServer('/api/v1/feedback/tickets/'+id)};expect(fetcher).toHaveBeenCalledTimes(2);expect(fetcher.mock.calls.map(c=>new Headers(c[1]?.headers).get('Cookie'))).toEqual(['mm_session_dev='+'A'.repeat(43),'mm_session_dev='+'B'.repeat(43)]);expect(fetcher.mock.calls[0]?.[1]).toMatchObject({cache:'no-store',redirect:'error'})});
