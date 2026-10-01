"use client";
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
            } command.message("Draft saved. Check readiness before submitting."); });
        }
        catch (error) {
            command.error(error instanceof Error ? error.message : "Content import failed.");
        }
        finally {
            ioRef.current = false;
            setIO(false);
        }
    }
    async function importFile(file: File | undefined) { if (!file || ioRef.current || command.busy)
        return; ioRef.current = true; setIO(true); try {
        if (file.size > 8 << 20)
            throw new Error("This content exceeds the request size limit.");
        const value = importDraftInput(new Uint8Array(await file.arrayBuffer()));
        change({ catalogueVersion: value.catalogueVersion, package: value.package, sourceMap: value.sourceMap });
        setLocal(value.assetBytes);
        command.message("JSON imported. Save this draft to run machine checks.");
    }
    catch (error) {
        command.error(error instanceof Error ? error.message : "Content import failed.");
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
        command.message("DraftInput exported.");
    }
    catch (error) {
        command.error(error instanceof Error ? error.message : "Export failed.");
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
        command.message("Illustration imported. Use new owner and unit versions when its meaning changes.");
    }
    catch (error) {
        command.error(error instanceof Error ? error.message : "Illustration import failed.");
    }
    finally {
        ioRef.current = false;
        setIO(false);
    } }
    return <section className={styles.workbench}><p className="eyebrow">CONTENT WORKSPACE</p><h1>Edit mathematical content</h1><p>Revision {view.revision} · {view.status}{dirty ? " · Unsaved changes" : ""}</p><p className={styles.metadata}>Authors: {view.authorIds.join(", ")} · Catalogue SHA: {view.catalogueSha256}{view.legacyUnattributed && " · Legacy authorship needs verification"}</p><p>Keep package and member versions explicit. Content changes require new immutable versions.</p><div className={styles.toolbar}><div className={styles.actions}><button className="button" disabled={!editable || busy} onClick={() => void save()}>Save draft</button><button className="button secondary" disabled={!editable || busy || dirty} onClick={() => void command.run({ kind: "validateDraft", id: view.id }, { expectedRevision: view.revision }, data => { setView(v => ({ ...v, gate: data as GateReport })); command.message("Readiness check complete."); })}>Check readiness</button><button className="button" disabled={!editable || busy || dirty || !view.gate.readyToSubmit} onClick={() => void command.run({ kind: "submitDraft", id: view.id }, { expectedRevision: view.revision, expectedDigest: view.gate.digest }, data => { setView(v => ({ ...v, status: "submitted" })); router.push("/review/" + (data as SubmissionView).id); router.refresh(); })}>Submit for review</button><button className="button secondary" disabled={busy} onClick={() => void exportFile()}>Export JSON</button></div><p>{dirty ? "Save before checking or submitting." : "Review always uses a frozen copy of the saved revision."}</p></div>{command.controls}<GatePanel gate={view.gate}/><fieldset disabled={!editable || busy}><legend>Draft fields</legend><label className={styles.field}>Import DraftInput JSON<input type="file" accept=".json,application/json" onChange={e => { void importFile(e.target.files?.[0]); e.target.value = ""; }}/></label><NumberField label="Catalogue version" value={input.catalogueVersion} onChange={catalogueVersion => change({ ...input, catalogueVersion })}/><PackageFields value={input.package} onChange={value => change({ ...input, package: value })}/><section><h2>Original illustrations</h2><Rows label="asset" items={input.package.assets} onChange={assets => { setLocal(values => values.filter(v => assets.some(a => a.id === v.id))); change({ ...input, package: { ...input.package, assets } }); }} create={() => ({ id: "new-asset", path: "original.svg", sha256: "", author: "", license: "", attribution: "", knowledge: { id: "new-knowledge", version: 1 } } as Asset)}>{(asset, i, update) => <fieldset><legend>Asset {i + 1}</legend>{(["id", "path", "sha256", "author", "license", "attribution"] as const).map(key => <TextField key={key} label={`Asset ${i + 1} ${key}`} value={asset[key]} onChange={v => update({ ...asset, [key]: v })}/>)}<RefFields label={`Asset ${i + 1} knowledge`} value={asset.knowledge} onChange={knowledge => update({ ...asset, knowledge })}/><label className={styles.field}>Asset {i + 1} SVG file<input type="file" accept=".svg,image/svg+xml" onChange={e => { void upload(e.target.files?.[0], i); e.target.value = ""; }}/></label></fieldset>}</Rows></section><SourceMapFields items={input.sourceMap} onChange={sourceMap => change({ ...input, sourceMap })}/></fieldset><ContentPreview value={input.package} assets={view.assets} assetScope={{ kind: "draft", id: view.id }}/><Link prefetch={false} href="/editor">Back to workspaces</Link></section>;
}
