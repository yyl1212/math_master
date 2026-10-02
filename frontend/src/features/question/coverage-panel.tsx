"use client";
import { useState } from "react";
import { requestQuestion } from "@/lib/question/client";
import type { CoverageReport } from "@/lib/question/types";
import styles from "@/styles/question.module.css";
export function CoveragePanel({ initial }: {
    initial: CoverageReport;
}) {
    const [report, setReport] = useState(initial), [stale, setStale] = useState(false), [busy, setBusy] = useState(false), [error, setError] = useState<string | null>(null);
    async function load(offset: number, refresh = false) { if (busy)
        return; setBusy(true); setError(null); try {
        const r = await requestQuestion<CoverageReport>({ kind: "readCoverage", query: { limit: report.nodes.limit, offset } });
        if (!r.ok) {
            setError(r.message);
            return;
        }
        if (!refresh && (r.data.knowledgeHead !== report.knowledgeHead || r.data.questionHead !== report.questionHead)) {
            setStale(true);
            return;
        }
        setReport(r.data);
        setStale(false);
    }
    finally {
        setBusy(false);
    } }
    return <section className={styles.card}><h2>Published question coverage</h2><p>Counts reflect approved, published questions whose current knowledge, units and illustrations remain available.</p><p className={styles.metadata}>Knowledge head: {report.knowledgeHead ?? "None"}<br />Question head: {report.questionHead ?? "None"}</p>{error && <p role="alert">{error}</p>}{stale ? <p role="status">Snapshots changed. Refresh from the first page before comparing coverage.</p> : <><p>Published trusted instances: {report.effectiveInstances}</p><p>Approved templates: {report.approvedTemplates} · Fixed questions: {report.fixedQuestions} · Published knowledge nodes: {report.publishedKnowledge} · Duplicate occurrences excluded: {report.duplicateInstances}</p><p>Node counts may overlap; global counts use distinct question identities.</p>{report.nodes.items.map((n, i) => <section className={styles.card} key={i}><h3>{n.knowledge.id} v{n.knowledge.version}</h3><p className={styles.badge}>{n.ready ? "Ready for five questions" : "Not ready for assessment"}</p><p>{n.blueprint ? `Blueprint ${n.blueprint.id} v${n.blueprint.version}` : "No assessment blueprint configured."}</p><p>Valid questions: {n.effectiveInstances} · Fixed: {n.fixedInstances} · Generated: {n.generatedInstances} · Blueprint pool: {n.assessmentInstances}</p><p>Core objectives: {n.coreObjectiveIndices.join(", ") || "None"} · Covered: {n.coveredObjectiveIndices.join(", ") || "None"}</p><p>Supplementary objectives: {n.supplementaryObjectiveIndices.join(", ") || "None"}</p>{n.reasons.map((r, j) => <p key={j}>{r.code}: {r.message || r.path}</p>)}</section>)}<p>{report.nodes.total} coverage rows · Offset {report.nodes.offset}</p></>}<div className={styles.actions}><button className="button secondary" disabled={busy} onClick={() => void load(0, true)}>Refresh coverage</button><button className="button secondary" disabled={busy || stale || report.nodes.offset === 0} onClick={() => void load(Math.max(0, report.nodes.offset - report.nodes.limit))}>Previous coverage nodes</button><button className="button secondary" disabled={busy || stale || report.nodes.offset + report.nodes.limit >= report.nodes.total} onClick={() => void load(report.nodes.offset + report.nodes.limit)}>Next coverage nodes</button></div></section>;
}
