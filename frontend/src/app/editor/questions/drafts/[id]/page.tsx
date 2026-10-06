import {getUiMetadata} from "@/lib/i18n/server";
import {UiPageTitle} from "@/components/ui-page-title";
export async function generateMetadata(){return getUiMetadata("page.editor.questions.drafts.id");}
import { questionPageAccess } from "@/features/question/page-access";
import { QuestionState } from "@/features/question/question-state";
import { readServerQuestion } from "@/lib/question/server-client";
export const dynamic = "force-dynamic";
import { DraftEditor } from "@/features/question/draft-editor";
import type { DraftView } from "@/lib/question/types";
import { questionUUID } from "@/lib/question/schemas";

export default async function Page({ params }: {
    params: Promise<{
        id: string;
    }>;
}) { const access = await questionPageAccess(["editor", "admin"]); if ("error" in access)
    return <><UiPageTitle messageKey="page.editor.questions.drafts.id"/>{access.error}</>; const { id } = await params; if (!questionUUID.test(id))
    return <><UiPageTitle messageKey="page.editor.questions.drafts.id"/>{<QuestionState status={404}/>}</>; const r = await readServerQuestion<DraftView>({ kind: "readDraft", id }, access.cookie); if (!r.ok)
    return <><UiPageTitle messageKey="page.editor.questions.drafts.id"/>{<QuestionState status={r.status} code={r.code}/>}</>; return <><UiPageTitle messageKey="page.editor.questions.drafts.id"/>{<DraftEditor initial={r.data} canEdit={access.user.roles.includes("editor") && access.user.id === r.data.ownerId}/>}</>; }
