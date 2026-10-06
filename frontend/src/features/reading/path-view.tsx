"use client";
import {UiText} from "@/lib/i18n/ui-text";
import {uiMessage,uiValue} from "@/lib/i18n/format";
import {useUiI18n} from "@/lib/i18n/provider";
import Link from "next/link";
import type { ApiResult, PathView as PathData } from "@/lib/api/types";
import { ContentState } from "@/components/content-state";
import { buildPathGraph, refKey } from "./path-graph";
import { PathConnections } from "./path-connections";
import styles from "@/styles/reading.module.css";
export function PathView({ result,personal }: { result: ApiResult<PathData>;personal?:React.ReactNode }) {
 const {t}=useUiI18n();

  if (!result.ok)
    return (
      <ContentState
        kind={result.kind === "not-found" ? "not-found" : "unavailable"}
      />
    );
  let graph: ReturnType<typeof buildPathGraph>;
  try {
    graph = buildPathGraph(result.data);
  } catch {
    return <ContentState kind="unavailable" />;
  }
  const { path, knowledge } = result.data,
    byRef = new Map(knowledge.map((v) => [refKey(v.knowledge), v.knowledge]));
  return (
    <>
      <nav className="breadcrumbs" aria-label={t("domain-view.breadcrumb.2bd873",{})}>
        <Link prefetch={false} href="/knowledge"><UiText notice={uiMessage("nav.knowledgeMap",{})}/></Link>
        <span aria-hidden="true">/</span>
        <span>{path.title}</span>
      </nav>
      <header className="page-heading">
        <p className="eyebrow"><UiText notice={uiMessage("path-view.learning.path.8c4705",{})}/></p>
        <h1 lang="en">{path.title}</h1>
        {path.titleZh && (
          <p className="zh" lang="zh-CN">
            {path.titleZh}
          </p>
        )}
        <div className={styles.meta}>
          <span className={styles.version}><UiText notice={uiMessage("domain-view.version.value.d2b5e7",{v0:uiValue(path.version)})}/></span>
          <span><UiText notice={uiMessage("path-view.value.knowledge.points.38db06",{v0:uiValue(path.nodes.length)})}/></span>
        </div>
        <p className={styles.pathIntro}><UiText notice={uiMessage("path-view.read.freely.and.follow.the.prerequisites.to.connect.the.ideas.ab5874",{})}/></p>
      </header>
      {personal}
      <section
        className={styles.pathGraph}
        aria-label={t("path-view.knowledge.prerequisites.5450ad",{})}
      >
        <PathConnections edges={graph.edges} />
        {graph.levels.map((level, index) => (
          <div className={styles.graphLevel} key={index}>
            {level.map((ref) => {
              const k = byRef.get(refKey(ref))!,
                prerequisites = k.relations.filter(
                  (r) => r.kind === "prerequisite",
                );
              return (
                <article
                  className={styles.node}
                  data-node-key={refKey(ref)}
                  key={refKey(ref)}
                >
                  <span className={styles.nodeType}>
                    {k.type.replaceAll("-", " ")}
                  </span>
                  <h2>
                    <Link prefetch={false} href={"/knowledge/" + k.id}>
                      {k.title}
                    </Link>
                  </h2>
                  {k.titleZh && (
                    <p lang="zh-CN" className="zh">
                      {k.titleZh}
                    </p>
                  )}
                  <span className={styles.version}><UiText notice={uiMessage("domain-view.version.value.d2b5e7",{v0:uiValue(k.version)})}/></span>
                  <div className={styles.prerequisites}>
                    <h3><UiText notice={uiMessage("path-view.prerequisites.865514",{})}/></h3>
                    {prerequisites.length ? (
                      <ul>
                        {prerequisites.map((r) => (
                          <li key={refKey(r.target)}>
                            <Link
                              prefetch={false}
                              href={"/knowledge/" + r.target.id}
                            >
                              {byRef.get(refKey(r.target))!.title}
                            </Link>{" "}
                            <span><UiText notice={uiMessage("path-view.vvalue.55a29f",{v0:uiValue(r.target.version)})}/></span>
                          </li>
                        ))}
                      </ul>
                    ) : (
                      <p><UiText notice={uiMessage("path-view.no.prior.knowledge.required.1da6b0",{})}/></p>
                    )}
                  </div>
                </article>
              );
            })}
          </div>
        ))}
      </section>
      <p className={styles.pathNote}><UiText notice={uiMessage("path-view.connections.show.prerequisites.for.this.path.version.reading.a.kn.477376",{})}/></p>
    </>
  );
}
