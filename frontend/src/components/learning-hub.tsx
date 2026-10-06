"use client";
import {UiText} from "@/lib/i18n/ui-text";
import {uiMessage,uiValue} from "@/lib/i18n/format";
import Link from "next/link";
import type { ApiResult, DomainList } from "@/lib/api/types";
import { ContentState } from "./content-state";
import { ContentStatus } from "./content-status";
import styles from "@/styles/layout.module.css";
export function LearningHub({ result }: { result: ApiResult<DomainList> }) {
  if (!result.ok)
    return (
      <ContentState
        kind={result.kind === "not-found" ? "not-found" : "unavailable"}
      />
    );
  if (!result.data.items.length) return <ContentState kind="empty" />;
  const domains = [...result.data.items].sort((a, b) => a.order - b.order);
  return (
    <>
      <section className={styles.hero}>
        <div className={styles.heroCopy}>
          <p className="eyebrow"><UiText notice={uiMessage("learning-hub.curious.minds.connected.ideas.55351d",{})}/></p>
          <h1><UiText notice={uiMessage("learning-hub.a.clear.path.through.983c9a",{})}/><br />
            <em><UiText notice={uiMessage("learning-hub.mathematics.4d3cd6",{})}/></em>
          </h1>
          <p><UiText notice={uiMessage("learning-hub.build.understanding.one.idea.at.a.time.explore.the.foundations.fo.88ab73",{})}/></p>
          <Link prefetch={false} href="/knowledge" className="button"><UiText notice={uiMessage("learning-hub.explore.knowledge.dfe344",{})}/><span aria-hidden="true">↗</span>
          </Link>
          <p><Link prefetch={false} href="/learn"><UiText notice={uiMessage("learning-hub.continue.your.learning.f95b7e",{})}/></Link></p>
          <p className={styles.heroNote}><UiText notice={uiMessage("learning-hub.from.first.principles.to.new.frontiers.15da58",{})}/></p>
        </div>
        <div className={styles.heroArt} aria-hidden="true">
          <div className={styles.artLabel}><UiText notice={uiMessage("learning-hub.every.idea.has.a.connection.69c99c",{})}/></div>
          <svg viewBox="0 0 460 340">
            <defs>
              <pattern
                id="idea-grid"
                width="24"
                height="24"
                patternUnits="userSpaceOnUse"
              >
                <circle cx="1" cy="1" r="1" fill="#cfdbc9" />
              </pattern>
            </defs>
            <rect width="460" height="340" fill="url(#idea-grid)" />
            <circle cx="235" cy="170" r="120" fill="none" stroke="#c1cfbc" />
            <circle cx="235" cy="170" r="83" fill="none" stroke="#d2decd" />
            <path
              d="M82 225L171 95L278 246L379 113M171 95L379 113M82 225L278 246"
              fill="none"
              stroke="#82967b"
              strokeWidth="1.5"
            />
            <path d="M235 72V270M135 170H335" stroke="#c1cfbc" />
            <circle cx="82" cy="225" r="25" fill="#e7dcca" />
            <circle cx="171" cy="95" r="34" fill="#315844" />
            <circle cx="278" cy="246" r="30" fill="#9fb69a" />
            <circle cx="379" cy="113" r="22" fill="#d5e1ce" />
            <text
              x="171"
              y="106"
              textAnchor="middle"
              fontSize="31"
              fill="white"
              fontFamily="Georgia,serif"
            >
              π
            </text>
            <text
              x="278"
              y="255"
              textAnchor="middle"
              fontSize="27"
              fill="#254a36"
              fontFamily="Georgia,serif"
            >
              ∞
            </text>
            <text
              x="82"
              y="233"
              textAnchor="middle"
              fontSize="24"
              fill="#705d46"
              fontFamily="Georgia,serif"
            >
              Σ
            </text>
          </svg>
          <div className={styles.artFooter}>
            <span><UiText notice={uiMessage("learning-hub.explore.e6e7d0",{})}/></span>
            <span><UiText notice={uiMessage("learning-hub.understand.57e3c1",{})}/></span>
            <span><UiText notice={uiMessage("learning-hub.connect.23c10a",{})}/></span>
          </div>
        </div>
      </section>
      <section aria-labelledby="domains-heading">
        <div className="section-heading">
          <div>
            <p className="eyebrow"><UiText notice={uiMessage("learning-hub.your.starting.point.7ae59f",{})}/></p>
            <h2 id="domains-heading"><UiText notice={uiMessage("learning-hub.value.learning.domains.b173e5",{v0:uiValue(result.data.total)})}/></h2>
          </div>
          <p><UiText notice={uiMessage("learning-hub.one.connected.map.many.ways.to.learn.e5a2a7",{})}/></p>
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
              <h3>
                <Link prefetch={false} href={"/domains/" + d.id}>
                  {d.name}
                </Link>
              </h3>
              <p className="zh" lang="zh-CN">
                {d.nameZh}
              </p>
              <p className="topic-summary">
                {d.topics.map((t) => t.name).join(" · ")}
              </p>
              <div className="card-bottom">
                <span><UiText notice={uiMessage(d.publishedKnowledgeCount===1?"public.publishedOne":"public.publishedMany",{count:d.publishedKnowledgeCount})}/></span>
                <span aria-hidden="true">↗</span>
              </div>
            </article>
          ))}
        </div>
      </section>
      <aside className={styles.learningNote}>
        <span aria-hidden="true">✦</span>
        <div>
          <h2><UiText notice={uiMessage("learning-hub.understanding.comes.first.fc29d3",{})}/></h2>
          <p><UiText notice={uiMessage("learning-hub.learning.domains.are.open.to.explore.mathematical.content.appears.a8c425",{})}/></p>
        </div>
      </aside>
    </>
  );
}
