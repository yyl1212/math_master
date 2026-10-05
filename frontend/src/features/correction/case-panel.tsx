"use client";
import {UiEnum,formatUiEnum} from "@/lib/i18n/enums";
import {UiText} from "@/lib/i18n/ui-text";
import {uiMessage,uiValue} from "@/lib/i18n/format";
import {useUiI18n} from "@/lib/i18n/provider";
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
 const {t,locale}=useUiI18n();

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
    const canAdmin = account?.roles.includes('admin'), canEdit = account?.roles.some(r => r === 'editor' || r === 'admin'), input = planInputSchema.safeParse({ expectedSequence: null, parent: null, algorithmVersion: 1, mappings, reason }), locked = create.busy || !!create.pending || !!account?.checking;
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
    return <main className={styles.workbench}><Link prefetch={false} href="/review/corrections"><UiText notice={uiMessage("page.review.corrections",{})}/></Link><h1><UiText notice={uiMessage("page.review.corrections.id",{})}/></h1><p className={styles.metadata}>{data.case.id} · {data.case.kind} · {data.case.hasApprovedPlan ? t("case-list.approved.basis.available.c1fcbf",{}) : t("case-panel.awaiting.approval.ae25c9",{})}</p>{data.case.cutoff && <p><UiText notice={uiMessage("case-panel.original.attempts.created.before.value.are.within.scope.007445",{v0:uiValue(new Date(data.case.cutoff).toLocaleString('en'))})}/></p>}<button type="button" className="button secondary" disabled={loading || account?.checking} onClick={() => void reload()}><UiText notice={uiMessage("case-panel.reload.case.status.665567",{})}/></button>{error && <CorrectionState error={error} onRetry={() => void reload()}/>}
 <section className={styles.card}><h2><UiText notice={uiMessage("case-panel.correction.plans.dfd490",{})}/></h2><ul className={styles.list}>{data.plans.items.map(p => <li key={p.ref.id + ':' + p.ref.version}><Link prefetch={false} href={'/review/corrections/' + data.case.id + '/plans/' + p.ref.id + '/' + p.ref.version}><UiText notice={uiMessage("case-panel.plan.version.value.value.b562a0",{v0:uiValue(p.ref.version),v1:formatUiEnum(locale,"correction.plan",p.status)})}/></Link><p><UiText notice={uiMessage("case-panel.sequence.value.value.exact.instance.mappings.7c40db",{v0:uiValue(p.sequence),v1:uiValue(p.mappingCount)})}/></p></li>)}</ul>{data.plans.items.length === 0 && <p><UiText notice={uiMessage("case-panel.no.correction.plan.yet.91fe2a",{})}/></p>}{data.plans.nextCursor && <button type="button" disabled={loading} onClick={() => void more('plans')}><UiText notice={uiMessage("case-panel.more.plans.4b0125",{})}/></button>}</section>
 {canEdit && <section className={styles.card}><h2><UiText notice={uiMessage("case-panel.draft.a.correction.plan.af8cb8",{})}/></h2><form onSubmit={e => {
                e.preventDefault();
                if (input.success)
                    void create.run({ kind: 'createPlan', caseId: data.case.id, input: input.data });
            }}><fieldset disabled={locked}><label className={styles.field}><UiText notice={uiMessage("case-panel.plan.reason.dc2c72",{})}/><textarea aria-label={t("case-panel.plan.reason.dc2c72",{})} rows={5} required value={reason} onChange={e => setReason(e.target.value)}/><small><UiText notice={uiMessage("new-form.value.4000.characters.9daa9f",{v0:uiValue([...reason].length)})}/></small></label><MappingFields value={mappings} onChange={setMappings}/><button className="button" disabled={!input.success}><UiText notice={uiMessage("case-panel.create.correction.plan.23b28c",{})}/></button></fieldset></form><CorrectionCommandStatus command={create}/>{created && <Link prefetch={false} href={'/review/corrections/' + data.case.id + '/plans/' + created.ref.id + '/' + created.ref.version}><UiText notice={uiMessage("case-panel.open.correction.plan.5cd0c5",{})}/></Link>}</section>}
 <section className={styles.card}><h2><UiText notice={uiMessage("case-panel.impact.processing.5455c8",{})}/></h2><p><UiText notice={uiMessage("case-panel.processing.retries.preserve.completed.evidence.and.the.audit.hist.e975f9",{})}/></p><ul className={styles.list}>{data.jobs.items.map(j => <li key={j.id}><strong><UiEnum group="correction.jobType" value={j.type}/> · <UiEnum group="correction.jobState" value={j.state}/></strong><p><UiText notice={uiMessage("case-panel.value.evidence.records.checked.attempt.value.8.5baf30",{v0:uiValue(j.processedCount),v1:uiValue(j.attempt)})}/></p>{j.errorClass && <p><UiText notice={uiMessage("case-panel.error.category.value.e53a05",{v0:uiValue(j.errorClass)})}/></p>}{j.nextRunAt && <p><UiText notice={uiMessage("case-panel.next.attempt.value.cd648a",{v0:uiValue(new Date(j.nextRunAt).toLocaleString('en'))})}/></p>}{canAdmin && ['failed', 'retry_wait'].includes(j.state) && <button type="button" className="button secondary" disabled={retry.busy || !!retry.pending} onClick={() => void retry.run({ kind: 'retryJob', jobId: j.id, input: { expectedSequence: j.sequence } })}><UiText notice={uiMessage("case-panel.retry.processing.593933",{})}/></button>}</li>)}</ul>{data.jobs.nextCursor && <button type="button" disabled={loading} onClick={() => void more('jobs')}><UiText notice={uiMessage("case-panel.more.jobs.a1464e",{})}/></button>}<CorrectionCommandStatus command={retry}/></section></main>;
}
