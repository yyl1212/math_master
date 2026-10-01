import Link from "next/link";
import Form from "next/form";
import type { ApiResult, DomainList } from "@/lib/api/types";
import { ContentState } from "@/components/content-state";
import { ContentStatus } from "@/components/content-status";
import type { CatalogueStatus } from "./query";
import styles from "@/styles/catalogue.module.css";
export function KnowledgeMap({
  result,
  q,
  status,
}: {
  result: ApiResult<DomainList>;
  q: string;
  status: CatalogueStatus;
}) {
  const domains = result.ok
    ? result.data.items
        .filter((d) => status === "all" || d.contentStatus === status)
        .sort((a, b) => a.order - b.order)
    : [];
  return (
    <>
      <div className="page-heading">
        <p className="eyebrow">A CONNECTED WORLD OF MATHEMATICS</p>
        <h1>Knowledge Map</h1>
        <p>Choose a domain. Explore its ideas, then follow a learning path.</p>
      </div>
      <Form
        role="search"
        action="/knowledge"
        prefetch={false}
        key={q + "\0" + status}
        className={styles.toolbar}
      >
        <div className={styles.search}>
          <label htmlFor="domain-search">Search learning domains</label>
          <div className={styles.searchInput}>
            <span aria-hidden="true">⌕</span>
            <input
              id="domain-search"
              type="search"
              name="q"
              defaultValue={q}
              placeholder="Search English or Chinese topics"
              maxLength={512}
            />
          </div>
        </div>
        <div className={styles.filter}>
          <label htmlFor="domain-status">Content status</label>
          <select id="domain-status" name="status" defaultValue={status}>
            <option value="all">All domains</option>
            <option value="planned">In development</option>
            <option value="published">Published</option>
          </select>
        </div>
        <button className="button" type="submit">
          Search
        </button>
      </Form>
      {!result.ok ? (
        <ContentState
          kind={result.kind === "not-found" ? "not-found" : "unavailable"}
        />
      ) : result.data.total === 0 && !q && status === "all" ? (
        <ContentState kind="empty" />
      ) : !domains.length ? (
        <ContentState kind="no-results" />
      ) : (
        <>
          <div className={styles.results}>
            <span>
              {domains.length} {domains.length === 1 ? "domain" : "domains"}
            </span>
            <span>Groups for learning, with room for connections.</span>
          </div>
          <div className="domain-grid">
            {domains.map((d) => (
              <article className="domain-card" key={d.id}>
                <div className="card-top">
                  <span className="domain-number">
                    {String(d.order).padStart(2, "0")}
                  </span>
                  <ContentStatus status={d.contentStatus} />
                </div>
                <h2 className={styles.cardTitle}>
                  <Link prefetch={false} href={"/domains/" + d.id}>
                    {d.name}
                  </Link>
                </h2>
                <p className="zh" lang="zh-CN">
                  {d.nameZh}
                </p>
                <ul className={styles.topicList}>
                  {d.topics.map((t) => (
                    <li key={t.id}>
                      <span>{t.name}</span>
                      <small lang="zh-CN">{t.nameZh}</small>
                    </li>
                  ))}
                </ul>
                <div className="card-bottom">
                  <span>
                    {d.publishedKnowledgeCount} published knowledge{" "}
                    {d.publishedKnowledgeCount === 1 ? "point" : "points"}
                  </span>
                  <span aria-hidden="true">↗</span>
                </div>
              </article>
            ))}
          </div>
        </>
      )}
      <p className={styles.mapNote}>
        These domains are practical learning groups. Connections across domains
        are part of the journey.
      </p>
    </>
  );
}
