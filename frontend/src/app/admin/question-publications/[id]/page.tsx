import {getUiMetadata} from "@/lib/i18n/server";
import {UiPageTitle} from "@/components/ui-page-title";
export async function generateMetadata(){return getUiMetadata("page.admin.question-publications.id");}
import { questionPageAccess } from "@/features/question/page-access";
import { QuestionState } from "@/features/question/question-state";
import { readServerQuestion } from "@/lib/question/server-client";
export const dynamic = "force-dynamic";
import { PublicationPanel } from "@/features/question/publication-panel";
import { readServerContent } from "@/lib/content/server-client";
import type { PublicationPage as ContentPublicationPage } from "@/lib/content/types";
import type { PublicationPage, PublicationSummary } from "@/lib/question/types";
import { questionUUID } from "@/lib/question/schemas";

export default async function Page({ params }: {
    params: Promise<{
        id: string;
    }>;
}) { const access = await questionPageAccess(["admin"]); if ("error" in access)
    return <><UiPageTitle messageKey="page.admin.question-publications.id"/>{access.error}</>; const { id } = await params; if (!questionUUID.test(id))
    return <><UiPageTitle messageKey="page.admin.question-publications.id"/>{<QuestionState status={404}/>}</>; const [selected, r, k] = await Promise.all([readServerQuestion<PublicationSummary>({ kind: "readPublication", id }, access.cookie), readServerQuestion<PublicationPage>({ kind: "listPublications", query: { limit: 20 } }, access.cookie), readServerContent<ContentPublicationPage>({ kind: "listPublications", query: { limit: 1 } }, access.cookie)]); if (!selected.ok)
    return <><UiPageTitle messageKey="page.admin.question-publications.id"/>{<QuestionState status={selected.status} code={selected.code}/>}</>; if (!r.ok)
    return <><UiPageTitle messageKey="page.admin.question-publications.id"/>{<QuestionState status={r.status} code={r.code}/>}</>; if (!k.ok)
    return <><UiPageTitle messageKey="page.admin.question-publications.id"/>{<QuestionState status={k.status} code={k.code}/>}</>; return <><UiPageTitle messageKey="page.admin.question-publications.id"/>{<PublicationPanel initial={r.data} knowledgeHead={k.data.head} selectedPublication={selected.data}/>}</>; }
