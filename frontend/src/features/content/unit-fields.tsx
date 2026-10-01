"use client";
import type { Unit } from "@/lib/content/types";
import { TextField, NumberField, TextList, RefFields, Rows } from "./field-controls";
export const emptyUnit = (): Unit => ({ id: "new-unit", version: 1, knowledge: { id: "new-knowledge", version: 1 }, angles: [], examples: [], counterexamples: [], assetIds: [] });
export function UnitFields({ value, label, onChange }: {
    value: Unit;
    label: string;
    onChange: (v: Unit) => void;
}) { return <fieldset><legend>{label}</legend><TextField label={label + " ID"} value={value.id} onChange={id => onChange({ ...value, id })}/><NumberField label={label + " version"} value={value.version} onChange={version => onChange({ ...value, version })}/><RefFields label={label + " knowledge"} value={value.knowledge} onChange={knowledge => onChange({ ...value, knowledge })}/><Rows label={label + " angle"} items={value.angles} onChange={angles => onChange({ ...value, angles })} create={() => ({ kind: "intuitive", body: "" })}>{(a, i, change) => <><TextField label={`${label} angle ${i + 1} kind`} value={a.kind} onChange={kind => change({ ...a, kind })}/><TextField label={`${label} angle ${i + 1} body`} value={a.body} onChange={body => change({ ...a, body })} multiline/></>}</Rows><TextList label={label + " example"} items={value.examples} onChange={examples => onChange({ ...value, examples })}/><TextList label={label + " counterexample"} items={value.counterexamples} onChange={counterexamples => onChange({ ...value, counterexamples })}/><TextList label={label + " asset ID"} items={value.assetIds} onChange={assetIds => onChange({ ...value, assetIds })}/></fieldset>; }
