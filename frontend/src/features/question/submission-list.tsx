"use client";
import {UiEnum,formatUiEnum} from "@/lib/i18n/enums";
import {uiError} from "@/lib/i18n/errors";
import type {UiNotice} from "@/lib/i18n/types";
import {UiText} from "@/lib/i18n/ui-text";
import {uiMessage,uiValue} from "@/lib/i18n/format";
import {useUiI18n} from "@/lib/i18n/provider";
import Link from "next/link";
import { useState } from "react";
import { requestQuestion } from "@/lib/question/client";
import type { SubmissionPage } from "@/lib/question/types";
import { FormMessage } from "@/features/auth/auth-state";
import styles from "@/styles/question.module.css";
export function SubmissionList({ initial, scope }: {
    initial: SubmissionPage;
    scope: "mine" | "review" | "all";
}) {
 const {t,locale}=useUiI18n();
 const [page, setPage] = useState(initial), [status, setStatus] = useState<"pending" | "approved" | "returned">("pending"), [error, setError] = useState<UiNotice | null>(null), [busy, setBusy] = useState(false); async function load(offset: number, next = status) { if (busy)
    return; setBusy(true); try {
    const r = await requestQuestion<SubmissionPage>({ kind: "listSubmissions", query: { scope, status: next, limit: page.limit, offset } });
    if (r.ok) {
        setPage(r.data);
        setStatus(next);
        setError(null);
    }
    else
        setError(uiError("question",r));
}
finally {
    setBusy(false);
} } return <section className={styles.workbench}><h2>{scope === "review" ? t("submission-list.independent.question.review.queue.006495",{}) : t("submission-list.frozen.question.submissions.250eef",{})}</h2><p><UiText notice={uiMessage("submission-list.review.uses.immutable.submitted.questions.and.every.bound.instanc.50a797",{})}/></p><FormMessage notice={error} error/><label><UiText notice={uiMessage("submission-list.status.8bdde2",{})}/><select value={status} disabled={busy} onChange={e => void load(0, e.target.value as typeof status)}><option value="pending"><UiText notice={uiMessage("submission-list.pending.331551",{})}/></option><option value="approved"><UiText notice={uiMessage("submission-list.approved.87b42e",{})}/></option><option value="returned"><UiText notice={uiMessage("submission-list.returned.361023",{})}/></option></select></label><ul className={styles.list}>{page.items.map(s => <li key={s.id}><Link prefetch={false} href={"/review/questions/" + s.id}><UiText notice={uiMessage("content-preview.value.vvalue.bde90b",{v0:uiValue(s.packageId),v1:uiValue(s.packageVersion)})}/></Link><p><UiText notice={uiMessage("submission-list.value.frozen.revision.value.cfefdf",{v0:formatUiEnum(locale,"question.state",s.status),v1:uiValue(s.revision)})}/></p><p className={styles.metadata}><UiText notice={uiMessage("submission-list.digest.value.b74ba8",{v0:uiValue(s.frozenDigest)})}/></p></li>)}</ul>{!page.items.length && <p><UiText notice={uiMessage("submission-list.no.submissions.in.this.view.28cce2",{})}/></p>}<div className={styles.actions}><button className="button secondary" disabled={busy || page.offset === 0} onClick={() => void load(Math.max(0, page.offset - page.limit))}><UiText notice={uiMessage("submission-list.previous.submissions.8d9ab0",{})}/></button><button className="button secondary" disabled={busy || page.offset + page.limit >= page.total} onClick={() => void load(page.offset + page.limit)}><UiText notice={uiMessage("submission-list.next.submissions.ea5832",{})}/></button></div></section>; }
