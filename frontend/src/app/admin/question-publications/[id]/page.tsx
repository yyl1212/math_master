import { questionPageAccess } from "@/features/question/page-access";
import { QuestionState } from "@/features/question/question-state";
import { readServerQuestion } from "@/lib/question/server-client";
export const dynamic = "force-dynamic";
import { PublicationPanel } from "@/features/question/publication-panel";
import { readServerContent } from "@/lib/content/server-client";
import type { PublicationPage as ContentPublicationPage } from "@/lib/content/types";
import type { PublicationPage, PublicationSummary } from "@/lib/question/types";
import { questionUUID } from "@/lib/question/schemas";
export const metadata = { title: "Fixed question snapshot" };
export default async function Page({ params }: {
    params: Promise<{
        id: string;
    }>;
}) { const access = await questionPageAccess(["admin"]); if ("error" in access)
    return access.error; const { id } = await params; if (!questionUUID.test(id))
    return <QuestionState status={404}/>; const [selected, r, k] = await Promise.all([readServerQuestion<PublicationSummary>({ kind: "readPublication", id }, access.cookie), readServerQuestion<PublicationPage>({ kind: "listPublications", query: { limit: 20 } }, access.cookie), readServerContent<ContentPublicationPage>({ kind: "listPublications", query: { limit: 1 } }, access.cookie)]); if (!selected.ok)
    return <QuestionState status={selected.status} code={selected.code}/>; if (!r.ok)
    return <QuestionState status={r.status} code={r.code}/>; if (!k.ok)
    return <QuestionState status={k.status} code={k.code}/>; return <PublicationPanel initial={r.data} knowledgeHead={k.data.head} selectedPublication={selected.data}/>; }
