"use client";
import {UiText} from "@/lib/i18n/ui-text";
import {uiMessage,uiValue} from "@/lib/i18n/format";
import { useState } from 'react';
import { decisionInputSchema } from '@/lib/correction/schemas';
import type { PlanMetadata } from '@/lib/correction/types';
import type { useCorrectionCommand } from './pending-command';
import styles from '@/styles/content.module.css';
export function ReviewPanel({ plan, command, enabled }: {
    plan: PlanMetadata;
    command: ReturnType<typeof useCorrectionCommand>;
    enabled: boolean;
}) {
    const [reason, setReason] = useState(''), valid = decisionInputSchema.safeParse({ expectedSequence: plan.sequence, decision: 'approve', reason }).success, locked = command.busy || !!command.pending;
    if (!enabled || plan.status !== 'pending')
        return null;
    return <section className={styles.card}><h2><UiText notice={uiMessage("page.review",{})}/></h2><p><UiText notice={uiMessage("review-panel.approve.only.after.checking.the.exact.published.sources.creators..0f5188",{})}/></p><fieldset disabled={locked}><label className={styles.field}><UiText notice={uiMessage("review-panel.independent.review.reason.08bd46",{})}/><textarea rows={4} required value={reason} onChange={e => setReason(e.target.value)}/></label><div className={styles.actions}><button type="button" className="button" disabled={!valid} onClick={() => void command.run({ kind: 'decidePlan', ref: plan.ref, input: { expectedSequence: plan.sequence, decision: 'approve', reason } })}><UiText notice={uiMessage("review-panel.approve.correction.d71667",{})}/></button><button type="button" className="button secondary" disabled={!valid} onClick={() => void command.run({ kind: 'decidePlan', ref: plan.ref, input: { expectedSequence: plan.sequence, decision: 'reject', reason } })}><UiText notice={uiMessage("review-panel.reject.correction.0558f4",{})}/></button></div></fieldset></section>;
}
