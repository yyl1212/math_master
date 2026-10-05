"use client";
import {enumOptionLabels} from "@/lib/i18n/enums";
import {useUiI18n} from "@/lib/i18n/provider";
import {UiText} from "@/lib/i18n/ui-text";
import {uiMessage,uiValue} from "@/lib/i18n/format";
import { TextField, NumberField, RefFields, Rows, SelectField } from "@/features/content/field-controls";
import { ObjectiveIndices, PublishedObjectives } from "./template-fields";
import type { Blueprint } from "@/lib/question/types";
export const newBlueprint = (): Blueprint => ({ id: "", version: 1, knowledge: { id: "", version: 1 }, coreObjectiveIndices: [], sources: [], coverageNote: "", ruleVersion: 1, questionCount: 5, passCount: 4 });
export function BlueprintFields({ value, label, onChange }: {
    value: Blueprint;
    label: string;
    onChange: (v: Blueprint) => void;
}) {
 const {t}=useUiI18n();
 const b = value, change = (p: Partial<Blueprint>) => onChange({ ...b, ...p }); return <fieldset><legend><UiText notice={uiMessage("blueprint-fields.value.five.distinct.questions.ac02dd",{v0:uiValue(label)})}/></legend><TextField label={t("path-fields.value.id.0ad111",{v0:uiValue(label)})} value={b.id} onChange={id => change({ id })}/><NumberField label={t("path-fields.value.version.f8a7ae",{v0:uiValue(label)})} value={b.version} onChange={version => change({ version })}/><RefFields label={t("unit-fields.value.knowledge.f59083",{v0:uiValue(label)})} value={b.knowledge} onChange={knowledge => change({ knowledge })}/><PublishedObjectives knowledge={b.knowledge}/><ObjectiveIndices label={t("blueprint-fields.value.core.97243e",{v0:uiValue(label)})} value={b.coreObjectiveIndices} onChange={coreObjectiveIndices => change({ coreObjectiveIndices })}/><Rows<Blueprint["sources"][number]> label={t("package-fields.value.source.5a7bf3",{v0:uiValue(label)})} items={b.sources} onChange={sources => change({ sources })} create={() => ({ kind: "template", ref: { id: "", version: 1 } })}>{(s, i, set) => <><SelectField label={t("blueprint-fields.value.source.value.kind.276627",{v0:uiValue(label),v1:uiValue(i + 1)})} value={s.kind} options={["template", "instance"]} optionLabels={enumOptionLabels("question.enum",["template", "instance"])} onChange={kind => set({ ...s, kind: kind as "template" | "instance" })}/><RefFields label={t("package-fields.value.source.value.40de80",{v0:uiValue(label),v1:uiValue(i + 1)})} value={s.ref} onChange={ref => set({ ...s, ref })}/></>}</Rows><p><UiText notice={uiMessage("blueprint-fields.instance.sources.identify.fixed.questions.generated.questions.ent.d26e6b",{})}/></p><TextField label={t("blueprint-fields.value.coverage.explanation.e9faf5",{v0:uiValue(label)})} value={b.coverageNote} onChange={coverageNote => change({ coverageNote })} multiline/><p><UiText notice={uiMessage("blueprint-fields.rule.version.1.5.questions.4.correct.answers.required.current.fiv.b49274",{})}/></p></fieldset>; }
