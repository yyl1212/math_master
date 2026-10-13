import {beforeEach,it,expect,vi} from 'vitest';
import {render,screen} from '@testing-library/react';
import {UiLocaleProvider} from '@/lib/i18n/provider';
import {ReviewWorkspacePage,ReviewHistoryPage} from './pages';
import {actorId,date} from '@/lib/study-review/test-fixtures';
vi.mock('next/navigation',()=>({useRouter:()=>({push:vi.fn(),refresh:vi.fn()}),usePathname:()=>'/learn/review'}));
vi.mock('next/headers',()=>({headers:async()=>new Headers({cookie:'isolated-session'})}));
const server=vi.hoisted(()=>({read:vi.fn(),user:vi.fn()}));
vi.mock('@/lib/study-review/server-client',()=>({readReview:server.read}));vi.mock('@/lib/auth/server-client',()=>({readServerSession:server.user}));
vi.mock('@/features/study/account-boundary',async()=>({...await vi.importActual<typeof import('@/features/study/account-boundary')>('@/features/study/account-boundary'),StudyAccountBoundary:({children}:{children:React.ReactNode})=><>{children}</>}));
beforeEach(()=>{server.read.mockReset();server.user.mockReset().mockResolvedValue({ok:true,data:{id:actorId,username:'learner',roles:['learner'],mustChangePassword:false}});server.read.mockImplementation(async(path:string)=>({ok:true,data:path.endsWith('/overview')?{actorId,asOf:date,total:0,unfinished:0,changed:0,due:0,unplanned:0,future:0}:path.includes('/history')?{actorId,asOf:date,items:[],nextCursor:null}:{actorId,asOf:date,items:[],total:0,limit:20,offset:0}}))});
it('SSR overview and queue run sequentially and all filters are preserved',async()=>{
 let overviewDone=false;server.read.mockImplementation(async(path:string)=>{if(path.endsWith('/overview')){await Promise.resolve();overviewDone=true;return {ok:true,data:{actorId,asOf:date,total:0,unfinished:0,changed:0,due:0,unplanned:0,future:0}}}expect(overviewDone).toBe(true);return {ok:true,data:{actorId,asOf:date,items:[],total:0,limit:20,offset:0}}});
 render(<UiLocaleProvider initialLocale="zh-CN">{await ReviewWorkspacePage({searchParams:Promise.resolve({bucket:'unplanned',q:'向量',topicKey:'15-XX'})})}</UiLocaleProvider>);expect(screen.getByRole('heading',{name:'复习工作区'})).toBeVisible();expect(server.read.mock.calls[1][0]).toBe('/api/v3/study/review/queue?bucket=unplanned&q=%E5%90%91%E9%87%8F&topicKey=15-XX');
});
it('SSR unknown/duplicate filters, another actor and force-password never expose a private list',async()=>{
 server.user.mockResolvedValue({ok:true,data:{id:actorId,mustChangePassword:true}});render(<UiLocaleProvider initialLocale="zh-CN">{await ReviewWorkspacePage({searchParams:Promise.resolve({})})}</UiLocaleProvider>);expect(server.read).not.toHaveBeenCalled();expect(screen.queryByRole('heading',{name:'复习工作区'})).toBeNull();
});
it('SSR history is personal metadata with its original learning-history link',async()=>{
 render(<UiLocaleProvider initialLocale="zh-CN">{await ReviewHistoryPage({searchParams:Promise.resolve({mode:'recall'})})}</UiLocaleProvider>);expect(screen.getByRole('heading',{name:'自评历史'})).toBeVisible();expect(server.read.mock.calls[0][0]).toBe('/api/v3/study/review/history?mode=recall');
});

it('rejects duplicate or unknown SSR filters before any private module request',async()=>{
 for(const filters of [{q:['a','b']},{unexpected:'x'}]){server.read.mockClear();await ReviewWorkspacePage({searchParams:Promise.resolve(filters)});expect(server.read).not.toHaveBeenCalled()}
});
it('rejects a different private actor instead of rendering queue metadata',async()=>{
 server.read.mockResolvedValue({ok:true,data:{actorId:'44444444-4444-4444-8444-444444444444',asOf:date,total:0,unfinished:0,changed:0,due:0,unplanned:0,future:0,items:[],limit:20,offset:0}});render(<UiLocaleProvider initialLocale="zh-CN">{await ReviewWorkspacePage({searchParams:Promise.resolve({})})}</UiLocaleProvider>);expect(screen.queryByRole('heading',{name:'复习工作区'})).toBeNull();expect(screen.getByText('请登录后使用个人复习。')).toBeVisible();
});
