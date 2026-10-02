"use client";
import Link from "next/link";
import { useState, useEffect } from "react";
import { requestQuestion } from "@/lib/question/client";
import type { PublicationPage, WithdrawalTarget, WithdrawalPreview, WithdrawalResult, MemberPage } from "@/lib/question/types";
import { questionInputError } from "@/lib/question/schemas";
import { TextField, NumberField, SelectField } from "@/features/content/field-controls";
import { useQuestionCommand } from "./command-controls";
import { ChangeItems } from "./diff-panel";
import styles from "@/styles/question.module.css";
export function WithdrawalPanel({ initial }: {
    initial: PublicationPage;
}) {
    const [head, setHead] = useState(initial.head), [target, setTarget] = useState<WithdrawalTarget>({ kind: "template", id: "", version: 1 }), [preview, setPreview] = useState<WithdrawalPreview | null>(null), [reason, setReason] = useState(""), [busy, setBusy] = useState(false), [notice, setNotice] = useState<string | null>(null), [members, setMembers] = useState<MemberPage | null>(null), command = useQuestionCommand();
    function change(next: WithdrawalTarget) { setTarget(next); setPreview(null); setNotice(null); }
    useEffect(() => { if (command.failure?.code === "QUESTION_PUBLICATION_STALE") {
        setPreview(null);
        setNotice("Preview changed. Read a fresh impact before withdrawing.");
    } }, [command.failure]);
    async function impact(offset = 0) { if (busy)
        return; setBusy(true); try {
        const r = await requestQuestion<WithdrawalPreview>({ kind: "previewWithdrawal", query: { limit: preview?.changes.limit ?? 20, offset } }, { target });
        if (!r.ok) {
            command.error(r.message);
            return;
        }
        if (JSON.stringify(r.data.target) !== JSON.stringify(target)) {
            setPreview(null);
            setNotice("Preview changed. Read a fresh impact before withdrawing.");
            return;
        }
        if (preview && offset !== 0 && (r.data.impactDigest !== preview.impactDigest || r.data.currentKnowledgeHead !== preview.currentKnowledgeHead || r.data.currentQuestionHead !== preview.currentQuestionHead)) {
            setPreview(null);
            setNotice("Preview changed. Read a fresh impact before withdrawing.");
            return;
        }
        setPreview(r.data);
        setNotice(null);
    }
    finally {
        setBusy(false);
    } }
    async function loadMembers(offset = 0) { if (!head || busy)
        return; setBusy(true); try {
        const r = await requestQuestion<MemberPage>({ kind: "listMembers", id: head, query: { limit: members?.limit ?? 20, offset } });
        if (r.ok && r.data.publicationId === head)
            setMembers(r.data);
        else
            command.error(r.ok ? "Snapshot identity does not match." : r.message);
    }
    finally {
        setBusy(false);
    } }
    const input = { target, expectedKnowledgeHead: preview?.currentKnowledgeHead ?? null, expectedQuestionHead: preview?.currentQuestionHead ?? null, reason };
    return <section className={styles.workbench}><p className="eyebrow">PERMANENT QUESTION WITHDRAWAL</p><h1>Withdraw an exact question version</h1><p>Template withdrawal removes its generated instances and dependent blueprints. Fixed-instance withdrawal removes explicit dependent blueprints. A single generated-instance withdrawal removes that instance; remaining pools are recalculated.</p><p className={styles.notice}>A node may no longer have five valid questions after withdrawal. Published explanations remain available. Historical bodies and approval records are preserved; withdrawn versions cannot be restored by republishing old approvals.</p><p className={styles.metadata}>Current question head: {head ?? "None"}</p><fieldset disabled={busy || command.blocked}><legend>Exact version to withdraw</legend><SelectField label="Target kind" value={target.kind} options={["template", "instance", "blueprint"]} onChange={kind => change({ ...target, kind })}/><TextField label="Target ID" value={target.id} onChange={id => change({ ...target, id })}/><NumberField label="Target version" value={target.version} onChange={version => change({ ...target, version })}/><p>You may identify a historical version that is no longer in the current bank.</p></fieldset><div className={styles.actions}><button className="button secondary" disabled={busy || command.blocked || questionInputError("previewWithdrawal", { target }) !== null} onClick={() => void impact()}>Preview withdrawal</button><button className="button secondary" disabled={busy || command.blocked || !head} onClick={() => void loadMembers()}>Browse current members</button></div>{members && <section className={styles.card}><h2>Current fixed members</h2>{members.items.map((m, i) => <p key={i}><button className="button secondary" disabled={busy || command.blocked} onClick={() => change({ kind: m.identity.kind, id: m.identity.id, version: m.identity.version })}>{m.identity.kind} · {m.identity.id} v{m.identity.version}</button></p>)}<div className={styles.actions}><button className="button secondary" disabled={busy || members.offset === 0} onClick={() => void loadMembers(Math.max(0, members.offset - members.limit))}>Previous current members</button><button className="button secondary" disabled={busy || members.offset + members.limit >= members.total} onClick={() => void loadMembers(members.offset + members.limit)}>Next current members</button></div></section>}{notice && <p role="status">{notice}</p>}{preview && <section className={styles.card}><h2>Complete impact totals</h2><p>Affected templates: {preview.affectedTemplates}</p><p>Affected instances: {preview.affectedInstances}</p><p>Affected blueprints: {preview.affectedBlueprints}</p><p className={styles.metadata}>Knowledge head: {preview.currentKnowledgeHead ?? "None"}<br />Question head: {preview.currentQuestionHead ?? "None"}<br />Impact digest: {preview.impactDigest}</p><p>{preview.changes.total} fixed changes · Offset {preview.changes.offset}. Totals include all pages.</p><ChangeItems items={preview.changes.items}/><div className={styles.actions}><button className="button secondary" disabled={busy || command.blocked || preview.changes.offset === 0} onClick={() => void impact(Math.max(0, preview.changes.offset - preview.changes.limit))}>Previous impact changes</button><button className="button secondary" disabled={busy || command.blocked || preview.changes.offset + preview.changes.limit >= preview.changes.total} onClick={() => void impact(preview.changes.offset + preview.changes.limit)}>Next impact changes</button></div></section>}<fieldset disabled={command.blocked}><legend>Withdrawal reason</legend><TextField label="Withdrawal reason" value={reason} onChange={setReason} multiline/></fieldset><button className="button" disabled={busy || command.blocked || !preview || questionInputError("withdrawVersion", input) !== null} onClick={() => void command.run({ kind: "withdrawVersion" }, input, data => { const result = data as WithdrawalResult; setHead(result.publication.id); setPreview(null); setMembers(null); setNotice("Exact version withdrawn permanently. Refresh published coverage to see remaining usable pools."); })}>Withdraw permanently</button>{command.controls}<p><Link prefetch={false} href="/admin/question-publications">Back to trusted question snapshots</Link></p></section>;
}
