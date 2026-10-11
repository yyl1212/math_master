import {getUiMetadata} from "@/lib/i18n/server";
import {UiPageTitle} from "@/components/ui-page-title";
export async function generateMetadata(){return getUiMetadata("page.admin.question-withdrawals");}
import { questionPageAccess } from "@/features/question/page-access";
import { QuestionState } from "@/features/question/question-state";
import { readServerQuestion } from "@/lib/question/server-client";
export const dynamic = "force-dynamic";
import { WithdrawalPanel } from "@/features/question/withdrawal-panel";
import type { PublicationPage } from "@/lib/question/types";

export default async function Page() { const access = await questionPageAccess(["admin"]); if ("error" in access)
    return <><UiPageTitle messageKey="page.admin.question-withdrawals"/>{access.error}</>; const r = await readServerQuestion<PublicationPage>({ kind: "listPublications", query: { limit: 1 } }, access.cookie); if (!r.ok)
    return <><UiPageTitle messageKey="page.admin.question-withdrawals"/>{<QuestionState status={r.status} code={r.code}/>}</>; return <><UiPageTitle messageKey="page.admin.question-withdrawals"/>{<WithdrawalPanel initial={r.data}/>}</>; }
