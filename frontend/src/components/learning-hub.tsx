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
          <p className="eyebrow">CURIOUS MINDS, CONNECTED IDEAS</p>
          <h1>
            A clear path through
            <br />
            <em>mathematics.</em>
          </h1>
          <p>
            Build understanding, one idea at a time. Explore the foundations,
            follow the connections, and find a direction that inspires you.
          </p>
          <Link prefetch={false} href="/knowledge" className="button">
            Explore knowledge <span aria-hidden="true">↗</span>
          </Link>
          <p className={styles.heroNote}>
            From first principles to new frontiers.
          </p>
        </div>
        <div className={styles.heroArt} aria-hidden="true">
          <div className={styles.artLabel}>EVERY IDEA HAS A CONNECTION</div>
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
            <span>Explore.</span>
            <span>Understand.</span>
            <span>Connect.</span>
          </div>
        </div>
      </section>
      <section aria-labelledby="domains-heading">
        <div className="section-heading">
          <div>
            <p className="eyebrow">YOUR STARTING POINT</p>
            <h2 id="domains-heading">{result.data.total} learning domains</h2>
          </div>
          <p>One connected map. Many ways to learn.</p>
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
                <span>
                  {d.publishedKnowledgeCount} published knowledge{" "}
                  {d.publishedKnowledgeCount === 1 ? "point" : "points"}
                </span>
                <span aria-hidden="true">↗</span>
              </div>
            </article>
          ))}
        </div>
      </section>
      <aside className={styles.learningNote}>
        <span aria-hidden="true">✦</span>
        <div>
          <h2>Understanding comes first.</h2>
          <p>
            Learning domains are open to explore. Mathematical content appears
            here after review, with its conditions, sources, and version.
          </p>
        </div>
      </aside>
    </>
  );
}
