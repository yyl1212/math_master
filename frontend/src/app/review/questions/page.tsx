import {getUiMetadata} from "@/lib/i18n/server";
import {UiPageTitle} from "@/components/ui-page-title";
export async function generateMetadata(){return getUiMetadata("page.review.questions");}
import { questionPageAccess } from "@/features/question/page-access";
import { QuestionState } from "@/features/question/question-state";
import { readServerQuestion } from "@/lib/question/server-client";
export const dynamic = "force-dynamic";
import { SubmissionList } from "@/features/question/submission-list";
import { CoveragePanel } from "@/features/question/coverage-panel";
import type { SubmissionPage, CoverageReport } from "@/lib/question/types";

export default async function Page() { const access = await questionPageAccess(["editor", "reviewer", "admin"]); if ("error" in access)
    return <><UiPageTitle messageKey="page.review.questions"/>{access.error}</>; const scope = access.user.roles.includes("admin") ? "all" : access.user.roles.includes("reviewer") ? "review" : "mine", r = await readServerQuestion<SubmissionPage>({ kind: "listSubmissions", query: { scope, status: "pending", limit: 20 } }, access.cookie); if (!r.ok)
    return <><UiPageTitle messageKey="page.review.questions"/>{<QuestionState status={r.status} code={r.code}/>}</>; const coverage = access.user.roles.some(role => role === "reviewer" || role === "admin") ? await readServerQuestion<CoverageReport>({ kind: "readCoverage", query: { limit: 20 } }, access.cookie) : null; return <><UiPageTitle messageKey="page.review.questions"/>{<><SubmissionList initial={r.data} scope={scope}/>{coverage && (coverage.ok ? <div className="container"><CoveragePanel initial={coverage.data}/></div> : <QuestionState status={coverage.status} code={coverage.code}/>)}</>}</>; }
