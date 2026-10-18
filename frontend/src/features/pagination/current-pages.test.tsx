import {render,screen,within} from '@testing-library/react';
import {it,expect,vi} from 'vitest';
import {UiLocaleProvider} from '@/lib/i18n/provider';
import {CurrentCatalogue,CurrentTopicPage} from '@/features/catalogue/current-catalogue';
const read=vi.hoisted(()=>vi.fn());
vi.mock('@/lib/knowledge-admin/server-client',()=>({readManaged:read}));
it.each([0,1,41])('current pages show accurate pagination for %s rows',async(total)=>{
 read.mockResolvedValue({ok:true,data:{items:[],total,limit:20,offset:total===41?20:0}});
 render(<UiLocaleProvider initialLocale="zh-CN">{await CurrentCatalogue({mode:'knowledge',offset:total===41?20:0})}</UiLocaleProvider>);
 expect(screen.getByText(`第 ${total===0?0:total===41?2:1} / ${Math.ceil(total/20)} 页，共 ${total} 条`)).toBeVisible();
 if(total===41)expect(screen.getByRole('link',{name:'尾页'})).toHaveAttribute('href','/knowledge?mode=knowledge&page=3');
});
it('independent topic and knowledge paging preserve the other section',async()=>{
 read.mockImplementation(async(path:string)=>({ok:true,data:path.startsWith('/api/v3/topics/')?{topicKey:'97-XX',title:'数学教育',titleEn:'Mathematics',kind:'primary',knowledgeCount:41,items:{items:[],total:41,limit:20,offset:20}}:{items:[],total:201,limit:100,offset:0}}));
 render(<UiLocaleProvider initialLocale="zh-CN">{await CurrentTopicPage({topicKey:'97-XX',searchParams:{knowledgePage:'2',q:'test'}})}</UiLocaleProvider>);
 const sections=screen.getAllByRole('navigation');
 const first=within(sections[0]).getByRole('link',{name:'下一页'});
 expect(first.getAttribute('href')).toContain('knowledgePage=2');expect(first.getAttribute('href')).toContain('page=2');
 expect(within(sections[1]).getByRole('link',{name:'下一页'}).getAttribute('href')).toContain('knowledgePage=3');
});
