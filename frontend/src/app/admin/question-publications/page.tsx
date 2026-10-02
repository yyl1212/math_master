import { questionPageAccess } from "@/features/question/page-access";
import { QuestionState } from "@/features/question/question-state";
import { readServerQuestion } from "@/lib/question/server-client";
export const dynamic = "force-dynamic";
import { PublicationPanel } from "@/features/question/publication-panel";
import { CoveragePanel } from "@/features/question/coverage-panel";
import { readServerContent } from "@/lib/content/server-client";
import type { PublicationPage as ContentPublicationPage } from "@/lib/content/types";
import type { PublicationPage, CoverageReport } from "@/lib/question/types";
export const metadata = { title: "Trusted question publications" };
export default async function Page() { const access = await questionPageAccess(["admin"]); if ("error" in access)
    return access.error; const [r, k] = await Promise.all([readServerQuestion<PublicationPage>({ kind: "listPublications", query: { limit: 20 } }, access.cookie), readServerContent<ContentPublicationPage>({ kind: "listPublications", query: { limit: 1 } }, access.cookie)]); if (!r.ok)
    return <QuestionState status={r.status} code={r.code}/>; if (!k.ok)
    return <QuestionState status={k.status} code={k.code}/>; const coverage = await readServerQuestion<CoverageReport>({ kind: "readCoverage", query: { limit: 20 } }, access.cookie); return <><PublicationPanel initial={r.data} knowledgeHead={k.data.head}/>{coverage.ok ? <div className="container"><CoveragePanel initial={coverage.data}/></div> : <QuestionState status={coverage.status} code={coverage.code}/>}</>; }
