import {render,screen,fireEvent} from '@testing-library/react';
import {it,expect,vi} from 'vitest';
import {UiLocaleProvider,useUiI18n} from '@/lib/i18n/provider';
import {CurrentCatalogue,CurrentTopicPage} from './current-catalogue';
const calls=vi.hoisted(()=>({read:vi.fn()}));
vi.mock('@/lib/knowledge-admin/server-client',()=>({readManaged:calls.read}));
const empty={items:[],total:0,limit:20,offset:0};
const root={topicKey:'97-XX',title:'数学教育',titleEn:'Mathematics education' as string|undefined,kind:'primary',knowledgeCount:0,items:empty};
const secondary={...root,topicKey:'97Fxx',title:'算术与数论教育',titleEn:'Arithmetic and number theory education',kind:'secondary'};
const specific={...root,topicKey:'97F40',title:'整数与有理数教育',titleEn:'Integers and rational numbers education',kind:'specific'};
function fixture(parent:unknown=root,child:unknown=secondary){calls.read.mockReset();calls.read.mockImplementation(async(path:string)=>({ok:true,data:path.startsWith('/api/v3/topics/')?parent:{items:[child],total:1,limit:100,offset:0}}))}
it('shows named themes and published count units without internal classification codes',async()=>{
 fixture(root,root);render(<UiLocaleProvider initialLocale="zh-CN">{await CurrentCatalogue({})}</UiLocaleProvider>);
 expect(screen.getByRole('link',{name:/数学教育/})).toHaveAttribute('href','/topics/97-XX');
 expect(screen.getByText('已发布 0 个知识点')).toBeVisible();
 expect(document.body.textContent).not.toContain('97-XX');
 expect(screen.getByRole('textbox',{name:'检索主题'})).toBeVisible();
});
it('shows second-level topics even when their parent has no published knowledge',async()=>{
 fixture();render(<UiLocaleProvider initialLocale="zh-CN">{await CurrentTopicPage({topicKey:'97-XX'})}</UiLocaleProvider>);
 expect(screen.getByRole('heading',{name:'二级主题'})).toBeVisible();
 expect(screen.getByRole('link',{name:/算术与数论教育/})).toHaveAttribute('href','/topics/97Fxx');
 expect(screen.getByRole('heading',{name:'可阅读知识点'})).toBeVisible();
 expect(screen.getByText('此主题暂无已发布知识点。')).toBeVisible();
 expect(document.body.textContent).not.toMatch(/97-XX|97Fxx/);
 expect(calls.read).toHaveBeenCalledWith('/api/v3/topics?topicKey=97-XX&limit=100&offset=0');
});
it('labels specific-topic navigation separately from empty knowledge in English',async()=>{
 fixture(secondary,specific);render(<UiLocaleProvider initialLocale="en">{await CurrentTopicPage({topicKey:'97Fxx'})}</UiLocaleProvider>);
 expect(screen.getByRole('heading',{name:'Specific themes'})).toBeVisible();
 expect(screen.getByRole('link',{name:/Integers and rational numbers education/})).toHaveAttribute('href','/topics/97F40');
 expect(screen.getAllByText('0 published knowledge points')).toHaveLength(2);
 expect(screen.getByText('This topic has no published knowledge yet.')).toBeVisible();
});

function Switch(){const{setLocale}=useUiI18n();return <><button onClick={()=>setLocale('en')}>English</button><button onClick={()=>setLocale('zh-CN')}>中文</button></>}
it('switches primary topic names immediately without changing search input or links',async()=>{
 fixture(root,root);render(<UiLocaleProvider initialLocale="en"><Switch/>{await CurrentCatalogue({q:'97'})}</UiLocaleProvider>);
 expect(screen.getByRole('link',{name:/Mathematics education/})).toHaveAttribute('href','/topics/97-XX');
 const input=screen.getByRole('textbox');fireEvent.change(input,{target:{value:'保留我的搜索'}});
 fireEvent.click(screen.getByRole('button',{name:'中文'}));
 expect(screen.getByRole('link',{name:/数学教育/})).toHaveAttribute('href','/topics/97-XX');
 expect(input).toHaveValue('保留我的搜索');
 fireEvent.click(screen.getByRole('button',{name:'English'}));
 expect(screen.getByRole('link',{name:/Mathematics education/})).toBeVisible();
 expect(input).toHaveValue('保留我的搜索');
});
it('switches detail and specific theme names, with bilingual knowledge titles',async()=>{
 const parent={...secondary,items:{...empty,total:1,items:[{id:'k-'+'a'.repeat(56),point:{title:'Linear systems',title_zh:'线性方程组',type:'definition',learning_difficulty:{difficulty_level:2}}}]}};
 fixture(parent,specific);render(<UiLocaleProvider initialLocale="en"><Switch/>{await CurrentTopicPage({topicKey:'97Fxx'})}</UiLocaleProvider>);
 expect(screen.getByRole('heading',{name:'Arithmetic and number theory education'})).toBeVisible();
 expect(screen.getByRole('link',{name:/Integers and rational numbers education/})).toHaveAttribute('href','/topics/97F40');
 expect(screen.getByRole('link',{name:'Linear systems'})).toBeVisible();
 fireEvent.click(screen.getByRole('button',{name:'中文'}));
 expect(screen.getByRole('heading',{name:'算术与数论教育'})).toBeVisible();
 expect(screen.getByRole('link',{name:/整数与有理数教育/})).toHaveAttribute('href','/topics/97F40');
 expect(screen.getByRole('link',{name:'线性方程组'})).toBeVisible();
});
it('switches project other and accepts older responses without an English field',async()=>{
 const other={...root,topicKey:'project:other',title:'项目其他',titleEn:'Project other',kind:'project-other'};
 fixture(other,other);const view=render(<UiLocaleProvider initialLocale="en"><Switch/>{await CurrentTopicPage({topicKey:'project:other'})}</UiLocaleProvider>);
 expect(screen.getByRole('heading',{name:'Project other'})).toBeVisible();
 fireEvent.click(screen.getByRole('button',{name:'中文'}));expect(screen.getByRole('heading',{name:'项目其他'})).toBeVisible();
 view.unmount();fixture({...root,titleEn:undefined},root);render(<UiLocaleProvider initialLocale="en">{await CurrentTopicPage({topicKey:'97-XX'})}</UiLocaleProvider>);
 expect(screen.getByRole('heading',{name:'数学教育'})).toBeVisible();
});
