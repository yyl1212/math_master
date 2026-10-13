import {it,expect} from 'vitest';import {render,screen,fireEvent} from '@testing-library/react';import {UiLocaleProvider} from '@/lib/i18n/provider';import {Comparison} from './comparison';
import {session,date} from '@/lib/study-review/test-fixtures';
it('compares frozen own answer and original reference without editing/scoring',()=>{
 const value=session({status:'compared',sequence:2,draftRevision:1,comparedAt:date,answer:'本人答案',reference:{sections:[{kind:'statement',items:['原始参考']}],sha256:'c'.repeat(64)}});render(<UiLocaleProvider initialLocale="zh-CN"><Comparison session={value}/></UiLocaleProvider>);expect(screen.getByText('本人答案')).toBeVisible();expect(screen.getByText('原始参考')).toBeInTheDocument();expect(screen.queryByRole('textbox')).toBeNull();expect(screen.queryByText(/得分|答对|答错/)).toBeNull();fireEvent.click(screen.getByRole('button',{name:'参考内容'}));expect(screen.getByRole('region',{name:'参考内容'})).toBeVisible();
});
it('unavailable comparison keeps own answer and hides official reference',()=>{
 const value=session({available:false,status:'completed',sequence:3,draftRevision:1,comparedAt:date,gradedAt:date,answer:'保留本人答案'});render(<UiLocaleProvider initialLocale="zh-CN"><Comparison session={value}/></UiLocaleProvider>);expect(screen.getByText('保留本人答案')).toBeVisible();expect(screen.queryByRole('region',{name:'参考内容'})).toBeNull();
});
