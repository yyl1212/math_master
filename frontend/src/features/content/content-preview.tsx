import type { Package, AssetView, AssetScope, GateReport } from "@/lib/content/types";
import { SafeMarkdown } from "@/features/reading/safe-markdown";
import styles from "@/styles/content.module.css";
export function GatePanel({ gate }: {
    gate: GateReport;
}) { return <section className={styles.card}><h2>Readiness check</h2><p>{gate.readyToSubmit ? "Machine checks passed. Reviewer approval is still required." : "Content needs work before review."}</p><p>Structural issues: {gate.structuralTotal} · Completeness issues: {gate.completenessTotal} · Human checks: {gate.humanReviewTotal}</p>{gate.truncated && <p>Showing the first 100 issues.</p>}<ul>{[...gate.structuralErrors, ...gate.completenessErrors, ...gate.humanReviewRequirements].map((v, i) => <li key={i}><strong>{v.code}</strong> {v.path}: {v.message}</li>)}</ul></section>; }
function sourceURL(value: string): string | null {
    try { const url = new URL(value); return url.protocol === "https:" && url.hostname && !url.username && !url.password ? value : null; } catch { return null; }
}
export function ContentPreview({ value, assets, assetScope }: { value: Package; assets: AssetView[]; assetScope: AssetScope }) {
    const markdown = (source: string) => <SafeMarkdown source={source} assets={assets} assetScope={assetScope}/>;
    return <section className={styles.card}>
        <h2>Safe preview</h2>
        {assetScope.kind === "draft" && <p>Save new illustrations to preview their bound bytes.</p>}
        {value.knowledge.map(k => <article key={k.id + ":" + k.version}>
            <h3>{k.title || k.id} <small>{k.titleZh}</small></h3>
            <p>{k.id} · {k.type} · Version {k.version}</p>
            <p>Domains: {k.domainIds.join(", ")} · Topics: {k.topicIds.join(", ")}</p>
            <h4>Statement</h4>{markdown(k.statement)}
            <h4>Scope</h4>{markdown(k.scope)}<h4>System</h4>{markdown(k.system)}
            <h4>Learning objectives</h4>{k.objectives.map((text,i)=><div key={i}>{markdown(text)}</div>)}
            <h4>Conditions</h4>{k.conditions.length ? k.conditions.map((text,i)=><div key={i}>{markdown(text)}</div>) : <p>No additional conditions declared.</p>}
            {k.proof && <><h4>Proof</h4>{markdown(k.proof)}</>}
            <h4>Sources</h4>{k.sources.length ? <ul>{k.sources.map((source,i)=><li key={i}>
                {sourceURL(source.url) ? <a href={source.url} rel="noreferrer">{source.title}</a> : <strong>{source.title}</strong>}
                <p>{source.author} · {source.kind} · {source.license}</p>
                {source.url && <p>{source.url}</p>}{source.accessedAt && <p>Accessed: {source.accessedAt}</p>}<p>{source.attribution}</p>
            </li>)}</ul> : <p>No sources yet.</p>}
            <h4>Knowledge relationships</h4>{k.relations.length ? <ul>{k.relations.map((relation,i)=><li key={i}>{relation.kind}: {relation.target.id} v{relation.target.version}</li>)}</ul> : <p>No relationships declared.</p>}
            {value.units.filter(u=>u.knowledge.id===k.id && u.knowledge.version===k.version).map(u=><section key={u.id+":"+u.version}>
                <h4>{u.id} · Version {u.version}</h4><p>Knowledge: {u.knowledge.id} v{u.knowledge.version} · Assets: {u.assetIds.join(", ")}</p>
                {u.angles.map((angle,i)=><div key={i}><h5>{angle.kind}</h5>{markdown(angle.body)}</div>)}
                <h5>Examples</h5>{u.examples.map((text,i)=><div key={i}>{markdown(text)}</div>)}
                <h5>Counterexamples</h5>{u.counterexamples.map((text,i)=><div key={i}>{markdown(text)}</div>)}
            </section>)}
        </article>)}
        <h3>Learning paths</h3>{value.paths.length ? value.paths.map(path=><section key={path.id+":"+path.version}>
            <h4>{path.title} <small>{path.titleZh}</small></h4><p>{path.id} v{path.version} · Domains: {path.domainIds.join(", ")}</p>
            <ol>{path.nodes.map((ref,i)=><li key={i}>{ref.id} v{ref.version}</li>)}</ol>
        </section>) : <p>No learning paths in this package.</p>}
    </section>;
}
