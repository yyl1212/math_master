"use client";
import {ContentTransferError,transferNotice} from "./transfer-error";
import {UiEnum,formatUiEnum,enumOptionLabels} from "@/lib/i18n/enums";

import {UiText} from "@/lib/i18n/ui-text";
import {uiMessage,uiValue} from "@/lib/i18n/format";
import {useUiI18n} from "@/lib/i18n/provider";
import Link from "next/link";
import { useState, useRef } from "react";
import { useRouter } from "next/navigation";
import type { DraftView, DraftInput, Asset, AssetInput, GateReport, SubmissionView } from "@/lib/content/types";
import { PackageFields } from "./package-fields";
import { SourceMapFields } from "./source-fields";
import { NumberField, TextField, RefFields, Rows } from "./field-controls";
import { ContentPreview, GatePanel } from "./content-preview";
import { useContentCommand } from "./command-controls";
import { importDraftInput, exportDraftInput, svgDigest, toBase64 } from "./asset-transfer";
import styles from "@/styles/content.module.css";
export function DraftEditor({ initial, canEdit = true }: {
    initial: DraftView;
    canEdit?: boolean;
}) {
 const {t,locale}=useUiI18n();

    const router = useRouter(), command = useContentCommand();
    const [view, setView] = useState(initial), [input, setInput] = useState<Omit<DraftInput, "assetBytes">>(() => ({ catalogueVersion: initial.catalogueVersion, package: structuredClone(initial.package), sourceMap: structuredClone(initial.sourceMap) })), [local, setLocal] = useState<AssetInput[]>([]), [dirty, setDirty] = useState(false), [io, setIO] = useState(false);
    const ioRef = useRef(false), generation = useRef(0);
    const editable = canEdit && view.status === "editing", busy = command.busy || io;
    function change(v: typeof input) { generation.current++; setInput(v); setDirty(true); }
    async function payload() { return JSON.parse(await exportDraftInput(input, { kind: "draft", id: view.id }, local)) as DraftInput; }
    async function save() {
        if (ioRef.current || command.busy)
            return;
        ioRef.current = true;
        setIO(true);
        const g = generation.current;
        try {
            const next = await payload();
            await command.run({ kind: "saveDraft", id: view.id }, { ...next, expectedRevision: view.revision }, data => { const saved = data as DraftView; setView(saved); if (g === generation.current) {
                setInput({ catalogueVersion: saved.catalogueVersion, package: saved.package, sourceMap: saved.sourceMap });
                setLocal([]);
                setDirty(false);
            } command.message(uiMessage("draft-editor.draft.saved.check.readiness.before.submitting.2f6566",{})); });
        }
        catch (error) {
            command.error(transferNotice(error));
        }
        finally {
            ioRef.current = false;
            setIO(false);
        }
    }
    async function importFile(file: File | undefined) { if (!file || ioRef.current || command.busy)
        return; ioRef.current = true; setIO(true); try {
        if (file.size > 8 << 20)
            throw new ContentTransferError("This content exceeds the request size limit.","PAYLOAD_TOO_LARGE");
        const value = importDraftInput(new Uint8Array(await file.arrayBuffer()));
        change({ catalogueVersion: value.catalogueVersion, package: value.package, sourceMap: value.sourceMap });
        setLocal(value.assetBytes);
        command.message(uiMessage("draft-editor.json.imported.save.this.draft.to.run.machine.checks.710f77",{}));
    }
    catch (error) {
        command.error(transferNotice(error));
    }
    finally {
        ioRef.current = false;
        setIO(false);
    } }
    async function exportFile() { if (ioRef.current || command.busy)
        return; ioRef.current = true; setIO(true); try {
        const text = await exportDraftInput(input, { kind: "draft", id: view.id }, local);
        const url = URL.createObjectURL(new Blob([text], { type: "application/json" }));
        const a = document.createElement("a");
        a.href = url;
        a.download = input.package.id + ".draft.json";
        a.click();
        setTimeout(() => URL.revokeObjectURL(url), 0);
        command.message(uiMessage("draft-editor.draftinput.exported.f36ce4",{}));
    }
    catch (error) {
        command.error(transferNotice(error));
    }
    finally {
        ioRef.current = false;
        setIO(false);
    } }
    async function upload(file: File | undefined, index: number) { if (!file || ioRef.current || command.busy)
        return; ioRef.current = true; setIO(true); try {
        if (file.size > 1 << 20)
            throw new Error("SVG files must be at most 1 MiB.");
        const bytes = new Uint8Array(await file.arrayBuffer()), sha256 = await svgDigest(bytes), asset = input.package.assets[index];
        change({ ...input, package: { ...input.package, assets: input.package.assets.map((a, i) => i === index ? { ...a, sha256 } : a) } });
        setLocal(list => [...list.filter(v => v.id !== asset.id), { id: asset.id, base64: toBase64(bytes) }]);
        command.message(uiMessage("draft-editor.illustration.imported.use.new.owner.and.unit.versions.when.its.me.7446be",{}));
    }
    catch (error) {
        command.error(transferNotice(error));
    }
    finally {
        ioRef.current = false;
        setIO(false);
    } }
    return <section className={styles.workbench}><p className="eyebrow"><UiText notice={uiMessage("draft-editor.content.workspace.af267d",{})}/></p><h1><UiText notice={uiMessage("page.editor.drafts.id",{})}/></h1><p><UiText notice={uiMessage("draft-editor.revision.value.valuevalue.7bf409",{v0:uiValue(view.revision),v1:formatUiEnum(locale,"content.enum",view.status),v2:uiValue(dirty ? t("audit.unsaved",{}) : "")})}/></p><p className={styles.metadata}><UiText notice={uiMessage("draft-editor.authors.value.catalogue.sha.valuevalue.bf61d0",{v0:uiValue(view.authorIds.join(", ")),v1:uiValue(view.catalogueSha256),v2:uiValue(view.legacyUnattributed && t("audit.legacy",{}))})}/></p><p><UiText notice={uiMessage("draft-editor.keep.package.and.member.versions.explicit.content.changes.require.b77a94",{})}/></p><div className={styles.toolbar}><div className={styles.actions}>{dirty || busy ? <button className="button secondary" disabled><UiText notice={uiMessage("draft-editor.read.saved.draft.daf801",{})}/></button> : <Link prefetch={false} className="button secondary" href={"/editor/drafts/" + view.id + "/preview"}><UiText notice={uiMessage("draft-editor.read.saved.draft.daf801",{})}/></Link>}<button className="button" disabled={!editable || busy} onClick={() => void save()}><UiText notice={uiMessage("draft-editor.save.draft.3de100",{})}/></button><button className="button secondary" disabled={!editable || busy || dirty} onClick={() => void command.run({ kind: "validateDraft", id: view.id }, { expectedRevision: view.revision }, data => { setView(v => ({ ...v, gate: data as GateReport })); command.message(uiMessage("draft-editor.readiness.check.complete.f8a29a",{})); })}><UiText notice={uiMessage("draft-editor.check.readiness.1e2cac",{})}/></button><button className="button" disabled={!editable || busy || dirty || !view.gate.readyToSubmit} onClick={() => void command.run({ kind: "submitDraft", id: view.id }, { expectedRevision: view.revision, expectedDigest: view.gate.digest }, data => { setView(v => ({ ...v, status: "submitted" })); router.push("/review/" + (data as SubmissionView).id); router.refresh(); })}><UiText notice={uiMessage("draft-editor.submit.for.review.40447e",{})}/></button><button className="button secondary" disabled={busy} onClick={() => void exportFile()}><UiText notice={uiMessage("draft-editor.export.json.ff2565",{})}/></button></div><p>{dirty ? t("draft-editor.save.before.reading.checking.or.submitting.3f949b",{}) : t("draft-editor.review.always.uses.a.frozen.copy.of.the.saved.revision.86413e",{})}</p></div>{command.controls}<GatePanel gate={view.gate}/><fieldset disabled={!editable || busy}><legend><UiText notice={uiMessage("draft-editor.draft.fields.476e2c",{})}/></legend><label className={styles.field}><UiText notice={uiMessage("draft-editor.import.draftinput.json.5ab8cb",{})}/><input type="file" accept=".json,application/json" onChange={e => { void importFile(e.target.files?.[0]); e.target.value = ""; }}/></label><NumberField label={t("draft-editor.catalogue.version.0654f3",{})} value={input.catalogueVersion} onChange={catalogueVersion => change({ ...input, catalogueVersion })}/><PackageFields value={input.package} onChange={value => change({ ...input, package: value })}/><section><h2><UiText notice={uiMessage("draft-editor.original.illustrations.558c7d",{})}/></h2><Rows label={t("draft-editor.asset.d59386",{})} items={input.package.assets} onChange={assets => { setLocal(values => values.filter(v => assets.some(a => a.id === v.id))); change({ ...input, package: { ...input.package, assets } }); }} create={() => ({ id: "new-asset", path: "original.svg", sha256: "", author: "", license: "", attribution: "", knowledge: { id: "new-knowledge", version: 1 } } as Asset)}>{(asset, i, update) => <fieldset><legend><UiText notice={uiMessage("draft-editor.asset.value.4a6e21",{v0:uiValue(i + 1)})}/></legend>{(["id", "path", "sha256", "author", "license", "attribution"] as const).map(key => <TextField key={key} label={t("draft-editor.asset.value.value.bb5eb5",{v0:uiValue(i + 1),v1:formatUiEnum(locale,"content.field",key)})} value={asset[key]} onChange={v => update({ ...asset, [key]: v })}/>)}<RefFields label={t("draft-editor.asset.value.knowledge.5e272b",{v0:uiValue(i + 1)})} value={asset.knowledge} onChange={knowledge => update({ ...asset, knowledge })}/><label className={styles.field}><UiText notice={uiMessage("draft-editor.asset.086710",{})}/>{i + 1}<UiText notice={uiMessage("draft-editor.svg.file.4b0cda",{})}/><input type="file" accept=".svg,image/svg+xml" onChange={e => { void upload(e.target.files?.[0], i); e.target.value = ""; }}/></label></fieldset>}</Rows></section><SourceMapFields items={input.sourceMap} onChange={sourceMap => change({ ...input, sourceMap })}/></fieldset><ContentPreview value={input.package} assets={view.assets} assetScope={{ kind: "draft", id: view.id }}/><Link prefetch={false} href="/editor"><UiText notice={uiMessage("draft-editor.back.to.workspaces.174743",{})}/></Link></section>;
}
