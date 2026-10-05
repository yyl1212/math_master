"use client";
import {UiText} from "@/lib/i18n/ui-text";
import {uiMessage,uiValue} from "@/lib/i18n/format";
import {useUiI18n} from "@/lib/i18n/provider";
import Link from "next/link";
import type {StaticMessageKey} from "@/lib/i18n/types";
import type {
  ApiResult,
  KnowledgeView as KnowledgeData,
} from "@/lib/api/types";
import { ContentState } from "@/components/content-state";
import { SafeMarkdown, safeTextUrl } from "./safe-markdown";
import styles from "@/styles/reading.module.css";
export function KnowledgeView({
  result, personal,
}: {
  result: ApiResult<KnowledgeData>;personal?:React.ReactNode;
}) {
 const {t}=useUiI18n();

  if (!result.ok)
    return (
      <ContentState
        kind={result.kind === "not-found" ? "not-found" : "unavailable"}
      />
    );
  const { knowledge: k, units, assets } = result.data;
  const sections = [
    k.statement && ["statement", "knowledge-view.core.statement.e943c8"],
    k.conditions.length && ["conditions", "knowledge-view.conditions.97d4be"],
    k.scope && ["scope", "knowledge-view.scope.system.6be795"],
    k.objectives.length && ["objectives", "knowledge-view.learning.goals.9e640a"],
    k.proof && ["proof", "knowledge-view.proof.7fbb3c"],
    units.length && ["explanations", "knowledge-view.explanations.de30ae"],
    k.relations.length && ["connections", "knowledge-view.connections.dc2731"],
    k.sources.length && ["sources", "knowledge-view.sources.use.31f2b5"],
  ].filter(Boolean) as [string,StaticMessageKey][];
  return (
    <>
      <nav className="breadcrumbs" aria-label={t("domain-view.breadcrumb.2bd873",{})}>
        <Link prefetch={false} href="/knowledge"><UiText notice={uiMessage("nav.knowledgeMap",{})}/></Link>
        <span aria-hidden="true">/</span>
        <span>{k.title}</span>
      </nav>
      <header className="page-heading">
        <div className={styles.meta}>
          <span className={styles.kind}>{k.type.replaceAll("-", " ")}</span>
          <span className={styles.version}><UiText notice={uiMessage("domain-view.version.value.d2b5e7",{v0:uiValue(k.version)})}/></span>
        </div>
        <h1 lang="en">{k.title}</h1>
        {k.titleZh && (
          <p className="zh" lang="zh-CN">
            {k.titleZh}
          </p>
        )}
        <p><UiText notice={uiMessage("knowledge-view.read.the.idea.understand.its.conditions.and.follow.the.connection.466280",{})}/></p>
      </header>
      {personal}
      <div className={styles.lessonLayout}>
        <nav className={styles.lessonNav} aria-label={t("knowledge-view.on.this.page.b5658f",{})}>
          <p className="eyebrow"><UiText notice={uiMessage("knowledge-view.on.this.page.073c74",{})}/></p>
          {sections.map(([id, label]) => (
            <a href={"#" + id} key={id}>
              <UiText notice={uiMessage(label,{})}/>
            </a>
          ))}
        </nav>
        <article className={styles.lessonBody}>
          {k.statement && (
            <section id="statement" className={styles.statement}>
              <p className="eyebrow"><UiText notice={uiMessage("knowledge-view.the.core.idea.6956da",{})}/></p>
              <h2><UiText notice={uiMessage("knowledge-view.core.statement.e943c8",{})}/></h2>
              <SafeMarkdown source={k.statement} assets={[]} />
            </section>
          )}
          {k.conditions.length > 0 && (
            <section id="conditions" className="panel">
              <h2><UiText notice={uiMessage("knowledge-view.conditions.97d4be",{})}/></h2>
              <ul className={styles.fieldList}>
                {k.conditions.map((v, i) => (
                  <li key={i}>
                    <SafeMarkdown source={v} assets={[]} />
                  </li>
                ))}
              </ul>
            </section>
          )}
          {(k.scope || k.system) && (
            <section id="scope" className="panel">
              <h2><UiText notice={uiMessage("knowledge-view.scope.system.6be795",{})}/></h2>
              {k.scope && <SafeMarkdown source={k.scope} assets={[]} />}{" "}
              {k.system && (
                <div className={styles.system}>
                  <h3><UiText notice={uiMessage("knowledge-view.mathematical.system.eec854",{})}/></h3>
                  <SafeMarkdown source={k.system} assets={[]} />
                </div>
              )}
            </section>
          )}
          {k.objectives.length > 0 && (
            <section id="objectives" className="panel">
              <h2><UiText notice={uiMessage("knowledge-view.learning.goals.9e640a",{})}/></h2>
              <ul className={styles.fieldList}>
                {k.objectives.map((v, i) => (
                  <li key={i}>
                    <SafeMarkdown source={v} assets={[]} />
                  </li>
                ))}
              </ul>
            </section>
          )}
          {k.proof && (
            <section id="proof" className="panel">
              <h2><UiText notice={uiMessage("knowledge-view.proof.7fbb3c",{})}/></h2>
              <SafeMarkdown source={k.proof} assets={[]} />
            </section>
          )}
          {units.length > 0 && (
            <section id="explanations" className="panel">
              <h2><UiText notice={uiMessage("knowledge-view.explanations.de30ae",{})}/></h2>
              {units.map((u) => {
                const bound = assets.filter(
                  (a) =>
                    u.assetIds.includes(a.id) &&
                    a.knowledge.id === k.id &&
                    a.knowledge.version === k.version,
                );
                return (
                  <div className={styles.unit} key={u.id + "@" + u.version}>
                    {u.angles.map((a, i) => (
                      <section key={i} className={styles.angle}>
                        <h3>{a.kind.replaceAll("-", " ")}</h3>
                        <SafeMarkdown source={a.body} assets={bound} />
                      </section>
                    ))}
                    {u.examples.length > 0 && (
                      <section className={styles.examples}>
                        <h3><UiText notice={uiMessage("knowledge-view.examples.e68ee0",{})}/></h3>
                        {u.examples.map((v, i) => (
                          <SafeMarkdown key={i} source={v} assets={bound} />
                        ))}
                      </section>
                    )}
                    {u.counterexamples.length > 0 && (
                      <section className={styles.counterexamples}>
                        <h3><UiText notice={uiMessage("knowledge-view.counterexamples.1e428c",{})}/></h3>
                        {u.counterexamples.map((v, i) => (
                          <SafeMarkdown key={i} source={v} assets={bound} />
                        ))}
                      </section>
                    )}
                  </div>
                );
              })}
            </section>
          )}
          {k.relations.length > 0 && (
            <section id="connections" className="panel">
              <h2><UiText notice={uiMessage("knowledge-view.connections.dc2731",{})}/></h2>
              {(["prerequisite", "derivation", "related"] as const).map(
                (kind) => {
                  const refs = k.relations.filter((r) => r.kind === kind);
                  return (
                    refs.length > 0 && (
                      <div className={styles.relationGroup} key={kind}>
                        <h3>
                          {kind === "prerequisite"
                            ? "Prerequisites"
                            : kind === "derivation"
                              ? "Derivations"
                              : "Related ideas"}
                        </h3>
                        <ul>
                          {refs.map((r) => (
                            <li key={r.target.id + "@" + r.target.version}>
                              <Link
                                prefetch={false}
                                href={"/knowledge/" + r.target.id}
                              >
                                {r.target.id}{" "}
                                <span><UiText notice={uiMessage("domain-view.version.value.d2b5e7",{v0:uiValue(r.target.version)})}/></span>
                                <span aria-hidden="true">↗</span>
                              </Link>
                            </li>
                          ))}
                        </ul>
                      </div>
                    )
                  );
                },
              )}
            </section>
          )}
          {k.sources.length > 0 && (
            <section id="sources" className="panel">
              <h2><UiText notice={uiMessage("knowledge-view.sources.use.31f2b5",{})}/></h2>
              {k.sources.map((source, i) => (
                <div className={styles.source} key={i}>
                  <h3>
                    {safeTextUrl(source.url) ? (
                      <a
                        href={safeTextUrl(source.url)}
                        rel="noopener noreferrer"
                      >
                        {source.title} <span aria-hidden="true">↗</span>
                      </a>
                    ) : (
                      source.title
                    )}
                  </h3>
                  <p>{source.author}</p>
                  <p className={styles.sourceLicense}>{source.license}</p>
                  {source.attribution && <p>{source.attribution}</p>}
                  {source.accessedAt && (
                    <p className="muted"><UiText notice={uiMessage("knowledge-view.accessed.value.908096",{v0:uiValue(source.accessedAt)})}/></p>
                  )}
                </div>
              ))}
            </section>
          )}
        </article>
      </div>
      <Link prefetch={false} className={styles.backLink} href="/knowledge"><UiText notice={uiMessage("knowledge-view.back.to.the.knowledge.map.8bf2a2",{})}/></Link>
    </>
  );
}
