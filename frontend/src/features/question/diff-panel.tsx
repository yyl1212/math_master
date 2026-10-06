"use client";

import {uiError} from "@/lib/i18n/errors";
import type {UiNotice} from "@/lib/i18n/types";
import {UiText} from "@/lib/i18n/ui-text";
import {uiMessage,uiValue} from "@/lib/i18n/format";
import {useUiI18n} from "@/lib/i18n/provider";
import { useEffect, useRef, useState } from "react";
import { requestQuestion } from "@/lib/question/client";
import type { ChangePage, MemberPage } from "@/lib/question/types";
import styles from "@/styles/question.module.css";
export function ChangeItems({ items }: {
    items: ChangePage["items"];
}) {
 const {t}=useUiI18n();
 return <ul className={styles.list}>{items.map((c, i) => <li key={i}><strong>{c.kind} · {c.id}</strong><p>{c.reason}</p><p className={styles.metadata}><UiText notice={uiMessage("diff-panel.before.a2bcd3",{})}/>{c.before ? `v${c.before.version} · ${c.before.sha256}` : t("diff-panel.none.dc937b",{})}<br /><UiText notice={uiMessage("diff-panel.after.84507c",{})}/>{c.after ? `v${c.after.version} · ${c.after.sha256}` : t("diff-panel.none.dc937b",{})}</p></li>)}</ul>; }
export function DiffPanel({ publicationId, manifestSha }: {
    publicationId: string;
    manifestSha: string;
}) {
 const {t}=useUiI18n();

    const [changes, setChanges] = useState<ChangePage | null>(null), [members, setMembers] = useState<MemberPage | null>(null), [error, setError] = useState<UiNotice | null>(null), [busy, setBusy] = useState(false);
    const activeRequest = useRef<AbortController | null>(null);
    useEffect(() => { const controller = new AbortController(); activeRequest.current = controller; setChanges(null); setMembers(null); setError(null); setBusy(true); void Promise.all([requestQuestion<ChangePage>({ kind: "listChanges", id: publicationId, query: { limit: 20 } }, undefined, undefined, controller.signal), requestQuestion<MemberPage>({ kind: "listMembers", id: publicationId, query: { limit: 20 } }, undefined, undefined, controller.signal)]).then(([a, b]) => { if (controller.signal.aborted)
        return; if (!a.ok || !b.ok) {
        setError(!a.ok?uiError("question",a):!b.ok?uiError("question",b):null);
        return;
    } if (a.data.publicationId !== publicationId || b.data.publicationId !== publicationId || a.data.manifestSha !== manifestSha || b.data.manifestSha !== manifestSha) {
        setError(uiMessage("diff-panel.snapshot.data.does.not.match.the.selected.manifest.707687",{}));
        return;
    } setChanges(a.data); setMembers(b.data); }).finally(() => { if (!controller.signal.aborted)
        setBusy(false); }); return () => { controller.abort(); activeRequest.current?.abort(); activeRequest.current = null; }; }, [publicationId, manifestSha]);
    async function load(kind: "listChanges" | "listMembers", offset: number) { if (busy)
        return; const controller = new AbortController(); activeRequest.current?.abort(); activeRequest.current = controller; setBusy(true); setError(null); try {
        const page = kind === "listChanges" ? changes : members;
        const r = await requestQuestion<ChangePage | MemberPage>({ kind, id: publicationId, query: { limit: page?.limit ?? 20, offset } }, undefined, undefined, controller.signal);
        if (controller.signal.aborted || activeRequest.current !== controller) return;
        if (!r.ok) {
            setError(uiError("question",r));
            return;
        }
        if (r.data.publicationId !== publicationId || r.data.manifestSha !== manifestSha) {
            setChanges(null);
            setMembers(null);
            setError(uiMessage("diff-panel.snapshot.data.does.not.match.the.selected.manifest.707687",{}));
            return;
        }
        if (kind === "listChanges")
            setChanges(r.data as ChangePage);
        else
            setMembers(r.data as MemberPage);
    }
    finally {
        if (!controller.signal.aborted && activeRequest.current === controller) setBusy(false);
    } }
    return <section><h3><UiText notice={uiMessage("diff-panel.fixed.mathematical.differences.15d782",{})}/></h3>{error && <p role="alert"><UiText notice={error}/></p>}{busy && !changes && <p><UiText notice={uiMessage("diff-panel.loading.fixed.differences.90b335",{})}/></p>}{changes && <><p><UiText notice={uiMessage("diff-panel.value.changes.offset.value.3a4d07",{v0:uiValue(changes.total),v1:uiValue(changes.offset)})}/></p><ChangeItems items={changes.items}/><div className={styles.actions}><button className="button secondary" disabled={busy || changes.offset === 0} onClick={() => void load("listChanges", Math.max(0, changes.offset - changes.limit))}><UiText notice={uiMessage("diff-panel.previous.changes.35f067",{})}/></button><button className="button secondary" disabled={busy || changes.offset + changes.limit >= changes.total} onClick={() => void load("listChanges", changes.offset + changes.limit)}><UiText notice={uiMessage("diff-panel.next.changes.d0a346",{})}/></button></div></>}<h3><UiText notice={uiMessage("diff-panel.fixed.members.and.approval.evidence.4d4bdb",{})}/></h3>{members && <><p><UiText notice={uiMessage("diff-panel.value.members.offset.value.approval.provenance.can.change.without.d13b68",{v0:uiValue(members.total),v1:uiValue(members.offset)})}/></p><ul className={styles.list}>{members.items.map((m, i) => <li key={i}><strong><UiText notice={uiMessage("diff-panel.value.value.vvalue.610089",{v0:uiValue(m.identity.kind),v1:uiValue(m.identity.id),v2:uiValue(m.identity.version)})}/></strong><p className={styles.metadata}><UiText notice={uiMessage("diff-panel.sha.483ee8",{})}/>{m.identity.sha256}<br /><UiText notice={uiMessage("diff-panel.package.98d9ee",{})}/>{m.identity.packageId}<UiText notice={uiMessage("publication-panel.v.e40f81",{})}/>{m.identity.packageVersion}<br /><UiText notice={uiMessage("diff-panel.submission.fb1f17",{})}/>{m.evidence.submissionId}<br /><UiText notice={uiMessage("diff-panel.decision.f78582",{})}/>{m.evidence.decisionId}<br /><UiText notice={uiMessage("diff-panel.frozen.digest.35e32d",{})}/>{m.evidence.frozenDigest}<br /><UiText notice={uiMessage("diff-panel.inherited.from.5be696",{})}/>{m.evidence.inheritedFrom ?? "Newly selected approval"}</p></li>)}</ul><div className={styles.actions}><button className="button secondary" disabled={busy || members.offset === 0} onClick={() => void load("listMembers", Math.max(0, members.offset - members.limit))}><UiText notice={uiMessage("diff-panel.previous.members.9423b1",{})}/></button><button className="button secondary" disabled={busy || members.offset + members.limit >= members.total} onClick={() => void load("listMembers", members.offset + members.limit)}><UiText notice={uiMessage("diff-panel.next.members.5909b2",{})}/></button></div></>}</section>;
}
