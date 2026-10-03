import { it, expect, vi, afterEach } from 'vitest';
import { render, screen, fireEvent, waitFor, act } from '@testing-library/react';
import { FeedbackAccountProvider, FeedbackAccountContext } from './feedback-account';
import { NewFeedbackForm } from './new-form';
import { FeedbackTicket } from './discussion-panel';
import { getAuthContext } from '@/lib/auth/client';
import { readFeedback, sendFeedback } from '@/lib/feedback/client';
import { FeedbackRequestError, type Context } from '@/lib/feedback/types';
import { id, otherId, metadata } from '@/lib/feedback/test-fixtures';
vi.mock('@/lib/auth/client', () => ({ getAuthContext: vi.fn(), notifyAuthChanged: vi.fn() }));
vi.mock('@/lib/feedback/client', () => ({ readFeedback: vi.fn(), sendFeedback: vi.fn() }));
const navigation = vi.hoisted(() => ({ push: vi.fn(), refresh: vi.fn() }));
vi.mock('next/navigation', () => ({ useRouter: () => navigation }));
const proof = (actorId = id) => ({ ok: true as const, data: { user: { id: actorId, username: 'original-test-user', roles: ['reviewer' as const], mustChangePassword: false }, csrfToken: 'A'.repeat(43) } });
const context = (version = 1): Context => ({ target: { kind: 'knowledge', identity: { id: 'math-root', version, sha256: 'a'.repeat(64) }, area: null, part: null }, source: { kind: 'publication', publicationId: version === 1 ? id : otherId, attemptId: null, position: null }, label: 'knowledge math-root · v' + version });
const sourcePath = '/api/v1/feedback/contexts/knowledge/math-root';
const invalidate = vi.fn();
const wrapper = ({ children }: { children: React.ReactNode }) => <FeedbackAccountContext.Provider value={{ actorId: id, invalidate }}>{children}</FeedbackAccountContext.Provider>;
afterEach(() => { vi.resetAllMocks(); vi.useRealTimers(); });

it('atomically removes the old private draft before a silent SSR account change is verified', async () => {
 vi.mocked(getAuthContext).mockResolvedValue(proof());
 const view = render(<FeedbackAccountProvider actorId={id}><NewFeedbackForm context={context()} sourcePath={sourcePath}/></FeedbackAccountProvider>);
 fireEvent.change(await screen.findByLabelText('Title'), { target: { value: 'A private draft' } });
 fireEvent.change(screen.getByLabelText('Details'), { target: { value: 'A private details' } });
 let resolve!: (value: ReturnType<typeof proof>) => void;
 vi.mocked(getAuthContext).mockReturnValueOnce(new Promise(r => { resolve = r; }));
 view.rerender(<FeedbackAccountProvider actorId={otherId}><NewFeedbackForm context={context()} sourcePath={sourcePath}/></FeedbackAccountProvider>);
 expect(screen.queryByLabelText('Title')).toBeNull();
 expect(screen.getByRole('status')).toHaveTextContent('Checking');
 await act(async () => resolve(proof(otherId)));
 expect(await screen.findByLabelText('Title')).toHaveValue('');
 expect(screen.getByLabelText('Details')).toHaveValue('');
 vi.mocked(getAuthContext).mockResolvedValue(proof(otherId));
 vi.mocked(sendFeedback).mockRejectedValue(new FeedbackRequestError());
 fireEvent.change(screen.getByLabelText('Title'), { target: { value: 'B original input' } });
 fireEvent.change(screen.getByLabelText('Details'), { target: { value: 'B original details' } });
 fireEvent.click(screen.getByRole('button', { name: 'Submit report' }));
 await waitFor(() => expect(sendFeedback).toHaveBeenCalledTimes(1));
 expect(vi.mocked(sendFeedback).mock.calls[0][0]).toMatchObject({ actorId: otherId, input: { title: 'B original input', message: 'B original details' } });
});

it('the offered Reload page action rechecks identity and recovers after a temporary error', async () => {
 vi.mocked(getAuthContext).mockResolvedValueOnce({ ok: false, status: 503, code: 'SERVICE_UNAVAILABLE', message: 'Service temporarily unavailable.' }).mockResolvedValue(proof());
 render(<FeedbackAccountProvider actorId={id}><p>recovered private page</p></FeedbackAccountProvider>);
 expect(await screen.findByRole('alert')).toHaveTextContent('temporarily unavailable');
 fireEvent.click(screen.getByRole('button', { name: 'Reload page' }));
 expect(await screen.findByText('recovered private page')).toBeVisible();
 expect(screen.queryByRole('alert')).toBeNull();
});

it('successful fresh identity on focus clears a previous temporary error', async () => {
 vi.mocked(getAuthContext).mockResolvedValueOnce({ ok: false, status: 503, code: 'SERVICE_UNAVAILABLE', message: 'Service temporarily unavailable.' }).mockResolvedValue(proof());
 render(<FeedbackAccountProvider actorId={id}><p>focus recovery</p></FeedbackAccountProvider>);
 await screen.findByRole('alert'); fireEvent(window, new Event('focus'));
 expect(await screen.findByText('focus recovery')).toBeVisible();
 expect(screen.queryByRole('alert')).toBeNull();
});

it('a same-account SSR refresh rechecks identity without clearing an existing draft', async () => {
 vi.mocked(getAuthContext).mockResolvedValue(proof());
 const view = render(<FeedbackAccountProvider actorId={id}><NewFeedbackForm context={context()} sourcePath={sourcePath}/></FeedbackAccountProvider>);
 fireEvent.change(await screen.findByLabelText('Title'), { target: { value: 'same account draft' } });
 view.rerender(<FeedbackAccountProvider actorId={id}><NewFeedbackForm context={context(2)} sourcePath={sourcePath}/></FeedbackAccountProvider>);
 await waitFor(() => expect(getAuthContext).toHaveBeenCalledTimes(2));
 expect(screen.getByLabelText('Title')).toHaveValue('same account draft');
});

it('refreshes a stale target while retaining the original text and waiting for explicit new submission', async () => {
 vi.mocked(getAuthContext).mockResolvedValue(proof());
 vi.mocked(sendFeedback).mockRejectedValue(new FeedbackRequestError('FEEDBACK_TARGET_STALE'));
 vi.mocked(readFeedback).mockResolvedValue({ actorId: id, data: context(2) });
 render(<NewFeedbackForm context={context()} sourcePath={sourcePath}/>, { wrapper });
 fireEvent.change(screen.getByLabelText('Title'), { target: { value: 'retained source report' } });
 fireEvent.change(screen.getByLabelText('Details'), { target: { value: '中'.repeat(4000) } });
 fireEvent.click(screen.getByRole('button', { name: 'Submit report' }));
 await screen.findByRole('alert');
 const original = vi.mocked(sendFeedback).mock.calls[0][0];
 fireEvent.click(screen.getByRole('button', { name: 'Refresh report target' }));
 expect(await screen.findByText('knowledge math-root · v2')).toBeVisible();
 expect(screen.getByLabelText('Title')).toHaveValue('retained source report');
 expect(screen.getByLabelText('Details')).toHaveValue('中'.repeat(4000));
 expect(sendFeedback).toHaveBeenCalledTimes(1);
 expect(vi.mocked(readFeedback).mock.calls[0][0]).toBe(sourcePath);
 fireEvent.click(screen.getByRole('button', { name: 'Submit report' }));
 await waitFor(() => expect(sendFeedback).toHaveBeenCalledTimes(2));
 const refreshed = vi.mocked(sendFeedback).mock.calls[1][0];
 expect(refreshed.key).not.toBe(original.key);
 expect(refreshed.input).toMatchObject({ target: context(2).target, source: context(2).source, title: 'retained source report', message: '中'.repeat(4000) });
 expect(original.input).toMatchObject({ target: context().target, source: context().source });
});

it('manual owner status refresh retains unsubmitted details and uses the refreshed sequence', async () => {
 vi.mocked(getAuthContext).mockResolvedValue(proof());
 vi.mocked(readFeedback).mockResolvedValue({ actorId: id, data: { ...metadata(), sequence: 2, status: 'processing' } });
 vi.mocked(sendFeedback).mockRejectedValue(new FeedbackRequestError());
 render(<FeedbackTicket initial={metadata()} review={false}/>, { wrapper });
 fireEvent.change(screen.getByLabelText('Additional details'), { target: { value: 'unsubmitted original supplement' } });
 fireEvent.click(screen.getByRole('button', { name: 'Reload report status' }));
 expect(await screen.findByText(/Event 2/)).toBeVisible();
 expect(screen.getByLabelText('Additional details')).toHaveValue('unsubmitted original supplement');
 fireEvent.click(screen.getByRole('button', { name: 'Add details' }));
 await waitFor(() => expect(sendFeedback).toHaveBeenCalledTimes(1));
 expect(vi.mocked(sendFeedback).mock.calls[0][0].input).toMatchObject({ expectedSequence: 2, message: 'unsubmitted original supplement' });
});

it('conflict recovery preserves handler draft and references while an old-key retry stays frozen', async () => {
 vi.mocked(getAuthContext).mockResolvedValue(proof());
 const initial = { ...metadata(), status: 'processing' as const, canHandle: true };
 vi.mocked(readFeedback).mockResolvedValue({ actorId: id, data: { ...initial, sequence: 2 } });
 vi.mocked(sendFeedback).mockRejectedValue(new FeedbackRequestError('FEEDBACK_CONFLICT'));
 render(<FeedbackTicket initial={initial} review={true}/>, { wrapper });
 fireEvent.change(screen.getByLabelText('Report status'), { target: { value: 'resolved' } });
 fireEvent.change(screen.getByLabelText('Review reply'), { target: { value: 'unsubmitted independent handling' } });
 fireEvent.change(screen.getByLabelText('Resolution basis'), { target: { value: 'revision_published' } });
 for (const [label, value] of [['Withdrawal event ID', id], ['Replacement ID', 'math-root'], ['Replacement version', '2'], ['Replacement SHA256', 'b'.repeat(64)], ['Replacement publication ID', otherId]]) fireEvent.change(screen.getByLabelText(label), { target: { value } });
 fireEvent.click(screen.getByRole('button', { name: 'Save handling result' }));
 await screen.findByRole('alert'); const original = vi.mocked(sendFeedback).mock.calls[0][0];
 fireEvent.click(screen.getByRole('button', { name: 'Reload report status' }));
 expect(await screen.findByText(/Event 2/)).toBeVisible();
 expect(screen.getByLabelText('Review reply')).toHaveValue('unsubmitted independent handling');
 expect(screen.getByLabelText('Resolution basis')).toHaveValue('revision_published');
 expect(screen.getByLabelText('Withdrawal event ID')).toHaveValue(id);
 expect(screen.getByLabelText('Replacement publication ID')).toHaveValue(otherId);
 fireEvent.click(screen.getByRole('button', { name: 'Retry same request' }));
 await waitFor(() => expect(sendFeedback).toHaveBeenCalledTimes(2));
 expect(vi.mocked(sendFeedback).mock.calls[1][0]).toEqual(original);
 fireEvent.click(screen.getByRole('button', { name: 'Edit as a new request' }));
 fireEvent.click(screen.getByRole('button', { name: 'Save handling result' }));
 await waitFor(() => expect(sendFeedback).toHaveBeenCalledTimes(3));
 expect(vi.mocked(sendFeedback).mock.calls[2][0].key).not.toBe(original.key);
 expect(vi.mocked(sendFeedback).mock.calls[2][0].input).toMatchObject({ expectedSequence: 2, message: 'unsubmitted independent handling', resolution: { kind: 'revision_published', withdrawal: { id }, replacement: { publicationId: otherId } } });
});

it('ignores a latest-metadata response arriving after the confirmed command deadline', async () => {
 vi.useFakeTimers(); vi.mocked(getAuthContext).mockResolvedValue(proof());
 vi.mocked(sendFeedback).mockResolvedValue({ actorId: id, data: { status: 201, ticket: metadata() } });
 let resolve!: (value: { actorId: string; data: ReturnType<typeof metadata> }) => void;
 vi.mocked(readFeedback).mockReturnValue(new Promise(r => { resolve = r; }));
 render(<NewFeedbackForm context={context()} sourcePath={sourcePath}/>, { wrapper });
 fireEvent.change(screen.getByLabelText('Title'), { target: { value: 'confirmed original report' } });
 fireEvent.change(screen.getByLabelText('Details'), { target: { value: 'original body' } });
 fireEvent.click(screen.getByRole('button', { name: 'Submit report' }));
 await act(async () => vi.advanceTimersByTimeAsync(10000));
 expect(screen.getByRole('alert')).toHaveTextContent('Your change was saved');
 expect(screen.getByRole('button', { name: 'Submit report' })).toBeDisabled();
 expect(screen.queryByRole('button', { name: 'Edit as a new request' })).toBeNull();
 await act(async () => resolve({ actorId: id, data: metadata() }));
 expect(navigation.push).not.toHaveBeenCalled();
 expect(screen.getByRole('button', { name: 'Retry same request' })).toBeVisible();
});
