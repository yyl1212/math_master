import { it, expect, vi, afterEach } from 'vitest';
import { render, screen, fireEvent, waitFor, act } from '@testing-library/react';
import { CorrectionAccountContext, CorrectionAccountProvider } from './correction-account';
import { CaseList } from './case-list';
import { ResultPanel } from './result-panel';
import { correctionClient } from '@/lib/correction/client';
import { getAuthContext } from '@/lib/auth/client';
import { resultMetadataSchema } from '@/lib/correction/schemas';
import { CorrectionRequestError } from '@/lib/correction/types';
import { id, otherId, caseMetadata, resultDetail, resultMetadata, context } from '@/lib/correction/test-fixtures';
vi.mock('next/navigation', () => ({ useRouter: () => ({ refresh: vi.fn() }) }));
vi.mock('@/lib/auth/client', () => ({ getAuthContext: vi.fn(), notifyAuthChanged: vi.fn() }));
vi.mock('@/lib/correction/client', async (original) => { const m = await original<typeof import('@/lib/correction/client')>(); return { ...m, correctionClient: { ...m.correctionClient, createCase: vi.fn(), readCase: vi.fn(), listCases: vi.fn(), readOwn: vi.fn(), readOwnDetail: vi.fn() } }; });
const invalidate = vi.fn(), wrapper = ({ children }: {
    children: React.ReactNode;
}) => <CorrectionAccountContext.Provider value={{ actorId: id, roles: ['admin'], checking: false, invalidate }}>{children}</CorrectionAccountContext.Provider>;
afterEach(() => { vi.resetAllMocks(); vi.useRealTimers(); });
it('CorrectionConfirmedPendingRefresh retains its key after committed POST and failed GET', async () => { vi.mocked(getAuthContext).mockResolvedValue({ ...context(), ok: true, data: { ...context().data, user: { ...context().data.user, username: 'test', roles: ['admin'] } } }); vi.mocked(correctionClient.createCase).mockResolvedValue({ actorId: id, data: { status: 201, case: caseMetadata(), plan: null, job: null } }); vi.mocked(correctionClient.readCase).mockRejectedValue(new CorrectionRequestError()); render(<CaseList initial={{ items: [], nextCursor: null }}/>, { wrapper }); fireEvent.click(screen.getByRole('button', { name: 'Register correction case' })); await screen.findByText(/Your change was saved/); const original = vi.mocked(correctionClient.createCase).mock.calls[0]; expect(screen.getByRole('button', { name: 'Register correction case' })).toBeDisabled(); expect(screen.queryByRole('button', { name: 'Edit as a new request' })).toBeNull(); fireEvent.click(screen.getByRole('button', { name: 'Retry same request' })); await waitFor(() => expect(correctionClient.createCase).toHaveBeenCalledTimes(2)); expect(vi.mocked(correctionClient.createCase).mock.calls[1]?.[0]).toEqual(original[0]); expect(vi.mocked(correctionClient.createCase).mock.calls[1]?.[1]).toMatchObject({ actorId: original[1].actorId, key: original[1].key }); });
it('shows six honest correction dispositions and loads protected details only on demand', async () => {
    const states = ['corrected_passed', 'corrected_failed', 'retake_required', 'review_material', 'checked_unaffected', 'awaiting_review'] as const;
    for (const status of states) {
        const original = resultMetadata();
        const result = status === 'corrected_failed' ? { ...original, status, score: 3, passed: false } : status === 'corrected_passed' ? original : { ...original, status, score: null, passed: null };
        const view = render(<ResultPanel initial={{ result: resultMetadataSchema.parse(result), items: [], planReason: null }}/>, { wrapper });
        if (status.startsWith('corrected_'))
            expect(screen.getByText(/Corrected result/)).toBeVisible();
        else
            expect(screen.queryByText(/Score:/)).toBeNull();
        expect(correctionClient.readOwnDetail).not.toHaveBeenCalled();
        view.unmount();
    }
});
it('delivers original answers and corrected explanations only through the protected read', async () => { vi.mocked(correctionClient.readOwnDetail).mockResolvedValue({ actorId: id, data: resultDetail() }); render(<ResultPanel initial={{ result: resultMetadata(), items: [], planReason: null }}/>, { wrapper }); expect(screen.queryByText('Adding one to one gives two.')).toBeNull(); fireEvent.click(screen.getByRole('button', { name: 'Review corrected answers' })); expect((await screen.findAllByText('Adding one to one gives two.')).length).toBe(5); expect(screen.getByRole('link', { name: 'Original result' })).toHaveAttribute('href', '/assessments/' + id + '/result'); });
it('CorrectionSSRActorSwitch removes all old private content before verifying a changed SSR actor', async () => { vi.mocked(getAuthContext).mockResolvedValue({ ...context(), ok: true, data: { ...context().data, user: { ...context().data.user, username: 'test', roles: ['admin'] } } }); const v = render(<CorrectionAccountProvider actorId={id}><p>A private content</p></CorrectionAccountProvider>); await screen.findByText('A private content'); vi.mocked(getAuthContext).mockReturnValueOnce(new Promise(() => { })); v.rerender(<CorrectionAccountProvider actorId={otherId}><p>B private content</p></CorrectionAccountProvider>); expect(screen.queryByText('A private content')).toBeNull(); expect(screen.queryByText('B private content')).toBeNull(); expect(screen.getByRole('status')).toHaveTextContent('Checking'); });
it('CorrectionActorDeadline covers the account boundary identity that never returns', async () => { vi.useFakeTimers(); vi.mocked(getAuthContext).mockReturnValue(new Promise(() => { })); render(<CorrectionAccountProvider actorId={id}><p>private</p></CorrectionAccountProvider>); await act(async () => vi.advanceTimersByTimeAsync(10001)); expect(screen.getByRole('alert')).toHaveTextContent('temporarily unavailable'); expect(screen.queryByText('private')).toBeNull(); });

it('CorrectionEffectiveAssets bind illustrations to the exact correction result', async () => {
 const d=resultDetail();d.result={...d.result,id:otherId};d.items[0].assets=[{id:'original-svg',sha256:'a'.repeat(64)}];
 vi.mocked(correctionClient.readOwnDetail).mockResolvedValue({actorId:id,data:d});
 render(<ResultPanel initial={{result:d.result,items:[],planReason:null}}/>,{wrapper});fireEvent.click(screen.getByRole('button',{name:'Review corrected answers'}));await screen.findByRole('img');
 expect(screen.getByRole('img')).toHaveAttribute('src',`/api/v1/corrections/results/${otherId}/assets/${'a'.repeat(64)}`);
});

it('CorrectionEditorCapabilities preserve plan editing without administrator commands', async () => {
 const {CasePanel}=await import('./case-panel');
 const editor=({children}:{children:React.ReactNode})=><CorrectionAccountContext.Provider value={{actorId:id,roles:['editor'],checking:false,invalidate}}>{children}</CorrectionAccountContext.Provider>;
 const list=render(<CaseList initial={{items:[],nextCursor:null}}/>,{wrapper:editor});
 expect(screen.queryByRole('button',{name:'Register correction case'})).toBeNull();list.unmount();
 const job={id,caseId:id,plan:null,type:'approved_plan' as const,state:'failed' as const,sequence:1,epoch:1,attempt:8,nextRunAt:null,processedCount:0,errorClass:'database' as const};
 render(<CasePanel initial={{case:caseMetadata(),plans:{items:[],nextCursor:null},jobs:{items:[job],nextCursor:null}}}/>,{wrapper:editor});
 fireEvent.change(screen.getByLabelText('Plan reason'),{target:{value:'My independent editor draft'}});
 expect(screen.getByRole('button',{name:'Create correction plan'})).toBeVisible();
 expect(screen.queryByRole('button',{name:'Retry processing'})).toBeNull();
 expect(screen.getByLabelText('Plan reason')).toHaveValue('My independent editor draft');
 expect(invalidate).not.toHaveBeenCalled();
});

it('CorrectionEditorHasNoAdminRetry while keeping an unsent plan draft', async () => {
 const {CasePanel}=await import('./case-panel'),job={id,caseId:id,plan:null,type:'approved_plan' as const,state:'failed' as const,sequence:1,epoch:1,attempt:8,nextRunAt:null,processedCount:0,errorClass:'database' as const};
 const editor=({children}:{children:React.ReactNode})=><CorrectionAccountContext.Provider value={{actorId:id,roles:['editor'],checking:false,invalidate}}>{children}</CorrectionAccountContext.Provider>;
 render(<CasePanel initial={{case:caseMetadata(),plans:{items:[],nextCursor:null},jobs:{items:[job],nextCursor:null}}}/>,{wrapper:editor});fireEvent.change(screen.getByLabelText('Plan reason'),{target:{value:'My unsent editor draft'}});
 expect(screen.queryByRole('button',{name:'Retry processing'})).toBeNull();expect(screen.getByLabelText('Plan reason')).toHaveValue('My unsent editor draft');expect(screen.getByRole('button',{name:'Create correction plan'})).toBeVisible();
});
