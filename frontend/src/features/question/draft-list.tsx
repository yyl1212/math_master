"use client";
import Link from "next/link";
import { useState } from "react";
import { useRouter } from "next/navigation";
import { TextField, NumberField } from "@/features/content/field-controls";
import { requestQuestion } from "@/lib/question/client";
import type { DraftPage, DraftView } from "@/lib/question/types";
import { useQuestionCommand } from "./command-controls";
import styles from "@/styles/question.module.css";
export function DraftList({ initial, canEdit }: {
    initial: DraftPage;
    canEdit: boolean;
}) {
    const [page, setPage] = useState(initial), [id, setID] = useState(""), [version, setVersion] = useState(1), [catalogue, setCatalogue] = useState(1), [reason, setReason] = useState(""), command = useQuestionCommand(), router = useRouter();
    async function load(offset: number) { const result = await requestQuestion<DraftPage>({ kind: "listDrafts", query: { scope: canEdit ? "mine" : "all", limit: page.limit, offset } }); if (result.ok)
        setPage(result.data);
    else
        command.error(result.message); }
    const open = (data: unknown) => router.push("/editor/questions/drafts/" + (data as DraftView).id);
    return <section className={styles.workbench}><p className="eyebrow">TRUSTED QUESTION BANK</p><h1>Question workspaces</h1><p>Build exact questions, validate the finite batch and submit it for independent review.</p>{command.controls}<ul className={styles.list}>{page.items.map(d => <li key={d.id}><Link prefetch={false} href={"/editor/questions/drafts/" + d.id}>{d.packageId} · Version {d.packageVersion}</Link><p>Revision {d.revision} · {d.status} · {d.structuralTotal + d.completenessTotal} machine issues</p></li>)}</ul>{!page.items.length && <p>No question workspaces yet.</p>}<div className={styles.actions}><button className="button secondary" disabled={command.busy || page.offset === 0} onClick={() => void load(Math.max(0, page.offset - page.limit))}>Previous workspaces</button><button className="button secondary" disabled={command.busy || page.offset + page.limit >= page.total} onClick={() => void load(page.offset + page.limit)}>Next workspaces</button></div>{canEdit ? <div className={styles.grid}><form className={styles.card} onSubmit={e => { e.preventDefault(); void command.run({ kind: "createDraft" }, { catalogueVersion: catalogue, questionPackage: { kind: "question-bank", schemaVersion: 1, id, version, templates: [], fixedQuestions: [], blueprints: [] }, sourceMap: [] }, open); }}><h2>Create question workspace</h2><fieldset disabled={command.blocked}><legend>New editable package</legend><TextField label="New question package ID" value={id} onChange={setID}/><NumberField label="New question package version" value={version} onChange={setVersion}/><NumberField label="New catalogue version" value={catalogue} onChange={setCatalogue}/></fieldset><button className="button" disabled={command.blocked}>Create draft</button></form><form className={styles.card} onSubmit={e => { e.preventDefault(); void command.run({ kind: "adoptDraft" }, { packageId: id, packageVersion: version, reason }, open); }}><h2>Adopt an imported question package</h2><p>Use the exact package ID and version above. Historical authorship is retained; independent approval is still required.</p><fieldset disabled={command.blocked}><legend>Adoption reason</legend><TextField label="Adoption reason" value={reason} onChange={setReason} multiline/></fieldset><button className="button secondary" disabled={command.blocked}>Adopt package</button></form></div> : <p>Editor permission is required to create or change questions.</p>}</section>;
}
