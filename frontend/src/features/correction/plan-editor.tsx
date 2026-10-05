"use client";
import {UiText} from "@/lib/i18n/ui-text";
import {uiMessage,uiValue} from "@/lib/i18n/format";
import {useUiI18n} from "@/lib/i18n/provider";
import {formatUiEnum} from "@/lib/i18n/enums";
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
 const {t,locale}=useUiI18n();

    const setIdentity = (index: number, side: 'original' | 'replacement', field: 'id' | 'version' | 'sha256', raw: string) => {
        const current = value[index];
        if (!current)
            return;
        const identity = side === 'original' ? current.original : current.replacement.identity;
        const updated = field === 'version' ? { ...identity, version: Number(raw) } : field === 'id' ? { ...identity, id: raw } : { ...identity, sha256: raw };
        onChange(value.map((m, i) => i !== index ? m : side === 'original' ? { ...m, original: updated } : { ...m, replacement: { ...m.replacement, identity: updated } }));
    };
    return <section><h3><UiText notice={uiMessage("plan-editor.exact.instance.mappings.bdfd3a",{})}/></h3><p><UiText notice={uiMessage("plan-editor.keep.question.meaning.and.parameters.unchanged.only.an.independen.89968d",{})}/></p>{value.map((m, i) => <fieldset key={i}><legend><UiText notice={uiMessage("plan-editor.mapping.value.fc087c",{v0:uiValue(i + 1)})}/></legend>{(['original', 'replacement'] as const).map(side => { const identity = side === 'original' ? m.original : m.replacement.identity; return <div key={side}><h4>{side === 'original' ? t("plan-editor.original.instance.4f3d6e",{}) : t("plan-editor.approved.replacement.f4ed84",{})}</h4>{(['id', 'version', 'sha256'] as const).map(field => <label className={styles.field} key={field}><UiText notice={uiMessage("audit.mapping.caption",{side:formatUiEnum(locale,"correction.side",side),field:formatUiEnum(locale,"correction.field",field)})}/><input aria-label={t("audit.mapping.label",{side:formatUiEnum(locale,"correction.side",side),field:formatUiEnum(locale,"correction.field",field),number:i+1})} type={field === 'version' ? 'number' : 'text'} value={identity[field]} onChange={e => setIdentity(i, side, field, e.target.value)}/></label>)}<label className={styles.field}><UiText notice={uiMessage("audit.mapping.publicationCaption",{side:formatUiEnum(locale,"correction.side",side)})}/><input aria-label={t("audit.mapping.publication",{side:formatUiEnum(locale,"correction.side",side),number:i+1})} value={side === 'original' ? m.originalPublicationId : m.replacement.publicationId} onChange={e => onChange(value.map((v, n) => n !== i ? v : side === 'original' ? { ...v, originalPublicationId: e.target.value } : { ...v, replacement: { ...v.replacement, publicationId: e.target.value } }))}/></label></div>; })}<button type="button" className={styles.remove} onClick={() => onChange(value.filter((_, n) => n !== i))}><UiText notice={uiMessage("plan-editor.remove.mapping.value.5a5ea7",{v0:uiValue(i + 1)})}/></button></fieldset>)}<button type="button" className="button secondary" disabled={value.length >= 50} onClick={() => onChange([...value, { original: { id: '', version: 1, sha256: '' }, originalPublicationId: '', replacement: { identity: { id: '', version: 1, sha256: '' }, publicationId: '' } }])}><UiText notice={uiMessage("plan-editor.add.instance.mapping.77231b",{})}/></button></section>;
}
export function PlanEditor({ initial }: {
    initial: PlanMetadataView;
}) {
 const {t,locale}=useUiI18n();

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
    return <main className={styles.workbench}><Link prefetch={false} href={'/review/corrections/' + metadata.plan.caseId}><UiText notice={uiMessage("page.review.corrections.id",{})}/></Link><h1><UiText notice={uiMessage("plan-editor.correction.plan.version.value.8aa1ce",{v0:uiValue(metadata.plan.ref.version)})}/></h1><p>{metadata.plan.status} · <span><UiText notice={uiMessage("plan-editor.sequence.value.b4d4eb",{v0:uiValue(metadata.plan.sequence)})}/></span> · {metadata.plan.mappingCount}<UiText notice={uiMessage("plan-editor.mappings.e7ca2a",{})}/></p><p className={styles.metadata}>{metadata.plan.digest ? 'Frozen basis SHA256: ' + metadata.plan.digest : t("plan-editor.this.draft.has.not.been.sealed.4a1611",{})}</p><div className={styles.actions}><button type="button" className="button secondary" disabled={loading || account?.checking} onClick={() => void refresh(false)}><UiText notice={uiMessage("plan-editor.reload.plan.status.082e03",{})}/></button><button type="button" className="button secondary" disabled={loading || locked || !!detail} onClick={() => void refresh(true)}><UiText notice={uiMessage("plan-editor.read.plan.details.5c6186",{})}/></button></div>{error && <CorrectionState error={error} onRetry={() => void refresh(false)}/>}
 {detail && <><section className={styles.card}><h2>{revision ? t("plan-editor.new.plan.version.a46c49",{}) : t("plan-editor.approved.basis.and.draft.538ca7",{})}</h2>{!editable && <p>{detail.reason}</p>}{detail.decisionReason && <p><UiText notice={uiMessage("plan-editor.review.decision.value.f55c2d",{v0:uiValue(detail.decisionReason)})}/></p>}{metadata.plan.status !== 'draft' && canEdit && !revision && <button type="button" className="button secondary" disabled={locked} onClick={() => setRevision(true)}><UiText notice={uiMessage("plan-editor.draft.a.new.plan.version.6ef060",{})}/></button>}<form onSubmit={e => {
                e.preventDefault();
                if (!input.success)
                    return;
                void command.run(revision ? { kind: 'createPlan', caseId: metadata.plan.caseId, input: input.data } : { kind: 'updatePlan', ref: metadata.plan.ref, input: input.data });
            }}><fieldset disabled={locked || !editable}><label className={styles.field}><UiText notice={uiMessage("case-panel.plan.reason.dc2c72",{})}/><textarea aria-label={t("case-panel.plan.reason.dc2c72",{})} rows={6} required value={reason} onChange={e => setReason(e.target.value)}/><small><UiText notice={uiMessage("new-form.value.4000.characters.9daa9f",{v0:uiValue([...reason].length)})}/></small></label><MappingFields value={mappings} onChange={setMappings}/>{editable && <button className="button" disabled={!input.success}>{revision ? t("plan-editor.create.new.plan.version.88490f",{}) : t("plan-editor.save.plan.draft.8fd425",{})}</button>}</fieldset></form>{canEdit && metadata.plan.status === 'draft' && !revision && <button type="button" className="button secondary" disabled={locked} onClick={() => void command.run({ kind: 'submitPlan', ref: metadata.plan.ref, input: { expectedSequence: metadata.plan.sequence } })}><UiText notice={uiMessage("plan-editor.submit.for.independent.review.7dad26",{})}/></button>}<CorrectionCommandStatus command={command}/>{created && <Link prefetch={false} href={'/review/corrections/' + created.caseId + '/plans/' + created.ref.id + '/' + created.ref.version}><UiText notice={uiMessage("plan-editor.open.new.plan.version.4e2487",{})}/></Link>}</section><ReviewPanel plan={metadata.plan} command={command} enabled={canReview && !revision}/></>}
 </main>;
}
