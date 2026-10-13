import {it,expect,vi,afterEach} from 'vitest';
import {render,screen,fireEvent,waitFor} from '@testing-library/react';
import {UiLocaleProvider,useUiI18n} from '@/lib/i18n/provider';
import {SessionPanel} from './session';
import {ReviewError} from '@/lib/study-review/types';
import {reviewCommand,readSession} from '@/lib/study-review/client';
import {session,receipt,date,actorId,knowledge} from '@/lib/study-review/test-fixtures';
vi.mock('next/navigation',()=>({usePathname:()=>'/learn/review/test',useRouter:()=>({refresh:vi.fn(),push:vi.fn()})}));
vi.mock('@/lib/study-review/client',()=>({reviewCommand:vi.fn(),readSession:vi.fn()}));
afterEach(()=>vi.resetAllMocks());
const reference={sections:[{kind:'statement' as const,items:['正确陈述与条件']}],sha256:'c'.repeat(64)};
function LocaleSwitch(){const{setLocale}=useUiI18n();return <button onClick={()=>setLocale('en')}>English</button>}
it('TestRecallFreezeAndCompare: own draft → save → frozen answer/reference → manual rating',async()=>{
 vi.mocked(reviewCommand).mockResolvedValueOnce(receipt({action:'draft',sequence:2,draftRevision:1})).mockResolvedValueOnce(receipt({action:'compare',status:'compared',sequence:3,draftRevision:2}));vi.mocked(readSession).mockResolvedValueOnce(session({answer:'我的作答 $x$',sequence:2,draftRevision:1,draftUpdatedAt:date})).mockResolvedValueOnce(session({answer:'我的作答 $x$',status:'compared',sequence:3,draftRevision:2,comparedAt:date,reference}));
 render(<UiLocaleProvider initialLocale="zh-CN"><SessionPanel initial={session()}/></UiLocaleProvider>);expect(screen.queryByText('正确陈述与条件')).toBeNull();fireEvent.change(screen.getByRole('textbox',{name:'我的作答'}),{target:{value:'我的作答 $x$'}});fireEvent.click(screen.getByRole('button',{name:'保存草稿'}));await waitFor(()=>expect(screen.getByRole('button',{name:'保存草稿'})).toBeEnabled());expect(vi.mocked(reviewCommand).mock.calls[0][2]).toEqual({expectedSequence:1,expectedDraftRevision:0,body:'我的作答 $x$'});fireEvent.click(screen.getByRole('button',{name:'答完，查看参考'}));await screen.findByText('正确陈述与条件');expect(screen.queryByRole('textbox',{name:'我的作答'})).toBeNull();expect(screen.getByRole('button',{name:'熟悉'})).toBeEnabled();expect(vi.mocked(reviewCommand).mock.calls[1][2]).toEqual({expectedSequence:2,body:'我的作答 $x$',unanswered:false});
});
it('TestProofPromptAndNoProof: prompt equations visible, expected proof absent before compare',()=>{
 const value=session({target:'proof',prompt:[{kind:'statement',items:['下式成立']},{kind:'equations',items:['$x^2>0$']} ]});render(<UiLocaleProvider initialLocale="zh-CN"><SessionPanel initial={value}/></UiLocaleProvider>);expect(screen.getByText('下式成立')).toBeVisible();expect(screen.getByRole('region',{name:'默写题干'})).toBeVisible();expect(screen.queryByRole('region',{name:'参考内容'})).toBeNull();
});
it('TestRecallChangedOrWithdrawnReference: all stages hide official content and retain own answer/draft',()=>{
 const value=session({target:'proof',answer:'我的未保存作答',prompt:[{kind:'statement',items:['旧题干']}]});const view=render(<UiLocaleProvider initialLocale="zh-CN"><SessionPanel initial={value}/></UiLocaleProvider>);fireEvent.change(screen.getByRole('textbox',{name:'我的作答'}),{target:{value:'继续修改的本地作答'}});view.rerender(<UiLocaleProvider initialLocale="zh-CN"><SessionPanel initial={{...value,available:false,prompt:[],reference:null}}/></UiLocaleProvider>);expect(screen.queryByText('旧题干')).toBeNull();expect(screen.getByRole('textbox',{name:'我的作答'})).toHaveValue('继续修改的本地作答');expect(screen.getByRole('button',{name:'答完，查看参考'})).toBeDisabled();expect(screen.getByRole('button',{name:'放弃本次会话'})).toBeEnabled();
});
it('blank answer uses explicit unanswered and only Forgotten; locale never discards typed input',async()=>{
 vi.mocked(reviewCommand).mockResolvedValue(receipt({action:'compare',status:'compared',sequence:2,draftRevision:1}));vi.mocked(readSession).mockResolvedValue(session({status:'compared',sequence:2,draftRevision:1,comparedAt:date,unanswered:true,reference}));render(<UiLocaleProvider initialLocale="zh-CN"><LocaleSwitch/><SessionPanel initial={session()}/></UiLocaleProvider>);expect(screen.getByRole('button',{name:'答完，查看参考'})).toBeDisabled();fireEvent.click(screen.getByRole('button',{name:'未写出，查看参考'}));await screen.findByRole('button',{name:'忘记'});expect(screen.getByRole('button',{name:'熟悉'})).toBeDisabled();expect(vi.mocked(reviewCommand).mock.calls[0][2]).toEqual({expectedSequence:1,body:'',unanswered:true});
});

it('forced-password response on manual refresh clears private answer immediately',async()=>{
 vi.mocked(readSession).mockRejectedValue(new ReviewError('PASSWORD_CHANGE_REQUIRED',403));render(<UiLocaleProvider initialLocale="zh-CN"><SessionPanel initial={session({answer:'这份私人作答必须清除'})}/></UiLocaleProvider>);fireEvent.click(screen.getByRole('button',{name:'刷新当前状态'}));await screen.findByText('账户已变化，请重新打开本页。');expect(screen.queryByDisplayValue('这份私人作答必须清除')).toBeNull();
});

it('TestRecallDraftNavigationAndLocale: first-edit cancelled link/menu preserves input and focus; refresh triggers unload guard',()=>{
 vi.spyOn(window,'confirm').mockReturnValue(false);const navigate=vi.fn();render(<UiLocaleProvider initialLocale="zh-CN"><SessionPanel initial={session()}/><a href="/knowledge" onClick={navigate}>Menu map</a></UiLocaleProvider>);const input=screen.getByRole('textbox',{name:'我的作答'});input.focus();fireEvent.change(input,{target:{value:'第一次编辑后的草稿'}});fireEvent.click(screen.getByRole('link',{name:'Menu map'}));expect(navigate).not.toHaveBeenCalled();expect(input).toHaveValue('第一次编辑后的草稿');expect(input).toHaveFocus();const unload=new Event('beforeunload',{cancelable:true});window.dispatchEvent(unload);expect(unload.defaultPrevented).toBe(true);
});
it('same-page preview/locale preserve draft and create no private browser-storage writes',()=>{
 const storage=vi.spyOn(Storage.prototype,'setItem');render(<UiLocaleProvider initialLocale="zh-CN"><LocaleSwitch/><SessionPanel initial={session()}/></UiLocaleProvider>);fireEvent.change(screen.getByRole('textbox',{name:'我的作答'}),{target:{value:'只留在内存的私人草稿'}});fireEvent.click(screen.getByRole('button',{name:'预览我的作答'}));fireEvent.click(screen.getByRole('button',{name:'English'}));fireEvent.click(screen.getByRole('button',{name:'Write'}));expect(screen.getByRole('textbox',{name:'My answer'})).toHaveValue('只留在内存的私人草稿');expect(storage.mock.calls.every(call=>!JSON.stringify(call).includes('只留在内存的私人草稿'))).toBe(true);expect(reviewCommand).not.toHaveBeenCalled();
});
