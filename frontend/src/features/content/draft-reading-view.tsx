"use client";

import Link from "next/link";
import { useState } from "react";
import type { DraftView, Knowledge } from "@/lib/content/types";
import { ContentPreview } from "./content-preview";
import styles from "@/styles/content.module.css";

function matchesSearch(point: Knowledge, term: string) {
    return [point.title, point.titleZh, point.id].some(text => text.toLowerCase().includes(term));
}

function knowledgeKey(point: Knowledge) {
    return point.id + ":" + point.version;
}

export function DraftReadingView({ initial }: { initial: DraftView }) {
    const [query, setQuery] = useState("");
    const [selectedKey, setSelectedKey] = useState(() => initial.package.knowledge[0] ? knowledgeKey(initial.package.knowledge[0]) : "");
    const term = query.trim().toLowerCase();
    const points = initial.package.knowledge.filter(point => matchesSearch(point, term));
    const selected = points.find(point => knowledgeKey(point) === selectedKey) ?? points[0];
    const selectedIndex = selected ? points.indexOf(selected) : -1;
    const reference = [
        "Draft: " + initial.id,
        "Saved revision: " + initial.revision,
        "Package: " + initial.package.id + " v" + initial.package.version,
        ...(selected ? ["Knowledge: " + selected.id + " v" + selected.version] : []),
    ].join("\n");

    function search(value: string) {
        setQuery(value);
        const nextTerm = value.trim().toLowerCase();
        const matches = initial.package.knowledge.filter(point => matchesSearch(point, nextTerm));
        setSelectedKey(matches.find(point => knowledgeKey(point) === selectedKey) ? selectedKey : matches[0] ? knowledgeKey(matches[0]) : "");
    }

    return <section className={styles.workbench}>
        <header>
            <p className="eyebrow">PRIVATE DRAFT PREVIEW</p>
            <h1>Read knowledge draft</h1>
            <p className={styles.metadata}>{initial.package.id} · Package version {initial.package.version}</p>
            <p>Saved revision {initial.revision} · Status: {initial.status}</p>
            <p className={styles.previewNotice}>Read-only preview of the saved revision. Unsaved edits are not included. This preview does not establish mathematical approval or publication, and does not record learning progress.</p>
            <div className={styles.actions}>
                <Link prefetch={false} className="button secondary" href={"/editor/drafts/" + initial.id}>Back to workspace</Link>
                <Link prefetch={false} href="/editor">All workspaces</Link>
            </div>
        </header>
        <div className={styles.readingLayout}>
            <aside className={styles.readingNav + " " + styles.card}>
                <h2>Browse knowledge</h2>
                <label className={styles.field}>Search knowledge points
                    <input type="search" value={query} onChange={event => search(event.target.value)} placeholder="English / 中文 / ID"/>
                </label>
                <p role="status">Showing {points.length} of {initial.package.knowledge.length} knowledge points.</p>
                <nav aria-label="Knowledge points">
                    <ul className={styles.knowledgeList}>
                        {points.map(point => <li key={knowledgeKey(point)}>
                            <button type="button" aria-pressed={point === selected} aria-controls="draft-knowledge-content" onClick={() => setSelectedKey(knowledgeKey(point))}>
                                <strong>{point.title || point.id}</strong>
                                {point.titleZh && <span lang="zh">{point.titleZh}</span>}
                                <small>{point.id} · v{point.version} · {point.type}</small>
                            </button>
                        </li>)}
                    </ul>
                </nav>
                {!initial.package.knowledge.length ? <p>No knowledge points in this saved draft.</p> : !points.length && <p>No matching knowledge points. Try a different title or ID.</p>}
            </aside>
            <div className={styles.readingContent}>
                <div className={styles.actions}>
                    <button type="button" className="button secondary" aria-controls="draft-knowledge-content" disabled={selectedIndex <= 0} onClick={() => setSelectedKey(knowledgeKey(points[selectedIndex - 1]))}>Previous knowledge point</button>
                    <button type="button" className="button secondary" aria-controls="draft-knowledge-content" disabled={selectedIndex < 0 || selectedIndex >= points.length - 1} onClick={() => setSelectedKey(knowledgeKey(points[selectedIndex + 1]))}>Next knowledge point</button>
                    {selected && <span>{selectedIndex + 1} / {points.length}</span>}
                </div>
                <div id="draft-knowledge-content" aria-live="polite">
                    {selected && <ContentPreview key={knowledgeKey(selected)} value={{ ...initial.package, knowledge: [selected] }} assets={initial.assets} assetScope={{ kind: "draft", id: initial.id }}/>}
                </div>
                <section className={styles.card}>
                    <h2>Feedback on this preview</h2>
                    <p>Copy this reference into your site feedback. Site feedback does not replace independent mathematical review.</p>
                    <label className={styles.field}>Preview reference<textarea readOnly rows={4} value={reference}/></label>
                    <Link prefetch={false} className="button secondary" href="/feedback/new?kind=site&area=other">Give site feedback</Link>
                </section>
            </div>
        </div>
    </section>;
}
