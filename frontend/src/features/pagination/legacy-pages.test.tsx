import {render,screen,fireEvent,waitFor,within} from '@testing-library/react';
import {it,expect,vi} from 'vitest';
import {DraftList} from '@/features/content/draft-list';
import {PublicationPanel} from '@/features/content/publication-panel';
const mocks=vi.hoisted(()=>({request:vi.fn(),push:vi.fn(),refresh:vi.fn(),error:vi.fn()}));
vi.mock('@/lib/content/client',()=>({contentRequest:mocks.request}));
vi.mock('next/navigation',()=>({useRouter:()=>mocks}));
vi.mock('@/features/content/command-controls',()=>({useContentCommand:()=>({busy:false,controls:null,error:mocks.error}),PasswordDialog:()=>null}));
it('legacy accessible lists use shared pagination and serialize page requests',async()=>{
 let done!:(v:unknown)=>void;mocks.request.mockReturnValueOnce(new Promise(r=>{done=r}));
 render(<DraftList initial={{items:[],total:41,limit:20,offset:0}} canEdit={false}/>);
 expect(screen.getByText('Page 1 / 3 · 41 results')).toBeVisible();
 fireEvent.click(screen.getByRole('button',{name:'Last page'}));fireEvent.click(screen.getByRole('button',{name:'Last page'}));
 expect(mocks.request).toHaveBeenCalledTimes(1);expect(mocks.request).toHaveBeenCalledWith({kind:'listDrafts',query:{scope:'all',limit:20,offset:40}});
 done({ok:true,data:{items:[],total:41,limit:20,offset:40}});await screen.findByText('Page 3 / 3 · 41 results');
});
it('two publication lists keep independent page',async()=>{
 mocks.request.mockReset();mocks.request.mockResolvedValue({ok:true,data:{items:[],total:41,limit:20,offset:0}});
 render(<PublicationPanel initial={{items:[],head:null,total:1,limit:20,offset:0}}/>);
 await screen.findByText('Page 1 / 3 · 41 results');expect(screen.getByText('Page 1 / 1 · 1 results')).toBeVisible();
 const pages=screen.getAllByRole('navigation');fireEvent.click(within(pages[0]).getByRole('button',{name:'Last page'}));
 await waitFor(()=>expect(mocks.request).toHaveBeenLastCalledWith({kind:'listSubmissions',query:{scope:'all',status:'approved',limit:20,offset:40}}));
 expect(within(pages[1]).getByRole('button',{name:'Last page'})).toBeDisabled();
});
