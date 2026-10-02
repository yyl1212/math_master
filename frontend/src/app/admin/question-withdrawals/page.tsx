import { questionPageAccess } from "@/features/question/page-access";
import { QuestionState } from "@/features/question/question-state";
import { readServerQuestion } from "@/lib/question/server-client";
export const dynamic = "force-dynamic";
import { WithdrawalPanel } from "@/features/question/withdrawal-panel";
import type { PublicationPage } from "@/lib/question/types";
export const metadata = { title: "Permanent question withdrawal" };
export default async function Page() { const access = await questionPageAccess(["admin"]); if ("error" in access)
    return access.error; const r = await readServerQuestion<PublicationPage>({ kind: "listPublications", query: { limit: 1 } }, access.cookie); if (!r.ok)
    return <QuestionState status={r.status} code={r.code}/>; return <WithdrawalPanel initial={r.data}/>; }
