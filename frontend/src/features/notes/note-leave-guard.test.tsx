import {it,expect,vi} from 'vitest';import {render,fireEvent,screen} from '@testing-library/react';import {NoteHistoryTracker,NoteLeaveGuard} from './note-leave-guard';
it('guards header-style links and leaves the draft on cancellation',()=>{
 const discard=vi.fn(),navigate=vi.fn();vi.spyOn(window,'confirm').mockReturnValue(false);render(<><NoteHistoryTracker/><NoteLeaveGuard dirty actorId="owner" onDiscard={discard}/><textarea defaultValue="unsaved note"/><a href="/knowledge" onClick={navigate}>Knowledge map</a></>);
 fireEvent.click(screen.getByRole('link'));expect(navigate).not.toHaveBeenCalled();expect(discard).not.toHaveBeenCalled();expect(screen.getByRole('textbox')).toHaveValue('unsaved note');expect(window.confirm).toHaveBeenCalledTimes(1);
});
it('allows same-page anchors and opening a new tab without discarding',()=>{
 const discard=vi.fn();vi.spyOn(window,'confirm').mockReturnValue(false);render(<><NoteLeaveGuard dirty actorId="owner" onDiscard={discard}/><a href="#section">Section</a><a href="/knowledge" target="_blank">New tab</a></>);fireEvent.click(screen.getByText('Section'));fireEvent.click(screen.getByText('New tab'));expect(window.confirm).not.toHaveBeenCalled();expect(discard).not.toHaveBeenCalled();
});
it('preserves the router history fields and restores traversal direction on cancellation',()=>{
 window.history.replaceState({__NA:true,tree:'unchanged'},'',window.location.href);const discard=vi.fn();vi.spyOn(window,'confirm').mockReturnValue(false);const go=vi.spyOn(window.history,'go').mockImplementation(()=>{});
 render(<><NoteHistoryTracker/><NoteLeaveGuard dirty actorId="owner" onDiscard={discard}/></>);window.history.pushState({__NA:true,tree:'note'},'',window.location.href);expect(window.history.state.tree).toBe('note');window.dispatchEvent(new PopStateEvent('popstate',{state:{__NA:true,tree:'unchanged',__mmNoteHistoryIndex:0}}));expect(go).toHaveBeenCalledWith(1);expect(discard).not.toHaveBeenCalled();
});
