import {it,expect,vi,afterEach} from 'vitest';
import {render,screen,fireEvent} from '@testing-library/react';
import {EvidenceLink} from './evidence-link';
import {LearningAccountContext} from '../learning/learning-account';
import {correctionClient} from '@/lib/correction/client';
import {id,otherId,resultMetadata} from '@/lib/correction/test-fixtures';
vi.mock('@/lib/correction/client',()=>({correctionClient:{listOwn:vi.fn()}}));
const invalidate=vi.fn(),wrapper=({children}:{children:React.ReactNode})=><LearningAccountContext.Provider value={{actorId:id,invalidate}}>{children}</LearningAccountContext.Provider>;
afterEach(()=>vi.resetAllMocks());
it('reads only metadata for the exact owned historical evidence when requested',async()=>{vi.mocked(correctionClient.listOwn).mockResolvedValue({actorId:id,data:{items:[resultMetadata()],nextCursor:null}});render(<EvidenceLink evidence={{kind:'assessment',id}}/>,{wrapper});expect(correctionClient.listOwn).not.toHaveBeenCalled();fireEvent.click(screen.getByRole('button',{name:'Check corrections'}));expect(await screen.findByRole('link',{name:'View correction'})).toHaveAttribute('href','/corrections/'+id);expect(vi.mocked(correctionClient.listOwn).mock.calls[0][0]).toEqual({kind:'assessment',id});expect(vi.mocked(correctionClient.listOwn).mock.calls[0][1]).toMatchObject({actorId:id})});
it('rejects a silently changed actor without displaying foreign correction metadata',async()=>{vi.mocked(correctionClient.listOwn).mockResolvedValue({actorId:otherId,data:{items:[resultMetadata()],nextCursor:null}});render(<EvidenceLink evidence={{kind:'assessment',id}}/>,{wrapper});fireEvent.click(screen.getByRole('button',{name:'Check corrections'}));await vi.waitFor(()=>expect(invalidate).toHaveBeenCalled());expect(screen.queryByRole('link',{name:'View correction'})).toBeNull()});

it('cancels a pending read when its exact evidence changes and enables the new reference',async()=>{vi.mocked(correctionClient.listOwn).mockImplementation(()=>new Promise(()=>{}));const v=render(<EvidenceLink evidence={{kind:'assessment',id}}/>,{wrapper});fireEvent.click(screen.getByRole('button',{name:'Check corrections'}));expect(screen.getByRole('button',{name:'Check corrections'})).toBeDisabled();v.rerender(<EvidenceLink evidence={{kind:'assessment',id:otherId}}/>);await vi.waitFor(()=>expect(screen.getByRole('button',{name:'Check corrections'})).toBeEnabled());expect(screen.queryByRole('link',{name:'View correction'})).toBeNull()});
