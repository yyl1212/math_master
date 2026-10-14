import {it,expect,vi,afterEach} from 'vitest';
import {render,screen,fireEvent,waitFor} from '@testing-library/react';
import {UiLocaleProvider} from '@/lib/i18n/provider';
import {ReadingReview,SelfRating} from './self-rating';
import {reviewCommand,readSession} from '@/lib/study-review/client';
import {actorId,sessionId,session,receipt,date,schedule} from '@/lib/study-review/test-fixtures';
vi.mock('next/navigation',()=>({usePathname:()=>'/learn/review/'+sessionId}));
vi.mock('@/lib/study-review/client',()=>({reviewCommand:vi.fn(),readSession:vi.fn()}));
afterEach(()=>vi.resetAllMocks());
const reference={sections:[{kind:'statement' as const,items:['先阅读参考命题 $x^2\\ge0$']}],sha256:'c'.repeat(64)};
const reading=()=>session({mode:'read',answer:null,reference});
it('TestReadingReviewGrade: read → explicit compare → self-rating, without score',async()=>{
 vi.mocked(reviewCommand).mockResolvedValueOnce(receipt({action:'compare',status:'compared',sequence:2,draftRevision:1})).mockResolvedValueOnce(receipt({action:'grade',status:'completed',sequence:3,draftRevision:1,grade:{rating:'familiar',gradedAt:date,schedule:schedule()}}));
 vi.mocked(readSession).mockResolvedValueOnce({...reading(),status:'compared',sequence:2,draftRevision:1,comparedAt:date}).mockResolvedValueOnce({...reading(),status:'completed',sequence:3,draftRevision:1,comparedAt:date,gradedAt:date,grade:{rating:'familiar',gradedAt:date,schedule:schedule()},schedule:schedule()});
 render(<UiLocaleProvider initialLocale="zh-CN"><ReadingReview initial={reading()}/></UiLocaleProvider>);expect(screen.queryByRole('button',{name:'熟悉'})).toBeNull();fireEvent.click(screen.getByRole('button',{name:'完成阅读，进行自评'}));await screen.findByRole('button',{name:'熟悉'});fireEvent.click(screen.getByRole('button',{name:'熟悉'}));await screen.findByText('本次复习已完成');expect(screen.getByText('当次安排')).toBeVisible();expect(screen.getByText('当前下次复习')).toBeVisible();expect(screen.queryByText(/正确率|得分/)).toBeNull();
 expect(vi.mocked(reviewCommand).mock.calls[1][2]).toEqual({expectedSequence:2,rating:'familiar'});
});
it('TestReviewUncertainCommandsAndAccountSwitch: confirmed write/read failure only rereads and auth change clears stale state',async()=>{
 vi.mocked(reviewCommand).mockResolvedValue(receipt({action:'compare',status:'compared',sequence:2,draftRevision:1}));vi.mocked(readSession).mockRejectedValueOnce(new Error('read failed')).mockResolvedValue({...reading(),status:'compared',sequence:2,draftRevision:1,comparedAt:date});
 render(<UiLocaleProvider initialLocale="zh-CN"><ReadingReview initial={reading()}/></UiLocaleProvider>);fireEvent.click(screen.getByRole('button',{name:'完成阅读，进行自评'}));await screen.findByText('操作已确认，重新读取');fireEvent.click(screen.getByRole('button',{name:'重新读取'}));await screen.findByRole('button',{name:'熟悉'});expect(reviewCommand).toHaveBeenCalledTimes(1);expect(readSession).toHaveBeenCalledTimes(2);
 fireEvent(window,new Event('math-master:auth-change'));await waitFor(()=>expect(screen.queryByRole('button',{name:'熟悉'})).toBeNull());expect(screen.queryByText('先阅读参考命题')).toBeNull();
});
it('empty recall disables non-forgotten grades, and unfinished work cannot grade',()=>{
 const onGrade=vi.fn();render(<UiLocaleProvider initialLocale="zh-CN"><SelfRating session={session({status:'compared',sequence:2,draftRevision:1,comparedAt:date,unanswered:true})} onGrade={onGrade} busy={false}/></UiLocaleProvider>);expect(screen.getByRole('button',{name:'忘记'})).toBeEnabled();expect(screen.getByRole('button',{name:'不熟'})).toBeDisabled();expect(screen.getByRole('button',{name:'熟悉'})).toBeDisabled();
});

it('refreshing the same session after withdrawal hides official text immediately',()=>{
 const value=reading();const view=render(<UiLocaleProvider initialLocale="zh-CN"><ReadingReview initial={value}/></UiLocaleProvider>);expect(screen.getByRole('region',{name:'参考内容'})).toBeVisible();view.rerender(<UiLocaleProvider initialLocale="zh-CN"><ReadingReview initial={{...value,available:false,prompt:[],reference:null}}/></UiLocaleProvider>);expect(screen.queryByRole('region',{name:'参考内容'})).toBeNull();expect(screen.getByText('知识暂不可用')).toBeVisible();
});
it('unknown compare confirms the original key; account-change ignores a late result',async()=>{
 vi.mocked(reviewCommand).mockRejectedValueOnce(new Error('unconfirmed')).mockResolvedValue(receipt({action:'compare',status:'compared',sequence:2,draftRevision:1}));vi.mocked(readSession).mockResolvedValue({...reading(),status:'compared',sequence:2,draftRevision:1,comparedAt:date});
 const view=render(<UiLocaleProvider initialLocale="zh-CN"><ReadingReview initial={reading()}/></UiLocaleProvider>);fireEvent.click(screen.getByRole('button',{name:'完成阅读，进行自评'}));await screen.findByText('请求结果未确认，请确认原请求。');expect(screen.getByRole('button',{name:'完成阅读，进行自评'})).toBeDisabled();fireEvent.click(screen.getByRole('button',{name:'确认原请求'}));await screen.findByRole('button',{name:'熟悉'});expect(vi.mocked(reviewCommand).mock.calls[0][3]).toBe(vi.mocked(reviewCommand).mock.calls[1][3]);view.unmount();
 let resolve!:(value:ReturnType<typeof receipt>)=>void;vi.mocked(reviewCommand).mockImplementation(()=>new Promise(r=>{resolve=r}));render(<UiLocaleProvider initialLocale="zh-CN"><ReadingReview initial={reading()}/></UiLocaleProvider>);fireEvent.click(screen.getByRole('button',{name:'完成阅读，进行自评'}));fireEvent(window,new Event('math-master:auth-change'));resolve(receipt({action:'compare',status:'compared',sequence:2,draftRevision:1}));await waitFor(()=>expect(screen.queryByRole('button',{name:'熟悉'})).toBeNull());expect(screen.getByText('账户已变化，请重新打开本页。')).toBeVisible();
});
