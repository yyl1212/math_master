import {render,screen} from '@testing-library/react';
import {it,expect,vi} from 'vitest';
import {UiLocaleProvider} from '@/lib/i18n/provider';
import {CurrentCatalogue,CurrentTopicPage} from './current-catalogue';
const calls=vi.hoisted(()=>({read:vi.fn()}));
vi.mock('@/lib/knowledge-admin/server-client',()=>({readManaged:calls.read}));
const empty={items:[],total:0,limit:20,offset:0};
const root={topicKey:'97-XX',title:'数学教育',kind:'primary',knowledgeCount:0,items:empty};
const secondary={...root,topicKey:'97Fxx',title:'算术与数论教育',kind:'secondary'};
const specific={...root,topicKey:'97F40',title:'整数与有理数教育',kind:'specific'};
function fixture(parent= root,child= secondary){calls.read.mockReset();calls.read.mockImplementation(async(path:string)=>({ok:true,data:path.startsWith('/api/v3/topics/')?parent:{items:[child],total:1,limit:100,offset:0}}))}
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
 expect(screen.getByRole('link',{name:/整数与有理数教育/})).toHaveAttribute('href','/topics/97F40');
 expect(screen.getAllByText('0 published knowledge points')).toHaveLength(2);
 expect(screen.getByText('This topic has no published knowledge yet.')).toBeVisible();
});
