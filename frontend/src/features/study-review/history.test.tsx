import {it,expect,vi} from 'vitest';
vi.mock('next/navigation',()=>({useRouter:()=>({push:vi.fn(),refresh:vi.fn()}),usePathname:()=>'/learn/review/history'}));
import {render,screen} from '@testing-library/react';
import {UiLocaleProvider} from '@/lib/i18n/provider';
import {ReviewHistory} from './history';
import {actorId,date,knowledge,sessionId,schedule} from '@/lib/study-review/test-fixtures';
it('TestReviewHistoryPrivateMetadata: self-ratings retain their historical schedule with private detail link',()=>{
 const page={actorId,asOf:date,items:[{sessionId,knowledge,title:'Statement',titleZh:'命题',knowledgeType:'theorem' as const,mode:'recall' as const,target:'statement' as const,grade:{rating:'familiar' as const,gradedAt:date,schedule:schedule()},available:false,materialChanged:true}],nextCursor:'next-page'};
 render(<UiLocaleProvider initialLocale="zh-CN"><ReviewHistory page={page} filters={{mode:'recall'}}/></UiLocaleProvider>);expect(screen.getByRole('heading',{name:'自评历史'})).toBeVisible();expect(screen.getByRole('link',{name:'命题'})).toHaveAttribute('href','/learn/review/'+sessionId);expect(screen.getByText('熟悉',{selector:'span'})).toBeVisible();expect(screen.getByText('知识暂不可用')).toBeVisible();expect(screen.getByRole('link',{name:'原学习历史'})).toHaveAttribute('href','/learn?view=history');const next=new URL(screen.getByRole('link',{name:'下一页'}).getAttribute('href')!,'https://example.test');expect(next.searchParams.get('mode')).toBe('recall');expect(next.searchParams.get('cursor')).toBe('next-page');
});
