"use client";
import Link from "next/link";
import { useState } from "react";
import { useRouter } from "next/navigation";
import { contentRequest } from "@/lib/content/client";
import type { DraftPage, DraftView } from "@/lib/content/types";
import { NumberField, TextField } from "./field-controls";
import { useContentCommand } from "./command-controls";
import styles from "@/styles/content.module.css";
export function DraftList({ initial, canEdit }: {
    initial: DraftPage;
    canEdit: boolean;
}) {
    const router = useRouter(), command = useContentCommand(), [page, setPage] = useState(initial), [catalogue, setCatalogue] = useState(0), [id, setID] = useState(""), [version, setVersion] = useState(1), [reason, setReason] = useState("");
    async function load(offset: number) { const result = await contentRequest<DraftPage>({ kind: "listDrafts", query: { scope: canEdit ? "mine" : "all", limit: page.limit, offset } }); if (result.ok)
        setPage(result.data);
    else
        command.error(result.message); }
    const open = (data: unknown) => router.push("/editor/drafts/" + (data as DraftView).id);
    return <section className={styles.workbench}><p className="eyebrow">AUTHORING</p><h1>Content workspaces</h1><p>Original content begins here. Saving and independent review have separate checks.</p>{command.controls}<ul className={styles.list}>{page.items.map(d => <li key={d.id}><Link prefetch={false} href={"/editor/drafts/" + d.id}>{d.packageId} · Version {d.packageVersion}</Link><p>Revision {d.revision} · {d.status} · {d.structuralTotal + d.completenessTotal} machine issues</p><Link prefetch={false} href={"/editor/drafts/" + d.id + "/preview"}>Read saved draft</Link></li>)}</ul>{!page.items.length && <p>No workspaces yet.</p>}<div className={styles.actions}><button className="button secondary" disabled={page.offset === 0} onClick={() => void load(Math.max(0, page.offset - page.limit))}>Previous</button><button className="button secondary" disabled={page.offset + page.limit >= page.total} onClick={() => void load(page.offset + page.limit)}>Next</button></div>{canEdit && <div className={styles.grid}><form className={styles.card} onSubmit={e => { e.preventDefault(); void command.run({ kind: "createDraft" }, { catalogueVersion: catalogue, package: { schemaVersion: 1, id, version, knowledge: [], units: [], paths: [], assets: [] }, assetBytes: [], sourceMap: [] }, open); }}><h2>Create workspace</h2><fieldset disabled={command.busy}><legend>New package</legend><TextField label="New package ID" value={id} onChange={setID}/><NumberField label="New package version" value={version} onChange={setVersion}/><NumberField label="New catalogue version" value={catalogue} onChange={setCatalogue}/></fieldset><p>Choose a trusted imported catalogue version.</p><button className="button" disabled={command.busy}>Create draft</button></form><form className={styles.card} onSubmit={e => { e.preventDefault(); void command.run({ kind: "adoptDraft" }, { packageId: id, packageVersion: version, reason }, open); }}><h2>Adopt an imported package</h2><p>The package ID and version above identify the immutable imported package.</p><TextField label="Adoption reason" value={reason} onChange={setReason} multiline/><button className="button secondary" disabled={command.busy}>Adopt package</button></form></div>}</section>;
}
