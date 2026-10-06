"use client";
import {UiText} from "@/lib/i18n/ui-text";
import {uiMessage,uiValue} from "@/lib/i18n/format";
import {useUiI18n} from "@/lib/i18n/provider";
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
  status,personal,
}: {
  result: ApiResult<DomainList>;
  q: string;
  status: CatalogueStatus;personal?:React.ReactNode;
}) {
 const {t}=useUiI18n();

  const domains = result.ok
    ? result.data.items
        .filter((d) => status === "all" || d.contentStatus === status)
        .sort((a, b) => a.order - b.order)
    : [];
  return (
    <>
      <div className="page-heading">
        <p className="eyebrow"><UiText notice={uiMessage("knowledge-map.a.connected.world.of.mathematics.4cd45c",{})}/></p>
        <h1><UiText notice={uiMessage("nav.knowledgeMap",{})}/></h1>
        <p><UiText notice={uiMessage("knowledge-map.choose.a.domain.explore.its.ideas.then.follow.a.learning.path.529a69",{})}/></p>
      </div>
      {personal}
      <Form
        role="search"
        action="/knowledge"
        prefetch={false}
        key={q + "\0" + status}
        className={styles.toolbar}
      >
        <div className={styles.search}>
          <label htmlFor="domain-search"><UiText notice={uiMessage("knowledge-map.search.learning.domains.13c880",{})}/></label>
          <div className={styles.searchInput}>
            <span aria-hidden="true">⌕</span>
            <input
              id="domain-search"
              type="search"
              name="q"
              defaultValue={q}
              placeholder={t("knowledge-map.search.english.or.chinese.topics.c67f21",{})}
              maxLength={512}
            />
          </div>
        </div>
        <div className={styles.filter}>
          <label htmlFor="domain-status"><UiText notice={uiMessage("knowledge-map.content.status.5efa57",{})}/></label>
          <select id="domain-status" name="status" defaultValue={status}>
            <option value="all"><UiText notice={uiMessage("knowledge-map.all.domains.0f97da",{})}/></option>
            <option value="planned"><UiText notice={uiMessage("knowledge-map.in.development.5259ae",{})}/></option>
            <option value="published"><UiText notice={uiMessage("knowledge-map.published.2ef42e",{})}/></option>
          </select>
        </div>
        <button className="button" type="submit"><UiText notice={uiMessage("knowledge-map.search.49c266",{})}/></button>
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
              <UiText notice={uiMessage(domains.length===1?"public.domainOne":"public.domainMany",{count:domains.length})}/>
            </span>
            <span><UiText notice={uiMessage("knowledge-map.groups.for.learning.with.room.for.connections.8ec81a",{})}/></span>
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
                  <span><UiText notice={uiMessage(d.publishedKnowledgeCount===1?"public.publishedOne":"public.publishedMany",{count:d.publishedKnowledgeCount})}/></span>
                  <span aria-hidden="true">↗</span>
                </div>
              </article>
            ))}
          </div>
        </>
      )}
      <p className={styles.mapNote}><UiText notice={uiMessage("knowledge-map.these.domains.are.practical.learning.groups.connections.across.do.a8e178",{})}/></p>
    </>
  );
}
