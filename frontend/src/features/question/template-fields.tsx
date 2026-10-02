"use client";
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
}) { return <Rows label={label + " source"} items={value} onChange={onChange} create={emptySource}>{(s, i, change) => <fieldset><legend>{label} source {i + 1}</legend>{(["kind", "author", "title", "url", "accessedAt", "license", "attribution"] as const).map(key => <TextField key={key} label={`${label} source ${i + 1} ${key}`} value={s[key]} onChange={v => change({ ...s, [key]: v })} multiline={key === "attribution"}/>)}</fieldset>}</Rows>; }
export function ObjectiveIndices({ label, value, onChange }: {
    label: string;
    value: number[];
    onChange: (v: number[]) => void;
}) { return <Rows label={label + " objective index"} items={value} onChange={onChange} create={() => 0}>{(v, i, change) => <label className={styles.check}>{label} objective index {i + 1}<input aria-label={`${label} objective index ${i + 1}`} type="number" min={0} max={2147483647} step={1} value={Number.isFinite(v) ? v : ""} onChange={e => change(e.target.value === "" ? NaN : Number(e.target.value))}/></label>}</Rows>; }
export function CoverageFields({ label, value, onChange }: {
    label: string;
    value: Template["coverage"];
    onChange: (v: Template["coverage"]) => void;
}) { return <Rows label={label + " coverage"} items={value} onChange={onChange} create={() => ({ knowledge: { id: "", version: 1 }, objectiveIndices: [0] })}>{(v, i, change) => <fieldset><legend>{label} coverage {i + 1}</legend><RefFields label={`${label} coverage ${i + 1} knowledge`} value={v.knowledge} onChange={knowledge => change({ ...v, knowledge })}/><ObjectiveIndices label={`${label} coverage ${i + 1}`} value={v.objectiveIndices} onChange={objectiveIndices => change({ ...v, objectiveIndices })}/></fieldset>}</Rows>; }
export function UnitFields({ label, value, onChange }: {
    label: string;
    value: Template["units"];
    onChange: (v: Template["units"]) => void;
}) { return <Rows label={label + " unit"} items={value} onChange={onChange} create={() => ({ id: "", version: 1 })}>{(v, i, change) => <RefFields label={`${label} unit ${i + 1}`} value={v} onChange={change}/>}</Rows>; }
export function AssetFields({ label, value, onChange }: {
    label: string;
    value: Template["assets"];
    onChange: (v: Template["assets"]) => void;
}) { return <Rows label={label + " fixed illustration"} items={value} onChange={onChange} create={() => ({ id: "", sha256: "" })}>{(v, i, change) => <><TextField label={`${label} illustration ${i + 1} ID`} value={v.id} onChange={id => change({ ...v, id })}/><TextField label={`${label} illustration ${i + 1} SHA`} value={v.sha256} onChange={sha256 => change({ ...v, sha256 })}/></>}</Rows>; }
export function PublishedObjectives({ knowledge }: {
    knowledge: Template["knowledge"];
}) {
    const [data, setData] = useState<KnowledgeView | null>(null), [busy, setBusy] = useState(false), [error, setError] = useState<string | null>(null), active = useRef<AbortController | null>(null);
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
            setError("This exact published knowledge version is unavailable. Validation will check the reference.");
    }
    finally {
        clearTimeout(timer);
        if (active.current === controller) {
            setBusy(false);
            active.current = null;
        }
    } }
    return <section><button className="button secondary" type="button" disabled={busy} onClick={() => void load()}>Load published objectives</button>{error && <p role="status">{error}</p>}{data && <><h4>{data.knowledge.title} · {data.knowledge.titleZh}</h4><ul>{data.knowledge.objectives.map((o, i) => <li key={i}>{i}: {o}</li>)}</ul></>}</section>;
}
export const newTemplate = (): Template => ({ id: "", version: 1, knowledge: { id: "", version: 1 }, coverage: [], units: [], type: "numeric", answerFormat: "rational", promptTemplate: "Calculate {{left}} + {{right}}.", explanationTemplate: "The sum is {{answer}}.", engine: { family: "rational_arithmetic", operation: "add", unknownSide: null, generatorVersion: 1, verifierVersion: 1 }, parameters: [{ name: "left", values: ["1"] }, { name: "right", values: ["2"] }], constraints: [], distractors: [], assets: [], sources: [] });
export function TemplateFields({ value, label, onChange }: {
    value: Template;
    label: string;
    onChange: (v: Template) => void;
}) {
    const t = value, change = (patch: Partial<Template>) => onChange({ ...t, ...patch });
    return <fieldset><legend>{label} · finite exact parameters</legend><div className={styles.grid}><TextField label={label + " ID"} value={t.id} onChange={id => change({ id })}/><NumberField label={label + " version"} value={t.version} onChange={version => change({ version })}/></div><RefFields label={label + " primary knowledge"} value={t.knowledge} onChange={knowledge => change({ knowledge })}/><PublishedObjectives knowledge={t.knowledge}/><CoverageFields label={label} value={t.coverage} onChange={coverage => change({ coverage })}/><SelectField label={label + " family"} value={t.engine.family} options={["rational_arithmetic", "rational_comparison", "missing_operand"]} onChange={family => change({ engine: { ...t.engine, family: family as Template["engine"]["family"], operation: family === "rational_comparison" ? "compare" : "add", unknownSide: family === "missing_operand" ? "left" : null }, parameters: family === "missing_operand" ? [{ name: "known", values: ["1"] }, { name: "result", values: ["2"] }] : [{ name: "left", values: ["1"] }, { name: "right", values: ["2"] }], constraints: [], distractors: [], ...(family === "rational_comparison" ? { type: "single_choice" as const, answerFormat: null } : {}) })}/><SelectField label={label + " operation"} value={t.engine.operation} options={t.engine.family === "rational_comparison" ? ["compare"] : ["add", "subtract", "multiply", "divide"]} onChange={operation => change({ engine: { ...t.engine, operation: operation as Template["engine"]["operation"] } })}/>{t.engine.family === "missing_operand" && <SelectField label={label + " unknown side"} value={t.engine.unknownSide ?? "left"} options={["left", "right"]} onChange={side => change({ engine: { ...t.engine, unknownSide: side as "left" | "right" } })}/>}<SelectField label={label + " answer type"} value={t.type} options={t.engine.family === "rational_comparison" ? ["single_choice"] : ["numeric", "single_choice"]} onChange={type => change({ type: type as Template["type"], answerFormat: type === "numeric" ? "rational" : null, distractors: type === "numeric" ? [] : ["plus_one"] })}/>{t.type === "numeric" && <SelectField label={label + " numeric format"} value={t.answerFormat ?? "rational"} options={["rational", "percentage"]} onChange={format => change({ answerFormat: format as "rational" | "percentage" })}/>}<TextField label={label + " prompt"} value={t.promptTemplate} onChange={promptTemplate => change({ promptTemplate })} multiline/><TextField label={label + " explanation"} value={t.explanationTemplate} onChange={explanationTemplate => change({ explanationTemplate })} multiline/><p>Answer placeholders belong in the explanation. Values remain exact text; Go generates and independently verifies the finite batch.</p><Rows label={label + " parameter"} items={t.parameters} onChange={parameters => change({ parameters })} create={() => ({ name: t.engine.family === "missing_operand" ? "known" : "left", values: [] })}>{(p, i, set) => <><TextField label={`${label} parameter ${i + 1} name`} value={p.name} onChange={name => set({ ...p, name })}/><TextField label={`${label} parameter ${i + 1} values`} value={p.values.join("\n")} onChange={v => set({ ...p, values: v === "" ? [] : v.split("\n") })} multiline/></>}</Rows><Rows<Template["constraints"][number]> label={label + " constraint"} items={t.constraints} onChange={constraints => change({ constraints })} create={() => "nonzero_divisor"}>{(v, i, set) => <SelectField label={`${label} constraint ${i + 1}`} value={v} options={["nonzero_divisor", "nonnegative_result", "distinct_operands"]} onChange={v => set(v as Template["constraints"][number])}/>}</Rows>{t.type === "single_choice" && t.engine.family !== "rational_comparison" && <Rows<Template["distractors"][number]> label={label + " distractor"} items={t.distractors} onChange={distractors => change({ distractors })} create={() => "plus_one"}>{(v, i, set) => <SelectField label={`${label} distractor ${i + 1}`} value={v} options={["negate", "plus_one", "minus_one", "reciprocal"]} onChange={v => set(v as Template["distractors"][number])}/>}</Rows>}<p>Generator version {t.engine.generatorVersion} · Verifier version {t.engine.verifierVersion}</p><UnitFields label={label} value={t.units} onChange={units => change({ units })}/><AssetFields label={label} value={t.assets} onChange={assets => change({ assets })}/><SourcesFields label={label} value={t.sources} onChange={sources => change({ sources })}/></fieldset>;
}
