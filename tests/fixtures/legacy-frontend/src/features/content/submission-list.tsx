"use client";
import {UiEnum,formatUiEnum} from "@/lib/i18n/enums";
import {uiError} from "@/lib/i18n/errors";
import type {UiNotice} from "@/lib/i18n/types";
import {UiText} from "@/lib/i18n/ui-text";
import {uiMessage,uiValue} from "@/lib/i18n/format";
import {useUiI18n} from "@/lib/i18n/provider";
import Link from "next/link";
import { useState } from "react";
import { contentRequest } from "@/lib/content/client";
import type { SubmissionPage } from "@/lib/content/types";
import styles from "@/styles/content.module.css";
import { FormMessage } from "@/features/auth/auth-state";
export function SubmissionList({ initial, scope }: {
    initial: SubmissionPage;
    scope: "mine" | "review" | "all";
}) {
 const {t,locale}=useUiI18n();

    const [page, setPage] = useState(initial), [status, setStatus] = useState("pending"), [busy, setBusy] = useState(false), [error, setError] = useState<UiNotice | null>(null);
    async function load(next: string, offset = 0) { if (busy)
        return; setBusy(true); try {
        const result = await contentRequest<SubmissionPage>({ kind: "listSubmissions", query: { scope, status: next as "pending", limit: page.limit, offset } });
        if (result.ok) {
            setPage(result.data);
            setStatus(next);
            setError(null);
        }
        else
            setError(uiError("content",result));
    }
    finally {
        setBusy(false);
    } }
    return <section className={styles.workbench}><h1>{scope === "review" ? t("submission-list.independent.review.queue.2ca777",{}) : t("submission-list.submissions.541db6",{})}</h1><p><UiText notice={uiMessage("submission-list.each.submission.fixes.the.entire.batch.sources.authors.and.illust.330fde",{})}/></p><label className={styles.field}><UiText notice={uiMessage("submission-list.submission.status.eeb12e",{})}/><select value={status} disabled={busy} onChange={e => void load(e.target.value)}>{["pending", "approved", "returned"].map(v => <option key={v} value={v}><UiEnum group="content.enum" value={v}/></option>)}</select></label><FormMessage notice={error} error/><ul className={styles.list}>{page.items.map(s => <li key={s.id}><Link prefetch={false} href={"/review/" + s.id}><UiText notice={uiMessage("content-preview.value.version.value.1d1973",{v0:uiValue(s.packageId),v1:uiValue(s.packageVersion)})}/></Link><p><UiText notice={uiMessage("submission-list.value.revision.value.6706c2",{v0:formatUiEnum(locale,"content.enum",s.status),v1:uiValue(s.revision)})}/></p><p className={styles.metadata}><UiText notice={uiMessage("submission-list.frozen.digest.value.bfb773",{v0:uiValue(s.frozenDigest)})}/></p></li>)}</ul>{!page.items.length && <p><UiText notice={uiMessage("submission-list.no.matching.submissions.0c6f73",{})}/></p>}<div className={styles.actions}><button className="button secondary" disabled={busy || !page.offset} onClick={() => void load(status, Math.max(0, page.offset - page.limit))}><UiText notice={uiMessage("admin-users.previous.a57b08",{})}/></button><button className="button secondary" disabled={busy || page.offset + page.limit >= page.total} onClick={() => void load(status, page.offset + page.limit)}><UiText notice={uiMessage("admin-users.next.1ff57a",{})}/></button></div></section>;
}
