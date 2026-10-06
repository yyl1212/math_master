import {getUiMetadata} from "@/lib/i18n/server";
import {UiPageTitle} from "@/components/ui-page-title";
export async function generateMetadata(){return getUiMetadata("page.editor.questions");}
import { questionPageAccess } from "@/features/question/page-access";
import { QuestionState } from "@/features/question/question-state";
import { readServerQuestion } from "@/lib/question/server-client";
export const dynamic = "force-dynamic";
import { DraftList } from "@/features/question/draft-list";
import type { DraftPage } from "@/lib/question/types";

export default async function Page() { const access = await questionPageAccess(["editor", "admin"]); if ("error" in access)
    return <><UiPageTitle messageKey="page.editor.questions"/>{access.error}</>; const canEdit = access.user.roles.includes("editor"), r = await readServerQuestion<DraftPage>({ kind: "listDrafts", query: { scope: canEdit ? "mine" : "all", limit: 20 } }, access.cookie); if (!r.ok)
    return <><UiPageTitle messageKey="page.editor.questions"/>{<QuestionState status={r.status} code={r.code}/>}</>; return <><UiPageTitle messageKey="page.editor.questions"/>{<DraftList initial={r.data} canEdit={canEdit}/>}</>; }
