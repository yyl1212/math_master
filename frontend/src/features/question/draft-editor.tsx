"use client";
import type {UiNotice} from "@/lib/i18n/types";
import {uiError} from "@/lib/i18n/errors";

import {UiText} from "@/lib/i18n/ui-text";
import {uiMessage,uiValue} from "@/lib/i18n/format";
import {useUiI18n} from "@/lib/i18n/provider";
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
 const {t}=useUiI18n();

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
        command.message(uiMessage("draft-editor.editable.question.data.imported.save.and.validate.this.revision.b.a2437c",{}));
    }
    catch {
        command.error(uiMessage("draft-editor.invalid.or.oversized.editable.question.json.the.current.input.was.899b5c",{}));
    } }
    function download() { const blob = new Blob([JSON.stringify(input, null, 2)], { type: "application/json" }); const url = URL.createObjectURL(blob), a = document.createElement("a"); a.href = url; a.download = "question-workspace.json"; a.click(); URL.revokeObjectURL(url); }
    return <section className={styles.workbench}><p className="eyebrow"><UiText notice={uiMessage("draft-editor.question.authoring.932915",{})}/></p><h1><UiText notice={uiMessage("content-preview.value.version.value.1d1973",{v0:uiValue(saved.questionPackage.id),v1:uiValue(saved.questionPackage.version)})}/></h1><p><UiText notice={uiMessage("draft-editor.saved.revision.value.value.value.b3456a",{v0:uiValue(saved.revision),v1:uiValue(saved.status),v2:uiValue(dirty && t("audit.unsavedQuestion",{}))})}/></p><p><UiText notice={uiMessage("draft-editor.saving.machine.validation.and.independent.approval.are.separate.s.d9d962",{})}/></p><p className={styles.metadata}><UiText notice={uiMessage("draft-editor.owner.value.authors.value.2be751",{v0:uiValue(saved.ownerId),v1:uiValue(saved.authorIds.join(", "))})}/></p>{saved.legacyUnattributed && <p className={styles.notice}><UiText notice={uiMessage("draft-editor.verify.historical.authorship.and.original.sources.before.approval.248a00",{})}/></p>}{!write && <p><UiText notice={uiMessage("draft-editor.this.workspace.is.read.only.for.this.account.or.has.already.been..be1962",{})}/></p>}{command.controls}<fieldset disabled={!write || command.blocked}><legend><UiText notice={uiMessage("draft-editor.editable.question.package.e30c58",{})}/></legend><div className={styles.grid}><NumberField label={t("draft-editor.catalogue.version.0654f3",{})} value={input.catalogueVersion} onChange={catalogueVersion => setInput(v => ({ ...v, catalogueVersion }))}/><TextField label={t("package-fields.package.id.7a35d0",{})} value={input.questionPackage.id} onChange={id => changePackage({ ...input.questionPackage, id })}/><NumberField label={t("package-fields.package.version.359c57",{})} value={input.questionPackage.version} onChange={version => changePackage({ ...input.questionPackage, version })}/></div><h2><UiText notice={uiMessage("draft-editor.finite.question.templates.3f552e",{})}/></h2><p><UiText notice={uiMessage("draft-editor.rational.arithmetic.rational.comparison.missing.operand.05a6d3",{})}/></p><Rows label={t("draft-editor.template.5cde0f",{})} items={input.questionPackage.templates} onChange={templates => changePackage({ ...input.questionPackage, templates })} create={newTemplate}>{(v, i, set) => <TemplateFields label={t("draft-editor.template.value.7d9545",{v0:uiValue(i + 1)})} value={v} onChange={set}/>}</Rows><h2><UiText notice={uiMessage("draft-editor.fixed.questions.67f48f",{})}/></h2><Rows label={t("draft-editor.fixed.question.b25887",{})} items={input.questionPackage.fixedQuestions} onChange={fixedQuestions => changePackage({ ...input.questionPackage, fixedQuestions })} create={newFixed}>{(v, i, set) => <FixedFields label={t("draft-editor.fixed.question.value.944706",{v0:uiValue(i + 1)})} value={v} onChange={set}/>}</Rows><h2><UiText notice={uiMessage("draft-editor.assessment.blueprints.fcd71c",{})}/></h2><Rows label={t("draft-editor.blueprint.b1ece0",{})} items={input.questionPackage.blueprints} onChange={blueprints => changePackage({ ...input.questionPackage, blueprints })} create={newBlueprint}>{(v, i, set) => <BlueprintFields label={t("draft-editor.blueprint.value.ee770e",{v0:uiValue(i + 1)})} value={v} onChange={set}/>}</Rows><details><summary><UiText notice={uiMessage("draft-editor.editable.json.and.source.mapping.ceb2f0",{})}/></summary><p><UiText notice={uiMessage("draft-editor.import.and.export.contain.editable.question.data.and.source.mappi.9b4d59",{})}/></p><TextField label={t("draft-editor.editable.question.json.ff0b13",{})} value={json} onChange={setJSON} multiline/><div className={styles.actions}><button type="button" className="button secondary" onClick={() => importJSON(new TextEncoder().encode(json))}><UiText notice={uiMessage("draft-editor.apply.editable.json.94e26f",{})}/></button><button type="button" className="button secondary" onClick={() => setJSON(JSON.stringify(input, null, 2))}><UiText notice={uiMessage("draft-editor.copy.current.fields.to.json.7379fb",{})}/></button><button type="button" className="button secondary" disabled={questionInputError("createDraft", input) !== null} onClick={download}><UiText notice={uiMessage("draft-editor.export.editable.json.65164d",{})}/></button><label><UiText notice={uiMessage("draft-editor.import.question.json.file.3dea76",{})}/><input aria-label={t("draft-editor.import.question.json.file.3dea76",{})} type="file" accept=".json,application/json" onChange={async (e) => { const file = e.target.files?.[0]; if (file) {
        if (file.size > 4194304)
            command.error(uiMessage("draft-editor.question.json.exceeds.the.request.size.limit.e70c01",{}));
        else
            importJSON(new Uint8Array(await file.arrayBuffer()));
    } e.target.value = ""; }}/></label></div></details></fieldset><div className={styles.toolbar}><div className={styles.actions}><button className="button" disabled={!write || command.blocked || questionInputError("saveDraft", saveInput) !== null} onClick={() => void command.run({ kind: "saveDraft", id: saved.id }, saveInput, data => { const next = data as DraftView; setSaved(next); setInput(editable(next)); setJSON(JSON.stringify(editable(next), null, 2)); setValidated(null); command.message(uiMessage("draft-editor.draft.saved.validate.the.new.saved.revision.3180f2",{})); })}><UiText notice={uiMessage("draft-editor.save.draft.3de100",{})}/></button><button className="button secondary" disabled={!write || command.blocked || dirty} onClick={() => void command.run({ kind: "validateDraft", id: saved.id }, { expectedRevision: saved.revision }, data => setValidated({ revision: saved.revision, report: data as ValidationReport }))}><UiText notice={uiMessage("draft-editor.validate.saved.revision.0603de",{})}/></button><button className="button" disabled={!write || command.blocked || dirty || validated?.revision !== saved.revision || !validated.report.readyToSubmit} onClick={() => { if (validated)
        void command.run({ kind: "submitDraft", id: saved.id }, { expectedRevision: saved.revision, expectedDigest: validated.report.digest }, data => { setSubmitted((data as SubmissionView).id); setSaved(v => ({ ...v, status: "submitted" })); command.message(uiMessage("draft-editor.the.saved.revision.was.frozen.for.independent.review.23df27",{})); router.refresh(); }); }}><UiText notice={uiMessage("draft-editor.submit.for.review.40447e",{})}/></button><button className="button secondary" disabled={command.busy} onClick={async () => { const result = await requestQuestion<DraftView>({ kind: "readDraft", id: saved.id }); if (result.ok) {
        setSaved(result.data);
        setValidated(null);
        command.clearPending();
        command.message(uiMessage("draft-editor.saved.version.reloaded.your.editable.input.was.preserved.c167eb",{}));
    }
    else
        command.error(uiError("question",result)); }}><UiText notice={uiMessage("draft-editor.reload.saved.version.3c845b",{})}/></button></div></div><GenerationPanel report={validated?.report ?? saved.gate} revision={validated?.revision ?? saved.revision}/>{submitted && <Link prefetch={false} href={"/review/questions/" + submitted}><UiText notice={uiMessage("draft-editor.open.frozen.submission.c1dfbc",{})}/></Link>}<p><Link prefetch={false} href="/editor/questions"><UiText notice={uiMessage("draft-editor.back.to.question.workspaces.7f7e32",{})}/></Link></p></section>;
}
