"use client";
import Link from "next/link";
import { useState } from "react";
import { useRouter } from "next/navigation";
import type { User } from "@/lib/auth/types";
import { requestQuestion } from "@/lib/question/client";
import { questionInputError } from "@/lib/question/schemas";
import type { SubmissionView, InstancePage, ReviewChecks, DraftView } from "@/lib/question/types";
import { TextField } from "@/features/content/field-controls";
import { GenerationPanel, InstancePreview, Sources } from "./generation-panel";
import { useQuestionCommand } from "./command-controls";
import styles from "@/styles/question.module.css";
export function ReviewPanel({ submission, initialInstances, user }: {
    submission: SubmissionView;
    initialInstances: InstancePage;
    user: User;
}) {
    const [view, setView] = useState(submission), [instances, setInstances] = useState(initialInstances), [pageBusy, setPageBusy] = useState(false), [checks, setChecks] = useState<ReviewChecks>({ mathematics: false, explanations: false, objectives: false, sources: false, illustrations: false, generation: false }), [independenceNote, setIndependence] = useState(""), [generationNote, setGeneration] = useState(""), [note, setNote] = useState(""), command = useQuestionCommand(), router = useRouter();
    const isAuthor = view.frozen.authorIds.includes(user.id), eligible = !user.mustChangePassword && user.roles.includes("reviewer") && !isAuthor && view.status === "pending";
    const review = (decision: "approve" | "return") => ({ decision, checks, independenceNote, generationNote, note });
    async function load(offset: number) { if (pageBusy)
        return; setPageBusy(true); try {
        const r = await requestQuestion<InstancePage>({ kind: "listInstances", id: view.id, query: { limit: instances.limit, offset } });
        if (r.ok) {
            const bound = new Map(view.frozen.instanceIdentities.map(i => [i.id + ":" + i.version, i.sha256]));
            if (r.data.total !== view.frozen.instanceIdentities.length || r.data.items.some(i => bound.get(i.identity.id + ":" + i.identity.version) !== i.identity.sha256)) {
                command.error("Instance data does not match this frozen submission.");
                return;
            }
            setInstances(r.data);
        }
        else
            command.error(r.message);
    }
    finally {
        setPageBusy(false);
    } }
    return <section className={styles.workbench}><p className="eyebrow">FROZEN QUESTION REVIEW</p><h1>{view.frozen.questionPackage.id} · Version {view.frozen.questionPackage.version}</h1><p>{view.status} · Frozen revision {view.revision} · Catalogue version {view.frozen.catalogueVersion}</p><p className={styles.metadata}>Frozen digest: {view.frozen.frozenDigest}<br />Catalogue SHA: {view.frozen.catalogueSha256}<br />Authors: {view.frozen.authorIds.join(", ")}<br />Generator versions: {view.frozen.generatorVersions.join(", ") || "None"} · Verifier versions: {view.frozen.verifierVersions.join(", ") || "None"}</p>{view.frozen.legacyUnattributed && <p className={styles.notice}>Verify historical authorship and original sources before approval.</p>}<h2>Frozen learning objectives</h2>{view.frozen.objectives.map((o, i) => <p key={i}>{o.knowledge.id} v{o.knowledge.version} · Index {o.objectiveIndex}: <span>{o.text}</span></p>)}<h2>Frozen source mapping</h2>{view.frozen.sourceMap.length ? <ul>{view.frozen.sourceMap.map((s, i) => <li key={i}><strong>{s.relativePath}</strong><p className={styles.metadata}>Batch SHA {s.batchSha256} · Source SHA {s.sha256}</p><p>{s.legacyId} · {s.note}</p></li>)}</ul> : <p>No source mapping declared. Verify original provenance independently.</p>}<GenerationPanel report={view.gate} revision={view.revision}/>{view.frozen.questionPackage.templates.map(t => <section className={styles.card} key={t.id}><h2>{t.id} · {t.engine.family}</h2><p>Exact parameters: {t.parameters.map(p => p.name + " = " + p.values.join(", ")).join("; ")}</p><p>Constraints: {t.constraints.join(", ") || "None"}</p><p>Distractors: {t.distractors.join(", ") || "None"}</p><Sources sources={t.sources}/><details><summary>Complete frozen template and engine specification</summary><pre>{JSON.stringify(t, null, 2)}</pre></details></section>)}<details><summary>Complete frozen package, blueprints and reference identities</summary><pre>{JSON.stringify(view.frozen, null, 2)}</pre></details><h2>Bound question instances</h2><p>Showing {instances.items.length} of {instances.total} instances · Offset {instances.offset}. Every instance is available through these pages.</p>{instances.items.map(i => <InstancePreview key={i.identity.id} instance={i}/>)}<div className={styles.actions}><button className="button secondary" disabled={pageBusy || instances.offset === 0} onClick={() => void load(Math.max(0, instances.offset - instances.limit))}>Previous instances</button><button className="button secondary" disabled={pageBusy || instances.offset + instances.limit >= instances.total} onClick={() => void load(instances.offset + instances.limit)}>Next instances</button></div>{isAuthor && <p>Authors cannot review their own or inherited questions.</p>}{eligible && <section className={styles.card}><h2>Independent review</h2><fieldset disabled={command.blocked}><legend>Complete all six checks before approval</legend>{(Object.keys(checks) as (keyof ReviewChecks)[]).map(key => <label className={styles.check} key={key}><input type="checkbox" checked={checks[key]} onChange={e => setChecks(v => ({ ...v, [key]: e.target.checked }))}/>{key[0].toUpperCase() + key.slice(1)}</label>)}<TextField label="Independence statement" value={independenceNote} onChange={setIndependence} multiline/><TextField label="Generation review statement" value={generationNote} onChange={setGeneration} multiline/><p>When no templates are present, explain explicitly why generation does not apply.</p><TextField label="Review note" value={note} onChange={setNote} multiline/></fieldset><div className={styles.actions}><button className="button" disabled={command.blocked || questionInputError("decideReview", review("approve")) !== null} onClick={() => void command.run({ kind: "decideReview", id: view.id }, review("approve"), data => { setView(data as SubmissionView); command.message("Independent review saved."); router.refresh(); })}>Approve submission</button><button className="button secondary" disabled={command.blocked || questionInputError("decideReview", review("return")) !== null} onClick={() => void command.run({ kind: "decideReview", id: view.id }, review("return"), data => { setView(data as SubmissionView); command.message("Submission returned for changes."); router.refresh(); })}>Return for changes</button></div></section>}{command.controls}{view.review && <section className={styles.card}><h2>Final review decision</h2><p>{view.review.decision} · {view.review.createdAt}</p><p>{view.review.independenceNote}</p><p>{view.review.generationNote}</p><p>{view.review.note}</p><p className={styles.metadata}>Reviewer: {view.review.reviewerId}</p></section>}{view.ownerId === user.id && user.roles.includes("editor") && (view.status === "approved" ? <button className="button secondary" disabled={command.blocked} onClick={() => void command.run({ kind: "reviseSubmission", id: view.id }, {}, data => router.push("/editor/questions/drafts/" + (data as DraftView).id))}>Create revision workspace</button> : view.status === "returned" ? <Link prefetch={false} href={"/editor/questions/drafts/" + view.workspaceId}>Resume returned workspace</Link> : null)}<p><Link prefetch={false} href="/review/questions">Back to question submissions</Link></p></section>;
}
