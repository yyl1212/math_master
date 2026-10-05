"use client";

import {uiError} from "@/lib/i18n/errors";

import {UiText} from "@/lib/i18n/ui-text";
import {uiMessage,uiValue} from "@/lib/i18n/format";
import {useUiI18n} from "@/lib/i18n/provider";
import Link from "next/link";
import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import { requestQuestion } from "@/lib/question/client";
import { contentRequest } from "@/lib/content/client";
import type { PublicationPage as ContentPublicationPage } from "@/lib/content/types";
import type { PublicationPage, PublicationSummary, SubmissionPage } from "@/lib/question/types";
import { questionInputError } from "@/lib/question/schemas";
import { TextField } from "@/features/content/field-controls";
import { useQuestionCommand } from "./command-controls";
import { DiffPanel } from "./diff-panel";
import styles from "@/styles/question.module.css";
export function PublicationPanel({ initial, knowledgeHead, selectedPublication }: {
    initial: PublicationPage;
    knowledgeHead: string | null;
    selectedPublication?: PublicationSummary;
}) {
 const {t}=useUiI18n();

    const [page, setPage] = useState(initial), [kHead, setKHead] = useState(knowledgeHead), [head, setHead] = useState<PublicationSummary | null>(null), [candidate, setCandidate] = useState<PublicationSummary | null>(selectedPublication ?? initial.items.find(p => p.status === "prepared") ?? initial.items[0] ?? null), [approved, setApproved] = useState<SubmissionPage | null>(null), [ids, setIDs] = useState<string[]>([]), [reason, setReason] = useState(""), [stale, setStale] = useState(false), [busy, setBusy] = useState(false), command = useQuestionCommand(), router = useRouter();
    useEffect(() => { const controller = new AbortController(); setHead(null); if (page.head)
        void requestQuestion<PublicationSummary>({ kind: "readPublication", id: page.head }, undefined, undefined, controller.signal).then(r => { if (controller.signal.aborted)
            return; if (r.ok && r.data.id === page.head)
            setHead(r.data);
        else
            command.error(r.ok ? uiMessage("publication-panel.current.snapshot.identity.does.not.match.f378f9",{}) : r.message); }); return () => controller.abort(); }, [page.head]);
    useEffect(() => { const controller = new AbortController(); void requestQuestion<SubmissionPage>({ kind: "listSubmissions", query: { scope: "all", status: "approved", limit: 20 } }, undefined, undefined, controller.signal).then(r => { if (controller.signal.aborted)
        return; if (r.ok)
        setApproved(r.data);
    else
        command.error(uiError("question",r)); }); return () => controller.abort(); }, []);
    useEffect(() => { if (command.failure?.code === "QUESTION_PUBLICATION_STALE")
        setStale(true); }, [command.failure]);
    async function refresh(offset = page.offset) { if (busy)
        return; setBusy(true); try {
        const [q, k] = await Promise.all([requestQuestion<PublicationPage>({ kind: "listPublications", query: { limit: page.limit, offset } }), contentRequest<ContentPublicationPage>({ kind: "listPublications", query: { limit: 1 } })]);
        if (!q.ok || !k.ok) {
            setStale(true);
            command.error(!q.ok ? q.message : !k.ok ? k.message : "");
            return;
        }
        setPage(q.data);
        setKHead(k.data.head);
        if (candidate && (candidate.baseKnowledgeHead !== k.data.head || candidate.baseQuestionHead !== q.data.head))
            setStale(true);
        router.refresh();
    }
    finally {
        setBusy(false);
    } }
    async function loadApproved(offset: number) { if (busy || !approved)
        return; setBusy(true); try {
        const r = await requestQuestion<SubmissionPage>({ kind: "listSubmissions", query: { scope: "all", status: "approved", limit: approved.limit, offset } });
        if (r.ok)
            setApproved(r.data);
        else
            command.error(uiError("question",r));
    }
    finally {
        setBusy(false);
    } }
    const prepare = { submissionIds: ids, expectedKnowledgeHead: kHead, expectedQuestionHead: page.head, reason }, activate = { expectedKnowledgeHead: kHead, expectedQuestionHead: page.head, expectedManifestSha: candidate?.manifestSha ?? "", reason };
    return <section className={styles.workbench}><p className="eyebrow"><UiText notice={uiMessage("publication-panel.question.publication.ef452a",{})}/></p><h1><UiText notice={uiMessage("publication-panel.trusted.question.snapshots.dca705",{})}/></h1><p className={styles.metadata}><UiText notice={uiMessage("publication-panel.current.knowledge.head.1a83a3",{})}/>{kHead ?? "None"}<br /><UiText notice={uiMessage("publication-panel.current.question.head.51281d",{})}/>{page.head ?? "None"}</p>{head && <p><UiText notice={uiMessage("publication-panel.current.published.bank.value.templates.value.trusted.instances.va.30647f",{v0:uiValue(head.templateCount),v1:uiValue(head.instanceCount),v2:uiValue(head.blueprintCount)})}/></p>}<p><Link prefetch={false} href="/admin/question-withdrawals"><UiText notice={uiMessage("publication-panel.withdraw.a.question.version.permanently.114e83",{})}/></Link></p><button className="button secondary" disabled={busy || command.blocked} onClick={() => void refresh()}><UiText notice={uiMessage("publication-panel.refresh.both.current.heads.13c170",{})}/></button>{command.controls}<fieldset disabled={command.blocked}><legend><UiText notice={uiMessage("publication-panel.publication.reason.8ae504",{})}/></legend><TextField label={t("publication-panel.publication.reason.8ae504",{})} value={reason} onChange={setReason} multiline/></fieldset><div className={styles.grid}><section className={styles.card}><h2><UiText notice={uiMessage("publication-panel.approved.submissions.86299b",{})}/></h2><p><UiText notice={uiMessage("publication-panel.select.1.20.independently.approved.submissions.inherited.members..a32964",{})}/></p>{!approved ? <p><UiText notice={uiMessage("publication-panel.loading.approved.submissions.523773",{})}/></p> : approved.items.length === 0 ? <p><UiText notice={uiMessage("publication-panel.no.approved.submissions.fc2830",{})}/></p> : approved.items.map(s => <label className={styles.check} key={s.id}><input type="checkbox" checked={ids.includes(s.id)} disabled={command.blocked || ids.length >= 20 && !ids.includes(s.id)} onChange={e => setIDs(v => e.target.checked ? [...v, s.id] : v.filter(id => id !== s.id))}/><span>{s.packageId}<UiText notice={uiMessage("publication-panel.v.e40f81",{})}/>{s.packageVersion} <Link prefetch={false} href={"/review/questions/" + s.id}><UiText notice={uiMessage("publication-panel.frozen.review.583f9b",{})}/></Link></span></label>)}{approved && <div className={styles.actions}><button className="button secondary" disabled={busy || command.blocked || approved.offset === 0} onClick={() => void loadApproved(Math.max(0, approved.offset - approved.limit))}><UiText notice={uiMessage("publication-panel.previous.approved.submissions.75d73f",{})}/></button><button className="button secondary" disabled={busy || command.blocked || approved.offset + approved.limit >= approved.total} onClick={() => void loadApproved(approved.offset + approved.limit)}><UiText notice={uiMessage("publication-panel.next.approved.submissions.d59bee",{})}/></button></div>}<p><UiText notice={uiMessage("publication-panel.value.submissions.selected.across.pages.f7aa8e",{v0:uiValue(ids.length)})}/></p><button className="button" disabled={busy || command.blocked || questionInputError("prepareRelease", prepare) !== null} onClick={() => void command.run({ kind: "prepareRelease" }, prepare, data => { const next = data as PublicationSummary; setCandidate(next); setStale(false); command.message(uiMessage("publication-panel.snapshot.prepared.inspect.its.complete.fixed.difference.and.evide.4818f7",{})); })}><UiText notice={uiMessage("publication-panel.prepare.snapshot.93466e",{})}/></button></section><section className={styles.card}><h2><UiText notice={uiMessage("publication-panel.snapshot.history.d81e02",{})}/></h2>{page.items.length === 0 ? <p><UiText notice={uiMessage("publication-panel.no.snapshots.on.this.page.2f19fb",{})}/></p> : <ul className={styles.list}>{page.items.map(p => <li key={p.id}><button className="button secondary" disabled={command.blocked} onClick={() => { setCandidate(p); setStale(false); }}>{p.status} · {p.id}</button><p><Link prefetch={false} href={"/admin/question-publications/" + p.id}><UiText notice={uiMessage("publication-panel.open.fixed.snapshot.d13edd",{})}/></Link></p></li>)}</ul>}<div className={styles.actions}><button className="button secondary" disabled={busy || command.blocked || page.offset === 0} onClick={() => void refresh(Math.max(0, page.offset - page.limit))}><UiText notice={uiMessage("publication-panel.previous.snapshots.1ead42",{})}/></button><button className="button secondary" disabled={busy || command.blocked || page.offset + page.limit >= page.total} onClick={() => void refresh(page.offset + page.limit)}><UiText notice={uiMessage("publication-panel.next.snapshots.28f079",{})}/></button></div></section></div>{candidate && <section className={styles.card}><h2><UiText notice={uiMessage("publication-panel.selected.value.snapshot.978a63",{v0:uiValue(candidate.status)})}/></h2><p className={styles.metadata}>{candidate.id}<br /><UiText notice={uiMessage("publication-panel.manifest.sha.aafd9c",{})}/>{candidate.manifestSha}<br /><UiText notice={uiMessage("publication-panel.base.knowledge.head.da6971",{})}/>{candidate.baseKnowledgeHead ?? "None"}<br /><UiText notice={uiMessage("publication-panel.base.question.head.c7e74a",{})}/>{candidate.baseQuestionHead ?? "None"}</p><p><UiText notice={uiMessage("publication-panel.value.templates.value.instances.value.blueprints.d899e0",{v0:uiValue(candidate.templateCount),v1:uiValue(candidate.instanceCount),v2:uiValue(candidate.blueprintCount)})}/></p><p><UiText notice={uiMessage("diff-panel.added.value.replaced.value.removed.value.83f44a",{v0:uiValue(candidate.diff.added),v1:uiValue(candidate.diff.replaced),v2:uiValue(candidate.diff.removed)})}/></p><DiffPanel publicationId={candidate.id} manifestSha={candidate.manifestSha}/>{(candidate.status === "prepared" && (stale || candidate.baseKnowledgeHead !== kHead || candidate.baseQuestionHead !== page.head)) && <p role="status"><UiText notice={uiMessage("publication-panel.published.snapshots.changed.refresh.both.heads.and.prepare.a.new..126274",{})}/></p>}<button className="button" disabled={busy || command.blocked || stale || candidate.status !== "prepared" || candidate.baseKnowledgeHead !== kHead || candidate.baseQuestionHead !== page.head || questionInputError("activateRelease", activate) !== null} onClick={() => void command.run({ kind: "activateRelease", id: candidate.id }, activate, async (data) => { await refresh(); setCandidate(data as PublicationSummary); setStale(false); command.message(uiMessage("publication-panel.snapshot.activated.both.current.heads.refreshed.550507",{})); })}><UiText notice={uiMessage("publication-panel.activate.snapshot.bd0268",{})}/></button></section>}</section>;
}
