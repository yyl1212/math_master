import {it,expect,vi} from 'vitest';
import {render,screen,fireEvent} from '@testing-library/react';
import {ReviewPanel} from './review-panel';
import {FeedbackAccountContext} from './feedback-account';
import {metadata,id} from '@/lib/feedback/test-fixtures';
vi.mock('@/lib/auth/client',()=>({getAuthContext:vi.fn()}));vi.mock('@/lib/feedback/client',()=>({sendFeedback:vi.fn()}));
it('a reviewer cannot handle their own report',()=>{render(<FeedbackAccountContext.Provider value={{actorId:id,invalidate:vi.fn()}}><ReviewPanel ticket={metadata()} onSuccess={vi.fn(async()=>{})}/></FeedbackAccountContext.Provider>);expect(screen.queryByRole('button',{name:'Save handling result'})).toBeNull()});
it('provides all eight explicit closure bases and existing proof fields',()=>{render(<FeedbackAccountContext.Provider value={{actorId:id,invalidate:vi.fn()}}><ReviewPanel ticket={{...metadata(),status:'processing',canHandle:true}} onSuccess={vi.fn(async()=>{})}/></FeedbackAccountContext.Provider>);fireEvent.change(screen.getByLabelText('Report status'),{target:{value:'resolved'}});expect(screen.getByLabelText('Resolution basis')).toBeVisible();fireEvent.change(screen.getByLabelText('Resolution basis'),{target:{value:'revision_published'}});expect(screen.getByLabelText('Withdrawal event ID')).toBeVisible();expect(screen.getByLabelText('Replacement publication ID')).toBeVisible();expect(screen.getByRole('button',{name:'Save handling result'})).toBeDisabled()});
