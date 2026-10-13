import {it,expect,vi,beforeEach} from 'vitest';import {render,screen,fireEvent,waitFor} from '@testing-library/react';
import {ManagedNoteEditor} from '@/features/study/managed-note-editor';import {UiLocaleProvider} from '@/lib/i18n/provider';import {LanguageSwitch} from '@/components/language-switch';
import {NoteShareError} from '@/lib/note-sharing/types';import {KnowledgeError} from '@/lib/knowledge-admin/types';
const mocks=vi.hoisted(()=>({save:vi.fn(),share:vi.fn()}));vi.mock('@/lib/knowledge-admin/client',()=>({managedRequest:mocks.save}));vi.mock('@/lib/note-sharing/client',()=>({noteShareRequest:mocks.share}));
const actor='51390b7d-a8e9-4e98-a3ae-22101c20b309',kid='k-'+'a'.repeat(56),ref={id:kid,contentSha256:'a'.repeat(64),sourceKind:'managed' as const};
const note=()=>({actorId:actor,knowledgeId:kid,body:'原有私人笔记',revision:1,knowledge:ref,deleted:false,available:true,updatedAt:'2026-10-13T00:00:00Z'});
const head=()=>({actorId:actor,knowledgeId:kid,status:'unshared',sequence:0,submissionId:null,approvedAt:null,lastDecisionReason:''});
beforeEach(()=>{mocks.save.mockReset();mocks.share.mockReset();mocks.share.mockResolvedValue(head())});
it('previews formulas without saving and retains draft across language changes',async()=>{
 const{container}=render(<UiLocaleProvider initialLocale="en"><LanguageSwitch/><ManagedNoteEditor initial={note()} knowledge={ref}/></UiLocaleProvider>);
 const input=screen.getByRole('textbox',{name:'Private note'});fireEvent.change(input,{target:{value:'自己的 **理解** $x^2$'}});fireEvent.click(screen.getByRole('button',{name:'Preview'}));expect(container.querySelector('.katex')).not.toBeNull();expect(mocks.save).not.toHaveBeenCalled();
 fireEvent.click(screen.getByRole('button',{name:'中文'}));fireEvent.click(screen.getByRole('button',{name:'编辑'}));expect(screen.getByRole('textbox',{name:'私人笔记'})).toHaveValue('自己的 **理解** $x^2$');
});
it('wraps the selected text and blocks a Unicode-overlong save',()=>{
 render(<ManagedNoteEditor initial={note()} knowledge={ref}/>);const input=screen.getByRole('textbox',{name:'Private note'}) as HTMLTextAreaElement;input.setSelectionRange(0,2);fireEvent.select(input);fireEvent.click(screen.getByRole('button',{name:'Bold'}));expect(input).toHaveValue('**原有**私人笔记');fireEvent.change(input,{target:{value:'😀'.repeat(16001)}});expect(screen.getByRole('button',{name:'Save note'})).toBeDisabled();
});
it('requires withdrawal before editing a shared note',async()=>{
 mocks.share.mockResolvedValue({...head(),status:'shared',sequence:2,submissionId:actor,approvedAt:'2026-10-13T00:00:00Z'});render(<ManagedNoteEditor initial={note()} knowledge={ref}/>);
 await screen.findByRole('button',{name:'Withdraw share'});expect(screen.getByRole('textbox',{name:'Private note'})).toBeDisabled();mocks.share.mockImplementation(async(_path:string,method?:string)=>method==='POST'?{submissionId:actor,knowledgeId:kid,action:'withdraw',status:'withdrawn',sequence:3}:{...head(),status:'withdrawn',sequence:3,submissionId:actor});
 fireEvent.click(screen.getByRole('button',{name:'Withdraw share'}));await waitFor(()=>expect(screen.getByRole('textbox',{name:'Private note'})).toBeEnabled());
});
it('keeps an uncertain save input and request key and blocks sharing',async()=>{
 mocks.save.mockRejectedValue(new KnowledgeError('REQUEST_INTERRUPTED',503));render(<ManagedNoteEditor initial={note()} knowledge={ref}/>);await waitFor(()=>expect(mocks.share).toHaveBeenCalled());fireEvent.change(screen.getByRole('textbox',{name:'Private note'}),{target:{value:'待确认的笔记'}});fireEvent.click(screen.getByRole('button',{name:'Save note'}));await screen.findByRole('button',{name:'Retry the same request'});expect(screen.getByRole('button',{name:'Request sharing'})).toBeDisabled();const first=mocks.save.mock.calls[0];fireEvent.click(screen.getByRole('button',{name:'Retry the same request'}));await waitFor(()=>expect(mocks.save).toHaveBeenCalledTimes(2));expect(mocks.save.mock.calls[1]).toEqual(first);
});
it('does not apply a late save after an account change',async()=>{
 let finish!:(v:unknown)=>void;mocks.save.mockImplementation(()=>new Promise(resolve=>{finish=resolve}));render(<ManagedNoteEditor initial={note()} knowledge={ref}/>);fireEvent.change(screen.getByRole('textbox',{name:'Private note'}),{target:{value:'原账户草稿'}});fireEvent.click(screen.getByRole('button',{name:'Save note'}));window.dispatchEvent(new Event('math-master:auth-change'));finish({...note(),body:'迟到的原账户正文'});await waitFor(()=>expect(document.body.textContent).not.toContain('迟到的原账户正文'));expect(screen.queryByDisplayValue('原账户草稿')).toBeNull();
});
it('keeps a note draft after sharing changed in another tab',async()=>{
 mocks.save.mockRejectedValue(new KnowledgeError('NOTE_SHARING_LOCKED',409));render(<ManagedNoteEditor initial={note()} knowledge={ref}/>);fireEvent.change(screen.getByRole('textbox',{name:'Private note'}),{target:{value:'不能丢弃的草稿'}});fireEvent.click(screen.getByRole('button',{name:'Save note'}));await waitFor(()=>expect(screen.getByRole('textbox',{name:'Private note'})).toHaveValue('不能丢弃的草稿'));
});
it('confirms an uncertain deletion with its original request and shows the current empty note',async()=>{
 mocks.save.mockRejectedValueOnce(new KnowledgeError('REQUEST_INTERRUPTED',503)).mockResolvedValueOnce({...note(),body:'',revision:2,deleted:true});render(<ManagedNoteEditor initial={note()} knowledge={ref}/>);fireEvent.click(screen.getByRole('button',{name:'Delete note'}));await screen.findByRole('button',{name:'Retry the same request'});const first=mocks.save.mock.calls[0];fireEvent.click(screen.getByRole('button',{name:'Retry the same request'}));await waitFor(()=>expect(mocks.save).toHaveBeenCalledTimes(2));expect(mocks.save.mock.calls[1]).toEqual(first);await waitFor(()=>expect(screen.getByRole('textbox',{name:'Private note'})).toHaveValue(''));
});
it('confirms the original uncertain save even when another tab submitted it for sharing',async()=>{
 mocks.save.mockRejectedValueOnce(new KnowledgeError('REQUEST_INTERRUPTED',503)).mockResolvedValueOnce({...note(),body:'已提交但待确认的正文',revision:2});render(<ManagedNoteEditor initial={note()} knowledge={ref}/>);fireEvent.change(screen.getByRole('textbox',{name:'Private note'}),{target:{value:'已提交但待确认的正文'}});fireEvent.click(screen.getByRole('button',{name:'Save note'}));await screen.findByRole('button',{name:'Retry the same request'});const first=mocks.save.mock.calls[0];mocks.share.mockResolvedValue({...head(),status:'pending',sequence:1,submissionId:actor});fireEvent(window,new Event('focus'));await screen.findByText('Awaiting approval');fireEvent.click(screen.getByRole('button',{name:'Retry the same request'}));await waitFor(()=>expect(mocks.save).toHaveBeenCalledTimes(2));expect(mocks.save.mock.calls[1]).toEqual(first);
});
