"use client";
import type { Path } from "@/lib/content/types";
import { TextField, NumberField, TextList, RefFields, Rows } from "./field-controls";
export const emptyPath = (): Path => ({ id: "new-path", version: 1, domainIds: [], title: "", titleZh: "", nodes: [] });
export function PathFields({ value, label, onChange }: {
    value: Path;
    label: string;
    onChange: (v: Path) => void;
}) { return <fieldset><legend>{label}</legend><TextField label={label + " ID"} value={value.id} onChange={id => onChange({ ...value, id })}/><NumberField label={label + " version"} value={value.version} onChange={version => onChange({ ...value, version })}/><TextField label={label + " title"} value={value.title} onChange={title => onChange({ ...value, title })}/><TextField label={label + " Chinese title"} value={value.titleZh} onChange={titleZh => onChange({ ...value, titleZh })}/><TextList label={label + " domain ID"} items={value.domainIds} onChange={domainIds => onChange({ ...value, domainIds })}/><Rows label={label + " node"} items={value.nodes} onChange={nodes => onChange({ ...value, nodes })} create={() => ({ id: "new-knowledge", version: 1 })}>{(v, i, change) => <RefFields label={`${label} node ${i + 1}`} value={v} onChange={change}/>}</Rows></fieldset>; }
