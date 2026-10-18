import {it,expect,vi} from 'vitest';
import {useState} from 'react';
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
it('fetches the actual last available page for an out of range legacy offset',async()=>{
 const change=vi.fn();function Results(){const[page,setPage]=useState({total:5,limit:20,offset:200,items:[] as number[]});return <><ul>{page.items.map(id=><li key={id}>Result {id}</li>)}</ul><Pagination meta={page} label="Results" navigation={{kind:'callback',onPageChange:async offset=>{change(offset);setPage({total:5,limit:20,offset,items:[1,2,3,4,5]})}}}/></>}
 render(<Results/>);await waitFor(()=>expect(change).toHaveBeenCalledExactlyOnceWith(0));
 expect(screen.getAllByRole('listitem')).toHaveLength(5);expect(screen.getByText('Page 1 / 1 · 5 results')).toBeVisible();
});
it('does not loop or display a false page summary when correction fails',async()=>{
 const change=vi.fn().mockRejectedValue(new Error('offline'));const props={meta:{total:5,limit:20,offset:200},label:'Results',navigation:{kind:'callback' as const,onPageChange:change}};
 const view=render(<Pagination {...props}/>);expect(await screen.findByRole('alert')).toHaveTextContent('Unable to change page');
 view.rerender(<Pagination {...props}/>);expect(change).toHaveBeenCalledTimes(1);expect(screen.queryByText('Page 1 / 1 · 5 results')).toBeNull();
 const input=screen.getByRole('textbox',{name:'Page number'});fireEvent.submit(input.closest('form')!);await waitFor(()=>expect(change).toHaveBeenCalledTimes(2));
});
it('corrects an old link once and retains its applied business filters',async()=>{
 const click=vi.spyOn(HTMLAnchorElement.prototype,'click').mockImplementation(()=>{});
 try{const props={meta:{total:41,limit:20,offset:200},label:'Results',navigation:{kind:'link' as const,pathname:'/admin/knowledge',query:{offset:'200',q:'applied',cursor:'old'},pageMode:true}};
 const view=render(<Pagination {...props}/>);await waitFor(()=>expect(click).toHaveBeenCalledTimes(1));
 const anchor=click.mock.instances[0] as HTMLAnchorElement;expect(anchor.getAttribute('href')).toBe('/admin/knowledge?q=applied&page=3');
 expect(screen.queryByText('Page 3 / 3 · 41 results')).toBeNull();view.rerender(<Pagination {...props}/>);expect(click).toHaveBeenCalledTimes(1);
 }finally{click.mockRestore()}
});
