"use client";
import {UiText} from "@/lib/i18n/ui-text";
import {uiMessage,uiValue} from "@/lib/i18n/format";
import {useUiI18n} from "@/lib/i18n/provider";

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
 const {t}=useUiI18n();

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
            <p className="eyebrow"><UiText notice={uiMessage("draft-reading-view.private.draft.preview.9b4ed0",{})}/></p>
            <h1><UiText notice={uiMessage("page.editor.drafts.id.preview",{})}/></h1>
            <p className={styles.metadata}><UiText notice={uiMessage("draft-reading-view.value.package.version.value.7321bb",{v0:uiValue(initial.package.id),v1:uiValue(initial.package.version)})}/></p>
            <p><UiText notice={uiMessage("draft-reading-view.saved.revision.value.status.value.c0ecaa",{v0:uiValue(initial.revision),v1:uiValue(initial.status)})}/></p>
            <p className={styles.previewNotice}><UiText notice={uiMessage("draft-reading-view.read.only.preview.of.the.saved.revision.unsaved.edits.are.not.inc.a4ec9b",{})}/></p>
            <div className={styles.actions}>
                <Link prefetch={false} className="button secondary" href={"/editor/drafts/" + initial.id}><UiText notice={uiMessage("draft-reading-view.back.to.workspace.0f0c0c",{})}/></Link>
                <Link prefetch={false} href="/editor"><UiText notice={uiMessage("draft-reading-view.all.workspaces.415c8e",{})}/></Link>
            </div>
        </header>
        <div className={styles.readingLayout}>
            <aside className={styles.readingNav + " " + styles.card}>
                <h2><UiText notice={uiMessage("draft-reading-view.browse.knowledge.7bae8a",{})}/></h2>
                <label className={styles.field}><UiText notice={uiMessage("draft-reading-view.search.knowledge.points.aa0709",{})}/><input type="search" value={query} onChange={event => search(event.target.value)} placeholder={t("draft-reading-view.english.id.caaeb8",{})}/>
                </label>
                <p role="status"><UiText notice={uiMessage("draft-reading-view.showing.value.of.value.knowledge.points.bcc8df",{v0:uiValue(points.length),v1:uiValue(initial.package.knowledge.length)})}/></p>
                <nav aria-label={t("draft-reading-view.knowledge.points.da18c7",{})}>
                    <ul className={styles.knowledgeList}>
                        {points.map(point => <li key={knowledgeKey(point)}>
                            <button type="button" aria-pressed={point === selected} aria-controls="draft-knowledge-content" onClick={() => setSelectedKey(knowledgeKey(point))}>
                                <strong>{point.title || point.id}</strong>
                                {point.titleZh && <span lang="zh">{point.titleZh}</span>}
                                <small><UiText notice={uiMessage("draft-reading-view.value.vvalue.value.a961d3",{v0:uiValue(point.id),v1:uiValue(point.version),v2:uiValue(point.type)})}/></small>
                            </button>
                        </li>)}
                    </ul>
                </nav>
                {!initial.package.knowledge.length ? <p><UiText notice={uiMessage("draft-reading-view.no.knowledge.points.in.this.saved.draft.982d22",{})}/></p> : !points.length && <p><UiText notice={uiMessage("draft-reading-view.no.matching.knowledge.points.try.a.different.title.or.id.81c59a",{})}/></p>}
            </aside>
            <div className={styles.readingContent}>
                <div className={styles.actions}>
                    <button type="button" className="button secondary" aria-controls="draft-knowledge-content" disabled={selectedIndex <= 0} onClick={() => setSelectedKey(knowledgeKey(points[selectedIndex - 1]))}><UiText notice={uiMessage("draft-reading-view.previous.knowledge.point.30303e",{})}/></button>
                    <button type="button" className="button secondary" aria-controls="draft-knowledge-content" disabled={selectedIndex < 0 || selectedIndex >= points.length - 1} onClick={() => setSelectedKey(knowledgeKey(points[selectedIndex + 1]))}><UiText notice={uiMessage("draft-reading-view.next.knowledge.point.9a368f",{})}/></button>
                    {selected && <span>{selectedIndex + 1} / {points.length}</span>}
                </div>
                <div id="draft-knowledge-content" aria-live="polite">
                    {selected && <ContentPreview key={knowledgeKey(selected)} value={{ ...initial.package, knowledge: [selected] }} assets={initial.assets} assetScope={{ kind: "draft", id: initial.id }}/>}
                </div>
                <section className={styles.card}>
                    <h2><UiText notice={uiMessage("draft-reading-view.feedback.on.this.preview.124f01",{})}/></h2>
                    <p><UiText notice={uiMessage("draft-reading-view.copy.this.reference.into.your.site.feedback.site.feedback.does.no.2e75aa",{})}/></p>
                    <label className={styles.field}><UiText notice={uiMessage("draft-reading-view.preview.reference.0fdc50",{})}/><textarea readOnly rows={4} value={reference}/></label>
                    <Link prefetch={false} className="button secondary" href="/feedback/new?kind=site&area=other"><UiText notice={uiMessage("draft-reading-view.give.site.feedback.26c82b",{})}/></Link>
                </section>
            </div>
        </div>
    </section>;
}
