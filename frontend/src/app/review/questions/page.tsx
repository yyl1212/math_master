import { questionPageAccess } from "@/features/question/page-access";
import { QuestionState } from "@/features/question/question-state";
import { readServerQuestion } from "@/lib/question/server-client";
export const dynamic = "force-dynamic";
import { SubmissionList } from "@/features/question/submission-list";
import { CoveragePanel } from "@/features/question/coverage-panel";
import type { SubmissionPage, CoverageReport } from "@/lib/question/types";
export const metadata = { title: "Independent question review" };
export default async function Page() { const access = await questionPageAccess(["editor", "reviewer", "admin"]); if ("error" in access)
    return access.error; const scope = access.user.roles.includes("admin") ? "all" : access.user.roles.includes("reviewer") ? "review" : "mine", r = await readServerQuestion<SubmissionPage>({ kind: "listSubmissions", query: { scope, status: "pending", limit: 20 } }, access.cookie); if (!r.ok)
    return <QuestionState status={r.status} code={r.code}/>; const coverage = access.user.roles.some(role => role === "reviewer" || role === "admin") ? await readServerQuestion<CoverageReport>({ kind: "readCoverage", query: { limit: 20 } }, access.cookie) : null; return <><SubmissionList initial={r.data} scope={scope}/>{coverage && (coverage.ok ? <div className="container"><CoveragePanel initial={coverage.data}/></div> : <QuestionState status={coverage.status} code={coverage.code}/>)}</>; }
