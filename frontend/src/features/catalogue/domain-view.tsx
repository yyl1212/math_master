import Link from "next/link";
import type { ApiResult, DomainDetail } from "@/lib/api/types";
import { ContentStatus } from "@/components/content-status";
import { ContentState } from "@/components/content-state";
import styles from "@/styles/catalogue.module.css";
export function DomainView({ result }: { result: ApiResult<DomainDetail> }) {
  if (!result.ok)
    return (
      <ContentState
        kind={result.kind === "not-found" ? "not-found" : "unavailable"}
      />
    );
  const d = result.data;
  return (
    <>
      <nav className="breadcrumbs" aria-label="Breadcrumb">
        <Link prefetch={false} href="/">
          Learning Hub
        </Link>
        <span aria-hidden="true">/</span>
        <Link prefetch={false} href="/knowledge">
          Knowledge Map
        </Link>
        <span aria-hidden="true">/</span>
        <span>{d.name}</span>
      </nav>
      <div className="page-heading">
        <p className="eyebrow">
          LEARNING DOMAIN {String(d.order).padStart(2, "0")}
        </p>
        <h1>{d.name}</h1>
        <p className="zh" lang="zh-CN">
          {d.nameZh}
        </p>
        <ContentStatus status={d.contentStatus} />
      </div>
      <div className={styles.detailGrid}>
        <div className={styles.detailMain}>
          <section className="panel">
            <h2>Topics to explore</h2>
            <div className={styles.topics}>
              {d.topics.map((t) => (
                <article key={t.id}>
                  <h3>{t.name}</h3>
                  <p className="zh" lang="zh-CN">
                    {t.nameZh}
                  </p>
                </article>
              ))}
            </div>
          </section>
          <section className="panel">
            <h2>Learning paths</h2>
            {d.paths.length ? (
              <div className={styles.paths}>
                {d.paths.map((path) => (
                  <Link
                    prefetch={false}
                    className={styles.pathCard}
                    key={path.id}
                    href={"/paths/" + path.id}
                  >
                    <div>
                      <h3>{path.title}</h3>
                      <p className="zh" lang="zh-CN">
                        {path.titleZh}
                      </p>
                      <span className={styles.version}>
                        Version {path.version}
                      </span>
                    </div>
                    <span aria-hidden="true">↗</span>
                  </Link>
                ))}
              </div>
            ) : (
              <div className={styles.inDevelopment}>
                <span aria-hidden="true">◇</span>
                <div>
                  <h3>Learning paths are in development.</h3>
                  <p>Reviewed content will appear here as this domain grows.</p>
                </div>
              </div>
            )}
          </section>
        </div>
        <aside className={styles.detailSide}>
          <section className="panel">
            <p className="eyebrow">CURRENT CONTENT</p>
            <p className={styles.knowledgeCount}>
              {d.publishedKnowledgeCount}
              <span>
                published knowledge{" "}
                {d.publishedKnowledgeCount === 1 ? "point" : "points"}
              </span>
            </p>
            <p className={styles.asideNote}>
              Explore the topics freely. Learning paths connect the ideas when
              reviewed content is available.
            </p>
          </section>
          {d.relatedDomainIds.length > 0 && (
            <section className="panel">
              <h2>Related domains</h2>
              <p className={styles.asideNote}>
                Explore another direction. These links describe connections,
                rather than prerequisites.
              </p>
              <ul className={styles.related}>
                {d.relatedDomainIds.map((id) => (
                  <li key={id}>
                    <Link prefetch={false} href={"/domains/" + id}>
                      {id}
                      <span aria-hidden="true">↗</span>
                    </Link>
                  </li>
                ))}
              </ul>
            </section>
          )}
        </aside>
      </div>
    </>
  );
}
