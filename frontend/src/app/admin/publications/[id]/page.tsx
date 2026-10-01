import { contentPageAccess } from "@/features/content/page-access";
import { ContentState } from "@/features/content/content-state";
import { readServerContent } from "@/lib/content/server-client";
import type { DraftPage, DraftView, SubmissionPage, SubmissionView, PublicationPage, PublicationView } from "@/lib/content/types";
import { contentUUID } from "@/lib/content/schemas";
import { DraftList } from "@/features/content/draft-list";
import { DraftEditor } from "@/features/content/draft-editor";
import { SubmissionList } from "@/features/content/submission-list";
import { ReviewPanel } from "@/features/content/review-panel";
import { PublicationPanel } from "@/features/content/publication-panel";
import { WithdrawalPanel } from "@/features/content/withdrawal-panel";
export const dynamic = "force-dynamic";
export const metadata = { title: "Fixed publication" };
export default async function Page({ params }: {
    params: Promise<{
        id: string;
    }>;
}) { const { id } = await params; if (!contentUUID.test(id))
    return <ContentState status={404}/>; const access = await contentPageAccess(["admin"]); if ("error" in access)
    return access.error; const [list, view] = await Promise.all([readServerContent<PublicationPage>({ kind: "listPublications", query: { limit: 100 } }, access.cookie), readServerContent<PublicationView>({ kind: "readPublication", id }, access.cookie)]); if (!list.ok)
    return <ContentState status={list.status} code={list.code}/>; if (!view.ok)
    return <ContentState status={view.status} code={view.code}/>; const initial = { ...list.data, items: [view.data, ...list.data.items.filter(v => v.id !== id)] }; return <PublicationPanel key={id} initial={initial} selectedID={id}/>; }
