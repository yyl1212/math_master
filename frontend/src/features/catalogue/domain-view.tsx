"use client";
import {UiText} from "@/lib/i18n/ui-text";
import {uiMessage,uiValue} from "@/lib/i18n/format";
import {useUiI18n} from "@/lib/i18n/provider";
import Link from "next/link";
import type { ApiResult, DomainDetail } from "@/lib/api/types";
import { ContentStatus } from "@/components/content-status";
import { ContentState } from "@/components/content-state";
import type {TopicDetail} from "@/lib/taxonomy/types";
import styles from "@/styles/catalogue.module.css";
export function DomainView({ result }: { result: ApiResult<DomainDetail> }) {
 const {t}=useUiI18n();

  if (!result.ok)
    return (
      <ContentState
        kind={result.kind === "not-found" ? "not-found" : "unavailable"}
      />
    );
  const d = result.data;
  return (
    <>
      <nav className="breadcrumbs" aria-label={t("domain-view.breadcrumb.2bd873",{})}>
        <Link prefetch={false} href="/"><UiText notice={uiMessage("site-header.learning.hub.0af988",{})}/></Link>
        <span aria-hidden="true">/</span>
        <Link prefetch={false} href="/knowledge"><UiText notice={uiMessage("nav.knowledgeMap",{})}/></Link>
        <span aria-hidden="true">/</span>
        <span>{d.name}</span>
      </nav>
      <div className="page-heading">
        <p className="eyebrow"><UiText notice={uiMessage("domain-view.learning.domain.value.8bbbb2",{v0:uiValue(String(d.order).padStart(2, "0"))})}/></p>
        <h1 lang="en">{d.name}</h1>
        <p className="zh" lang="zh-CN">
          {d.nameZh}
        </p>
        <ContentStatus status={d.contentStatus} />
      </div>
      <div className={styles.detailGrid}>
        <div className={styles.detailMain}>
          <section className="panel">
            <h2><UiText notice={uiMessage("domain-view.topics.to.explore.773408",{})}/></h2>
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
            <h2><UiText notice={uiMessage("domain-view.learning.paths.5f23ab",{})}/></h2>
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
                      <span className={styles.version}><UiText notice={uiMessage("domain-view.version.value.d2b5e7",{v0:uiValue(path.version)})}/></span>
                    </div>
                    <span aria-hidden="true">↗</span>
                  </Link>
                ))}
              </div>
            ) : (
              <div className={styles.inDevelopment}>
                <span aria-hidden="true">◇</span>
                <div>
                  <h3><UiText notice={uiMessage("domain-view.learning.paths.are.in.development.a67b76",{})}/></h3>
                  <p><UiText notice={uiMessage("domain-view.reviewed.content.will.appear.here.as.this.domain.grows.ad70f5",{})}/></p>
                </div>
              </div>
            )}
          </section>
        </div>
        <aside className={styles.detailSide}>
          <section className="panel">
            <p className="eyebrow"><UiText notice={uiMessage("domain-view.current.content.f491db",{})}/></p>
            <p className={styles.knowledgeCount}>
              <UiText notice={uiMessage(d.publishedKnowledgeCount===1?"public.publishedOne":"public.publishedMany",{count:d.publishedKnowledgeCount})}/>
            </p>
            <p className={styles.asideNote}><UiText notice={uiMessage("domain-view.explore.the.topics.freely.learning.paths.connect.the.ideas.when.r.7e8cfb",{})}/></p>
          </section>
          {d.relatedDomainIds.length > 0 && (
            <section className="panel">
              <h2><UiText notice={uiMessage("domain-view.related.domains.97e705",{})}/></h2>
              <p className={styles.asideNote}><UiText notice={uiMessage("domain-view.explore.another.direction.these.links.describe.connections.rather.a7437c",{})}/></p>
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

export function TopicDomainView({details}:{details:TopicDetail[]}){const {t,locale}=useUiI18n();return <><div className="page-heading"><h1>{t("topic.map.explore",{})}</h1><p>{t("topic.alias.note",{})}</p></div><div className="domain-grid">{details.map(({summary:n})=><article className="domain-card" key={n.id}><code>{n.code}</code><h2><Link prefetch={false} href={"/topics/"+n.id}>{locale==="zh-CN"&&n.nameZh?n.nameZh:n.name}</Link></h2><p>{t(n.publishedKnowledgeCount===1?"public.publishedOne":"public.publishedMany",{count:n.publishedKnowledgeCount})}</p></article>)}</div><Link prefetch={false} href="/knowledge">{t("nav.knowledgeMap",{})}</Link></>}
