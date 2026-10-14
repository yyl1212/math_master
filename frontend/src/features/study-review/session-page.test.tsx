import {it,expect,vi} from 'vitest';
import {renderToString} from 'react-dom/server';
import {UiLocaleProvider} from '@/lib/i18n/provider';
import {ReviewSessionPage} from './pages';
import {actorId,sessionId,session} from '@/lib/study-review/test-fixtures';
const transport=vi.hoisted(()=>({read:vi.fn()}));
vi.mock('next/navigation',()=>({usePathname:()=>'/learn/review/own',useRouter:()=>({refresh:vi.fn(),push:vi.fn()})}));
vi.mock('next/headers',()=>({headers:async()=>new Headers({cookie:'private-test-session'})}));
vi.mock('@/lib/auth/server-client',()=>({readServerSession:async()=>({ok:true,data:{id:actorId,username:'learner',roles:['learner'],mustChangePassword:false}})}));
vi.mock('@/lib/study-review/server-client',()=>({readReview:transport.read}));
it('TestRecallSSRHasNoExpected: initial client props, HTML and serialized fields contain no target',async()=>{
 transport.read.mockResolvedValue({ok:true,data:session({answer:'本人已保存思路'})});const page=await ReviewSessionPage({params:Promise.resolve({id:sessionId})});
 const props=(page.props.children as React.ReactElement<{initial:ReturnType<typeof session>}>).props.initial;expect(props.reference).toBeNull();expect(props.prompt).toEqual([]);expect(Object.keys(props)).not.toContain('expected');expect(JSON.stringify(props)).not.toContain('测试目标正确内容');
 const html=renderToString(<UiLocaleProvider initialLocale="zh-CN">{page}</UiLocaleProvider>);expect(html).not.toContain('测试目标正确内容');expect(transport.read).toHaveBeenCalledWith('/api/v3/study/review/sessions/'+sessionId,'private-test-session');
});
it('another actor or unavailable proof projection is never passed as active source',async()=>{
 transport.read.mockResolvedValue({ok:true,data:session({actorId:sessionId})});const wrong=await ReviewSessionPage({params:Promise.resolve({id:sessionId})});expect(wrong.props).toMatchObject({code:'AUTHENTICATION_REQUIRED'});
 transport.read.mockResolvedValue({ok:true,data:session({target:'proof',available:false,prompt:[],reference:null})});const down=await ReviewSessionPage({params:Promise.resolve({id:sessionId})});const props=(down.props.children as React.ReactElement<{initial:ReturnType<typeof session>}>).props.initial;expect(props.available).toBe(false);expect(props.prompt).toEqual([]);expect(props.reference).toBeNull();
});
