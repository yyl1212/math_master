"use client";
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
    return <section className={styles.card}><h2>Independent review</h2><p>Approve only after checking the exact published sources. Creators, editors of this draft and authors of its mathematical sources cannot approve it.</p><fieldset disabled={locked}><label className={styles.field}>Independent review reason<textarea rows={4} required value={reason} onChange={e => setReason(e.target.value)}/></label><div className={styles.actions}><button type="button" className="button" disabled={!valid} onClick={() => void command.run({ kind: 'decidePlan', ref: plan.ref, input: { expectedSequence: plan.sequence, decision: 'approve', reason } })}>Approve correction</button><button type="button" className="button secondary" disabled={!valid} onClick={() => void command.run({ kind: 'decidePlan', ref: plan.ref, input: { expectedSequence: plan.sequence, decision: 'reject', reason } })}>Reject correction</button></div></fieldset></section>;
}
