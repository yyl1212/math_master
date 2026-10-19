import {beforeEach,it,expect,vi} from 'vitest';import {render,screen,fireEvent,waitFor,within} from '@testing-library/react';import {PublishBatch} from './publish-batch';
const mocks=vi.hoisted(()=>({create:vi.fn(),list:vi.fn(),read:vi.fn(),retry:vi.fn(),run:vi.fn(),auth:vi.fn()}));
vi.mock('@/lib/knowledge-admin/publish-batch',()=>({createPublishJob:mocks.create,listPublishJobs:mocks.list,readPublishJob:mocks.read,readPublishRetryKnowledge:mocks.retry}));
vi.mock('@/lib/auth/client',()=>({getAuthContext:mocks.auth}));
vi.mock('@/lib/knowledge-admin/publish-runner',()=>({runPublishJob:mocks.run}));
const actorId='11111111-1111-4111-8111-111111111111';
beforeEach(()=>{vi.resetAllMocks();sessionStorage.clear()});
it('loads collapsed personal operations only after opening the panel',async()=>{
 mocks.list.mockResolvedValue({actorId,items:[],total:0,limit:20,offset:0});
 render(<PublishBatch actorId={actorId} selection={new Map()} onStarted={vi.fn()} onSettled={vi.fn()}/>);
 const summary=await screen.findByText('My publishing operations',{selector:'summary'});
 expect(mocks.list).not.toHaveBeenCalled();fireEvent.click(summary);
 await waitFor(()=>expect(mocks.list).toHaveBeenCalledTimes(1));
});
it('full publish ignores current filters and uses server current progress',async()=>{const job={actorId,id:actorId,mode:'all',sequence:1,total:0,pending:0,published:0,skipped:0,failed:0,createdAt:'2026-10-19T00:00:00Z'};mocks.list.mockResolvedValue({actorId,items:[],total:0,limit:20,offset:0});mocks.create.mockResolvedValue(job);mocks.read.mockResolvedValue({job,items:[],total:0,limit:20,offset:0});mocks.run.mockResolvedValue(job);render(<PublishBatch actorId={actorId} selection={new Map()} onStarted={vi.fn()} onSettled={vi.fn()}/>);fireEvent.click(screen.getByRole('button',{name:'Publish all unpublished knowledge'}));await waitFor(()=>expect(mocks.create).toHaveBeenCalled());expect(mocks.create.mock.calls[0][1]).toEqual({mode:'all'});expect(mocks.run).toHaveBeenCalledTimes(1)});

it('account changes cancel recovery and clear actor-specific saved metadata',async()=>{sessionStorage.clear();mocks.list.mockResolvedValue({actorId,items:[],total:0,limit:20,offset:0});mocks.auth.mockResolvedValue({ok:true,data:{user:{id:'22222222-2222-4222-8222-222222222222',roles:['admin']}}});const invalidated=vi.fn();render(<PublishBatch actorId={actorId} selection={new Map()} onStarted={vi.fn()} onSettled={vi.fn()} onInvalidated={invalidated}/>);sessionStorage.setItem('math-master:publish:'+actorId,JSON.stringify({createKey:'original',jobId:actorId}));window.dispatchEvent(new Event('math-master:auth-change'));await screen.findByRole('alert');expect(invalidated).toHaveBeenCalledOnce();expect(sessionStorage.getItem('math-master:publish:'+actorId)).toBeNull();expect(screen.queryByRole('button',{name:'Publish all unpublished knowledge'})).toBeNull()});

it('keeps the recent operation page after navigation and inspecting its job',async()=>{
 const job={actorId,id:actorId,mode:'all',sequence:1,total:999,pending:0,published:999,skipped:0,failed:0,createdAt:'2026-10-19T00:00:00Z'};
 mocks.list.mockImplementation(async(_actor,query)=>({actorId,items:[{...job,total:query.offset===20?999:1}],total:21,limit:20,offset:query.offset}));
 mocks.read.mockResolvedValue({job,items:[],total:0,limit:20,offset:0});
 render(<PublishBatch actorId={actorId} selection={new Map()} onStarted={vi.fn()} onSettled={vi.fn()}/>);
 const summary=await screen.findByText('My publishing operations',{selector:'summary'});fireEvent.click(summary);
 const recent=summary.closest('details')!;fireEvent.click(await within(recent).findByRole('button',{name:'Next page'}));
 await waitFor(()=>expect(within(recent).getByText('Page 2 / 2 · 21 results')).toBeVisible());
 expect(within(recent).getByText(/999 total, 0 pending/)).toBeVisible();
 fireEvent.click(within(recent).getByRole('button',{name:'View / recover'}));
 await waitFor(()=>expect(mocks.read).toHaveBeenCalled());
 await waitFor(()=>expect(within(recent).getByRole('button',{name:'View / recover'})).toBeEnabled());
 expect(within(recent).getByText('Page 2 / 2 · 21 results')).toBeVisible();
 expect(mocks.list.mock.calls.at(-1)?.[1].offset).toBe(20);
});

it('checks beyond the first 1000 original failures after they were published by retry jobs',async()=>{
 const job={actorId,id:actorId,mode:'all',sequence:52,total:1001,pending:0,published:0,skipped:0,failed:1001,createdAt:'2026-10-19T00:00:00Z'};
 const items=Array.from({length:1001},(_,index)=>({id:index.toString(16).padStart(8,'0')+'-1111-4111-8111-111111111111',title:'Failed '+index,state:'failed',errorCode:'EDIT_TOKEN_STALE'}));
 mocks.list.mockResolvedValue({actorId,items:[job],total:1,limit:20,offset:0});
 mocks.read.mockImplementation(async(_actor,_job,query)=>({job,items:items.slice(query.offset,query.offset+query.limit),total:1001,limit:query.limit,offset:query.offset}));
 mocks.retry.mockImplementation(async(_actor,id)=>({id,title:'Current '+id,editToken:'fresh',published:id!==items[1000].id,deleted:false}));
 sessionStorage.setItem('math-master:publish:'+actorId,JSON.stringify({jobId:actorId}));
 render(<PublishBatch actorId={actorId} selection={new Map()} onStarted={vi.fn()} onSettled={vi.fn()}/>);
 fireEvent.click(await screen.findByRole('button',{name:'Check failed items after corrections'}));
 expect(await screen.findByRole('button',{name:'Publish these 1 failed items'})).toBeEnabled();
 expect(mocks.read.mock.calls.some(call=>call[2].offset===1000)).toBe(true);
 expect(mocks.create).not.toHaveBeenCalled();
});
