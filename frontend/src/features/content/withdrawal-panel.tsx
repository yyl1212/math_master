"use client";
import {UiEnum,formatUiEnum,enumOptionLabels} from "@/lib/i18n/enums";
import {uiError} from "@/lib/i18n/errors";

import {UiText} from "@/lib/i18n/ui-text";
import {uiMessage,uiValue} from "@/lib/i18n/format";
import {useUiI18n} from "@/lib/i18n/provider";
import Link from "next/link";
import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import { contentRequest } from "@/lib/content/client";
import type { PublicationPage, PublicationView, WithdrawalTarget, WithdrawalPreview, WithdrawalResult } from "@/lib/content/types";
import { validContentInput } from "@/lib/content/schemas";
import { TextField, NumberField, SelectField } from "./field-controls";
import { DiffPanel } from "./diff-panel";
import { useContentCommand } from "./command-controls";
import styles from "@/styles/content.module.css";
export function WithdrawalPanel({ initial }: {
    initial?: PublicationPage;
}) {
 const {t}=useUiI18n();

    const router = useRouter(), command = useContentCommand(), [page, setPage] = useState<PublicationPage | null>(initial ?? null), [kind, setKind] = useState("knowledge"), [id, setID] = useState(""), [version, setVersion] = useState(1), [sha, setSHA] = useState(""), [reason, setReason] = useState(""), [preview, setPreview] = useState<WithdrawalPreview | null>(null);
    const target: WithdrawalTarget = kind === "asset" ? { kind: "asset", sha256: sha } : { kind: kind as "knowledge" | "unit" | "path", id, version };
    const current = page?.items.find(p => p.id === page.head);
    useEffect(() => { if (command.failure?.code === "PUBLICATION_STALE") {
        setPreview(null);
        command.clearPending();
    } }, [command.failure]);
    async function refresh() { const result = await contentRequest<PublicationPage>({ kind: "listPublications", query: { limit: 100 } }); if (result.ok) {
        const next = result.data;
        if (next.head && !next.items.some(v => v.id === next.head)) {
            const head = await contentRequest<PublicationView>({ kind: "readPublication", id: next.head });
            if (!head.ok) { command.error(uiError("content",head)); return false; }
            next.items = [head.data, ...next.items];
        }
        setPage(next);
        return true;
    } command.error(uiError("content",result)); return false; }
    useEffect(() => { if (!initial)
        void refresh(); }, []);
    function changed(fn: () => void) { fn(); setPreview(null); command.clearPending(); }
    return <section className={styles.workbench}><p className="eyebrow"><UiText notice={uiMessage("withdrawal-panel.content.correction.c9be22",{})}/></p><h1><UiText notice={uiMessage("publication-panel.withdraw.a.fixed.version.2cca43",{})}/></h1><p><UiText notice={uiMessage("withdrawal-panel.withdrawal.is.permanent.for.this.target.dependent.content.leaves..ba7e1a",{})}/></p><p>{page?.head ? "Current head: " + page.head : t("publication-panel.no.content.has.been.published.yet.e8904f",{})}</p>{command.controls}<section className={styles.card}><h2><UiText notice={uiMessage("withdrawal-panel.problem.version.ff5f6e",{})}/></h2>{current && current.manifest.members.length > 0 && <label className={styles.field}><UiText notice={uiMessage("withdrawal-panel.published.member.2d0d52",{})}/><select defaultValue="" disabled={command.busy} onChange={e => { const m = current.manifest.members[Number(e.target.value)]?.identity; if (!m)
        return; changed(() => { setKind(m.kind); setID(m.id); setVersion(m.version); setSHA(m.sha256); }); }}><option value=""><UiText notice={uiMessage("withdrawal-panel.choose.a.published.member.95163c",{})}/></option>{current.manifest.members.map((m, i) => <option key={m.identity.kind + "/" + m.identity.id} value={i}><UiText notice={uiMessage("content-preview.value.value.vvalue.afac6e",{v0:uiValue(m.identity.kind),v1:uiValue(m.identity.id),v2:uiValue(m.identity.version)})}/></option>)}</select></label>}<fieldset disabled={command.busy}><legend><UiText notice={uiMessage("withdrawal-panel.exact.target.93f4b6",{})}/></legend><SelectField label={t("withdrawal-panel.target.kind.607b91",{})} value={kind} options={["knowledge", "unit", "path", "asset"]} optionLabels={enumOptionLabels("content.enum",["knowledge", "unit", "path", "asset"])} onChange={v => changed(() => setKind(v))}/>{kind === "asset" ? <TextField label={t("withdrawal-panel.target.svg.sha.7c995e",{})} value={sha} onChange={v => changed(() => setSHA(v))}/> : <><TextField label={t("withdrawal-panel.target.id.32a090",{})} value={id} onChange={v => changed(() => setID(v))}/><NumberField label={t("withdrawal-panel.target.version.2acb4e",{})} value={version} onChange={v => changed(() => setVersion(v))}/></>}<TextField label={t("withdrawal-panel.withdrawal.reason.6fb3ad",{})} value={reason} onChange={setReason} multiline/></fieldset><button className="button secondary" disabled={command.busy || !validContentInput("previewWithdrawal", { target })} onClick={() => void command.run({ kind: "previewWithdrawal" }, { target }, data => { setPreview(data as WithdrawalPreview); command.message(uiMessage("withdrawal-panel.inspect.the.full.removal.before.withdrawing.518633",{})); })}><UiText notice={uiMessage("withdrawal-panel.preview.withdrawal.69c4c0",{})}/></button></section>{preview && <section className={styles.card}><h2><UiText notice={uiMessage("withdrawal-panel.withdrawal.preview.6e7687",{})}/></h2><p className={styles.metadata}><UiText notice={uiMessage("withdrawal-panel.based.on.head.value.afc16c",{v0:uiValue(preview.currentHead ?? "none")})}/></p><DiffPanel diff={preview.diff}/></section>}<div className={styles.actions}><button className="button" disabled={command.busy || !preview || !validContentInput("withdrawVersion", { target: preview.target, expectedHead: preview.currentHead, reason })} onClick={() => { if (preview)
        void command.run({ kind: "withdrawVersion" }, { target: preview.target, expectedHead: preview.currentHead, reason }, async (data) => { setPreview(null); await refresh(); router.refresh(); command.message(uiMessage("withdrawal-panel.withdrawal.complete.current.head.refreshed.0518f3",{})); }); }}><UiText notice={uiMessage("withdrawal-panel.withdraw.version.8e7a90",{})}/></button><button className="button secondary" disabled={command.busy} onClick={() => { setPreview(null); command.clearPending(); void refresh(); }}><UiText notice={uiMessage("withdrawal-panel.refresh.head.and.clear.preview.43cd61",{})}/></button></div><Link prefetch={false} href="/admin/publications"><UiText notice={uiMessage("withdrawal-panel.back.to.publications.ebb489",{})}/></Link></section>;
}
