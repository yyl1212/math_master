"use client";
import {UiEnum,formatUiEnum,enumOptionLabels} from "@/lib/i18n/enums";

import {UiText} from "@/lib/i18n/ui-text";
import {uiMessage,uiValue} from "@/lib/i18n/format";
import {useUiI18n} from "@/lib/i18n/provider";
import type { Source, SourceLink } from "@/lib/content/types";
import { TextField, Rows, RefFields, SelectField } from "./field-controls";
export const emptySource = (): Source => ({ kind: "original", author: "", title: "", url: "", accessedAt: "", license: "", attribution: "" });
export function SourceFields({ label, value, onChange }: {
    label: string;
    value: Source;
    onChange: (v: Source) => void;
}) {
 const {t,locale}=useUiI18n();
 return <fieldset><legend>{label}</legend><SelectField label={t("source-fields.value.kind.033e78",{v0:uiValue(label)})} value={value.kind} options={["original", "external"]} optionLabels={enumOptionLabels("content.enum",["original", "external"])} onChange={kind => onChange({ ...value, kind: kind as Source["kind"] })}/>{(["author", "title", "url", "accessedAt", "license", "attribution"] as const).map(key => <TextField key={key} label={uiMessage("field.named",{label,field:formatUiEnum(locale,"content.field",key)})} value={value[key]} onChange={v => onChange({ ...value, [key]: v })}/>)}</fieldset>; }
export function SourceMapFields({ items, onChange }: {
    items: SourceLink[];
    onChange: (v: SourceLink[]) => void;
}) {
 const {t,locale}=useUiI18n();
 return <section><h2><UiText notice={uiMessage("source-fields.private.source.mapping.a8b054",{})}/></h2><Rows label={t("source-fields.source.mapping.a69a8e",{})} items={items} onChange={onChange} create={() => ({ knowledge: { id: "new-knowledge", version: 1 }, batchSha256: "", relativePath: "", sha256: "", legacyId: "", note: "" })}>{(v, i, change) => <fieldset><legend><UiText notice={uiMessage("source-fields.source.mapping.value.830c78",{v0:uiValue(i + 1)})}/></legend><RefFields label={t("source-fields.source.mapping.value.knowledge.49b15e",{v0:uiValue(i + 1)})} value={v.knowledge} onChange={knowledge => change({ ...v, knowledge })}/>{(["batchSha256", "relativePath", "sha256", "legacyId", "note"] as const).map(key => <TextField key={key} label={t("source-fields.source.mapping.value.value.b90966",{v0:uiValue(i + 1),v1:formatUiEnum(locale,"content.field",key)})} value={v[key]} onChange={text => change({ ...v, [key]: text })}/>)}</fieldset>}</Rows></section>; }
