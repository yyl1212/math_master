import type { Package, AssetView, AssetScope, GateReport } from "@/lib/content/types";
import { SafeMarkdown } from "@/features/reading/safe-markdown";
import styles from "@/styles/content.module.css";
export function GatePanel({ gate }: {
    gate: GateReport;
}) { return <section className={styles.card}><h2>Readiness check</h2><p>{gate.readyToSubmit ? "Machine checks passed. Independent review is still required." : "Content needs work before review."}</p><p>Structural issues: {gate.structuralTotal} · Completeness issues: {gate.completenessTotal} · Human checks: {gate.humanReviewTotal}</p>{gate.truncated && <p>Showing the first 100 issues.</p>}<ul>{[...gate.structuralErrors, ...gate.completenessErrors, ...gate.humanReviewRequirements].map((v, i) => <li key={i}><strong>{v.code}</strong> {v.path}: {v.message}</li>)}</ul></section>; }
export function ContentPreview({ value, assets, assetScope }: {
    value: Package;
    assets: AssetView[];
    assetScope: AssetScope;
}) { return <section className={styles.card}><h2>Safe preview</h2><p>Save new illustrations to preview their bound bytes.</p>{value.knowledge.map(k => <article key={k.id + ":" + k.version}><h3>{k.title || k.id} <small>{k.titleZh}</small></h3><p>{k.type} · Version {k.version}</p>{[k.statement, k.scope, k.system, ...k.objectives, ...k.conditions, k.proof].filter(Boolean).map((source, i) => <SafeMarkdown key={i} source={source} assets={assets} assetScope={assetScope}/>)}{value.units.filter(u => u.knowledge.id === k.id && u.knowledge.version === k.version).map(u => <section key={u.id + ":" + u.version}><h4>{u.id} · Version {u.version}</h4>{u.angles.map((a, i) => <div key={i}><h5>{a.kind}</h5><SafeMarkdown source={a.body} assets={assets} assetScope={assetScope}/></div>)}{[...u.examples, ...u.counterexamples].map((source, i) => <SafeMarkdown key={i} source={source} assets={assets} assetScope={assetScope}/>)}</section>)}</article>)}</section>; }
