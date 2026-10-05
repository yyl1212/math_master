"use client";
import {UiEnum,formatUiEnum} from "@/lib/i18n/enums";
import {uiError} from "@/lib/i18n/errors";
import {UiText} from "@/lib/i18n/ui-text";
import {uiMessage,uiValue} from "@/lib/i18n/format";
import {useUiI18n} from "@/lib/i18n/provider";
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
 const {t,locale}=useUiI18n();

    const router = useRouter(), command = useContentCommand(), [page, setPage] = useState(initial), [catalogue, setCatalogue] = useState(0), [id, setID] = useState(""), [version, setVersion] = useState(1), [reason, setReason] = useState("");
    async function load(offset: number) { const result = await contentRequest<DraftPage>({ kind: "listDrafts", query: { scope: canEdit ? "mine" : "all", limit: page.limit, offset } }); if (result.ok)
        setPage(result.data);
    else
        command.error(uiError("content",result)); }
    const open = (data: unknown) => router.push("/editor/drafts/" + (data as DraftView).id);
    return <section className={styles.workbench}><p className="eyebrow"><UiText notice={uiMessage("draft-list.authoring.67b82c",{})}/></p><h1><UiText notice={uiMessage("page.editor",{})}/></h1><p><UiText notice={uiMessage("draft-list.original.content.begins.here.saving.and.independent.review.have.s.c2ee37",{})}/></p>{command.controls}<ul className={styles.list}>{page.items.map(d => <li key={d.id}><Link prefetch={false} href={"/editor/drafts/" + d.id}><UiText notice={uiMessage("content-preview.value.version.value.1d1973",{v0:uiValue(d.packageId),v1:uiValue(d.packageVersion)})}/></Link><p><UiText notice={uiMessage("draft-list.revision.value.value.value.machine.issues.0d005f",{v0:uiValue(d.revision),v1:formatUiEnum(locale,"content.enum",d.status),v2:uiValue(d.structuralTotal + d.completenessTotal)})}/></p><Link prefetch={false} href={"/editor/drafts/" + d.id + "/preview"}><UiText notice={uiMessage("draft-editor.read.saved.draft.daf801",{})}/></Link></li>)}</ul>{!page.items.length && <p><UiText notice={uiMessage("draft-list.no.workspaces.yet.ded050",{})}/></p>}<div className={styles.actions}><button className="button secondary" disabled={page.offset === 0} onClick={() => void load(Math.max(0, page.offset - page.limit))}><UiText notice={uiMessage("admin-users.previous.a57b08",{})}/></button><button className="button secondary" disabled={page.offset + page.limit >= page.total} onClick={() => void load(page.offset + page.limit)}><UiText notice={uiMessage("admin-users.next.1ff57a",{})}/></button></div>{canEdit && <div className={styles.grid}><form className={styles.card} onSubmit={e => { e.preventDefault(); void command.run({ kind: "createDraft" }, { catalogueVersion: catalogue, package: { schemaVersion: 1, id, version, knowledge: [], units: [], paths: [], assets: [] }, assetBytes: [], sourceMap: [] }, open); }}><h2><UiText notice={uiMessage("draft-list.create.workspace.4b8922",{})}/></h2><fieldset disabled={command.busy}><legend><UiText notice={uiMessage("draft-list.new.package.afcdcc",{})}/></legend><TextField label={t("draft-list.new.package.id.c42b51",{})} value={id} onChange={setID}/><NumberField label={t("draft-list.new.package.version.12a769",{})} value={version} onChange={setVersion}/><NumberField label={t("draft-list.new.catalogue.version.a016bc",{})} value={catalogue} onChange={setCatalogue}/></fieldset><p><UiText notice={uiMessage("draft-list.choose.a.trusted.imported.catalogue.version.5f3f9e",{})}/></p><button className="button" disabled={command.busy}><UiText notice={uiMessage("draft-list.create.draft.07d07a",{})}/></button></form><form className={styles.card} onSubmit={e => { e.preventDefault(); void command.run({ kind: "adoptDraft" }, { packageId: id, packageVersion: version, reason }, open); }}><h2><UiText notice={uiMessage("draft-list.adopt.an.imported.package.70604f",{})}/></h2><p><UiText notice={uiMessage("draft-list.the.package.id.and.version.above.identify.the.immutable.imported..a9c4df",{})}/></p><TextField label={t("draft-list.adoption.reason.e89c31",{})} value={reason} onChange={setReason} multiline/><button className="button secondary" disabled={command.busy}><UiText notice={uiMessage("draft-list.adopt.package.60672c",{})}/></button></form></div>}</section>;
}
