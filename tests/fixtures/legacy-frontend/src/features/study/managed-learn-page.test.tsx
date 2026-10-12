import {beforeEach,expect,it,vi} from 'vitest';
import {fireEvent,render,screen,within} from '@testing-library/react';
import {ManagedLearnPage} from './managed-pages';
import {UiLocaleProvider,useUiI18n} from '@/lib/i18n/provider';
import type {Overview,ProgressPage,StudyPage,TopicPage} from '@/lib/knowledge-admin/types';

vi.mock('server-only',()=>({}));
vi.mock('next/headers',()=>({headers:async()=>new Headers({cookie:'test-session'})}));
vi.mock('@/lib/auth/server-client',()=>({readServerSession:async()=>({ok:true,data:{id:'owner',mustChangePassword:false}})}));
vi.mock('./account-boundary',()=>({StudyAccountBoundary:({children}:{children:React.ReactNode})=><>{children}</>}));
const request=vi.hoisted(()=>vi.fn());
vi.mock('@/lib/knowledge-admin/server-client',()=>({readManaged:request}));
const overview:Overview={actorId:'owner',total:4,completed:1,learning:1,reviewing:1,unavailable:0,unclassified:0,materialChanged:0,reminders:[]};
const topics:ProgressPage={items:[{topicKey:'15-XX',total:4,completed:1,learning:1,reviewing:1,completedRatio:.5},{topicKey:'65-XX',total:0,completed:0,learning:0,reviewing:0,completedRatio:null}],total:2,limit:100,offset:0};
const directory:TopicPage={items:[{topicKey:'15-XX',title:'线性代数与矩阵理论',titleEn:'Linear algebra and matrix theory',kind:'primary',knowledgeCount:4,items:{items:[],limit:20,offset:0,total:4}},{topicKey:'65-XX',title:'数值分析',titleEn:'Numerical analysis',kind:'primary',knowledgeCount:0,items:{items:[],limit:20,offset:0,total:0}}],total:2,limit:100,offset:0};
const knowledgeId='k-'+'a'.repeat(56);
const list:StudyPage={items:[{actorId:'owner',available:true,materialChanged:false,currentKnowledge:{id:knowledgeId,externalId:'point-a',topicKeys:['15A03','15A06'],point:{title:'Column space',title_zh:'列空间',type:'concept',learning_difficulty:{difficulty_level:2,rationale:"定义及简单计算",prerequisites:[],review_status:"body_assessed",rubric_version:"learning-foundation-v1-20261008",proof_scope_note:"不涉及证明"}},sources:[],ref:{id:knowledgeId,contentSha256:'a'.repeat(64),sourceKind:'managed'},updatedAt:'2026-10-10T00:00:00Z'},record:{knowledgeId,state:'learning',sequence:1,firstStartedAt:null,firstCompletedAt:null,lastCompletedAt:null,lastReadAt:null,lastReviewedAt:null,completedRef:null,lastReviewRef:null,lastReviewId:null,activeReviewId:null}}],total:4,limit:2,offset:0};
beforeEach(()=>{request.mockReset();request.mockImplementation(async(path:string)=>({ok:true,data:path.startsWith('/api/v3/study/overview')?overview:path.startsWith('/api/v3/study/topics')?topics:path.startsWith('/api/v3/topics')?directory:list}));});
function LocaleSwitch(){const{setLocale}=useUiI18n();return <button onClick={()=>setLocale('en')}>English</button>;}
async function page(filters:Record<string,string>={}){render(<UiLocaleProvider initialLocale="zh-CN"><LocaleSwitch/>{await ManagedLearnPage({searchParams:Promise.resolve(filters)})}</UiLocaleProvider>);}

it('names themes and separates labelled progress from the knowledge list',async()=>{
 await page();
 const navigation=screen.getByRole('navigation',{name:'按主题学习'});
 expect(within(navigation).getByText('线性代数与矩阵理论')).toBeVisible();
 expect(within(navigation).getByText('已完成 2 / 4')).toBeVisible();
 expect(within(navigation).getByRole('progressbar')).toHaveAccessibleName('线性代数与矩阵理论：已完成 2 / 4');
 expect(screen.getByRole('heading',{name:'知识点'})).toBeVisible();
 expect(within(screen.getByRole('region',{name:'知识点'})).getByText('共 4 个知识点')).toBeVisible();
 expect(screen.queryByText('15-XX',{exact:true})).toBeNull();
 expect(screen.queryByText('65-XX',{exact:true})).toBeNull();
 expect(screen.queryByText(/15A03/)).toBeNull();
 const stats=screen.getByRole('region',{name:'学习概览'});
 expect(within(stats).getByText('未学习')).toBeVisible();
 expect(within(stats).getByText('复习中')).toBeVisible();
});
it('switches theme names, knowledge titles and labels without discarding search input',async()=>{
 await page({q:'列空间'});fireEvent.change(screen.getByRole('searchbox',{name:'搜索知识点'}),{target:{value:'保留我的搜索'}});
 fireEvent.click(screen.getByRole('button',{name:'English'}));
 expect(screen.getByRole('heading',{name:'Knowledge points'})).toBeVisible();
 expect(screen.getByRole('link',{name:'Column space'})).toBeVisible();
 expect(within(screen.getByRole('navigation',{name:'Learn by theme'})).getByRole('progressbar')).toHaveAccessibleName('Linear algebra and matrix theory: 2 / 4 completed');
 expect(screen.getByRole('searchbox',{name:'Search knowledge'})).toHaveValue('保留我的搜索');
});
it('preserves review, search and state filters when choosing themes or paging',async()=>{
 await page({q:'linear',state:'completed',mode:'review',offset:'2'});
 const link=within(screen.getByRole('navigation',{name:'按主题学习'})).getByRole('link',{name:/线性代数/});
 const url=new URL(link.getAttribute('href')!,'https://example.test');
 expect(Object.fromEntries(url.searchParams)).toEqual({q:'linear',state:'completed',mode:'review',topicKey:'15-XX'});
 const next=new URL(screen.getByRole('link',{name:'下一页'}).getAttribute('href')!,'https://example.test');
 expect(next.searchParams.get('mode')).toBe('review');expect(next.searchParams.has('reviewOnly')).toBe(false);
 expect(screen.getByRole('combobox',{name:'主题'})).toBeVisible();
});
it('shows an actionable empty state and never changes learning data on a read',async()=>{
 request.mockImplementation(async(path:string)=>({ok:true,data:path.startsWith('/api/v3/study/overview')?overview:path.startsWith('/api/v3/study/topics')?topics:path.startsWith('/api/v3/topics')?directory:{...list,items:[],total:0}}));
 await page({state:'completed'});
 expect(screen.getByText('没有符合条件的知识点。')).toBeVisible();
 expect(screen.getByRole('link',{name:'清除筛选'})).toHaveAttribute('href','/learn');
 expect(request.mock.calls.every(c=>c[0].startsWith('/api/v3/'))).toBe(true);
});
it('fails closed when another actor owns the private list',async()=>{
 request.mockImplementation(async(path:string)=>({ok:true,data:path.startsWith('/api/v3/study/overview')?overview:path.startsWith('/api/v3/study/topics')?topics:path.startsWith('/api/v3/topics')?directory:{...list,items:[{...list.items[0],actorId:'someone-else'}]}}));
 await page();expect(screen.queryByText('列空间')).toBeNull();
});
it('keeps a specific theme filter named instead of silently widening it',async()=>{
 const current=request.getMockImplementation()!;
 request.mockImplementation((path:string)=>path.startsWith('/api/v3/topics/15A06')?Promise.resolve({ok:true,data:{...directory.items[0],topicKey:'15A06',title:'线性方程',titleEn:'Linear equations',kind:'specific'}}):current(path));
 await page({topicKey:'15A06',q:'方程'});
 expect(screen.getByRole('combobox',{name:'主题'})).toHaveValue('15A06');
 expect(screen.getByRole('option',{name:'线性方程'})).toBeVisible();
 expect(screen.queryByText('15A06',{exact:true})).toBeNull();
 expect(screen.getByRole('searchbox',{name:'搜索知识点'})).toHaveValue('方程');
});
it('does not render mismatched directory codes as theme names',async()=>{
 const current=request.getMockImplementation()!;
 request.mockImplementation((path:string)=>path.startsWith('/api/v3/topics')?Promise.resolve({ok:true,data:{...directory,items:[]}}):current(path));
 await page();expect(screen.queryByRole('navigation',{name:'按主题学习'})).toBeNull();expect(screen.queryByText('列空间')).toBeNull();
});
it.each(['15','msc-15','15-xx','MSC-15-XX'])('preserves the existing primary prefix or alias %s',async(topicKey)=>{
 await page({topicKey});
 expect(screen.getByRole('combobox',{name:'主题'})).toHaveValue('15-XX');
 expect(request).toHaveBeenCalledWith('/api/v3/study/knowledge?topicKey=15-XX','test-session');
});
it.each(['15A','msc-15a','15axx'])('preserves the existing secondary prefix or alias %s',async(topicKey)=>{
 const current=request.getMockImplementation()!;
 request.mockImplementation((path:string)=>path.startsWith('/api/v3/topics/15Axx')?Promise.resolve({ok:true,data:{...directory.items[0],topicKey:'15Axx',title:'基本线性代数',titleEn:'Basic linear algebra',kind:'secondary'}}):current(path));
 await page({topicKey});
 expect(screen.getByRole('combobox',{name:'主题'})).toHaveValue('15Axx');
 expect(screen.getByRole('option',{name:'基本线性代数'})).toBeVisible();
});
