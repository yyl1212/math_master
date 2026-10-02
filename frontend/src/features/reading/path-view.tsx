import Link from "next/link";
import type { ApiResult, PathView as PathData } from "@/lib/api/types";
import { ContentState } from "@/components/content-state";
import { buildPathGraph, refKey } from "./path-graph";
import { PathConnections } from "./path-connections";
import styles from "@/styles/reading.module.css";
export function PathView({ result,personal }: { result: ApiResult<PathData>;personal?:React.ReactNode }) {
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
      <nav className="breadcrumbs" aria-label="Breadcrumb">
        <Link prefetch={false} href="/knowledge">
          Knowledge Map
        </Link>
        <span aria-hidden="true">/</span>
        <span>{path.title}</span>
      </nav>
      <header className="page-heading">
        <p className="eyebrow">LEARNING PATH</p>
        <h1>{path.title}</h1>
        {path.titleZh && (
          <p className="zh" lang="zh-CN">
            {path.titleZh}
          </p>
        )}
        <div className={styles.meta}>
          <span className={styles.version}>Version {path.version}</span>
          <span>{path.nodes.length} knowledge points</span>
        </div>
        <p className={styles.pathIntro}>
          Read freely and follow the prerequisites to connect the ideas.
        </p>
      </header>
      {personal}
      <section
        className={styles.pathGraph}
        aria-label="Knowledge prerequisites"
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
                  <span className={styles.version}>Version {k.version}</span>
                  <div className={styles.prerequisites}>
                    <h3>Prerequisites</h3>
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
                            <span>v{r.target.version}</span>
                          </li>
                        ))}
                      </ul>
                    ) : (
                      <p>No prior knowledge required.</p>
                    )}
                  </div>
                </article>
              );
            })}
          </div>
        ))}
      </section>
      <p className={styles.pathNote}>
        Connections show prerequisites for this path version. Reading a
        knowledge point does not create a learning or assessment record.
      </p>
    </>
  );
}
