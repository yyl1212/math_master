"use client";
import type { Source, SourceLink } from "@/lib/content/types";
import { TextField, Rows, RefFields, SelectField } from "./field-controls";
export const emptySource = (): Source => ({ kind: "original", author: "", title: "", url: "", accessedAt: "", license: "", attribution: "" });
export function SourceFields({ label, value, onChange }: {
    label: string;
    value: Source;
    onChange: (v: Source) => void;
}) { return <fieldset><legend>{label}</legend><SelectField label={label + " kind"} value={value.kind} options={["original", "external"]} onChange={kind => onChange({ ...value, kind: kind as Source["kind"] })}/>{(["author", "title", "url", "accessedAt", "license", "attribution"] as const).map(key => <TextField key={key} label={label + " " + key} value={value[key]} onChange={v => onChange({ ...value, [key]: v })}/>)}</fieldset>; }
export function SourceMapFields({ items, onChange }: {
    items: SourceLink[];
    onChange: (v: SourceLink[]) => void;
}) { return <section><h2>Private source mapping</h2><Rows label="source mapping" items={items} onChange={onChange} create={() => ({ knowledge: { id: "new-knowledge", version: 1 }, batchSha256: "", relativePath: "", sha256: "", legacyId: "", note: "" })}>{(v, i, change) => <fieldset><legend>Source mapping {i + 1}</legend><RefFields label={`Source mapping ${i + 1} knowledge`} value={v.knowledge} onChange={knowledge => change({ ...v, knowledge })}/>{(["batchSha256", "relativePath", "sha256", "legacyId", "note"] as const).map(key => <TextField key={key} label={`Source mapping ${i + 1} ${key}`} value={v[key]} onChange={text => change({ ...v, [key]: text })}/>)}</fieldset>}</Rows></section>; }
