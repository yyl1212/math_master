"use client";
import Link from 'next/link';
import { useContext, useState, useEffect } from 'react';
import { correctionClient } from '@/lib/correction/client';
import { planInputSchema } from '@/lib/correction/schemas';
import { CorrectionRequestError, type CaseMetadata, type Page, type PlanMetadata, type JobMetadata, type Mapping } from '@/lib/correction/types';
import { CorrectionAccountContext, useCorrectionRead } from './correction-account';
import { useCorrectionCommand } from './pending-command';
import { CorrectionCommandStatus, CorrectionState } from './status';
import { MappingFields } from './plan-editor';
import styles from '@/styles/content.module.css';
type Data = {
    case: CaseMetadata;
    plans: Page<PlanMetadata>;
    jobs: Page<JobMetadata>;
};
export function CasePanel({ initial }: {
    initial: Data;
}) {
    const account = useContext(CorrectionAccountContext), read = useCorrectionRead(), [data, setData] = useState(initial), [reason, setReason] = useState(''), [mappings, setMappings] = useState<Mapping[]>([]), [created, setCreated] = useState<PlanMetadata | null>(null), [loading, setLoading] = useState(false), [error, setError] = useState<CorrectionRequestError | null>(null);
    useEffect(() => { setData(initial); }, [initial]);
    const create = useCorrectionCommand(account?.actorId ?? '', async (receipt, signal) => {
        const plan = receipt.plan;
        if (!plan)
            throw new CorrectionRequestError();
        const latest = await read(a => correctionClient.readPlan(plan.ref, a), signal);
        if (!latest || signal.aborted)
            throw new CorrectionRequestError();
        setCreated(latest.data.plan);
        setData(v => ({ ...v, plans: { ...v.plans, items: [latest.data.plan, ...v.plans.items.filter(i => i.ref.id !== plan.ref.id || i.ref.version !== plan.ref.version)] } }));
        setReason('');
        setMappings([]);
    });
    const retry = useCorrectionCommand(account?.actorId ?? '', async (receipt, signal) => {
        if (!receipt.job)
            throw new CorrectionRequestError();
        const out = await read(a => correctionClient.listJobs(data.case.id, {}, a), signal);
        if (!out || signal.aborted)
            throw new CorrectionRequestError();
        setData(v => ({ ...v, jobs: out.data }));
    });
    const canEdit = account?.roles.some(r => r === 'editor' || r === 'admin'), input = planInputSchema.safeParse({ expectedSequence: null, parent: null, algorithmVersion: 1, mappings, reason }), locked = create.busy || !!create.pending || !!account?.checking;
    async function reload() {
        if (loading)
            return;
        setLoading(true);
        setError(null);
        try {
            const out = await read(async (a) => {
                const [c, p, j] = await Promise.all([correctionClient.readCase(data.case.id, a), correctionClient.listPlans(data.case.id, {}, a), correctionClient.listJobs(data.case.id, {}, a)]);
                if (c.actorId !== p.actorId || c.actorId !== j.actorId)
                    throw new CorrectionRequestError('AUTHENTICATION_REQUIRED');
                return { actorId: c.actorId, data: { case: c.data, plans: p.data, jobs: j.data } };
            });
            if (out)
                setData(out.data);
        }
        catch (e) {
            setError(e instanceof CorrectionRequestError ? e : new CorrectionRequestError());
        }
        finally {
            setLoading(false);
        }
    }
    async function more(kind: 'plans' | 'jobs') {
        const cursor = data[kind].nextCursor;
        if (!cursor || loading)
            return;
        setLoading(true);
        setError(null);
        try {
            if (kind === 'plans') {
                const out = await read(a => correctionClient.listPlans(data.case.id, { cursor }, a));
                if (out)
                    setData(v => ({ ...v, plans: { items: [...v.plans.items, ...out.data.items], nextCursor: out.data.nextCursor } }));
            }
            else {
                const out = await read(a => correctionClient.listJobs(data.case.id, { cursor }, a));
                if (out)
                    setData(v => ({ ...v, jobs: { items: [...v.jobs.items, ...out.data.items], nextCursor: out.data.nextCursor } }));
            }
        }
        catch (e) {
            setError(e instanceof CorrectionRequestError ? e : new CorrectionRequestError());
        }
        finally {
            setLoading(false);
        }
    }
    return <main className={styles.workbench}><Link prefetch={false} href="/review/corrections">Correction cases</Link><h1>Correction case</h1><p className={styles.metadata}>{data.case.id} · {data.case.kind} · {data.case.hasApprovedPlan ? 'Approved basis available' : 'Awaiting approval'}</p>{data.case.cutoff && <p>Original attempts created before {new Date(data.case.cutoff).toLocaleString('en')} are within scope.</p>}<button type="button" className="button secondary" disabled={loading || account?.checking} onClick={() => void reload()}>Reload case status</button>{error && <CorrectionState error={error} onRetry={() => void reload()}/>}
 <section className={styles.card}><h2>Correction plans</h2><ul className={styles.list}>{data.plans.items.map(p => <li key={p.ref.id + ':' + p.ref.version}><Link prefetch={false} href={'/review/corrections/' + data.case.id + '/plans/' + p.ref.id + '/' + p.ref.version}>Plan version {p.ref.version} · {p.status}</Link><p>Sequence {p.sequence} · {p.mappingCount} exact instance mappings</p></li>)}</ul>{data.plans.items.length === 0 && <p>No correction plan yet.</p>}{data.plans.nextCursor && <button type="button" disabled={loading} onClick={() => void more('plans')}>More plans</button>}</section>
 {canEdit && <section className={styles.card}><h2>Draft a correction plan</h2><form onSubmit={e => {
                e.preventDefault();
                if (input.success)
                    void create.run({ kind: 'createPlan', caseId: data.case.id, input: input.data });
            }}><fieldset disabled={locked}><label className={styles.field}>Plan reason<textarea aria-label="Plan reason" rows={5} required value={reason} onChange={e => setReason(e.target.value)}/><small>{[...reason].length}/4000 characters</small></label><MappingFields value={mappings} onChange={setMappings}/><button className="button" disabled={!input.success}>Create correction plan</button></fieldset></form><CorrectionCommandStatus command={create}/>{created && <Link prefetch={false} href={'/review/corrections/' + data.case.id + '/plans/' + created.ref.id + '/' + created.ref.version}>Open correction plan</Link>}</section>}
 <section className={styles.card}><h2>Impact processing</h2><p>Processing retries preserve completed evidence and the audit history.</p><ul className={styles.list}>{data.jobs.items.map(j => <li key={j.id}><strong>{j.type.replaceAll('_', ' ')} · {j.state}</strong><p>{j.processedCount} evidence records checked · Attempt {j.attempt}/8</p>{j.errorClass && <p>Error category: {j.errorClass}</p>}{j.nextRunAt && <p>Next attempt: {new Date(j.nextRunAt).toLocaleString('en')}</p>}{canEdit && ['failed', 'retry_wait'].includes(j.state) && <button type="button" className="button secondary" disabled={retry.busy || !!retry.pending} onClick={() => void retry.run({ kind: 'retryJob', jobId: j.id, input: { expectedSequence: j.sequence } })}>Retry processing</button>}</li>)}</ul>{data.jobs.nextCursor && <button type="button" disabled={loading} onClick={() => void more('jobs')}>More jobs</button>}<CorrectionCommandStatus command={retry}/></section></main>;
}
