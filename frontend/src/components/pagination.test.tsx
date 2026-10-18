import {it,expect,vi} from 'vitest';
import {render,screen,fireEvent,waitFor} from '@testing-library/react';
import {UiLocaleProvider} from '@/lib/i18n/provider';
import {Pagination} from './pagination';
vi.mock('next/navigation',()=>({useRouter:()=>({push:vi.fn()})}));
it('shows real page counts and navigates to the last page',async()=>{
 const change=vi.fn();render(<UiLocaleProvider initialLocale="zh-CN"><Pagination meta={{total:41,limit:20,offset:20}} label="知识" navigation={{kind:'callback',onPageChange:change}}/></UiLocaleProvider>);
 expect(screen.getByText('第 2 / 3 页，共 41 条')).toBeVisible();
 fireEvent.click(screen.getByRole('button',{name:'尾页'}));await waitFor(()=>expect(change).toHaveBeenCalledWith(40));
});
it('rejects fractional and out of range jumps and submits a valid page',async()=>{
 const change=vi.fn();render(<Pagination meta={{total:41,limit:20,offset:0}} label="Results" navigation={{kind:'callback',onPageChange:change}}/>);
 const input=screen.getByRole('textbox',{name:'Page number'});
 for(const value of ['','-1','1.5','4','9007199254740992']){fireEvent.change(input,{target:{value}});fireEvent.submit(input.closest('form')!);expect(change).not.toHaveBeenCalled();}
 fireEvent.change(input,{target:{value:'3'}});fireEvent.submit(input.closest('form')!);await waitFor(()=>expect(change).toHaveBeenCalledWith(40));
});
it('keeps empty and single-page controls visible and disabled',()=>{
 render(<Pagination meta={{total:0,limit:20,offset:0}} label="Results" navigation={{kind:'callback',onPageChange:vi.fn()}}/>);
 expect(screen.getByText('Page 0 / 0 · 0 results')).toBeVisible();expect(screen.getByRole('button',{name:'Next page'})).toBeDisabled();
});
