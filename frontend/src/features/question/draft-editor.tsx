"use client";
import Link from "next/link";
import { useState } from "react";
import { useRouter } from "next/navigation";
import { TextField, NumberField, Rows } from "@/features/content/field-controls";
import { requestQuestion } from "@/lib/question/client";
import { questionInputError, parseQuestionJSON, questionCanonicalJSON } from "@/lib/question/schemas";
import type { DraftInput, DraftView, SubmissionView, ValidationReport } from "@/lib/question/types";
import { TemplateFields, newTemplate } from "./template-fields";
import { FixedFields, newFixed } from "./fixed-fields";
import { BlueprintFields, newBlueprint } from "./blueprint-fields";
import { GenerationPanel } from "./generation-panel";
import { useQuestionCommand } from "./command-controls";
import styles from "@/styles/question.module.css";
const editable = (d: DraftView): DraftInput => ({ catalogueVersion: d.catalogueVersion, questionPackage: d.questionPackage, sourceMap: d.sourceMap });
export function DraftEditor({ initial, canEdit = false }: {
    initial: DraftView;
    canEdit?: boolean;
}) {
    const router = useRouter(), command = useQuestionCommand(), [saved, setSaved] = useState(initial), [input, setInput] = useState<DraftInput>(() => editable(initial)), [validated, setValidated] = useState<{
        revision: number;
        report: ValidationReport;
    } | null>(null), [submitted, setSubmitted] = useState<string | null>(null), [json, setJSON] = useState(() => JSON.stringify(editable(initial), null, 2));
    const dirty = questionCanonicalJSON(input) !== questionCanonicalJSON(editable(saved)), write = canEdit && saved.status === "editing", saveInput = { ...input, expectedRevision: saved.revision };
    const changePackage = (p: DraftInput["questionPackage"]) => setInput(v => ({ ...v, questionPackage: p }));
    function importJSON(raw: Uint8Array) { try {
        if (raw.byteLength > 4194304)
            throw new Error();
        const next = parseQuestionJSON(raw);
        if (questionInputError("createDraft", next))
            throw new Error();
        setInput(next as DraftInput);
        setValidated(null);
        setJSON(JSON.stringify(next, null, 2));
        command.message("Editable question data imported. Save and validate this revision before submitting.");
    }
    catch {
        command.error("Invalid or oversized editable question JSON. The current input was preserved.");
    } }
    function download() { const blob = new Blob([JSON.stringify(input, null, 2)], { type: "application/json" }); const url = URL.createObjectURL(blob), a = document.createElement("a"); a.href = url; a.download = "question-workspace.json"; a.click(); URL.revokeObjectURL(url); }
    return <section className={styles.workbench}><p className="eyebrow">QUESTION AUTHORING</p><h1>{saved.questionPackage.id} · Version {saved.questionPackage.version}</h1><p>Saved revision {saved.revision} · {saved.status} {dirty && "· Unsaved changes"}</p><p>Saving, machine validation and independent approval are separate steps.</p><p className={styles.metadata}>Owner: {saved.ownerId} · Authors: {saved.authorIds.join(", ")}</p>{saved.legacyUnattributed && <p className={styles.notice}>Verify historical authorship and original sources before approval.</p>}{!write && <p>This workspace is read-only for this account or has already been submitted.</p>}{command.controls}<fieldset disabled={!write || command.blocked}><legend>Editable question package</legend><div className={styles.grid}><NumberField label="Catalogue version" value={input.catalogueVersion} onChange={catalogueVersion => setInput(v => ({ ...v, catalogueVersion }))}/><TextField label="Package ID" value={input.questionPackage.id} onChange={id => changePackage({ ...input.questionPackage, id })}/><NumberField label="Package version" value={input.questionPackage.version} onChange={version => changePackage({ ...input.questionPackage, version })}/></div><h2>Finite question templates</h2><p>Rational arithmetic · Rational comparison · Missing operand</p><Rows label="template" items={input.questionPackage.templates} onChange={templates => changePackage({ ...input.questionPackage, templates })} create={newTemplate}>{(v, i, set) => <TemplateFields label={`Template ${i + 1}`} value={v} onChange={set}/>}</Rows><h2>Fixed questions</h2><Rows label="fixed question" items={input.questionPackage.fixedQuestions} onChange={fixedQuestions => changePackage({ ...input.questionPackage, fixedQuestions })} create={newFixed}>{(v, i, set) => <FixedFields label={`Fixed question ${i + 1}`} value={v} onChange={set}/>}</Rows><h2>Assessment blueprints</h2><Rows label="blueprint" items={input.questionPackage.blueprints} onChange={blueprints => changePackage({ ...input.questionPackage, blueprints })} create={newBlueprint}>{(v, i, set) => <BlueprintFields label={`Blueprint ${i + 1}`} value={v} onChange={set}/>}</Rows><details><summary>Editable JSON and source mapping</summary><p>Import and export contain editable question data and source mapping. Original author and review evidence are retained by the server.</p><TextField label="Editable question JSON" value={json} onChange={setJSON} multiline/><div className={styles.actions}><button type="button" className="button secondary" onClick={() => importJSON(new TextEncoder().encode(json))}>Apply editable JSON</button><button type="button" className="button secondary" onClick={() => setJSON(JSON.stringify(input, null, 2))}>Copy current fields to JSON</button><button type="button" className="button secondary" disabled={questionInputError("createDraft", input) !== null} onClick={download}>Export editable JSON</button><label>Import question JSON file<input aria-label="Import question JSON file" type="file" accept=".json,application/json" onChange={async (e) => { const file = e.target.files?.[0]; if (file) {
        if (file.size > 4194304)
            command.error("Question JSON exceeds the request size limit.");
        else
            importJSON(new Uint8Array(await file.arrayBuffer()));
    } e.target.value = ""; }}/></label></div></details></fieldset><div className={styles.toolbar}><div className={styles.actions}><button className="button" disabled={!write || command.blocked || questionInputError("saveDraft", saveInput) !== null} onClick={() => void command.run({ kind: "saveDraft", id: saved.id }, saveInput, data => { const next = data as DraftView; setSaved(next); setInput(editable(next)); setJSON(JSON.stringify(editable(next), null, 2)); setValidated(null); command.message("Draft saved. Validate the new saved revision."); })}>Save draft</button><button className="button secondary" disabled={!write || command.blocked || dirty} onClick={() => void command.run({ kind: "validateDraft", id: saved.id }, { expectedRevision: saved.revision }, data => setValidated({ revision: saved.revision, report: data as ValidationReport }))}>Validate saved revision</button><button className="button" disabled={!write || command.blocked || dirty || validated?.revision !== saved.revision || !validated.report.readyToSubmit} onClick={() => { if (validated)
        void command.run({ kind: "submitDraft", id: saved.id }, { expectedRevision: saved.revision, expectedDigest: validated.report.digest }, data => { setSubmitted((data as SubmissionView).id); setSaved(v => ({ ...v, status: "submitted" })); command.message("The saved revision was frozen for independent review."); router.refresh(); }); }}>Submit for review</button><button className="button secondary" disabled={command.busy} onClick={async () => { const result = await requestQuestion<DraftView>({ kind: "readDraft", id: saved.id }); if (result.ok) {
        setSaved(result.data);
        setValidated(null);
        command.clearPending();
        command.message("Saved version reloaded. Your editable input was preserved.");
    }
    else
        command.error(result.message); }}>Reload saved version</button></div></div><GenerationPanel report={validated?.report ?? saved.gate} revision={validated?.revision ?? saved.revision}/>{submitted && <Link prefetch={false} href={"/review/questions/" + submitted}>Open frozen submission</Link>}<p><Link prefetch={false} href="/editor/questions">Back to question workspaces</Link></p></section>;
}
