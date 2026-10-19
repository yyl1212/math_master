import {it,expect,vi} from 'vitest';
vi.mock('next/navigation',()=>({useRouter:()=>({push:vi.fn(),refresh:vi.fn()}),usePathname:()=>'/learn/review'}));
import {render,screen,fireEvent} from '@testing-library/react';
import {UiLocaleProvider,useUiI18n} from '@/lib/i18n/provider';
import {ReviewQueue} from './queue';
import {actorId,date,knowledge} from '@/lib/study-review/test-fixtures';
import type {Overview,QueuePage} from '@/lib/study-review/types';
const overview:Overview={actorId,asOf:date,total:1,unfinished:0,changed:0,due:0,unplanned:1,future:0};
const page:QueuePage={actorId,asOf:date,items:[{knowledge,title:'Vector space',titleZh:'向量空间',knowledgeType:'concept',topicKeys:['15A03'],bucket:'unplanned',materialChanged:false,proofAvailable:false,activeSessionId:null,schedule:null}],total:1,limit:20,offset:0};
function LocaleSwitch(){const{setLocale}=useUiI18n();return <button onClick={()=>setLocale('en')}>English</button>}
it('TestReviewWorkspaceSections: exclusive counts, named queue, filters and reading/recall actions',()=>{
 render(<UiLocaleProvider initialLocale="zh-CN"><LocaleSwitch/><ReviewQueue overview={overview} page={page} filters={{bucket:'unplanned'}}/></UiLocaleProvider>);
 expect(screen.getByRole('heading',{name:'复习工作区'})).toBeVisible();for(const text of ['现在待复习','待安排','未完成','全部'])expect(screen.getByRole('link',{name:new RegExp(text)})).toBeVisible();
 expect(screen.getByRole('link',{name:'向量空间'})).toHaveAttribute('href','/knowledge/'+knowledge.id);
 expect(screen.getByRole('button',{name:'阅读复习'})).toBeEnabled();expect(screen.getByRole('button',{name:'陈述与条件默写'})).toBeEnabled();expect(screen.queryByRole('button',{name:'证明默写'})).toBeNull();
 expect(screen.getByRole('combobox',{name:'每页条数'})).toHaveValue('20');expect(screen.getByText(/按服务器此次查询时间/)).toBeVisible();
 fireEvent.change(screen.getByRole('searchbox',{name:'搜索知识点'}),{target:{value:'保留筛选'}});fireEvent.click(screen.getByRole('button',{name:'English'}));expect(screen.getByRole('link',{name:'Vector space'})).toBeVisible();expect(screen.getByRole('searchbox',{name:'Search knowledge'})).toHaveValue('保留筛选');
});
it('paging keeps filters and does not expand an empty queue into other buckets',()=>{
 render(<UiLocaleProvider initialLocale="zh-CN"><ReviewQueue overview={overview} page={{...page,items:[],total:41,limit:20,offset:20}} filters={{bucket:'future',q:'公式',topicKey:'15-XX',offset:'20'}}/></UiLocaleProvider>);
 const next=new URL(screen.getByRole('link',{name:'下一页'}).getAttribute('href')!,'https://example.test');expect(Object.fromEntries(next.searchParams)).toEqual({bucket:'future',q:'公式',topicKey:'15-XX',limit:'20',page:'3'});expect(screen.getByText('此队列暂无知识点。')).toBeVisible();
});
