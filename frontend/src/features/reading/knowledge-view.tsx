import Link from "next/link";
import type {
  ApiResult,
  KnowledgeView as KnowledgeData,
} from "@/lib/api/types";
import { ContentState } from "@/components/content-state";
import { SafeMarkdown, safeTextUrl } from "./safe-markdown";
import styles from "@/styles/reading.module.css";
export function KnowledgeView({
  result,
}: {
  result: ApiResult<KnowledgeData>;
}) {
  if (!result.ok)
    return (
      <ContentState
        kind={result.kind === "not-found" ? "not-found" : "unavailable"}
      />
    );
  const { knowledge: k, units, assets } = result.data;
  const sections = [
    k.statement && ["statement", "Core statement"],
    k.conditions.length && ["conditions", "Conditions"],
    k.scope && ["scope", "Scope & system"],
    k.objectives.length && ["objectives", "Learning goals"],
    k.proof && ["proof", "Proof"],
    units.length && ["explanations", "Explanations"],
    k.relations.length && ["connections", "Connections"],
    k.sources.length && ["sources", "Sources & use"],
  ].filter(Boolean) as string[][];
  return (
    <>
      <nav className="breadcrumbs" aria-label="Breadcrumb">
        <Link prefetch={false} href="/knowledge">
          Knowledge Map
        </Link>
        <span aria-hidden="true">/</span>
        <span>{k.title}</span>
      </nav>
      <header className="page-heading">
        <div className={styles.meta}>
          <span className={styles.kind}>{k.type.replaceAll("-", " ")}</span>
          <span className={styles.version}>Version {k.version}</span>
        </div>
        <h1>{k.title}</h1>
        {k.titleZh && (
          <p className="zh" lang="zh-CN">
            {k.titleZh}
          </p>
        )}
        <p>
          Read the idea, understand its conditions, and follow the connections.
        </p>
      </header>
      <div className={styles.lessonLayout}>
        <nav className={styles.lessonNav} aria-label="On this page">
          <p className="eyebrow">ON THIS PAGE</p>
          {sections.map(([id, label]) => (
            <a href={"#" + id} key={id}>
              {label}
            </a>
          ))}
        </nav>
        <article className={styles.lessonBody}>
          {k.statement && (
            <section id="statement" className={styles.statement}>
              <p className="eyebrow">THE CORE IDEA</p>
              <h2>Core statement</h2>
              <SafeMarkdown source={k.statement} assets={[]} />
            </section>
          )}
          {k.conditions.length > 0 && (
            <section id="conditions" className="panel">
              <h2>Conditions</h2>
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
              <h2>Scope & system</h2>
              {k.scope && <SafeMarkdown source={k.scope} assets={[]} />}{" "}
              {k.system && (
                <div className={styles.system}>
                  <h3>Mathematical system</h3>
                  <SafeMarkdown source={k.system} assets={[]} />
                </div>
              )}
            </section>
          )}
          {k.objectives.length > 0 && (
            <section id="objectives" className="panel">
              <h2>Learning goals</h2>
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
              <h2>Proof</h2>
              <SafeMarkdown source={k.proof} assets={[]} />
            </section>
          )}
          {units.length > 0 && (
            <section id="explanations" className="panel">
              <h2>Explanations</h2>
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
                        <h3>Examples</h3>
                        {u.examples.map((v, i) => (
                          <SafeMarkdown key={i} source={v} assets={bound} />
                        ))}
                      </section>
                    )}
                    {u.counterexamples.length > 0 && (
                      <section className={styles.counterexamples}>
                        <h3>Counterexamples</h3>
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
              <h2>Connections</h2>
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
                                <span>Version {r.target.version}</span>
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
              <h2>Sources & use</h2>
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
                    <p className="muted">Accessed {source.accessedAt}</p>
                  )}
                </div>
              ))}
            </section>
          )}
        </article>
      </div>
      <Link prefetch={false} className={styles.backLink} href="/knowledge">
        ← Back to the knowledge map
      </Link>
    </>
  );
}
