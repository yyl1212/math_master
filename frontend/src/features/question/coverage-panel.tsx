"use client";
import {uiError} from "@/lib/i18n/errors";
import type {UiNotice} from "@/lib/i18n/types";
import {UiText} from "@/lib/i18n/ui-text";
import {uiMessage,uiValue} from "@/lib/i18n/format";
import {useUiI18n} from "@/lib/i18n/provider";
import { useState } from "react";
import { requestQuestion } from "@/lib/question/client";
import type { CoverageReport } from "@/lib/question/types";
import styles from "@/styles/question.module.css";
export function CoveragePanel({ initial }: {
    initial: CoverageReport;
}) {
 const {t}=useUiI18n();

    const [report, setReport] = useState(initial), [stale, setStale] = useState(false), [busy, setBusy] = useState(false), [error, setError] = useState<UiNotice | null>(null);
    async function load(offset: number, refresh = false) { if (busy)
        return; setBusy(true); setError(null); try {
        const r = await requestQuestion<CoverageReport>({ kind: "readCoverage", query: { limit: report.nodes.limit, offset } });
        if (!r.ok) {
            setError(uiError("question",r));
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
    return <section className={styles.card}><h2><UiText notice={uiMessage("coverage-panel.published.question.coverage.567402",{})}/></h2><p><UiText notice={uiMessage("coverage-panel.counts.reflect.approved.published.questions.whose.current.knowled.630f4d",{})}/></p><p className={styles.metadata}><UiText notice={uiMessage("coverage-panel.knowledge.head.b7020e",{})}/>{report.knowledgeHead ?? "None"}<br /><UiText notice={uiMessage("coverage-panel.question.head.6001f1",{})}/>{report.questionHead ?? "None"}</p>{error && <p role="alert"><UiText notice={error}/></p>}{stale ? <p role="status"><UiText notice={uiMessage("coverage-panel.snapshots.changed.refresh.from.the.first.page.before.comparing.co.a168f3",{})}/></p> : <><p><UiText notice={uiMessage("coverage-panel.published.trusted.instances.value.0943fe",{v0:uiValue(report.effectiveInstances)})}/></p><p><UiText notice={uiMessage("coverage-panel.approved.templates.value.fixed.questions.value.published.knowledg.47b5a4",{v0:uiValue(report.approvedTemplates),v1:uiValue(report.fixedQuestions),v2:uiValue(report.publishedKnowledge),v3:uiValue(report.duplicateInstances)})}/></p><p><UiText notice={uiMessage("coverage-panel.node.counts.may.overlap.global.counts.use.distinct.question.ident.746aef",{})}/></p>{report.nodes.items.map((n, i) => <section className={styles.card} key={i}><h3><UiText notice={uiMessage("content-preview.value.vvalue.bde90b",{v0:uiValue(n.knowledge.id),v1:uiValue(n.knowledge.version)})}/></h3><p className={styles.badge}>{n.ready ? t("coverage-panel.ready.for.five.questions.16340b",{}) : t("coverage-panel.not.ready.for.assessment.386442",{})}</p><p>{n.blueprint ? `Blueprint ${n.blueprint.id} v${n.blueprint.version}` : t("coverage-panel.no.assessment.blueprint.configured.723360",{})}</p><p><UiText notice={uiMessage("coverage-panel.valid.questions.value.fixed.value.generated.value.blueprint.pool..dd1cd1",{v0:uiValue(n.effectiveInstances),v1:uiValue(n.fixedInstances),v2:uiValue(n.generatedInstances),v3:uiValue(n.assessmentInstances)})}/></p><p><UiText notice={uiMessage("coverage-panel.core.objectives.value.covered.value.e199da",{v0:uiValue(n.coreObjectiveIndices.join(", ") || "None"),v1:uiValue(n.coveredObjectiveIndices.join(", ") || "None")})}/></p><p><UiText notice={uiMessage("coverage-panel.supplementary.objectives.value.3fa75c",{v0:uiValue(n.supplementaryObjectiveIndices.join(", ") || "None")})}/></p>{n.reasons.map((r, j) => <p key={j}>{r.code}: {r.message || r.path}</p>)}</section>)}<p><UiText notice={uiMessage("coverage-panel.value.coverage.rows.offset.value.d50be7",{v0:uiValue(report.nodes.total),v1:uiValue(report.nodes.offset)})}/></p></>}<div className={styles.actions}><button className="button secondary" disabled={busy} onClick={() => void load(0, true)}><UiText notice={uiMessage("coverage-panel.refresh.coverage.7fbd1b",{})}/></button><button className="button secondary" disabled={busy || stale || report.nodes.offset === 0} onClick={() => void load(Math.max(0, report.nodes.offset - report.nodes.limit))}><UiText notice={uiMessage("coverage-panel.previous.coverage.nodes.67c2fb",{})}/></button><button className="button secondary" disabled={busy || stale || report.nodes.offset + report.nodes.limit >= report.nodes.total} onClick={() => void load(report.nodes.offset + report.nodes.limit)}><UiText notice={uiMessage("coverage-panel.next.coverage.nodes.dff886",{})}/></button></div></section>;
}
