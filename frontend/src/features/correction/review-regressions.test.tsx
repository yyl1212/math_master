import { it, expect, vi, afterEach } from 'vitest';
import { render, screen, fireEvent, waitFor, act } from '@testing-library/react';
import { CorrectionAccountProvider, CorrectionAccountContext } from './correction-account';
import { CasePanel } from './case-panel';
import { PlanEditor } from './plan-editor';
import { correctionClient } from '@/lib/correction/client';
import { getAuthContext } from '@/lib/auth/client';
import { CorrectionRequestError } from '@/lib/correction/types';
import { id, otherId, caseMetadata, planMetadata, context } from '@/lib/correction/test-fixtures';
vi.mock('next/navigation', () => ({ useRouter: () => ({ refresh: vi.fn() }) }));
vi.mock('@/lib/auth/client', () => ({ getAuthContext: vi.fn(), notifyAuthChanged: vi.fn() }));
vi.mock('@/lib/correction/client', async (original) => { const m = await original<typeof import('@/lib/correction/client')>(); return { ...m, correctionClient: { ...m.correctionClient, createPlan: vi.fn(), readPlan: vi.fn(), readCase: vi.fn(), listPlans: vi.fn(), listJobs: vi.fn() } }; });
afterEach(() => { vi.resetAllMocks(); vi.useRealTimers(); });
const initial = () => ({ case: caseMetadata(), plans: { items: [], nextCursor: null }, jobs: { items: [], nextCursor: null } });
const wrapper = ({ children }: {
    children: React.ReactNode;
}) => <CorrectionAccountContext.Provider value={{ actorId: id, roles: ['admin'], checking: false, invalidate: vi.fn() }}>{children}</CorrectionAccountContext.Provider>;
const proof = (actor = id) => ({ ...context(actor), ok: true as const, data: { ...context(actor).data, user: { ...context(actor).data.user, username: 'test', roles: ['admin' as const] } } });
it('CorrectionPrivateDraftRefresh preserves inputs across a same-actor SSR refresh', async () => { vi.mocked(getAuthContext).mockResolvedValue(proof()); const v = render(<CorrectionAccountProvider actorId={id} management><CasePanel initial={initial()}/></CorrectionAccountProvider>); fireEvent.change(await screen.findByLabelText('Plan reason'), { target: { value: 'Same account private reason' } }); v.rerender(<CorrectionAccountProvider actorId={id} management><CasePanel initial={initial()}/></CorrectionAccountProvider>); await waitFor(() => expect(getAuthContext).toHaveBeenCalledTimes(2)); expect(screen.getByLabelText('Plan reason')).toHaveValue('Same account private reason'); });
it('disables duplicate commands during identity checking and retains its key on timeout', async () => { vi.useFakeTimers(); vi.mocked(getAuthContext).mockReturnValue(new Promise(() => { })); render(<CasePanel initial={initial()}/>, { wrapper }); fireEvent.change(screen.getByLabelText('Plan reason'), { target: { value: 'Original request' } }); fireEvent.click(screen.getByRole('button', { name: 'Create correction plan' })); fireEvent.click(screen.getByRole('button', { name: 'Create correction plan' })); expect(screen.getByRole('button', { name: 'Create correction plan' })).toBeDisabled(); await act(async () => vi.advanceTimersByTimeAsync(10001)); expect(correctionClient.createPlan).not.toHaveBeenCalled(); expect(screen.getByRole('button', { name: 'Retry same request' })).toBeVisible(); expect(screen.getByLabelText('Plan reason')).toHaveValue('Original request'); });
it('ignores a metadata response after the confirmed deadline and retries the original snapshot', async () => {
    vi.useFakeTimers();
    vi.mocked(getAuthContext).mockResolvedValue(proof());
    vi.mocked(correctionClient.createPlan).mockResolvedValue({ actorId: id, data: { status: 201, case: null, plan: planMetadata(), job: null } });
    let release!: (v: {
        actorId: string;
        data: {
            plan: ReturnType<typeof planMetadata>;
            parent: null;
            mappings: [
            ];
            reason: null;
            decisionReason: null;
        };
    }) => void;
    vi.mocked(correctionClient.readPlan).mockReturnValue(new Promise(r => { release = r; }));
    render(<CasePanel initial={initial()}/>, { wrapper });
    fireEvent.change(screen.getByLabelText('Plan reason'), { target: { value: 'Confirmed draft' } });
    fireEvent.click(screen.getByRole('button', { name: 'Create correction plan' }));
    await act(async () => vi.advanceTimersByTimeAsync(10001));
    expect(screen.getByRole('alert')).toHaveTextContent('Your change was saved');
    await act(async () => release({ actorId: id, data: { plan: planMetadata(), parent: null, mappings: [], reason: null, decisionReason: null } }));
    expect(screen.queryByRole('link', { name: 'Open correction plan' })).toBeNull();
    expect(screen.getByRole('button', { name: 'Retry same request' })).toBeVisible();
});
it('a live account change clears pending input and all private reason text', async () => { vi.mocked(getAuthContext).mockResolvedValue(proof()); vi.mocked(correctionClient.createPlan).mockRejectedValue(new CorrectionRequestError()); render(<CorrectionAccountProvider actorId={id} management><CasePanel initial={initial()}/></CorrectionAccountProvider>); fireEvent.change(await screen.findByLabelText('Plan reason'), { target: { value: 'Old account private text' } }); fireEvent.click(screen.getByRole('button', { name: 'Create correction plan' })); await screen.findByRole('alert'); vi.mocked(getAuthContext).mockResolvedValue(proof(otherId)); fireEvent(window, new Event('math-master:auth-change')); expect(screen.queryByLabelText('Plan reason')).toBeNull(); expect(screen.queryByDisplayValue('Old account private text')).toBeNull(); });
it('a manual metadata read also ends after ten seconds', async () => { vi.useFakeTimers(); vi.mocked(correctionClient.readPlan).mockReturnValue(new Promise(() => { })); render(<PlanEditor initial={{ plan: planMetadata(), parent: null, mappings: [], reason: null, decisionReason: null }}/>, { wrapper }); fireEvent.click(screen.getByRole('button', { name: 'Reload plan status' })); expect(screen.getByRole('button', { name: 'Reload plan status' })).toBeDisabled(); await act(async () => vi.advanceTimersByTimeAsync(10001)); expect(screen.getByRole('alert')).toHaveTextContent('temporarily unavailable'); expect(screen.getByRole('button', { name: 'Reload plan status' })).toBeEnabled(); });
vi.mock('server-only', () => ({}));
const server = vi.hoisted(() => ({ cookies: vi.fn(), session: vi.fn() }));
vi.mock('next/headers', () => ({ cookies: server.cookies }));
vi.mock('@/lib/auth/server-client', () => ({ readServerSession: server.session }));
import { correctionPage, correctionPageQuery } from './page-data';
it('SSRActorSwitch rejects a silently changed response actor before rendering private data', async () => { server.cookies.mockResolvedValue({ toString: () => '' }); server.session.mockResolvedValue({ ok: true, data: proof().data.user }); const read = vi.fn(async () => ({ actorId: otherId, data: 'B private' })); await expect(correctionPage(read, true)).rejects.toMatchObject({ code: 'AUTHENTICATION_REQUIRED' }); expect(read.mock.calls[0]).toEqual([{ actorId: id, signal: expect.any(AbortSignal) }]); });
it('SSR identity cookies and session reads share the ten-second total budget', async () => { vi.useFakeTimers(); server.cookies.mockReturnValue(new Promise(() => { })); const read = vi.fn(); const observed = correctionPage(read).catch(e => e); await vi.advanceTimersByTimeAsync(10001); expect(await observed).toMatchObject({ code: 'SERVICE_UNAVAILABLE' }); expect(read).not.toHaveBeenCalled(); });
it('SSR queries reject duplicate arrays, unknown fields and noncanonical numbers', () => { for (const q of [{ limit: ['1', '2'] }, { cursor: 'bad' }, { limit: '01' }, { extra: 'private' }])
    expect(() => correctionPageQuery(q)).toThrow(); });

it('a fresh same-actor SSR snapshot recovers an invalidated boundary without reviving its old draft',async()=>{vi.mocked(getAuthContext).mockResolvedValue(proof());const v=render(<CorrectionAccountProvider actorId={id} management><CasePanel initial={initial()}/></CorrectionAccountProvider>);fireEvent.change(await screen.findByLabelText('Plan reason'),{target:{value:'Discarded old private draft'}});fireEvent(window,new Event('math-master:auth-change'));expect(screen.queryByLabelText('Plan reason')).toBeNull();fireEvent(window,new Event('focus'));expect(screen.queryByLabelText('Plan reason')).toBeNull();v.rerender(<CorrectionAccountProvider actorId={id} management><CasePanel initial={initial()}/></CorrectionAccountProvider>);expect(await screen.findByLabelText('Plan reason')).toHaveValue('');expect(screen.queryByDisplayValue('Discarded old private draft')).toBeNull()});
