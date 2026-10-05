"use client";
import {uiError} from "@/lib/i18n/errors";
import {UiText} from "@/lib/i18n/ui-text";
import {uiMessage,uiValue} from "@/lib/i18n/format";
import {useUiI18n} from "@/lib/i18n/provider";
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
 const {t}=useUiI18n();

    const [page, setPage] = useState(initial), [id, setID] = useState(""), [version, setVersion] = useState(1), [catalogue, setCatalogue] = useState(1), [reason, setReason] = useState(""), command = useQuestionCommand(), router = useRouter();
    async function load(offset: number) { const result = await requestQuestion<DraftPage>({ kind: "listDrafts", query: { scope: canEdit ? "mine" : "all", limit: page.limit, offset } }); if (result.ok)
        setPage(result.data);
    else
        command.error(uiError("question",result)); }
    const open = (data: unknown) => router.push("/editor/questions/drafts/" + (data as DraftView).id);
    return <section className={styles.workbench}><p className="eyebrow"><UiText notice={uiMessage("draft-list.trusted.question.bank.a62caf",{})}/></p><h1><UiText notice={uiMessage("page.editor.questions",{})}/></h1><p><UiText notice={uiMessage("draft-list.build.exact.questions.validate.the.finite.batch.and.submit.it.for.03a844",{})}/></p>{command.controls}<ul className={styles.list}>{page.items.map(d => <li key={d.id}><Link prefetch={false} href={"/editor/questions/drafts/" + d.id}><UiText notice={uiMessage("content-preview.value.version.value.1d1973",{v0:uiValue(d.packageId),v1:uiValue(d.packageVersion)})}/></Link><p><UiText notice={uiMessage("draft-list.revision.value.value.value.machine.issues.0d005f",{v0:uiValue(d.revision),v1:uiValue(d.status),v2:uiValue(d.structuralTotal + d.completenessTotal)})}/></p></li>)}</ul>{!page.items.length && <p><UiText notice={uiMessage("draft-list.no.question.workspaces.yet.84226e",{})}/></p>}<div className={styles.actions}><button className="button secondary" disabled={command.busy || page.offset === 0} onClick={() => void load(Math.max(0, page.offset - page.limit))}><UiText notice={uiMessage("draft-list.previous.workspaces.fa8a2d",{})}/></button><button className="button secondary" disabled={command.busy || page.offset + page.limit >= page.total} onClick={() => void load(page.offset + page.limit)}><UiText notice={uiMessage("draft-list.next.workspaces.e03049",{})}/></button></div>{canEdit ? <div className={styles.grid}><form className={styles.card} onSubmit={e => { e.preventDefault(); void command.run({ kind: "createDraft" }, { catalogueVersion: catalogue, questionPackage: { kind: "question-bank", schemaVersion: 1, id, version, templates: [], fixedQuestions: [], blueprints: [] }, sourceMap: [] }, open); }}><h2><UiText notice={uiMessage("draft-list.create.question.workspace.2b8d7c",{})}/></h2><fieldset disabled={command.blocked}><legend><UiText notice={uiMessage("draft-list.new.editable.package.9d8c64",{})}/></legend><TextField label={t("draft-list.new.question.package.id.fc85e8",{})} value={id} onChange={setID}/><NumberField label={t("draft-list.new.question.package.version.6b2659",{})} value={version} onChange={setVersion}/><NumberField label={t("draft-list.new.catalogue.version.a016bc",{})} value={catalogue} onChange={setCatalogue}/></fieldset><button className="button" disabled={command.blocked}><UiText notice={uiMessage("draft-list.create.draft.07d07a",{})}/></button></form><form className={styles.card} onSubmit={e => { e.preventDefault(); void command.run({ kind: "adoptDraft" }, { packageId: id, packageVersion: version, reason }, open); }}><h2><UiText notice={uiMessage("draft-list.adopt.an.imported.question.package.35dce8",{})}/></h2><p><UiText notice={uiMessage("draft-list.use.the.exact.package.id.and.version.above.historical.authorship..5ae8a4",{})}/></p><fieldset disabled={command.blocked}><legend><UiText notice={uiMessage("draft-list.adoption.reason.e89c31",{})}/></legend><TextField label={t("draft-list.adoption.reason.e89c31",{})} value={reason} onChange={setReason} multiline/></fieldset><button className="button secondary" disabled={command.blocked}><UiText notice={uiMessage("draft-list.adopt.package.60672c",{})}/></button></form></div> : <p><UiText notice={uiMessage("draft-list.editor.permission.is.required.to.create.or.change.questions.0931c2",{})}/></p>}</section>;
}
