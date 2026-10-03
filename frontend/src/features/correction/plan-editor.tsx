"use client";
import Link from 'next/link';
import { useContext, useState, useEffect } from 'react';
import { correctionClient } from '@/lib/correction/client';
import { planInputSchema } from '@/lib/correction/schemas';
import { CorrectionRequestError, type Mapping, type PlanMetadataView, type PlanDetail, type PlanMetadata } from '@/lib/correction/types';
import { CorrectionAccountContext, useCorrectionRead } from './correction-account';
import { useCorrectionCommand } from './pending-command';
import { CorrectionState, CorrectionCommandStatus } from './status';
import { ReviewPanel } from './review-panel';
import styles from '@/styles/content.module.css';
export function MappingFields({ value, onChange }: {
    value: Mapping[];
    onChange: (value: Mapping[]) => void;
}) {
    const setIdentity = (index: number, side: 'original' | 'replacement', field: 'id' | 'version' | 'sha256', raw: string) => {
        const current = value[index];
        if (!current)
            return;
        const identity = side === 'original' ? current.original : current.replacement.identity;
        const updated = field === 'version' ? { ...identity, version: Number(raw) } : field === 'id' ? { ...identity, id: raw } : { ...identity, sha256: raw };
        onChange(value.map((m, i) => i !== index ? m : side === 'original' ? { ...m, original: updated } : { ...m, replacement: { ...m.replacement, identity: updated } }));
    };
    return <section><h3>Exact instance mappings</h3><p>Keep question meaning and parameters unchanged. Only an independently published correction to answers or explanations may replace an instance.</p>{value.map((m, i) => <fieldset key={i}><legend>Mapping {i + 1}</legend>{(['original', 'replacement'] as const).map(side => { const identity = side === 'original' ? m.original : m.replacement.identity; return <div key={side}><h4>{side === 'original' ? 'Original instance' : 'Approved replacement'}</h4>{(['id', 'version', 'sha256'] as const).map(field => <label className={styles.field} key={field}>{side} {field}<input aria-label={side + ' ' + field + ' ' + (i + 1)} type={field === 'version' ? 'number' : 'text'} value={identity[field]} onChange={e => setIdentity(i, side, field, e.target.value)}/></label>)}<label className={styles.field}>{side} publication ID<input aria-label={side + ' publication ' + (i + 1)} value={side === 'original' ? m.originalPublicationId : m.replacement.publicationId} onChange={e => onChange(value.map((v, n) => n !== i ? v : side === 'original' ? { ...v, originalPublicationId: e.target.value } : { ...v, replacement: { ...v.replacement, publicationId: e.target.value } }))}/></label></div>; })}<button type="button" className={styles.remove} onClick={() => onChange(value.filter((_, n) => n !== i))}>Remove mapping {i + 1}</button></fieldset>)}<button type="button" className="button secondary" disabled={value.length >= 50} onClick={() => onChange([...value, { original: { id: '', version: 1, sha256: '' }, originalPublicationId: '', replacement: { identity: { id: '', version: 1, sha256: '' }, publicationId: '' } }])}>Add instance mapping</button></section>;
}
export function PlanEditor({ initial }: {
    initial: PlanMetadataView;
}) {
    const account = useContext(CorrectionAccountContext), read = useCorrectionRead(), [metadata, setMetadata] = useState(initial), [detail, setDetail] = useState<PlanDetail | null>(null), [reason, setReason] = useState(''), [mappings, setMappings] = useState<Mapping[]>([]), [loading, setLoading] = useState(false), [error, setError] = useState<CorrectionRequestError | null>(null), [revision, setRevision] = useState(false), [created, setCreated] = useState<PlanMetadata | null>(null);
    useEffect(() => { setMetadata(initial); }, [initial]);
    const command = useCorrectionCommand(account?.actorId ?? '', async (receipt, signal) => {
        const plan = receipt.plan;
        if (!plan)
            throw new CorrectionRequestError();
        const latest = await read(a => correctionClient.readPlan(plan.ref, a), signal);
        if (!latest || signal.aborted)
            throw new CorrectionRequestError();
        if (plan.ref.id === metadata.plan.ref.id && plan.ref.version === metadata.plan.ref.version)
            setMetadata(latest.data);
        else
            setCreated(latest.data.plan);
    });
    const canEdit = !!account?.roles.some(r => r === 'editor' || r === 'admin'), canReview = !!account?.roles.some(r => r === 'reviewer' || r === 'admin'), locked = command.busy || !!command.pending || !!account?.checking;
    async function refresh(protectedBody: boolean) {
        if (loading)
            return;
        setLoading(true);
        setError(null);
        try {
            if (protectedBody) {
                const out = await read(a => correctionClient.readPlanDetail(metadata.plan.ref, a));
                if (out) {
                    setDetail(out.data);
                    setMetadata({ plan: out.data.plan, parent: out.data.parent, mappings: [], reason: null, decisionReason: null });
                    setReason(out.data.reason);
                    setMappings(out.data.mappings);
                }
            }
            else {
                const out = await read(a => correctionClient.readPlan(metadata.plan.ref, a));
                if (out)
                    setMetadata(out.data);
            }
        }
        catch (e) {
            setError(e instanceof CorrectionRequestError ? e : new CorrectionRequestError());
        }
        finally {
            setLoading(false);
        }
    }
    const editable = canEdit && (metadata.plan.status === 'draft' || revision), input = planInputSchema.safeParse({ expectedSequence: revision ? null : metadata.plan.sequence, parent: revision ? metadata.plan.ref : metadata.parent, algorithmVersion: 1, mappings, reason });
    return <main className={styles.workbench}><Link prefetch={false} href={'/review/corrections/' + metadata.plan.caseId}>Correction case</Link><h1>Correction plan · Version {metadata.plan.ref.version}</h1><p>{metadata.plan.status} · <span>Sequence {metadata.plan.sequence}</span> · {metadata.plan.mappingCount} mappings</p><p className={styles.metadata}>{metadata.plan.digest ? 'Frozen basis SHA256: ' + metadata.plan.digest : 'This draft has not been sealed.'}</p><div className={styles.actions}><button type="button" className="button secondary" disabled={loading || account?.checking} onClick={() => void refresh(false)}>Reload plan status</button><button type="button" className="button secondary" disabled={loading || locked || !!detail} onClick={() => void refresh(true)}>Read plan details</button></div>{error && <CorrectionState error={error} onRetry={() => void refresh(false)}/>}
 {detail && <><section className={styles.card}><h2>{revision ? 'New plan version' : 'Approved basis and draft'}</h2>{!editable && <p>{detail.reason}</p>}{detail.decisionReason && <p>Review decision: {detail.decisionReason}</p>}{metadata.plan.status !== 'draft' && canEdit && !revision && <button type="button" className="button secondary" disabled={locked} onClick={() => setRevision(true)}>Draft a new plan version</button>}<form onSubmit={e => {
                e.preventDefault();
                if (!input.success)
                    return;
                void command.run(revision ? { kind: 'createPlan', caseId: metadata.plan.caseId, input: input.data } : { kind: 'updatePlan', ref: metadata.plan.ref, input: input.data });
            }}><fieldset disabled={locked || !editable}><label className={styles.field}>Plan reason<textarea aria-label="Plan reason" rows={6} required value={reason} onChange={e => setReason(e.target.value)}/><small>{[...reason].length}/4000 characters</small></label><MappingFields value={mappings} onChange={setMappings}/>{editable && <button className="button" disabled={!input.success}>{revision ? 'Create new plan version' : 'Save plan draft'}</button>}</fieldset></form>{canEdit && metadata.plan.status === 'draft' && !revision && <button type="button" className="button secondary" disabled={locked} onClick={() => void command.run({ kind: 'submitPlan', ref: metadata.plan.ref, input: { expectedSequence: metadata.plan.sequence } })}>Submit for independent review</button>}<CorrectionCommandStatus command={command}/>{created && <Link prefetch={false} href={'/review/corrections/' + created.caseId + '/plans/' + created.ref.id + '/' + created.ref.version}>Open new plan version</Link>}</section><ReviewPanel plan={metadata.plan} command={command} enabled={canReview && !revision}/></>}
 </main>;
}
