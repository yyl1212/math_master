"use client";
import {formatUiEnum} from "@/lib/i18n/enums";
import {enumOptionLabels} from "@/lib/i18n/enums";

import type {UiNotice} from "@/lib/i18n/types";
import {useUiI18n} from "@/lib/i18n/provider";
import {UiText} from "@/lib/i18n/ui-text";
import {uiMessage,uiValue} from "@/lib/i18n/format";
import { useEffect, useRef, useState } from "react";
import { TextField, NumberField, SelectField, Rows, RefFields } from "@/features/content/field-controls";
import { knowledgeViewSchema } from "@/lib/api/schemas";
import { readContentBytes } from "@/lib/content/bytes";
import type { KnowledgeView } from "@/lib/api/types";
import type { Template, FixedQuestion } from "@/lib/question/types";
import styles from "@/styles/question.module.css";
const emptySource = () => ({ kind: "original", author: "", title: "", url: "", accessedAt: "", license: "", attribution: "" });
export function SourcesFields({ label, value, onChange }: {
    label: string;
    value: Template["sources"];
    onChange: (v: Template["sources"]) => void;
}) {
 const {t:uiT,locale}=useUiI18n();
 return <Rows label={uiT("package-fields.value.source.5a7bf3",{v0:uiValue(label)})} items={value} onChange={onChange} create={emptySource}>{(s, i, change) => <fieldset><legend><UiText notice={uiMessage("package-fields.value.source.value.40de80",{v0:uiValue(label),v1:uiValue(i + 1)})}/></legend>{(["kind", "author", "title", "url", "accessedAt", "license", "attribution"] as const).map(key => <TextField key={key} label={uiT("template-fields.value.source.value.value.83babd",{v0:uiValue(label),v1:uiValue(i + 1),v2:formatUiEnum(locale,"content.field",key)})} value={s[key]} onChange={v => change({ ...s, [key]: v })} multiline={key === "attribution"}/>)}</fieldset>}</Rows>; }
export function ObjectiveIndices({ label, value, onChange }: {
    label: string;
    value: number[];
    onChange: (v: number[]) => void;
}) {
 const {t:uiT,locale}=useUiI18n();
 return <Rows label={uiT("template-fields.value.objective.index.b0b812",{v0:uiValue(label)})} items={value} onChange={onChange} create={() => 0}>{(v, i, change) => <label className={styles.check}>{label}<UiText notice={uiMessage("template-fields.objective.index.eab754",{})}/>{i + 1}<input aria-label={uiT("audit.objective.label",{label,number:i+1})} type="number" min={0} max={2147483647} step={1} value={Number.isFinite(v) ? v : ""} onChange={e => change(e.target.value === "" ? NaN : Number(e.target.value))}/></label>}</Rows>; }
export function CoverageFields({ label, value, onChange }: {
    label: string;
    value: Template["coverage"];
    onChange: (v: Template["coverage"]) => void;
}) {
 const {t:uiT,locale}=useUiI18n();
 return <Rows label={uiT("template-fields.value.coverage.3b6324",{v0:uiValue(label)})} items={value} onChange={onChange} create={() => ({ knowledge: { id: "", version: 1 }, objectiveIndices: [0] })}>{(v, i, change) => <fieldset><legend><UiText notice={uiMessage("template-fields.value.coverage.value.f228fd",{v0:uiValue(label),v1:uiValue(i + 1)})}/></legend><RefFields label={uiT("template-fields.value.coverage.value.knowledge.6e063b",{v0:uiValue(label),v1:uiValue(i + 1)})} value={v.knowledge} onChange={knowledge => change({ ...v, knowledge })}/><ObjectiveIndices label={uiT("template-fields.value.coverage.value.f228fd",{v0:uiValue(label),v1:uiValue(i + 1)})} value={v.objectiveIndices} onChange={objectiveIndices => change({ ...v, objectiveIndices })}/></fieldset>}</Rows>; }
export function UnitFields({ label, value, onChange }: {
    label: string;
    value: Template["units"];
    onChange: (v: Template["units"]) => void;
}) {
 const {t:uiT,locale}=useUiI18n();
 return <Rows label={uiT("template-fields.value.unit.f2dadb",{v0:uiValue(label)})} items={value} onChange={onChange} create={() => ({ id: "", version: 1 })}>{(v, i, change) => <RefFields label={uiT("template-fields.value.unit.value.7da153",{v0:uiValue(label),v1:uiValue(i + 1)})} value={v} onChange={change}/>}</Rows>; }
export function AssetFields({ label, value, onChange }: {
    label: string;
    value: Template["assets"];
    onChange: (v: Template["assets"]) => void;
}) {
 const {t:uiT,locale}=useUiI18n();
 return <Rows label={uiT("template-fields.value.fixed.illustration.022155",{v0:uiValue(label)})} items={value} onChange={onChange} create={() => ({ id: "", sha256: "" })}>{(v, i, change) => <><TextField label={uiT("template-fields.value.illustration.value.id.ba47fd",{v0:uiValue(label),v1:uiValue(i + 1)})} value={v.id} onChange={id => change({ ...v, id })}/><TextField label={uiT("template-fields.value.illustration.value.sha.a322ab",{v0:uiValue(label),v1:uiValue(i + 1)})} value={v.sha256} onChange={sha256 => change({ ...v, sha256 })}/></>}</Rows>; }
export function PublishedObjectives({ knowledge }: {
    knowledge: Template["knowledge"];
}) {
 const {t:uiT,locale}=useUiI18n();

    const [data, setData] = useState<KnowledgeView | null>(null), [busy, setBusy] = useState(false), [error, setError] = useState<UiNotice | null>(null), active = useRef<AbortController | null>(null);
    useEffect(() => { active.current?.abort(); setData(null); setError(null); return () => active.current?.abort(); }, [knowledge.id, knowledge.version]);
    async function load() { if (busy)
        return; const controller = new AbortController(); active.current = controller; const timer = setTimeout(() => controller.abort(), 5000); setBusy(true); setError(null); try {
        if (!/^[a-z][a-z0-9]*(-[a-z0-9]+)*$/.test(knowledge.id) || knowledge.id.length > 64)
            throw new Error();
        const response = await fetch("/api/v1/knowledge/" + knowledge.id, { method: "GET", credentials: "omit", cache: "no-store", redirect: "error", signal: controller.signal });
        if (!response.ok || response.headers.has("Set-Cookie") || response.redirected)
            throw new Error();
        const parsed = JSON.parse(new TextDecoder("utf-8", { fatal: true }).decode(await readContentBytes(response, 10485760, controller.signal)));
        const view = knowledgeViewSchema.parse(parsed.data);
        if (view.knowledge.id !== knowledge.id || view.knowledge.version !== knowledge.version)
            throw new Error();
        setData(view);
    }
    catch {
        if (!controller.signal.aborted)
            setError(uiMessage("template-fields.this.exact.published.knowledge.version.is.unavailable.validation..44e7ec",{}));
    }
    finally {
        clearTimeout(timer);
        if (active.current === controller) {
            setBusy(false);
            active.current = null;
        }
    } }
    return <section><button className="button secondary" type="button" disabled={busy} onClick={() => void load()}><UiText notice={uiMessage("template-fields.load.published.objectives.a9108a",{})}/></button>{error && <p role="status"><UiText notice={error}/></p>}{data && <><h4>{data.knowledge.title} · {data.knowledge.titleZh}</h4><ul>{data.knowledge.objectives.map((o, i) => <li key={i}>{i}: {o}</li>)}</ul></>}</section>;
}
export const newTemplate = (): Template => ({ id: "", version: 1, knowledge: { id: "", version: 1 }, coverage: [], units: [], type: "numeric", answerFormat: "rational", promptTemplate: "Calculate {{left}} + {{right}}.", explanationTemplate: "The sum is {{answer}}.", engine: { family: "rational_arithmetic", operation: "add", unknownSide: null, generatorVersion: 1, verifierVersion: 1 }, parameters: [{ name: "left", values: ["1"] }, { name: "right", values: ["2"] }], constraints: [], distractors: [], assets: [], sources: [] });
export function TemplateFields({ value, label, onChange }: {
    value: Template;
    label: string;
    onChange: (v: Template) => void;
}) {
 const {t:uiT,locale}=useUiI18n();

    const t = value, change = (patch: Partial<Template>) => onChange({ ...t, ...patch });
    return <fieldset><legend><UiText notice={uiMessage("template-fields.value.finite.exact.parameters.572b12",{v0:uiValue(label)})}/></legend><div className={styles.grid}><TextField label={uiT("path-fields.value.id.0ad111",{v0:uiValue(label)})} value={t.id} onChange={id => change({ id })}/><NumberField label={uiT("path-fields.value.version.f8a7ae",{v0:uiValue(label)})} value={t.version} onChange={version => change({ version })}/></div><RefFields label={uiT("template-fields.value.primary.knowledge.99f3ec",{v0:uiValue(label)})} value={t.knowledge} onChange={knowledge => change({ knowledge })}/><PublishedObjectives knowledge={t.knowledge}/><CoverageFields label={label} value={t.coverage} onChange={coverage => change({ coverage })}/><SelectField label={uiT("template-fields.value.family.ea310d",{v0:uiValue(label)})} value={t.engine.family} options={["rational_arithmetic", "rational_comparison", "missing_operand"]} optionLabels={enumOptionLabels("question.enum",["rational_arithmetic", "rational_comparison", "missing_operand"])} onChange={family => change({ engine: { ...t.engine, family: family as Template["engine"]["family"], operation: family === "rational_comparison" ? "compare" : "add", unknownSide: family === "missing_operand" ? "left" : null }, parameters: family === "missing_operand" ? [{ name: "known", values: ["1"] }, { name: "result", values: ["2"] }] : [{ name: "left", values: ["1"] }, { name: "right", values: ["2"] }], constraints: [], distractors: [], ...(family === "rational_comparison" ? { type: "single_choice" as const, answerFormat: null } : {}) })}/><SelectField label={uiT("template-fields.value.operation.e1cc41",{v0:uiValue(label)})} value={t.engine.operation} options={t.engine.family === "rational_comparison" ? ["compare"] : ["add", "subtract", "multiply", "divide"]} optionLabels={enumOptionLabels("question.enum",t.engine.family === "rational_comparison" ? ["compare"] : ["add", "subtract", "multiply", "divide"])} onChange={operation => change({ engine: { ...t.engine, operation: operation as Template["engine"]["operation"] } })}/>{t.engine.family === "missing_operand" && <SelectField label={uiT("template-fields.value.unknown.side.48d786",{v0:uiValue(label)})} value={t.engine.unknownSide ?? "left"} options={["left", "right"]} optionLabels={enumOptionLabels("question.enum",["left", "right"])} onChange={side => change({ engine: { ...t.engine, unknownSide: side as "left" | "right" } })}/>}<SelectField label={uiT("template-fields.value.answer.type.e7124b",{v0:uiValue(label)})} value={t.type} options={t.engine.family === "rational_comparison" ? ["single_choice"] : ["numeric", "single_choice"]} optionLabels={enumOptionLabels("question.enum",t.engine.family === "rational_comparison" ? ["single_choice"] : ["numeric", "single_choice"])} onChange={type => change({ type: type as Template["type"], answerFormat: type === "numeric" ? "rational" : null, distractors: type === "numeric" ? [] : ["plus_one"] })}/>{t.type === "numeric" && <SelectField label={uiT("template-fields.value.numeric.format.fa8de6",{v0:uiValue(label)})} value={t.answerFormat ?? "rational"} options={["rational", "percentage"]} optionLabels={enumOptionLabels("question.enum",["rational", "percentage"])} onChange={format => change({ answerFormat: format as "rational" | "percentage" })}/>}<TextField label={uiT("template-fields.value.prompt.31f4c5",{v0:uiValue(label)})} value={t.promptTemplate} onChange={promptTemplate => change({ promptTemplate })} multiline/><TextField label={uiT("template-fields.value.explanation.cedb6c",{v0:uiValue(label)})} value={t.explanationTemplate} onChange={explanationTemplate => change({ explanationTemplate })} multiline/><p><UiText notice={uiMessage("template-fields.answer.placeholders.belong.in.the.explanation.values.remain.exact.c1d000",{})}/></p><Rows label={uiT("template-fields.value.parameter.b27ed8",{v0:uiValue(label)})} items={t.parameters} onChange={parameters => change({ parameters })} create={() => ({ name: t.engine.family === "missing_operand" ? "known" : "left", values: [] })}>{(p, i, set) => <><TextField label={uiT("template-fields.value.parameter.value.name.93d5c0",{v0:uiValue(label),v1:uiValue(i + 1)})} value={p.name} onChange={name => set({ ...p, name })}/><TextField label={uiT("template-fields.value.parameter.value.values.1a245e",{v0:uiValue(label),v1:uiValue(i + 1)})} value={p.values.join("\n")} onChange={v => set({ ...p, values: v === "" ? [] : v.split("\n") })} multiline/></>}</Rows><Rows<Template["constraints"][number]> label={uiT("template-fields.value.constraint.9fe5b9",{v0:uiValue(label)})} items={t.constraints} onChange={constraints => change({ constraints })} create={() => "nonzero_divisor"}>{(v, i, set) => <SelectField label={uiT("template-fields.value.constraint.value.27d9e1",{v0:uiValue(label),v1:uiValue(i + 1)})} value={v} options={["nonzero_divisor", "nonnegative_result", "distinct_operands"]} optionLabels={enumOptionLabels("question.enum",["nonzero_divisor", "nonnegative_result", "distinct_operands"])} onChange={v => set(v as Template["constraints"][number])}/>}</Rows>{t.type === "single_choice" && t.engine.family !== "rational_comparison" && <Rows<Template["distractors"][number]> label={uiT("template-fields.value.distractor.46ec27",{v0:uiValue(label)})} items={t.distractors} onChange={distractors => change({ distractors })} create={() => "plus_one"}>{(v, i, set) => <SelectField label={uiT("template-fields.value.distractor.value.acc9f0",{v0:uiValue(label),v1:uiValue(i + 1)})} value={v} options={["negate", "plus_one", "minus_one", "reciprocal"]} optionLabels={enumOptionLabels("question.enum",["negate", "plus_one", "minus_one", "reciprocal"])} onChange={v => set(v as Template["distractors"][number])}/>}</Rows>}<p><UiText notice={uiMessage("template-fields.generator.version.value.verifier.version.value.bda518",{v0:uiValue(t.engine.generatorVersion),v1:uiValue(t.engine.verifierVersion)})}/></p><UnitFields label={label} value={t.units} onChange={units => change({ units })}/><AssetFields label={label} value={t.assets} onChange={assets => change({ assets })}/><SourcesFields label={label} value={t.sources} onChange={sources => change({ sources })}/></fieldset>;
}
