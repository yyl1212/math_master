"use client";
import {UiText} from "@/lib/i18n/ui-text";
import {uiMessage,uiValue} from "@/lib/i18n/format";
import {useUiI18n} from "@/lib/i18n/provider";
import type { Package, AssetView, AssetScope, GateReport } from "@/lib/content/types";
import { SafeMarkdown } from "@/features/reading/safe-markdown";
import styles from "@/styles/content.module.css";
export function GatePanel({ gate }: {
    gate: GateReport;
}) {
 const {t}=useUiI18n();
 return <section className={styles.card}><h2><UiText notice={uiMessage("content-preview.readiness.check.2f70ae",{})}/></h2><p>{gate.readyToSubmit ? t("content-preview.machine.checks.passed.reviewer.approval.is.still.required.184ba1",{}) : t("content-preview.content.needs.work.before.review.bb2a94",{})}</p><p><UiText notice={uiMessage("content-preview.structural.issues.value.completeness.issues.value.human.checks.va.6fe5e4",{v0:uiValue(gate.structuralTotal),v1:uiValue(gate.completenessTotal),v2:uiValue(gate.humanReviewTotal)})}/></p>{gate.truncated && <p><UiText notice={uiMessage("content-preview.showing.the.first.100.issues.be469a",{})}/></p>}<ul>{[...gate.structuralErrors, ...gate.completenessErrors, ...gate.humanReviewRequirements].map((v, i) => <li key={i}><strong>{v.code}</strong> {v.path}: {v.message}</li>)}</ul></section>; }
function sourceURL(value: string): string | null {
    try { const url = new URL(value); return url.protocol === "https:" && url.hostname && !url.username && !url.password ? value : null; } catch { return null; }
}
export function ContentPreview({ value, assets, assetScope }: { value: Package; assets: AssetView[]; assetScope: AssetScope }) {
    const markdown = (source: string) => <SafeMarkdown source={source} assets={assets} assetScope={assetScope}/>;
    return <section className={styles.card}>
        <h2><UiText notice={uiMessage("content-preview.safe.preview.eeb51b",{})}/></h2>
        {assetScope.kind === "draft" && <p><UiText notice={uiMessage("content-preview.save.new.illustrations.to.preview.their.bound.bytes.deb444",{})}/></p>}
        {value.knowledge.map(k => <article key={k.id + ":" + k.version}>
            <h3>{k.title || k.id} <small>{k.titleZh}</small></h3>
            <p><UiText notice={uiMessage("content-preview.value.value.version.value.f7a6b1",{v0:uiValue(k.id),v1:uiValue(k.type),v2:uiValue(k.version)})}/></p>
            <p><UiText notice={uiMessage("content-preview.domains.value.topics.value.e9a533",{v0:uiValue(k.domainIds.join(", ")),v1:uiValue(k.topicIds.join(", "))})}/></p>
            <h4><UiText notice={uiMessage("content-preview.statement.6171b2",{})}/></h4>{markdown(k.statement)}
            <h4><UiText notice={uiMessage("content-preview.scope.b073f6",{})}/></h4>{markdown(k.scope)}<h4><UiText notice={uiMessage("content-preview.system.6725e7",{})}/></h4>{markdown(k.system)}
            <h4><UiText notice={uiMessage("content-preview.learning.objectives.d38eb0",{})}/></h4>{k.objectives.map((text,i)=><div key={i}>{markdown(text)}</div>)}
            <h4><UiText notice={uiMessage("knowledge-view.conditions.97d4be",{})}/></h4>{k.conditions.length ? k.conditions.map((text,i)=><div key={i}>{markdown(text)}</div>) : <p><UiText notice={uiMessage("content-preview.no.additional.conditions.declared.c0637b",{})}/></p>}
            {k.proof && <><h4><UiText notice={uiMessage("knowledge-view.proof.7fbb3c",{})}/></h4>{markdown(k.proof)}</>}
            <h4><UiText notice={uiMessage("content-preview.sources.caf85b",{})}/></h4>{k.sources.length ? <ul>{k.sources.map((source,i)=><li key={i}>
                {sourceURL(source.url) ? <a href={source.url} rel="noreferrer">{source.title}</a> : <strong>{source.title}</strong>}
                <p>{source.author} · {source.kind} · {source.license}</p>
                {source.url && <p>{source.url}</p>}{source.accessedAt && <p><UiText notice={uiMessage("content-preview.accessed.value.5dc1f8",{v0:uiValue(source.accessedAt)})}/></p>}<p>{source.attribution}</p>
            </li>)}</ul> : <p><UiText notice={uiMessage("content-preview.no.sources.yet.810fee",{})}/></p>}
            <h4><UiText notice={uiMessage("content-preview.knowledge.relationships.242c96",{})}/></h4>{k.relations.length ? <ul>{k.relations.map((relation,i)=><li key={i}><UiText notice={uiMessage("content-preview.value.value.vvalue.afac6e",{v0:uiValue(relation.kind),v1:uiValue(relation.target.id),v2:uiValue(relation.target.version)})}/></li>)}</ul> : <p><UiText notice={uiMessage("content-preview.no.relationships.declared.4d1ca5",{})}/></p>}
            {value.units.filter(u=>u.knowledge.id===k.id && u.knowledge.version===k.version).map(u=><section key={u.id+":"+u.version}>
                <h4><UiText notice={uiMessage("content-preview.value.version.value.1d1973",{v0:uiValue(u.id),v1:uiValue(u.version)})}/></h4><p><UiText notice={uiMessage("content-preview.knowledge.value.vvalue.assets.value.523132",{v0:uiValue(u.knowledge.id),v1:uiValue(u.knowledge.version),v2:uiValue(u.assetIds.join(", "))})}/></p>
                {u.angles.map((angle,i)=><div key={i}><h5>{angle.kind}</h5>{markdown(angle.body)}</div>)}
                <h5><UiText notice={uiMessage("knowledge-view.examples.e68ee0",{})}/></h5>{u.examples.map((text,i)=><div key={i}>{markdown(text)}</div>)}
                <h5><UiText notice={uiMessage("knowledge-view.counterexamples.1e428c",{})}/></h5>{u.counterexamples.map((text,i)=><div key={i}>{markdown(text)}</div>)}
            </section>)}
        </article>)}
        <h3><UiText notice={uiMessage("domain-view.learning.paths.5f23ab",{})}/></h3>{value.paths.length ? value.paths.map(path=><section key={path.id+":"+path.version}>
            <h4>{path.title} <small>{path.titleZh}</small></h4><p><UiText notice={uiMessage("content-preview.value.vvalue.domains.value.206a60",{v0:uiValue(path.id),v1:uiValue(path.version),v2:uiValue(path.domainIds.join(", "))})}/></p>
            <ol>{path.nodes.map((ref,i)=><li key={i}><UiText notice={uiMessage("content-preview.value.vvalue.bde90b",{v0:uiValue(ref.id),v1:uiValue(ref.version)})}/></li>)}</ol>
        </section>) : <p><UiText notice={uiMessage("content-preview.no.learning.paths.in.this.package.09d6d0",{})}/></p>}
    </section>;
}
