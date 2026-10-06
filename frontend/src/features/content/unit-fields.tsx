"use client";
import {UiText} from "@/lib/i18n/ui-text";
import {uiMessage,uiValue} from "@/lib/i18n/format";
import {useUiI18n} from "@/lib/i18n/provider";
import type { Unit } from "@/lib/content/types";
import { TextField, NumberField, TextList, RefFields, Rows } from "./field-controls";
export const emptyUnit = (): Unit => ({ id: "new-unit", version: 1, knowledge: { id: "new-knowledge", version: 1 }, angles: [], examples: [], counterexamples: [], assetIds: [] });
export function UnitFields({ value, label, onChange }: {
    value: Unit;
    label: string;
    onChange: (v: Unit) => void;
}) {
 const {t}=useUiI18n();
 return <fieldset><legend>{label}</legend><TextField label={t("path-fields.value.id.0ad111",{v0:uiValue(label)})} value={value.id} onChange={id => onChange({ ...value, id })}/><NumberField label={t("path-fields.value.version.f8a7ae",{v0:uiValue(label)})} value={value.version} onChange={version => onChange({ ...value, version })}/><RefFields label={t("unit-fields.value.knowledge.f59083",{v0:uiValue(label)})} value={value.knowledge} onChange={knowledge => onChange({ ...value, knowledge })}/><Rows label={t("unit-fields.value.angle.df71ec",{v0:uiValue(label)})} items={value.angles} onChange={angles => onChange({ ...value, angles })} create={() => ({ kind: "intuitive", body: "" })}>{(a, i, change) => <><TextField label={t("unit-fields.value.angle.value.kind.c74f64",{v0:uiValue(label),v1:uiValue(i + 1)})} value={a.kind} onChange={kind => change({ ...a, kind })}/><TextField label={t("unit-fields.value.angle.value.body.e39e0c",{v0:uiValue(label),v1:uiValue(i + 1)})} value={a.body} onChange={body => change({ ...a, body })} multiline/></>}</Rows><TextList label={t("unit-fields.value.example.7998ee",{v0:uiValue(label)})} items={value.examples} onChange={examples => onChange({ ...value, examples })}/><TextList label={t("unit-fields.value.counterexample.c715cc",{v0:uiValue(label)})} items={value.counterexamples} onChange={counterexamples => onChange({ ...value, counterexamples })}/><TextList label={t("unit-fields.value.asset.id.8ccbdd",{v0:uiValue(label)})} items={value.assetIds} onChange={assetIds => onChange({ ...value, assetIds })}/></fieldset>; }
