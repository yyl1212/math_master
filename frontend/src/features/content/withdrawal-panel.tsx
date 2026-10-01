"use client";
import Link from "next/link";
import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import { contentRequest } from "@/lib/content/client";
import type { PublicationPage, WithdrawalTarget, WithdrawalPreview, WithdrawalResult } from "@/lib/content/types";
import { validContentInput } from "@/lib/content/schemas";
import { TextField, NumberField, SelectField } from "./field-controls";
import { DiffPanel } from "./diff-panel";
import { useContentCommand } from "./command-controls";
import styles from "@/styles/content.module.css";
export function WithdrawalPanel({ initial }: {
    initial?: PublicationPage;
}) {
    const router = useRouter(), command = useContentCommand(), [page, setPage] = useState<PublicationPage | null>(initial ?? null), [kind, setKind] = useState("knowledge"), [id, setID] = useState(""), [version, setVersion] = useState(1), [sha, setSHA] = useState(""), [reason, setReason] = useState(""), [preview, setPreview] = useState<WithdrawalPreview | null>(null);
    const target: WithdrawalTarget = kind === "asset" ? { kind: "asset", sha256: sha } : { kind: kind as "knowledge" | "unit" | "path", id, version };
    const current = page?.items.find(p => p.id === page.head);
    useEffect(() => { if (command.failure?.code === "PUBLICATION_STALE") {
        setPreview(null);
        command.clearPending();
    } }, [command.failure]);
    async function refresh() { const result = await contentRequest<PublicationPage>({ kind: "listPublications", query: { limit: 100 } }); if (result.ok) {
        setPage(result.data);
        return true;
    } command.error(result.message); return false; }
    useEffect(() => { if (!initial)
        void refresh(); }, []);
    function changed(fn: () => void) { fn(); setPreview(null); command.clearPending(); }
    return <section className={styles.workbench}><p className="eyebrow">CONTENT CORRECTION</p><h1>Withdraw a fixed version</h1><p>Withdrawal is permanent for this target. Dependent content leaves the current publication without changing its history.</p><p>{page?.head ? "Current head: " + page.head : "No content has been published yet."}</p>{command.controls}<section className={styles.card}><h2>Problem version</h2>{current && current.manifest.members.length > 0 && <label className={styles.field}>Published member<select defaultValue="" disabled={command.busy} onChange={e => { const m = current.manifest.members[Number(e.target.value)]?.identity; if (!m)
        return; changed(() => { setKind(m.kind); setID(m.id); setVersion(m.version); setSHA(m.sha256); }); }}><option value="">Choose a published member</option>{current.manifest.members.map((m, i) => <option key={m.identity.kind + "/" + m.identity.id} value={i}>{m.identity.kind}: {m.identity.id} v{m.identity.version}</option>)}</select></label>}<fieldset disabled={command.busy}><legend>Exact target</legend><SelectField label="Target kind" value={kind} options={["knowledge", "unit", "path", "asset"]} onChange={v => changed(() => setKind(v))}/>{kind === "asset" ? <TextField label="Target SVG SHA" value={sha} onChange={v => changed(() => setSHA(v))}/> : <><TextField label="Target ID" value={id} onChange={v => changed(() => setID(v))}/><NumberField label="Target version" value={version} onChange={v => changed(() => setVersion(v))}/></>}<TextField label="Withdrawal reason" value={reason} onChange={setReason} multiline/></fieldset><button className="button secondary" disabled={command.busy || !validContentInput("previewWithdrawal", { target })} onClick={() => void command.run({ kind: "previewWithdrawal" }, { target }, data => { setPreview(data as WithdrawalPreview); command.message("Inspect the full removal before withdrawing."); })}>Preview withdrawal</button></section>{preview && <section className={styles.card}><h2>Withdrawal preview</h2><p className={styles.metadata}>Based on head: {preview.currentHead ?? "none"}</p><DiffPanel diff={preview.diff}/></section>}<div className={styles.actions}><button className="button" disabled={command.busy || !preview || !validContentInput("withdrawVersion", { target: preview.target, expectedHead: preview.currentHead, reason })} onClick={() => { if (preview)
        void command.run({ kind: "withdrawVersion" }, { target: preview.target, expectedHead: preview.currentHead, reason }, async (data) => { setPreview(null); await refresh(); router.refresh(); command.message("Withdrawal complete. Current head refreshed."); }); }}>Withdraw version</button><button className="button secondary" disabled={command.busy} onClick={() => { setPreview(null); command.clearPending(); void refresh(); }}>Refresh head and clear preview</button></div><Link prefetch={false} href="/admin/publications">Back to publications</Link></section>;
}
