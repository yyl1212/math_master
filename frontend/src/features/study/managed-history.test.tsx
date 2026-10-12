import {it,expect,vi} from 'vitest';import {render,screen,fireEvent} from '@testing-library/react';import {ManagedHistory} from './managed-history';import {UiLocaleProvider} from '@/lib/i18n/provider';
vi.mock('@/components/topic-picker',()=>({TopicPicker:()=> <span>主题名称选择</span>}));
const actor='51390b7d-a8e9-4e98-a3ae-22101c20b309',id='k-'+'a'.repeat(56);
const event={id:actor,title:'Vector space',titleZh:'向量空间',available:false,knowledge:{id,contentSha256:'a'.repeat(64),sourceKind:'managed' as const},topicKeys:['15A03'],kind:'completed' as const,noteRevision:null,reviewId:null,recordedAt:'2026-10-10T08:00:00Z',sourceKind:'managed' as const};
it('keeps unavailable history named but unlinked, and keeps the other paging cursor',()=>{render(<UiLocaleProvider initialLocale="zh-CN"><ManagedHistory page={{actorId:actor,items:[event],nextCursor:'current-next'}} filters={{cursor:'current-old',legacyCursor:'early-old'}}/></UiLocaleProvider>);expect(screen.getByText(/向量空间/)).toBeVisible();expect(screen.queryByRole('link',{name:'向量空间'})).toBeNull();expect(screen.queryByRole('textbox',{name:'数据ID'})).toBeNull();expect(screen.getByRole('link',{name:'下一页'})).toHaveAttribute('href','/learn?view=history&cursor=current-next&legacyCursor=early-old');fireEvent.click(screen.getByText('查看记录标识与主题归属'));expect(screen.getByText(/15A03/)).toBeVisible();});
it('keeps date and knowledge filters inside learning when submitting a new history filter',()=>{
 render(<UiLocaleProvider initialLocale="zh-CN"><ManagedHistory page={{actorId:actor,items:[],nextCursor:null}} filters={{knowledgeId:id,from:'2026-10-01T00:00:00Z',to:'2026-10-11T00:00:00Z',cursor:'old-page',legacyCursor:'early-page'}}/></UiLocaleProvider>);
 expect(screen.getByRole('heading',{level:1,name:'我的学习'})).toBeVisible();expect(screen.getByRole('heading',{level:2,name:'学习历史'})).toBeVisible();
 const form=screen.getByRole('button',{name:'检索知识点'}).closest('form')!;
 expect(form).toHaveAttribute('action','/learn');expect(Object.fromEntries(new FormData(form))).toMatchObject({view:'history',knowledgeId:id,from:'2026-10-01T00:00:00Z',to:'2026-10-11T00:00:00Z'});
 expect(new FormData(form).has('cursor')).toBe(false);expect(new FormData(form).has('legacyCursor')).toBe(false);
 expect(screen.getByRole('link',{name:'清除筛选'})).toHaveAttribute('href','/learn?view=history');
});
it('clears a changed action before it has been submitted to the history URL',()=>{
 render(<UiLocaleProvider initialLocale="zh-CN"><ManagedHistory page={{actorId:actor,items:[],nextCursor:null}}/></UiLocaleProvider>);
 const action=screen.getByRole('combobox',{name:'操作'});fireEvent.change(action,{target:{value:'started'}});expect(action).toHaveValue('started');const clear=screen.getByRole('link',{name:'清除筛选'});clear.addEventListener('click',event=>event.preventDefault());fireEvent.click(clear);expect(action).toHaveValue('');
});
