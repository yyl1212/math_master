"use client";
import {enumOptionLabels} from "@/lib/i18n/enums";

import type {UiNotice} from "@/lib/i18n/types";
import {uiError} from "@/lib/i18n/errors";
import {UiText} from "@/lib/i18n/ui-text";
import {uiMessage,uiValue} from "@/lib/i18n/format";
import {useUiI18n} from "@/lib/i18n/provider";
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
 const {t}=useUiI18n();

    const [head, setHead] = useState(initial.head), [target, setTarget] = useState<WithdrawalTarget>({ kind: "template", id: "", version: 1 }), [preview, setPreview] = useState<WithdrawalPreview | null>(null), [reason, setReason] = useState(""), [busy, setBusy] = useState(false), [notice, setNotice] = useState<UiNotice | null>(null), [members, setMembers] = useState<MemberPage | null>(null), command = useQuestionCommand();
    function change(next: WithdrawalTarget) { setTarget(next); setPreview(null); setNotice(null); }
    useEffect(() => { if (command.failure?.code === "QUESTION_PUBLICATION_STALE") {
        setPreview(null);
        setNotice(uiMessage("withdrawal-panel.preview.changed.read.a.fresh.impact.before.withdrawing.25242a",{}));
    } }, [command.failure]);
    async function impact(offset = 0) { if (busy)
        return; setBusy(true); try {
        const r = await requestQuestion<WithdrawalPreview>({ kind: "previewWithdrawal", query: { limit: preview?.changes.limit ?? 20, offset } }, { target });
        if (!r.ok) {
            command.error(uiError("question",r));
            return;
        }
        if (JSON.stringify(r.data.target) !== JSON.stringify(target)) {
            setPreview(null);
            setNotice(uiMessage("withdrawal-panel.preview.changed.read.a.fresh.impact.before.withdrawing.25242a",{}));
            return;
        }
        if (preview && offset !== 0 && (r.data.impactDigest !== preview.impactDigest || r.data.currentKnowledgeHead !== preview.currentKnowledgeHead || r.data.currentQuestionHead !== preview.currentQuestionHead)) {
            setPreview(null);
            setNotice(uiMessage("withdrawal-panel.preview.changed.read.a.fresh.impact.before.withdrawing.25242a",{}));
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
            command.error(r.ok?uiMessage("common.unavailable",{}):uiError("question",r));
    }
    finally {
        setBusy(false);
    } }
    const input = { target, expectedKnowledgeHead: preview?.currentKnowledgeHead ?? null, expectedQuestionHead: preview?.currentQuestionHead ?? null, reason };
    return <section className={styles.workbench}><p className="eyebrow"><UiText notice={uiMessage("withdrawal-panel.permanent.question.withdrawal.33c686",{})}/></p><h1><UiText notice={uiMessage("withdrawal-panel.withdraw.an.exact.question.version.d8e270",{})}/></h1><p><UiText notice={uiMessage("withdrawal-panel.template.withdrawal.removes.its.generated.instances.and.dependent.865351",{})}/></p><p className={styles.notice}><UiText notice={uiMessage("withdrawal-panel.a.node.may.no.longer.have.five.valid.questions.after.withdrawal.p.54a61b",{})}/></p><p className={styles.metadata}><UiText notice={uiMessage("withdrawal-panel.current.question.head.value.2cd82b",{v0:uiValue(head ?? "None")})}/></p><fieldset disabled={busy || command.blocked}><legend><UiText notice={uiMessage("withdrawal-panel.exact.version.to.withdraw.f7625e",{})}/></legend><SelectField label={t("withdrawal-panel.target.kind.607b91",{})} value={target.kind} options={["template", "instance", "blueprint"]} optionLabels={enumOptionLabels("question.enum",["template", "instance", "blueprint"])} onChange={kind => change({ ...target, kind })}/><TextField label={t("withdrawal-panel.target.id.32a090",{})} value={target.id} onChange={id => change({ ...target, id })}/><NumberField label={t("withdrawal-panel.target.version.2acb4e",{})} value={target.version} onChange={version => change({ ...target, version })}/><p><UiText notice={uiMessage("withdrawal-panel.you.may.identify.a.historical.version.that.is.no.longer.in.the.cu.67dc63",{})}/></p></fieldset><div className={styles.actions}><button className="button secondary" disabled={busy || command.blocked || questionInputError("previewWithdrawal", { target }) !== null} onClick={() => void impact()}><UiText notice={uiMessage("withdrawal-panel.preview.withdrawal.69c4c0",{})}/></button><button className="button secondary" disabled={busy || command.blocked || !head} onClick={() => void loadMembers()}><UiText notice={uiMessage("withdrawal-panel.browse.current.members.944018",{})}/></button></div>{members && <section className={styles.card}><h2><UiText notice={uiMessage("withdrawal-panel.current.fixed.members.2494e0",{})}/></h2>{members.items.map((m, i) => <p key={i}><button className="button secondary" disabled={busy || command.blocked} onClick={() => change({ kind: m.identity.kind, id: m.identity.id, version: m.identity.version })}><UiText notice={uiMessage("diff-panel.value.value.vvalue.610089",{v0:uiValue(m.identity.kind),v1:uiValue(m.identity.id),v2:uiValue(m.identity.version)})}/></button></p>)}<div className={styles.actions}><button className="button secondary" disabled={busy || members.offset === 0} onClick={() => void loadMembers(Math.max(0, members.offset - members.limit))}><UiText notice={uiMessage("withdrawal-panel.previous.current.members.eb0309",{})}/></button><button className="button secondary" disabled={busy || members.offset + members.limit >= members.total} onClick={() => void loadMembers(members.offset + members.limit)}><UiText notice={uiMessage("withdrawal-panel.next.current.members.9edb05",{})}/></button></div></section>}{notice && <p role="status"><UiText notice={notice}/></p>}{preview && <section className={styles.card}><h2><UiText notice={uiMessage("withdrawal-panel.complete.impact.totals.a7beee",{})}/></h2><p><UiText notice={uiMessage("withdrawal-panel.affected.templates.value.9c33ae",{v0:uiValue(preview.affectedTemplates)})}/></p><p><UiText notice={uiMessage("withdrawal-panel.affected.instances.value.d7268d",{v0:uiValue(preview.affectedInstances)})}/></p><p><UiText notice={uiMessage("withdrawal-panel.affected.blueprints.value.30f428",{v0:uiValue(preview.affectedBlueprints)})}/></p><p className={styles.metadata}><UiText notice={uiMessage("coverage-panel.knowledge.head.b7020e",{})}/>{preview.currentKnowledgeHead ?? "None"}<br /><UiText notice={uiMessage("coverage-panel.question.head.6001f1",{})}/>{preview.currentQuestionHead ?? "None"}<br /><UiText notice={uiMessage("withdrawal-panel.impact.digest.c7887e",{})}/>{preview.impactDigest}</p><p><UiText notice={uiMessage("withdrawal-panel.value.fixed.changes.offset.value.totals.include.all.pages.6666c8",{v0:uiValue(preview.changes.total),v1:uiValue(preview.changes.offset)})}/></p><ChangeItems items={preview.changes.items}/><div className={styles.actions}><button className="button secondary" disabled={busy || command.blocked || preview.changes.offset === 0} onClick={() => void impact(Math.max(0, preview.changes.offset - preview.changes.limit))}><UiText notice={uiMessage("withdrawal-panel.previous.impact.changes.ab7233",{})}/></button><button className="button secondary" disabled={busy || command.blocked || preview.changes.offset + preview.changes.limit >= preview.changes.total} onClick={() => void impact(preview.changes.offset + preview.changes.limit)}><UiText notice={uiMessage("withdrawal-panel.next.impact.changes.15deab",{})}/></button></div></section>}<fieldset disabled={command.blocked}><legend><UiText notice={uiMessage("withdrawal-panel.withdrawal.reason.6fb3ad",{})}/></legend><TextField label={t("withdrawal-panel.withdrawal.reason.6fb3ad",{})} value={reason} onChange={setReason} multiline/></fieldset><button className="button" disabled={busy || command.blocked || !preview || questionInputError("withdrawVersion", input) !== null} onClick={() => void command.run({ kind: "withdrawVersion" }, input, data => { const result = data as WithdrawalResult; setHead(result.publication.id); setPreview(null); setMembers(null); setNotice(uiMessage("withdrawal-panel.exact.version.withdrawn.permanently.refresh.published.coverage.to.fc0086",{})); })}><UiText notice={uiMessage("withdrawal-panel.withdraw.permanently.07fd73",{})}/></button>{command.controls}<p><Link prefetch={false} href="/admin/question-publications"><UiText notice={uiMessage("withdrawal-panel.back.to.trusted.question.snapshots.d7fd67",{})}/></Link></p></section>;
}
