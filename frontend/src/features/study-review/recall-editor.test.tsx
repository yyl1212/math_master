import {it,expect,vi} from 'vitest';import {render,screen,fireEvent} from '@testing-library/react';import {useState} from 'react';
import {UiLocaleProvider,useUiI18n} from '@/lib/i18n/provider';import {RecallEditor} from './recall-editor';
function Editor(){const[body,setBody]=useState('初始思路'),{setLocale}=useUiI18n();return <><button onClick={()=>setLocale('en')}>English</button><RecallEditor value={body} onChange={setBody} disabled={false} dirty/><a href="/learn/review">Queue</a></>}
it('TestRecallDraftNavigationAndLocale: preview is only own input and language retains it',()=>{
 render(<UiLocaleProvider initialLocale="zh-CN"><Editor/></UiLocaleProvider>);fireEvent.change(screen.getByRole('textbox',{name:'我的作答'}),{target:{value:'**自己写的** $x^2$'}});fireEvent.click(screen.getByRole('button',{name:'预览我的作答'}));expect(screen.getByText('自己写的',{selector:'strong'})).toBeVisible();expect(screen.queryByText('参考内容')).toBeNull();fireEvent.click(screen.getByRole('button',{name:'English'}));fireEvent.click(screen.getByRole('button',{name:'Write'}));expect(screen.getByRole('textbox',{name:'My answer'})).toHaveValue('**自己写的** $x^2$');
});
it('counts Unicode codepoints and reports exceeding16k without truncating',()=>{
 function Long(){const[value,setValue]=useState('');return <RecallEditor value={value} onChange={setValue} disabled={false} dirty/>}render(<UiLocaleProvider initialLocale="zh-CN"><Long/></UiLocaleProvider>);const input=screen.getByRole('textbox',{name:'我的作答'});fireEvent.change(input,{target:{value:'😀'.repeat(16000)}});expect(screen.queryByText('作答超过字符或字节限制。')).toBeNull();fireEvent.change(input,{target:{value:'字'.repeat(16001)}});expect(screen.getByText('作答超过字符或字节限制。')).toBeVisible();expect(input).toHaveValue('字'.repeat(16001));
});
